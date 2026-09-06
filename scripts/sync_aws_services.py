#!/usr/bin/env python3
"""
AWS Commander Service Synchronization Script.

This script synchronizes aws-commander configurations with botocore service
definitions and generates YAML configuration files that match the real AWS CLI
command names and response shapes.
"""

import argparse
import os
import shutil
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Optional

import yaml


class AWSServiceSync:
    """Synchronizes aws-commander with botocore service definitions."""

    TOKEN_MEMBER_NAMES = {
        "NextToken",
        "nextToken",
        "Marker",
        "NextMarker",
        "LastEvaluatedKey",
        "LastEvaluatedTableName",
        "ContinuationToken",
        "NextContinuationToken",
        "IsTruncated",
    }

    def __init__(self, aws_commander_path: str, verbose: bool = False):
        self.aws_commander_path = Path(aws_commander_path)
        self.config_dir = self.aws_commander_path / "configurations"
        self.verbose = verbose
        self.existing_configs = {}
        self.botocore: Any = None
        self.session: Any = None

    def log(self, message: str):
        if self.verbose:
            print(f"[INFO] {message}")

    def load_botocore(self) -> bool:
        try:
            import botocore  # type: ignore[import-not-found]
            import botocore.session  # type: ignore[import-not-found]

            self.botocore = botocore
            self.session = botocore.session.Session()
            return True
        except ImportError:
            print(
                "ERROR: botocore not installed. Create a virtualenv and install botocore + PyYAML."
            )
            return False

    def get_all_services(self) -> list[str]:
        available_services = []
        service_names = self.session.get_available_services()
        skip_services = {"s3control"}

        for service_name in service_names:
            if service_name not in skip_services:
                available_services.append(service_name)

        return sorted(available_services)

    def normalize_command_name(self, command_name: str) -> str:
        if command_name == command_name.lower() and "_" not in command_name:
            return command_name
        return self.botocore.xform_name(command_name, "-")

    def get_paginator_config(self, service_name: str, operation_name: str):
        try:
            paginator_model = self.session.get_paginator_model(service_name)
            return paginator_model.get_paginator(operation_name)
        except Exception:
            return None

    def has_required_input(self, operation_model) -> bool:
        if operation_model.input_shape is None:
            return False
        required_members = getattr(operation_model.input_shape, "required_members", [])
        return len(required_members) > 0

    def is_writer_operation(self, operation_name: str) -> bool:
        op_lower = operation_name.lower()
        return op_lower.startswith(
            (
                "create",
                "delete",
                "update",
                "put",
                "remove",
                "add",
                "tag",
                "untag",
                "start",
                "stop",
            )
        )

    def select_output_member(self, operation_model, paginator_config):
        if operation_model.output_shape is None:
            return None, None

        members = operation_model.output_shape.members
        if not members:
            return None, None

        if paginator_config is not None:
            result_key = paginator_config.get("result_key")
            if isinstance(result_key, list):
                result_key = result_key[0] if result_key else None
            if result_key in members:
                return result_key, members[result_key]

        candidate_names = [
            name for name in members if name not in self.TOKEN_MEMBER_NAMES
        ]

        for member_name in candidate_names:
            member = members[member_name]
            if member.type_name == "list":
                return member_name, member

        for member_name in candidate_names:
            member = members[member_name]
            if member.type_name in {"structure", "map"}:
                return member_name, member

        return None, None

    def determine_parse_type(self, output_member) -> str:
        if output_member.type_name == "list":
            item_shape = getattr(output_member, "member", None)
            if item_shape is not None and item_shape.type_name in {
                "string",
                "integer",
                "long",
                "float",
                "double",
                "boolean",
            }:
                return "list"
        return "object"

    def guess_resource_name(self, output_member_name: str, output_member) -> str:
        if output_member.type_name != "list":
            return ""

        item_shape = getattr(output_member, "member", None)
        if item_shape is None or item_shape.type_name != "string":
            return ""

        if output_member_name.endswith("Names"):
            return output_member_name[:-1]
        if output_member_name.endswith("Ids"):
            return output_member_name[:-1]
        return ""

    def build_pagination_config(self, paginator_config):
        if paginator_config is None:
            return None

        input_token = paginator_config.get("input_token")
        output_token = paginator_config.get("output_token")

        if isinstance(input_token, list):
            input_token = input_token[0] if input_token else None
        if isinstance(output_token, list):
            output_token = output_token[0] if output_token else None

        if not input_token:
            return None

        return {
            "enabled": True,
            "nextTokenParam": f"--{self.botocore.xform_name(input_token, '-')}",
            "nextTokenJsonPath": output_token or "",
        }

    def get_service_operations(self, service_name: str) -> list[dict]:
        operations = []

        try:
            service_model = self.session.get_service_model(service_name)

            for operation_name in service_model.operation_names:
                operation_model = service_model.operation_model(operation_name)
                paginator_config = self.get_paginator_config(
                    service_name, operation_name
                )
                output_member_name, output_member = self.select_output_member(
                    operation_model, paginator_config
                )

                if self.is_writer_operation(operation_name):
                    continue
                if self.has_required_input(operation_model):
                    continue
                if output_member_name is None or output_member is None:
                    continue

                operations.append(
                    {
                        "api_name": operation_name,
                        "cli_name": self.botocore.xform_name(operation_name, "-"),
                        "output_member_name": output_member_name,
                        "output_member": output_member,
                        "parse_type": self.determine_parse_type(output_member),
                        "pagination": self.build_pagination_config(paginator_config),
                        "resource_name": self.guess_resource_name(
                            output_member_name, output_member
                        ),
                    }
                )

        except Exception as error:
            self.log(f"Error getting operations for {service_name}: {error}")

        return operations

    def load_existing_configs(self) -> dict:
        configs = {}

        if not self.config_dir.exists():
            return configs

        for yaml_file in self.config_dir.glob("*.yaml"):
            service_name = yaml_file.stem
            try:
                with open(yaml_file, "r", encoding="utf-8") as file_handle:
                    configs[service_name] = yaml.safe_load(file_handle)
            except Exception as error:
                self.log(f"Error loading {yaml_file}: {error}")

        return configs

    def generate_yaml_template(self, service_name: str) -> Optional[dict]:
        operations = self.get_service_operations(service_name)
        if not operations:
            return None

        default_command = ""
        commands = []

        sorted_operations = sorted(
            operations,
            key=lambda operation: (
                not operation["cli_name"].startswith("list-"),
                not operation["cli_name"].startswith("describe-"),
                not operation["cli_name"].startswith("get-"),
                operation["cli_name"],
            ),
        )

        for operation in sorted_operations[:20]:
            command = {
                "name": operation["cli_name"],
                "arguments": [
                    "--output",
                    "json",
                    "--cli-read-timeout",
                    "10",
                    "--cli-connect-timeout",
                    "5",
                ],
                "view": "tableView",
                "showJsonViewer": True,
                "parse": {
                    "type": operation["parse_type"],
                    "attributeName": operation["output_member_name"],
                },
                "rerunOnBack": False,
                "favouritable": True,
            }

            if operation["resource_name"]:
                command["resourceName"] = operation["resource_name"]

            if operation["pagination"] is not None:
                command["pagination"] = operation["pagination"]

            if not default_command and operation["cli_name"].startswith(
                ("list-", "describe-", "get-")
            ):
                default_command = operation["cli_name"]

            commands.append(command)

        if not commands:
            return None

        return {
            "name": service_name,
            "defaultCommand": default_command,
            "commands": commands,
        }

    def detect_changes(self, service_name: str, generated: dict) -> dict:
        existing = self.existing_configs.get(service_name, {})
        changes = {
            "new": False,
            "modified": False,
            "new_commands": [],
            "removed_commands": [],
        }

        if not existing:
            changes["new"] = True
            return changes

        existing_commands = {
            command["name"] for command in existing.get("commands", [])
        }
        generated_commands = {
            command["name"] for command in generated.get("commands", [])
        }

        changes["new_commands"] = list(generated_commands - existing_commands)
        changes["removed_commands"] = list(existing_commands - generated_commands)

        if changes["new_commands"] or changes["removed_commands"]:
            changes["modified"] = True

        return changes

    def apply_changes(
        self, service_name: str, config: dict, backup: bool = False
    ) -> bool:
        try:
            output_file = self.config_dir / f"{service_name}.yaml"

            if backup and output_file.exists():
                backup_dir = (
                    self.config_dir.parent
                    / f"configurations.backup.{datetime.now().strftime('%Y%m%d_%H%M%S')}"
                )
                backup_dir.mkdir(exist_ok=True)
                shutil.copy2(output_file, backup_dir / f"{service_name}.yaml")
                self.log(f"Backed up {output_file} to {backup_dir}")

            with open(output_file, "w", encoding="utf-8") as file_handle:
                yaml.dump(
                    config, file_handle, default_flow_style=False, sort_keys=False
                )

            self.log(f"Written {output_file}")
            return True
        except Exception as error:
            print(f"ERROR: Failed to write {service_name}.yaml: {error}")
            return False

    def should_drop_legacy_generated_command(
        self, command: dict, generated_names: set[str]
    ) -> bool:
        command_name = command.get("name", "")
        normalized_name = self.normalize_command_name(command_name)
        return (
            command_name != normalized_name and normalized_name not in generated_names
        )

    def _merge_configs(self, existing: dict, generated: dict) -> dict:
        merged = existing.copy()
        merged["name"] = generated["name"]

        existing_commands = list(existing.get("commands", []))
        generated_commands = generated.get("commands", [])
        generated_names = {command["name"] for command in generated_commands}

        for generated_command in generated_commands:
            exact_match_index = next(
                (
                    index
                    for index, existing_command in enumerate(existing_commands)
                    if existing_command.get("name") == generated_command["name"]
                ),
                None,
            )
            if exact_match_index is not None:
                continue

            normalized_match_index = next(
                (
                    index
                    for index, existing_command in enumerate(existing_commands)
                    if existing_command.get("name") != generated_command["name"]
                    and self.normalize_command_name(existing_command.get("name", ""))
                    == generated_command["name"]
                ),
                None,
            )

            if normalized_match_index is not None:
                existing_commands[normalized_match_index] = generated_command
            else:
                existing_commands.append(generated_command)

        existing_commands = [
            command
            for command in existing_commands
            if not self.should_drop_legacy_generated_command(command, generated_names)
        ]

        merged["commands"] = existing_commands

        existing_default = merged.get("defaultCommand", "")
        generated_default = generated.get("defaultCommand", "")
        if (
            not existing_default
            or self.normalize_command_name(existing_default) == generated_default
        ):
            merged["defaultCommand"] = generated_default

        return merged

    def run(
        self,
        dry_run: bool = False,
        apply: bool = False,
        backup: bool = False,
        service: Optional[str] = None,
    ) -> int:
        if not self.load_botocore():
            return 1

        self.existing_configs = self.load_existing_configs()
        self.log(f"Loaded {len(self.existing_configs)} existing configurations")

        if service:
            services = [service]
        else:
            services = self.get_all_services()

        self.log(f"Found {len(services)} AWS services in botocore")

        total_new = 0
        total_modified = 0
        total_unchanged = 0

        report_lines = [
            "=" * 60,
            "AWS Commander Sync Report",
            f"Generated: {datetime.now().isoformat()}",
            "=" * 60,
            "",
        ]

        for service_name in services:
            generated = self.generate_yaml_template(service_name)
            if not generated:
                continue

            changes = self.detect_changes(service_name, generated)

            if changes["new"]:
                report_lines.append(
                    f"[NEW] {service_name}: {len(generated['commands'])} commands"
                )
                total_new += 1
                if apply:
                    self.apply_changes(service_name, generated, backup)
            elif changes["modified"]:
                report_lines.append(
                    f"[MODIFIED] {service_name}: +{len(changes['new_commands'])} new commands"
                )
                total_modified += 1
                if apply:
                    existing = self.existing_configs.get(service_name, {})
                    merged = self._merge_configs(existing, generated)
                    self.apply_changes(service_name, merged, backup)
            else:
                total_unchanged += 1

        report_lines.extend(
            [
                "",
                "-" * 60,
                f"Summary: {total_new} new, {total_modified} modified, {total_unchanged} unchanged",
                "",
            ]
        )

        print("\n".join(report_lines))

        if dry_run:
            print("\n[DRY RUN] No changes were applied.")

        return 0


def main():
    parser = argparse.ArgumentParser(
        description="Sync aws-commander configurations with botocore"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would change without making changes",
    )
    parser.add_argument(
        "--report",
        action="store_true",
        help="Generate a detailed report of differences",
    )
    parser.add_argument(
        "--apply", action="store_true", help="Apply changes to configurations"
    )
    parser.add_argument(
        "--backup", action="store_true", help="Create backup before applying changes"
    )
    parser.add_argument("--service", help="Only process specific service")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose output")
    parser.add_argument(
        "--aws-cli-path",
        default=os.environ.get("AWS_CLI_PATH", ""),
        help="Deprecated compatibility flag. botocore is used as the source of truth.",
    )
    parser.add_argument(
        "--aws-commander-path",
        default=os.environ.get(
            "AWS_COMMANDER_PATH",
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
        ),
        help="Path to aws-commander (default: parent of script)",
    )

    args = parser.parse_args()
    aws_commander_path = args.aws_commander_path

    if not os.path.isdir(aws_commander_path):
        print(f"ERROR: aws-commander path not found: {aws_commander_path}")
        print("Set AWS_COMMANDER_PATH environment variable or use --aws-commander-path")
        return 1

    if not os.path.isdir(os.path.join(aws_commander_path, "configurations")):
        print(f"ERROR: configurations directory not found in {aws_commander_path}")
        return 1

    sync = AWSServiceSync(aws_commander_path, args.verbose)
    return sync.run(
        dry_run=args.dry_run, apply=args.apply, backup=args.backup, service=args.service
    )


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""
AWS Commander Service Synchronization Script

This script synchronizes aws-commander configurations with the AWS CLI source.
It reads service definitions from botocore and generates YAML configuration files.

Usage:
    python sync_aws_services.py [options]

Options:
    --dry-run          Show what would change without making changes
    --report           Generate a detailed report of differences
    --apply            Apply changes to configurations
    --backup           Create backup before applying changes
    --service SERVICE  Only process specific service
    --verbose          Enable verbose output
"""

import argparse
import json
import os
import shutil
import sys
import glob
from datetime import datetime
from pathlib import Path
from typing import Any, Optional

import yaml


class AWSServiceSync:
    """Synchronizes aws-commander with AWS CLI service definitions."""

    # Common result attribute names for list operations
    LIST_ATTRIBUTES = {
        'tables': 'TableNames',
        'buckets': 'Buckets',
        'instances': 'Instances',
        'volumes': 'Volumes',
        'vpcs': 'Vpcs',
        'subnets': 'Subnets',
        'security-groups': 'SecurityGroups',
        'queues': 'Queues',
        'topics': 'Topics',
        'functions': 'Functions',
        'logs': 'logEvents',
    }

    def __init__(self, aws_cli_path: str, aws_commander_path: str, verbose: bool = False):
        self.aws_cli_path = Path(aws_cli_path)
        self.aws_commander_path = Path(aws_commander_path)
        self.config_dir = self.aws_commander_path / 'configurations'
        self.verbose = verbose
        self.services = {}
        self.existing_configs = {}
        self.botocore = None
        
    def log(self, message: str):
        """Print message if verbose mode is enabled."""
        if self.verbose:
            print(f"[INFO] {message}")

    def load_botocore(self) -> bool:
        """Load botocore to get service definitions."""
        try:
            import botocore
            import botocore.session
            self.botocore = botocore
            self.session = botocore.session.Session()
            return True
        except ImportError:
            print("ERROR: botocore not installed. Run: pip install botocore")
            return False

    def get_all_services(self) -> list[str]:
        """Get list of all available AWS services from botocore."""
        available_services = []
        
        # Get services from botocore's service catalog
        service_names = self.session.get_available_services()
        
        # Filter out internal/non-CLI services
        skip_services = {'s3control'}  # s3control is covered by s3api
        
        for svc in service_names:
            if svc not in skip_services:
                available_services.append(svc)
        
        return sorted(available_services)

    def get_service_operations(self, service_name: str) -> list[dict]:
        """Get operations for a specific service."""
        operations = []
        
        try:
            # Get the service metadata
            service_model = self.session.get_service_model(service_name)
            
            # Get operation names
            for op_name in service_model.operation_names:
                op_model = service_model.operation_model(op_name)
                
                # Determine if this is a writer operation (create, delete, update)
                # Check both 'writer' attribute and operation type
                is_writer = False
                try:
                    if hasattr(op_model, 'writer') and op_model.writer is not None:
                        is_writer = True
                except Exception:
                    pass
                    
                # Alternative: check if operation name suggests it's a write operation
                op_lower = op_name.lower()
                if op_lower.startswith(('create', 'delete', 'update', 'put', 'remove', 'add')):
                    is_writer = True
                
                operations.append({
                    'name': op_name,
                    'name_camel': self._camel_to_snake(op_name),
                    'is_writer': is_writer,
                    'has_paginator': self._has_paginator(service_name, op_name),
                })
                
        except Exception as e:
            self.log(f"Error getting operations for {service_name}: {e}")
            
        return operations

    def _camel_to_snake(self, name: str) -> str:
        """Convert CamelCase to snake_case."""
        import re
        s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', name)
        return re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()

    def _has_paginator(self, service_name: str, operation_name: str) -> bool:
        """Check if operation has pagination."""
        try:
            paginators = self.session.get_paginator_models(service_name)
            return operation_name in paginators
        except Exception:
            return False

    def classify_operation(self, operation_name: str) -> dict:
        """Determine view type, parse config based on operation name."""
        op_lower = operation_name.lower()
        
        # Default classification
        classification = {
            'view': 'tableView',
            'showJsonViewer': True,
            'parse_type': 'object',
            'parse_attribute': '',
            'rerunOnBack': False,
            'favouritable': True,
        }
        
        # List operations
        if op_lower.startswith('list') or op_lower.startswith('describe') or op_lower.startswith('get'):
            classification['view'] = 'tableView'
            classification['parse_type'] = 'object'
        # Write operations
        elif op_lower.startswith('create') or op_lower.startswith('delete') or op_lower.startswith('update') or op_lower.startswith('put'):
            classification['view'] = 'contentView'
            classification['parse_type'] = 'object'
        # Other operations
        else:
            classification['view'] = 'contentView'
            classification['parse_type'] = 'object'
            
        return classification

    def get_default_attribute(self, service_name: str, operation_name: str) -> str:
        """Guess the result attribute name based on service and operation."""
        op_lower = operation_name.lower()
        
        # Common patterns
        if 'list' in op_lower:
            # Try to extract the resource name
            parts = op_lower.replace('list_', '').replace('list', '').strip('_')
            if parts:
                # Try singular form
                if parts.endswith('s'):
                    return parts[:-1]  # Remove trailing 's'
                return parts + 's'  # Add 's'
            return operation_name.replace('list-', '').replace('list', '') + 's'
        
        if 'describe' in op_lower or 'get' in op_lower:
            return service_name.title()
            
        return ''

    def load_existing_configs(self) -> dict:
        """Load existing YAML configurations."""
        configs = {}
        
        if not self.config_dir.exists():
            return configs
            
        for yaml_file in self.config_dir.glob('*.yaml'):
            service_name = yaml_file.stem
            try:
                with open(yaml_file, 'r') as f:
                    configs[service_name] = yaml.safe_load(f)
            except Exception as e:
                self.log(f"Error loading {yaml_file}: {e}")
                
        return configs

    def generate_yaml_template(self, service_name: str) -> dict:
        """Generate YAML configuration for a service."""
        operations = self.get_service_operations(service_name)
        
        if not operations:
            return None
            
        # Find a good default command (first list or describe operation)
        default_command = ''
        commands = []
        
        # Sort operations: list/describe first, then others
        sorted_ops = sorted(operations, key=lambda x: (
            not x['name'].lower().startswith('list'),
            not x['name'].lower().startswith('describe'),
            not x['name'].lower().startswith('get'),
            x['name']
        ))
        
        # Limit to first 20 operations to avoid overwhelming configs
        # These are typically the most useful ones
        for op in sorted_ops[:20]:
            classification = self.classify_operation(op['name'])
            
            # Skip writer operations (create, delete, update) for auto-generation
            # These typically require input and return status
            if classification['view'] == 'contentView' and op['is_writer']:
                continue
                
            cmd = {
                'name': op['name'],
                'arguments': [
                    '--output', 'json',
                    '--cli-read-timeout', '10',
                    '--cli-connect-timeout', '5',
                ],
                'view': classification['view'],
                'showJsonViewer': classification['showJsonViewer'],
                'parse': {
                    'type': classification['parse_type'],
                    'attributeName': self.get_default_attribute(service_name, op['name']),
                },
                'rerunOnBack': classification['rerunOnBack'],
                'favouritable': classification['favouritable'],
            }
            
            # Add pagination if available
            if op['has_paginator']:
                cmd['pagination'] = {
                    'enabled': True,
                    'nextTokenParam': '--starting-token',
                    'nextTokenJsonPath': 'NextToken',
                }
                # Remove --no-paginate from arguments
                cmd['arguments'] = [a for a in cmd['arguments'] if a != '--no-paginate']
            
            # Set default command
            if not default_command and (op['name'].startswith('list') or op['name'].startswith('describe')):
                default_command = op['name']
                # Add resourceName for list operations
                if op['name'].startswith('list'):
                    resource_name = op['name'].replace('list-', '').replace('list', '').strip('-')
                    if resource_name:
                        cmd['resourceName'] = resource_name.title()
            
            commands.append(cmd)
            
        if not commands:
            return None
            
        config = {
            'name': service_name,
            'defaultCommand': default_command,
            'commands': commands,
        }
        
        return config

    def detect_changes(self, service_name: str, generated: dict) -> dict:
        """Compare existing and generated configs."""
        existing = self.existing_configs.get(service_name, {})
        
        changes = {
            'new': False,
            'modified': False,
            'new_commands': [],
            'removed_commands': [],
        }
        
        if not existing:
            changes['new'] = True
            return changes
            
        existing_commands = {cmd['name'] for cmd in existing.get('commands', [])}
        generated_commands = {cmd['name'] for cmd in generated.get('commands', [])}
        
        changes['new_commands'] = list(generated_commands - existing_commands)
        changes['removed_commands'] = list(existing_commands - generated_commands)
        
        if changes['new_commands'] or changes['removed_commands']:
            changes['modified'] = True
            
        return changes

    def apply_changes(self, service_name: str, config: dict, backup: bool = False) -> bool:
        """Write configuration file."""
        try:
            output_file = self.config_dir / f'{service_name}.yaml'
            
            # Create backup if requested
            if backup and output_file.exists():
                backup_dir = self.config_dir.parent / f'configurations.backup.{datetime.now().strftime("%Y%m%d_%H%M%S")}'
                backup_dir.mkdir(exist_ok=True)
                shutil.copy2(output_file, backup_dir / f'{service_name}.yaml')
                self.log(f"Backed up {output_file} to {backup_dir}")
            
            # Write new config
            with open(output_file, 'w') as f:
                yaml.dump(config, f, default_flow_style=False, sort_keys=False)
                
            self.log(f"Written {output_file}")
            return True
            
        except Exception as e:
            print(f"ERROR: Failed to write {service_name}.yaml: {e}")
            return False

    def run(self, dry_run: bool = False, apply: bool = False, 
            backup: bool = False, service: Optional[str] = None) -> int:
        """Run the synchronization."""
        
        # Load botocore
        if not self.load_botocore():
            return 1
            
        # Load existing configs
        self.existing_configs = self.load_existing_configs()
        self.log(f"Loaded {len(self.existing_configs)} existing configurations")
        
        # Get services to process
        if service:
            services = [service]
        else:
            services = self.get_all_services()
            
        self.log(f"Found {len(services)} AWS services in botocore")
        
        # Track changes
        total_new = 0
        total_modified = 0
        total_unchanged = 0
        
        report_lines = []
        report_lines.append("=" * 60)
        report_lines.append("AWS Commander Sync Report")
        report_lines.append(f"Generated: {datetime.now().isoformat()}")
        report_lines.append("=" * 60)
        report_lines.append("")
        
        for svc in services:
            # Skip if service already exists and we're only adding new
            # But we still check for new commands
            
            generated = self.generate_yaml_template(svc)
            if not generated:
                continue
                
            changes = self.detect_changes(svc, generated)
            
            if changes['new']:
                report_lines.append(f"[NEW] {svc}: {len(generated['commands'])} commands")
                total_new += 1
                
                if apply:
                    self.apply_changes(svc, generated, backup)
                    
            elif changes['modified']:
                new_cmds = len(changes['new_commands'])
                report_lines.append(f"[MODIFIED] {svc}: +{new_cmds} new commands")
                total_modified += 1
                
                if apply:
                    # Merge with existing - keep customizations
                    existing = self.existing_configs.get(svc, {})
                    merged = self._merge_configs(existing, generated)
                    self.apply_changes(svc, merged, backup)
                    
            else:
                total_unchanged += 1
                
        report_lines.append("")
        report_lines.append("-" * 60)
        report_lines.append(f"Summary: {total_new} new, {total_modified} modified, {total_unchanged} unchanged")
        report_lines.append("")
        
        # Print report
        report = '\n'.join(report_lines)
        print(report)
        
        if dry_run:
            print("\n[DRY RUN] No changes were applied.")
            
        return 0

    def _merge_configs(self, existing: dict, generated: dict) -> dict:
        """Merge existing config with generated, preserving customizations."""
        merged = existing.copy()
        
        existing_commands = {cmd['name']: cmd for cmd in existing.get('commands', [])}
        
        for cmd in generated.get('commands', []):
            if cmd['name'] not in existing_commands:
                # Add new command
                existing_commands[cmd['name']] = cmd
                
        merged['commands'] = list(existing_commands.values())
        
        return merged


def main():
    parser = argparse.ArgumentParser(
        description='Sync aws-commander configurations with AWS CLI'
    )
    parser.add_argument(
        '--dry-run', 
        action='store_true',
        help='Show what would change without making changes'
    )
    parser.add_argument(
        '--report',
        action='store_true', 
        help='Generate a detailed report of differences'
    )
    parser.add_argument(
        '--apply',
        action='store_true',
        help='Apply changes to configurations'
    )
    parser.add_argument(
        '--backup',
        action='store_true',
        help='Create backup before applying changes'
    )
    parser.add_argument(
        '--service',
        help='Only process specific service'
    )
    parser.add_argument(
        '--verbose',
        action='store_true',
        help='Enable verbose output'
    )
    parser.add_argument(
        '--aws-cli-path',
        default=os.environ.get('AWS_CLI_PATH', os.path.expanduser('~/code/aws-cli')),
        help='Path to AWS CLI source (default: ~/code/aws-cli)'
    )
    parser.add_argument(
        '--aws-commander-path',
        default=os.environ.get('AWS_COMMANDER_PATH', os.path.dirname(os.path.dirname(os.path.abspath(__file__)))),
        help='Path to aws-commander (default: parent of script)'
    )
    
    args = parser.parse_args()
    
    # Validate paths
    aws_cli_path = args.aws_cli_path
    aws_commander_path = args.aws_commander_path
    
    if not os.path.isdir(aws_cli_path):
        print(f"ERROR: AWS CLI path not found: {aws_cli_path}")
        print("Set AWS_CLI_PATH environment variable or use --aws-cli-path")
        return 1
        
    if not os.path.isdir(aws_commander_path):
        print(f"ERROR: aws-commander path not found: {aws_commander_path}")
        print("Set AWS_COMMANDER_PATH environment variable or use --aws-commander-path")
        return 1
        
    if not os.path.isdir(os.path.join(aws_commander_path, 'configurations')):
        print(f"ERROR: configurations directory not found in {aws_commander_path}")
        return 1
        
    sync = AWSServiceSync(aws_cli_path, aws_commander_path, args.verbose)
    
    return sync.run(
        dry_run=args.dry_run,
        apply=args.apply,
        backup=args.backup,
        service=args.service
    )


if __name__ == '__main__':
    sys.exit(main())

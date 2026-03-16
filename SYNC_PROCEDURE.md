# AWS Commander Sync Plan

## Overview

This document describes the procedure for synchronizing aws-commander with the AWS CLI source code to maintain feature parity with all AWS services.

## Implemented Components

### Scripts Created

1. **`scripts/sync_aws_services.py`** - Python script that:
   - Reads AWS service definitions from botocore
   - Generates YAML configuration templates
   - Detects changes between existing and new configs
   - Supports dry-run, report, and apply modes
   - Creates backups before applying changes

2. **`scripts/sync-aws-services.sh`** - Bash wrapper that:
   - Provides convenient CLI interface
   - Sets up environment variables
   - Installs botocore if missing
   - Validates paths and options
   - Supports all Python script options

## Background

- **aws-commander**: Go TUI application wrapping AWS CLI
- **AWS CLI source**: Python application at https://github.com/aws/aws-cli
- **Current coverage**: 14 services (cloudfront, cloudtrail, dynamodb, ec2, ecs, eks, events, iam, lambda, logs, s3api, sns, sqs, stepfunctions)
- **Total AWS services**: ~190 services
- **Gap**: ~176 services not yet covered

## Directory Structure

```
$AWS_COMMANDER_PATH/
├── configurations/          # YAML configs for each service
│   ├── dynamodb.yaml
│   ├── s3api.yaml
│   └── ...
├── scripts/                # Sync scripts (to be created)
│   └── sync-aws-services.sh
│   └── sync_aws_services.py
└── SYNC_PROCEDURE.md       # This document
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `AWS_CLI_PATH` | Path to cloned aws-cli repository | Yes | `$HOME/code/aws-cli` |
| `AWS_COMMANDER_PATH` | Path to aws-commander repository | Yes | Parent of this file |
| `BOTOCORE_VERSION` | Specific botocore version to use | No | Latest installed |

## Phase 1: Analysis

### 1.1 Service Discovery

The AWS CLI uses botocore for service definitions. Each service has:
- Service name (e.g., `dynamodb`, `s3api`)
- Operations (e.g., `list-tables`, `describe-table`)
- Paginators (for pagination support)
- Waiters (for async operations)

### 1.2 Operation Classification

Operations are classified by name patterns:

| Pattern | View Type | Description |
|---------|-----------|-------------|
| `list-*` | tableView | Returns collection of items |
| `describe-*` | tableView | Returns description of resources |
| `get-*` | tableView/contentView | Returns single or multiple items |
| `create-*` | contentView | Creates resource, returns result |
| `delete-*` | contentView | Deletes resource, returns result |
| `update-*` | contentView | Updates resource, returns result |

### 1.3 Result Parsing

For each operation, determine:
- **type**: `list` or `object`
- **attributeName**: JSON key containing results

Common patterns:
- `list-tables` → `TableNames` (list)
- `describe-instances` → `Reservations[].Instances[]` (object with nested)
- `list-objects` → `Contents` (object)

## Phase 2: Sync Script Implementation

### 2.1 Script: `scripts/sync-aws-services.sh`

Bash wrapper script providing:
- Environment setup
- Dry-run mode
- Report generation
- Apply mode with backup

### 2.2 Script: `scripts/sync_aws_services.py`

Python script that:
1. Loads botocore service definitions
2. Generates YAML config templates
3. Compares with existing configs
4. Outputs diff report or applies changes

#### Key Functions

```python
def get_all_services() -> List[str]:
    """Get list of all available AWS services from botocore."""

def get_service_operations(service_name: str) -> List[dict]:
    """Get operations for a specific service with metadata."""

def get_service_paginators(service_name: str) -> dict:
    """Get pagination configuration for a service."""

def classify_operation(operation_name: str) -> dict:
    """Determine view type, parse config based on operation name."""

def generate_yaml_template(service_name: str) -> str:
    """Generate YAML configuration for a service."""

def detect_changes(existing: dict, generated: dict) -> dict:
    """Compare existing and generated configs."""

def apply_changes(configs: dict, backup: bool = True) -> None:
    """Write new/updated configuration files."""
```

## Phase 3: Sync Procedure

### 3.1 Manual Sync Workflow

```bash
# Step 1: Ensure AWS CLI source is up to date
cd $AWS_CLI_PATH
git pull origin develop

# Step 2: Navigate to aws-commander
cd $AWS_COMMANDER_PATH

# Step 3: Dry-run to see what would change
AWS_CLI_PATH=$HOME/code/aws-cli \
AWS_COMMANDER_PATH=$HOME/code/aws-commander \
./scripts/sync-aws-services.sh --dry-run

# Step 4: Review the report
./scripts/sync-aws-services.sh --report

# Step 5: Apply changes with backup
AWS_CLI_PATH=$HOME/code/aws-cli \
AWS_COMMANDER_PATH=$HOME/code/aws-commander \
./scripts/sync-aws-services.sh --apply --backup

# Step 6: Verify the application builds
go build ./...
```

### 3.2 Automated Sync (Optional)

For CI/CD automation:
```bash
# Run weekly on schedule
0 0 * * 0 AWS_CLI_PATH=$HOME/code/aws-cli \
    AWS_COMMANDER_PATH=$HOME/code/aws-commander \
    $AWS_COMMANDER_PATH/scripts/sync-aws-services.sh --apply --backup
```

## Phase 4: Configuration Generation Rules

### 4.1 Default Arguments

All commands include:
```yaml
arguments:
  - "--output"
  - "json"
  - "--cli-read-timeout"
  - "10"
  - "--cli-connect-timeout"
  - "5"
```

### 4.2 Service Default Command

First `list-*` or `describe-*` operation becomes `defaultCommand`.

### 4.3 Resource Selection

For operations requiring resource ID:
- Use `resourceName` field
- Depends on parent operation

### 4.4 Pagination

Enable pagination if:
- Service has paginators defined
- Common token fields: `NextToken`, `LastEvaluatedKey`, `Marker`

## Phase 5: Safety Features

### 5.1 Backup

Before applying changes:
```bash
# Create timestamped backup
backup_dir="configurations.backup.$(date +%Y%m%d_%H%M%S)"
cp -r configurations "$backup_dir"
```

### 5.2 Dry-Run

Always show what would change without modifying files.

### 5.3 Validation

After sync, validate:
```bash
go build ./...
go vet ./...
go test ./...
```

## Phase 6: Handling Special Cases

### 6.1 Existing Custom Configurations

Some services have custom UIs (DynamoDB, S3):
- **Do not overwrite** existing complex configs
- Only add missing commands
- Use `--merge` flag to combine

### 6.2 Deprecated Services

AWS CLI may have deprecated services:
- Keep existing config if service removed from CLI
- Mark as deprecated in comments

### 6.3 New Services

New services from AWS:
- Generate base config automatically
- Human review recommended before use

## Maintenance

### 7.1 Troubleshooting

| Issue | Solution |
|-------|----------|
| Python import errors | Ensure botocore is installed: `pip install botocore` |
| Missing services | Update botocore: `pip install --upgrade botocore` |
| Invalid YAML | Run linter: `yamllint configurations/` |
| Build failures | Run: `go vet ./...` and fix issues |

### 7.2 Common Commands

```bash
# Full sync with backup
./scripts/sync-aws-services.sh --apply --backup

# Dry-run only
./scripts/sync-aws-services.sh --dry-run

# Generate report
./scripts/sync-aws-services.sh --report

# Restore from backup
cp -r configurations.backup.*/configurations/* configurations/
```

## Appendix: Service Coverage Status

### Currently Covered (14 services)
- cloudfront
- cloudtrail
- dynamodb
- ec2
- ecs
- eks
- events
- iam
- lambda
- logs
- s3api
- sns
- sqs
- stepfunctions

### Not Yet Covered (~176 services)
accessanalyzer, acm, acm-pca, apigateway, apigatewayv2, appconfig,
application-autoscaling, application-signals, appmesh, apprunner, athena,
autoscaling, autoscaling-plans, backup, batch, budgets, ce, chime, cloud9,
cloudcontrol, cloudformation, cloudsearchdomain, cognito-identity,
cognito-idp, comprehend, comprehendmedical, configservice, connect, cur,
datapipeline, datasync, dax, deploy, detective, devicefarm,
directconnect, discovery, dms, docdb, ds, ds-data, dynamodbstreams,
ec2-instance-connect, ecr, ecr-public, efs, elasticache,
elasticbeanstalk, elb, elbv2, emr, emr-containers, es, firehose, fis, fms,
gamelift, glacier, globalaccelerator, glue, grafana, greengrass,
greengrassv2, guardduty, health, healthlake, iam, imagebuilder,
importexport, inspector, inspector2, iot, iot-data, iotdevice advisor,
iotevents, iotevents-data, iotsitewise, iotthingsgraph, iotwireless, ivs,
ivs-realtime, ivschat, kafka, kendra, kinesis, kms, lakeformation,
license-manager, lightsail, macie2, mediaconnect, medialive, mediamatic,
mediapackage, mediapackage-vod, mediastore, mediastore-data, medical-imaging,
memorydb, networkflowmonitor, networkmanager, networkmonitor, oam,
observabilityadmin, omics, organizations, outposts, payment-cryptography,
payment-cryptography-data, pi, pinpoint, pipes, polly, pricing, proton,
ram, rds, rds-data, redshift, rekognition, resource-explorer-2,
resource-groups, resourcegroupstaggingapi, route53, route53domains,
route53profiles, route53resolver, s3, s3control, secretsmanager,
securityhub, securitylake, serverlessrepo, service-quotas, servicecatalog,
servicecatalog-appregistry, servicediscovery, ses, shield, signer,
snowball, ssm, ssm-contacts, ssm-incidents, storagegateway, sts, support,
swf, synthetics, textract, transcribe, translate, trustedadvisor,
verifiedpermissions, vpc-lattice, waf, waf-regional, wafv2, workdocs,
workmail, workmailmessageflow, workspaces, xray

---

Document Version: 1.0
Last Updated: 2026-03-16

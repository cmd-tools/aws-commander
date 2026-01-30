# AWS Commander

A terminal-based user interface (TUI) for interacting with AWS services. AWS Commander provides an intuitive, keyboard-driven interface for browsing and managing AWS resources without leaving your terminal.

## Features

- **Interactive TUI**: Navigate AWS resources using keyboard shortcuts
- **Multiple AWS Service Support**: 
  - DynamoDB (tables, scan, query with support for GSI/LSI)
  - S3 (buckets, objects, folder navigation)
  - SQS (queues, messages)
  - And more...
- **DynamoDB Query Builder**: Interactive form-based query builder with automatic key detection
- **Smart JSON Viewer**: 
  - Tree-based JSON visualization
  - Toggle between DynamoDB format and regular JSON (press 'n')
  - Support for nested JSON parsing
  - Base64 gzip decompression
- **Reserved Word Handling**: Automatic handling of DynamoDB reserved words in queries
- **Pagination Support**: Navigate through large result sets with next/previous page
- **Profile Management**: Switch between AWS profiles
- **Search**: Quick search across results (press ':')
- **Copy to Clipboard**: Copy data with 'y' key

## Table of contents
1. [What the Project Does](#what-the-project-does)
2. [Key Bindings](#key-bindings)
3. [Development](#development)
   1. [Prerequisites](#prerequisites)
   1. [Getting Started](#getting-started)
   1. [Running the Application](#running-the-application)
4. [Versioning](#versioning)

## What the Project Does

AWS Commander is a terminal UI that wraps the AWS CLI, providing:

1. **Easy AWS Resource Navigation**: Browse services → resources → details without remembering CLI commands
2. **DynamoDB Query Interface**: 
   - Automatically detects table keys (PK/SK) from table schema
   - Builds proper query expressions with expression attribute names/values
   - Handles DynamoDB reserved words (like STATUS, DATA, NAME, etc.)
   - Supports querying Global and Local Secondary Indexes
3. **Smart JSON Inspection**:
   - View DynamoDB items in both DynamoDB JSON format (`{"S": "value"}`) and regular JSON format
   - Toggle between formats with the 'n' key
   - Expand stringified JSON fields
   - Decompress base64-gzipped data
4. **S3 Navigation**: Browse buckets and folders like a file system
5. **Result Caching**: Fast navigation with intelligent result caching

## Key Bindings

| Key | Context | Description |
|-----|---------|-------------|
| `ESC` | Global | Go back / Navigate up |
| `:` | Global | Open search bar |
| `n` | Table view | Next page (pagination) |
| `p` | Table view | Previous page (pagination) |
| `v` | JSON viewer | Toggle DynamoDB/Normal JSON format |
| `y` | Any view | Copy (yank) current selection to clipboard |
| `Ctrl+C` | Any view | Copy current selection to clipboard |
| `Enter` | Table view | View item details or navigate into selection |
| `Enter` | JSON viewer | Expand stringified JSON or decompress gzip |
| `?` | Global | Show help |

## Development

### Prerequisites

This project requires:
* [AWS cli](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html).
* `Makefile` 
  * `Windows` ([MinGW](http://www.mingw.org/) or [Cygwin](https://www.cygwin.com/));
  * `Linux`: `apt install make`;
  * `Mac`: `brew install make`.
* [Docker](https://docs.docker.com/engine/install/);
* [Go](https://go.dev/doc/install).

### Getting Started

1. **Setup local development environment**:
   ```bash
   make up
   ```
   This starts a LocalStack environment with pre-configured AWS resources (DynamoDB tables, S3 buckets, SQS queues) for testing.

2. **Build the application**:
   ```bash
   go build -o aws-commander .
   ```

3. **Run the application**:
   ```bash
   ./aws-commander
   ```
   
   Or run directly with Go:
   ```bash
   go run .
   ```

4. **Enable logging** (optional):
   - Add `--logview` flag to show logs in the application
   - Or tail the log file: `tail -f aws-commander.log`

### Running the Application

#### With LocalStack (Development)
When using LocalStack, AWS Commander automatically uses the `localstack` profile configured in the project.

```bash
# Start LocalStack
make up

# Run AWS Commander
./aws-commander
```

#### With Real AWS Account
Ensure you have AWS CLI configured with valid credentials:

```bash
# Configure AWS CLI (if not already done)
aws configure

# Run AWS Commander
./aws-commander
```

The application will prompt you to select an AWS profile from your `~/.aws/credentials` file.

#### Navigation Flow Example
1. Start the application
2. Select a profile (e.g., `localstack`, `default`, or your custom profile)
3. Select a service (e.g., `dynamodb`, `s3`, `sqs`)
4. Select a resource (e.g., table name, bucket name)
5. Select an action (e.g., `scan`, `query`, `list-objects`)
6. View results in table format
7. Press `Enter` on a row to view JSON details
8. Press `v` in JSON view to toggle between DynamoDB and regular JSON format
9. Press `ESC` to go back
10. Press `:` to search within results

## Versioning

We use [SemVer](http://semver.org/) for versioning.

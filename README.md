# AWS Commander

A terminal-based user interface (TUI) for interacting with AWS services. AWS Commander provides an intuitive, keyboard-driven interface for browsing and managing AWS resources without leaving your terminal.

## Features

- **Interactive TUI**: Navigate AWS resources using keyboard shortcuts
- **Multiple AWS Service Support**: 
  - DynamoDB (tables, scan, query with support for GSI/LSI)
  - S3API (buckets, objects)
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
1. [Installation](#installation)
2. [What the Project Does](#what-the-project-does)
3. [Key Bindings](#key-bindings)
4. [Development](#development)
   1. [Prerequisites](#prerequisites)
   2. [Getting Started](#getting-started)
   3. [Running the Application](#running-the-application)
   4. [Project Structure](#project-structure)
   5. [Build and Test](#build-and-test)
5. [Versioning](#versioning)

## Installation

### Homebrew

```bash
brew tap cmd-tools/homebrew-tap
brew install aws-commander
```

### From source

```bash
go install github.com/cmd-tools/aws-commander@latest
```

### Manual download

Download the latest binary for your platform from the [Releases](https://github.com/cmd-tools/aws-commander/releases) page.

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
4. **S3API Navigation**: Browse buckets and objects
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
* [Go](https://go.dev/doc/install) 1.24.0 or later
* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
* [Docker](https://docs.docker.com/engine/install/) (optional, for LocalStack)
* `make` (optional, for LocalStack)
  * `Windows`: [MinGW](http://www.mingw.org/) or [Cygwin](https://www.cygwin.com/)
  * `Linux`: `apt install make`
  * `Mac`: `brew install make`

### Getting Started

1. **Clone the repository**:
   ```bash
   git clone https://github.com/cmd-tools/aws-commander.git
   cd aws-commander
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the application**:
   ```bash
   go build -o aws-commander .
   ```

4. **Run the application**:
   ```bash
   ./aws-commander
   ```
   
   Or run directly with Go:
   ```bash
   go run .
   ```

5. **Enable logging** (optional):
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

# Stop LocalStack
make down
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
3. Select a service (e.g., `dynamodb`, `s3api`, `sqs`)
4. Select a resource (e.g., table name, bucket name)
5. Select an action (e.g., `scan`, `query`, `list-objects`)
6. View results in table format
7. Press `Enter` on a row to view JSON details
8. Press `v` in JSON view to toggle between DynamoDB and regular JSON format
9. Press `ESC` to go back
10. Press `:` to search within results

### Project Structure

| Package | Purpose |
|---------|---------|
| `main` (root) | Entry point, handlers, views, navigation, search |
| `cmd/` | Domain types, config loading, UI state |
| `cmd/profile/` | AWS profile discovery and SSO handling |
| `parser/` | Command output parsing and content view creation |
| `ui/` | Reusable TUI components (table, list, tree, modal, toast) |
| `executor/` | Shell command execution wrapper |
| `logger/` | Logging singleton |
| `constants/` | Shared string constants |
| `helpers/` | String utilities, AWS version detection |
| `configurations/` | YAML service definitions for AWS commands |

### Build and Test

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run a specific test
go test ./parser/ -run Test_ParseCommand_Object

# Static analysis
go vet ./...

# Format check
gofmt -l .
```

## Versioning

We use [SemVer](http://semver.org/) for versioning.

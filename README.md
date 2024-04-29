# AWS Commander

## Table of contents
1. [Development](#development)
   1. [Prerequisites](#prerequisites)
   1. [Getting Started](#getting-started)
2. [Versioning](#versioning)

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

1. Run the command `make up` to setup and startup local development environment using `localstack`;
2. Execute the application using `go run main.go`. 
3. Add `--logview` to show logs directly in `aws-commander` or check development logs by tailing the related log file `aws-commander.log`.

## Versioning

We use [SemVer](http://semver.org/) for versioning.

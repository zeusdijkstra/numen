# Numen – Advanced File Processing Utility

Numen is a fast, reliable, and developer-friendly file processing tool written in Go. It’s built for anyone who needs efficient file management with powerful filtering, batch operations, and dependable error handling — from everyday users to enterprise environments.

## Features

### Core Functionality

* **Smart File Listing**: Recursively scans directories with flexible, intelligent filters.
* **Batch File Operations**: Safely delete files with full logging and rollback-ready workflows.
* **Advanced Filtering**: Filter by extension, size, custom patterns, or mix and match.
* **Flexible Logging**: Output logs to both stdout and file, using structured, readable formatting.
* **Performance Optimized**: Designed for speed and low memory use, even with huge file sets.

## Installation

### Prerequisites

* Go 1.25.3 or newer
* Git

### Quick Install

```bash
git clone https://github.com/yourusername/numen.git
cd numen

# Build the binary
go build -ldflags="-s -w" -o numen .

# Optional: install globally
sudo mv numen /usr/local/bin/
```

### Development Install

```bash
go mod tidy
go mod download

# Verify everything works
go test -v ./...
```

## Usage Guide

### Basic Syntax

```bash
numen -root <directory> [options]
```

### Command Line Options

| Option  | Type   | Required | Default  | Description                       |
| ------- | ------ | -------- | -------- | --------------------------------- |
| `-root` | string | Yes    | –        | Directory to process              |
| `-list` | bool   | No     | `false`  | List files (default behavior)     |
| `-ext`  | string | No     | `""`     | Space-separated extension filter  |
| `-min`  | int64  | No     | `0`      | Minimum file size (bytes)         |
| `-del`  | string | No     | `""`     | Files to delete (space-separated) |
| `-log`  | string | No     | `stdout` | Log file path                     |

### Examples

#### Basic File Listing

```bash
numen -root /home/user/documents -list
numen -root /var/log -list -log /tmp/file_scan.log
```

#### Filtering

```bash
numen -root /src -list -ext ".go .js .py .md"
numen -root /downloads -list -min 1048576
numen -root /projects -list -ext ".go .rs" -min 1024
```

#### Safe Deletion

```bash
numen -root /tmp -del "temp1.txt temp2.tmp" -log /tmp/cleanup.log
numen -root /logs -del "*.log *.tmp" -log cleanup.log
```

#### Common Workflows

```bash
numen -root /var/log/app -del "*.log.1 *.log.2" \
  -log /var/log/cleanup/$(date +%Y%m%d).log

numen -root ./build -del "*.o *.exe" -log build_cleanup.log

numen -root /home -list -min 104857600 -log large_files_audit.log
```

## Output Formats

### File Listing

```
src/main.go
src/utils/helper.go
docs/README.md
tests/integration_test.go
```

### Deletion Log

```
2024-01-15 10:30:45 DELETED FILE: [/tmp/temp1.txt /tmp/temp2.tmp]
successfully deleted: [/tmp/temp1.txt /tmp/temp2.tmp]
```

### Error Examples

```
INVALID_SIZE: file size cannot be negative
ROOT_NOT_FOUND: root directory does not exist (/nonexistent/path)
```

## Configuration

### Environment Variables

```bash
export NUMEN_LOG_LEVEL=info
export NUMEN_DEFAULT_LOG=/var/log/numen.log
export NUMEN_DEBUG=true
```

### Optional Config File (`~/.numen.yaml`)

```yaml
default:
  log_file: "~/.numen.log"
  min_size: 0
  extensions: []

presets:
  cleanup: "temp *.log *.tmp"
  source: "*.go *.js *.py *.rs"
  docs: "*.md *.txt *.rst"
```

## Error Handling

### Categories

* **Configuration Errors**: Missing fields, invalid arguments
* **File System Errors**: Permission issues, missing files, disk errors
* **Validation Errors**: Invalid paths or extensions
* **Runtime Errors**: Unexpected internal conditions

### Error Codes

| Code                 | Description            | Fix                     |
| -------------------- | ---------------------- | ----------------------- |
| `INVALID_SIZE`       | Negative size          | Use positive integer    |
| `EMPTY_ROOT`         | No root path provided  | Add `-root`             |
| `ROOT_NOT_FOUND`     | Path doesn't exist     | Create it or correct it |
| `ROOT_NOT_DIR`       | Path isn’t a directory | Provide a directory     |
| `NO_FILES_TO_DELETE` | Nothing to delete      | Specify files           |

### Troubleshooting

```bash
NUMEN_DEBUG=true numen -root /path -list
ls -la /path/to/directory

# Dry run
numen -root /path -list -ext "*.tmp"
numen -root /path -del "*.tmp" -log dry_run.log
```

## Testing

### Running Tests

```bash
go test -v
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
go test -race -v
go test -bench=. -v
```

### Test Types

* Unit tests
* Integration tests
* Performance and scalability tests
* Failure and edge-case testing

### Coverage Example

```
PASS
coverage: 96.7% of statements
ok      numen   2.456s
```

## Architecture

### Directory Layout

```
numen/
├── main.go
├── config.go
├── fileops.go
├── errors.go
├── main_test.go
├── go.mod
├── README.md
├── LICENSE
└── examples/
    ├── cleanup.sh
    ├── audit.sh
    └── backup.sh
```

### Design Patterns Used

* **Strategy** for filtering
* **Command** for CLI operations
* **Observer** for logging hooks
* **Factory** for processor creation
* **Error chaining** for rich error context

### Performance Notes

* **O(1)** memory usage
* **O(n)** traversal time
* Tested with **1M+ files**
* Thread-safe for concurrent execution

## Security

### Path Protection

* Rejects unsafe path traversal
* Blocks `../` and other escape attempts
* Sanitizes user-supplied patterns

### Permissions

* Obeys system-level permissions
* Gracefully handles denied access
* Logs all deletion attempts

### Input Validation

* Strict parameter checks
* Safe string handling
* Limits to prevent overflow issues

## Contributing

### Dev Setup

```bash
git clone https://github.com/yourusername/numen.git
cd numen
git checkout -b feature/amazing-feature

go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

golangci-lint run
goimports -w .
go test -v ./...
```

### Pull Requests

1. Branch from `main`
2. Add tests
3. Pass all checks
4. Update docs
5. Submit PR with clear explanation

## License

MIT License — see the [LICENSE](LICENSE) file.


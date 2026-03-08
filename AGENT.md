# Codeplug Management Tool Context

## Overview

This project is a Go-based CLI and web application for managing Amateur Radio codeplugs. It ingests data from various sources (Repeaterbook, RadioID.net, existing radio CSVs), stores it in a central SQLite database, and exports to specific radio formats.

## Architecture

- **Language**: Go 1.26.1
- **Database**: SQLite (using `modernc.org/sqlite` to avoid CGO dependencies)
- **ORM**: GORM (`gorm.io/gorm`) for data modeling and migrations
- **Frontend**: Vue 3 + Vite + Bun + Tailwind CSS v4
- **Build**: Taskfile (go-task) for task automation
- **CI/CD**: GitHub Actions for testing, releases, and contact generation
- **Linting**: golangci-lint with comprehensive configuration

## Project Structure

```
.
├── cmd/                      # CLI commands
│   └── generate_contacts.go  # Contact list generation
├── models/                   # GORM data models (Channel, Contact, Zone, etc.)
├── importer/                 # CSV import logic (DM32UV, AnyTone 890, Chirp, etc.)
├── exporter/                 # Radio-specific export formats
├── database/                 # DB connection and setup
├── api/                      # REST API and WebSocket handlers
├── services/                 # Business logic
├── frontend/                 # Vue 3 web UI
├── filters/                  # Filter files for contact generation
├── generated/                # Generated contact lists (git-ignored)
├── scripts/                  # Utility scripts (install-hooks.sh)
├── .github/workflows/        # CI/CD workflows
├── Taskfile.yml             # Task definitions
└── .golangci.yml            # Linter configuration
```

## Supported Radios

- **Radioddity DB25-D** - Via generic CSV import/export
- **Baofeng DM32UV** - Full import/export support with roaming
- **AnyTone 890** - Export support with roaming and scan lists

## Current Capabilities

### Import
- Import channels from CSV files with various formats
- Import from Repeaterbook JSON
- Import from ZIP archives containing radio CSVs
- Import DMR contacts/talkgroups from multiple formats

### Export
- Export to DM32UV format (channels, talkgroups, zones, roaming)
- Export to AnyTone 890 format (channels, contacts, zones, scan lists, roaming)
- Export to Chirp CSV format
- Export to DB25-D CSV format

### Contact Generation
Generate filtered contact lists from RadioID.net data:
```bash
./codeplugs --generate-contacts \
  --filter-file filters/my-contacts.csv \
  --source-file user.csv \
  --output-file contacts.csv \
  --contact-format dm32uv  # or radioid (default), at890
```

**Supported formats:**
- `radioid` - RadioID.net format (default)
- `dm32uv` - Baofeng DM32UV format
- `at890` - AnyTone 890 format (quoted fields, CRLF)

## Development Workflow

### Prerequisites
- Go 1.26.1+
- Bun (for frontend)
- Task (`brew install go-task` or `npm install -g @go-task/cli`)

### Build
```bash
task build          # Build with frontend
task fast-build     # Quick build without frontend
```

### Testing
```bash
task test           # Run all tests
task test-verbose   # Verbose output
task test-race      # With race detector
```

### Code Quality
```bash
task lint           # Run golangci-lint
task vet            # Run go vet
task fmt            # Format code
task ci             # Run all CI checks
```

### Git Hooks
Pre-commit hook runs golangci-lint before each commit:
```bash
./scripts/install-hooks.sh  # Install hooks
git commit --no-verify      # Bypass (emergency only)
```

## CI/CD

### Workflows
- **test.yml** - Runs on PRs/pushes: tests, linting, build verification
- **release.yml** - Triggered on tags: cross-platform binary builds
- **generate-contacts.yml** - Automated contact filtering on filter changes

### Release Process
1. Tag with version: `git tag v1.0.0`
2. Push tag: `git push origin v1.0.0`
3. Release workflow builds binaries for Linux, macOS, Windows (AMD64/ARM64)

## Data Models

- **Channel** - Frequency, mode, power, DMR details (color code, time slot, contact)
- **Contact** - DMR contacts/talkgroups (ID, name, type: Group/Private/All Call)
- **Zone** - Organized groups of channels
- **ScanList** - Custom scan lists
- **RoamingChannel/RoamingZone** - DMR roaming configuration
- **ContactList** - Filter lists for contact generation

## Development Methodology

### Test Driven Development (TDD)
- Tests SHOULD be written before implementation when practical
- Maintain high coverage on core logic (importers, exporters, API handlers, services)

### Code Quality
- All code must pass `golangci-lint` (configured in `.golangci.yml`)
- Pre-commit hook enforces linting
- CI runs full test suite and linting

### Frontend Standards
- **Framework**: Vue 3 + Vite
- **Runtime**: Bun
- **Styling**: Tailwind CSS v4
- **Design**: Premium aesthetic (Dark mode, glassmorphism, responsive)

## CLI Examples

### Import
```bash
# Import generic CSV
./codeplugs --import channels.csv

# Import DM32UV directory
./codeplugs --import path/to/dm32uv/ --radio dm32uv

# Import with zone
./codeplugs --import channels.csv --zone "My Zone"
```

### Export
```bash
# Export DM32UV
./codeplugs --export output/ --radio dm32uv

# Export AnyTone 890
./codeplugs --export output/ --radio at890

# Export with filter
./codeplugs --export contacts.csv --use-list "My Filter List"
```

### Web UI
```bash
./codeplugs --serve --port 8080
# Open http://localhost:8080
```

## Configuration

### Environment Variables
- `GO111MODULE=on`
- `CGO_ENABLED=0`

### Filter Files
Create filter files in `filters/` directory:
```csv
Radio ID,Callsign,Name
3100000,W9XXX,John Doe
```

CI automatically generates filtered contacts when filter files change.

## Future Work

- **Web UI Enhancements** - Drag-and-drop zone/channel management
- **Additional Radios** - Expand radio support (Yaesu System Fusion explicitly OUT OF SCOPE)
- **Repeater Integration** - Direct API integration with repeater databases
- **Contact Sync** - Automated contact list synchronization

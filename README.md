# Codeplugs

A Go-based tool for managing Amateur Radio codeplugs. It supports importing from various sources (CSV, Repeaterbook) and exporting to specific radio formats, with a focus on DMR radios.

## Supported Radios

- **Radioddity DB25-D** (via generic CSV import/export)
- **Baofeng DM32UV** (Full Import/Export support)
- **AnyTone 890** (Export support, validated against CPS)

## Features

- **Central Database**: Stores channels, contacts, and zones in a local SQLite database.
- **Web UI**: Interface for managing channels and zones (WIP).
- **CLI**: Powerful command-line interface for batch operations.
- **Exporters**: Customizable exporters that handle radio-specific formatting (e.g., AnyTone quoting rules).

## Installation

Install [go-task](https://taskfile.dev/) (recommended):

```bash
# macOS
brew install go-task

# Node.js
npm install -g @go-task/cli
```

Build the binary:

```bash
task build
```

## Usage

### Taskfile (Recommended)

The project uses [go-task](https://taskfile.dev/) to manage development tasks.

| Command | Description |
|---------|-------------|
| `task build` | Build the Go binary (includes frontend) |
| `task fast-build` | Quick build without rebuilding frontend |
| `task test` | Run all tests |
| `task run` | Build and run the server |
| `task clean` | Remove build artifacts and temporary files |
| `task fmt` | Run `go fmt` |
| `task vet` | Run `go vet` |
| `task lint` | Run linter (golangci-lint or vet) |
| `task check` | Run all checks (fmt, vet, lint, test, build) |
| `task ci` | Run CI checks |
| `task frontend-install` | Install frontend dependencies |
| `task frontend-build` | Build the frontend |
| `task generate-contacts` | Generate filtered contact list (set FORMAT=dm32uv/at890) |

### CLI Commands

#### Baofeng DM32UV

**Import from Directory** (containing `channels.csv`, `talkgroups.csv`, etc.):

```bash
./codeplugs --import path/to/dm32uv_csv_folder --radio dm32uv
```

**Export to Directory** (generates generic/DM32UV compatible CSVs):

```bash
./codeplugs --export path/to/output_folder --radio dm32uv
```

#### AnyTone 890

**Export to Directory** (Generates `Channel.CSV`, `DMRTalkGroups.CSV`, etc. for CPS import):

```bash
./codeplugs --export path/to/output_folder --radio at890
```

#### General / DB25-D

**Import Single File**:

```bash
./codeplugs --import my_channels.csv
```

**Export Single File**:

```bash
./codeplugs --export my_new_codeplug.csv
```

### Web UI

Start the server:

```bash
task run
```

Access the UI at `http://localhost:8080`.

## Development

Run tests:

```bash
task test
```

### Filtered Contact Generation

Generate filtered contact lists from RadioID.net data for specific radios:

**Basic usage** (RadioID.net format):
```bash
./codeplugs --generate-contacts \
  --filter-file filters/my-contacts.csv \
  --source-file user.csv \
  --output-file contacts.csv
```

**DM32UV format**:
```bash
./codeplugs --generate-contacts \
  --filter-file filters/my-contacts.csv \
  --source-file user.csv \
  --output-file digital_contacts.csv \
  --contact-format dm32uv
```

**AnyTone 890 format**:
```bash
./codeplugs --generate-contacts \
  --filter-file filters/my-contacts.csv \
  --source-file user.csv \
  --output-file DMRDigitalContactList.CSV \
  --contact-format at890
```

Or use Taskfile:
```bash
# Default (RadioID format)
task generate-contacts

# DM32UV format
FORMAT=dm32uv task generate-contacts

# AnyTone 890 format
FORMAT=at890 task generate-contacts
```

See `filters/README.md` for filter file format details.

### Makefile (Legacy)

A `Makefile` is provided for backward compatibility but is considered deprecated. Use `task` for the latest features.

```bash
make build
make test
```

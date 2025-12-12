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

```bash
go build -o codeplugs
```

## Usage

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
./codeplugs --server
```

Access the UI at `http://localhost:8080`.

## Development

Run tests:

```bash
go test ./...
```

# Codeplug Management Tool Context

## Overview

This project is a Go-based CLI tool designed to manage Amateur Radio codeplugs. It ingests data from various sources (like Repeaterbook or existing radio CSVs), stores it in a central SQLite database, and exports it to specific radio formats.

## Architecture

- **Language**: Go
- **Database**: SQLite (using `modernc.org/sqlite` to avoid CGO dependencies).
- **ORM**: GORM (`gorm.io/gorm`) for data modeling and migrations.
- **Structure**:
  - `cmd/`: CLI commands (currently in `main.go`).
  - `models/`: GORM data models (Channel, Contact, Zone).
  - `importer/`: Logic for importing CSVs.
  - `exporter/`: Logic for exporting to radio formats (currently DB25-D).
  - `database/`: DB connection and setup.

## Current Capabilities

- **Import**: Can import channels from CSV files.
  - Supports generic headers (Name, Frequency, Mode).
  - Supports DB25-D specific headers (RX Group, Contacts, Color Code, Time Slot).
- **Export**: Can export channels to the Radioddity DB25-D CSV format.
  - Handles DMR fields (Color Code, Time Slot, Group/Private calls).
  - Formats frequencies and other fields to match the radio's software requirements.
- **Data Models**:
  - `Channel`: Stores frequency, mode, power, and DMR-specific details.
  - `Contact`: (Planned) For managing DMR contacts/talkgroups.
  - `Zone`: (Planned) For organizing channels into zones.

## Usage

### Build

```bash
go build -o codeplugs
```

### Import

```bash
./codeplugs --import <csv_file> --region "<RegionName>"
```

Example: `./codeplugs --import db25d/Ann_Arbor_Area.csv --region "Ann Arbor"`

### Export

```bash
./codeplugs --export <output_csv> --region "<RegionName>"
```

Example: `./codeplugs --export my_codeplug.csv --region "Ann Arbor"`

## Future Work

- **Zone Management**: Fully implement Zone-to-Channel relationships.
- **Contact Management**: Separate Contacts from Channels for better DMR management.
- **More Radios**: Add exporters for AnyTone, Yaesu System Fusion, etc.
- **UI**: Add a TUI or Web UI for easier data manipulation.

## Development Methodology

### Test Driven Development (TDD)

- **Requirement**: All new logic must be tested.
- **Workflow**: Write the test case *first*, ensure it fails, then write the implementation to make the test pass.
- **Coverage**: Aim for high coverage on core logic (importers, exporters, data models).

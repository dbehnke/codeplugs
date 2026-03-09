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

#### BrandMeister Contact Generation

Generate contacts filtered by BrandMeister Last Heard activity (all formats at once):

```bash
# Generate contacts for active BrandMeister users
# Downloads RadioID.net data, applies filters/filter-brandmeister.csv
# Creates three output files in outputs/ directory
task generate-brandmeister

# Force re-download of RadioID.net data
task generate-brandmeister-clean
```

This creates:
- `outputs/brandmeister-radioid-{timestamp}.csv` - Standard RadioID.net format
- `outputs/brandmeister-dm32uv-{timestamp}.csv` - Baofeng DM32UV format
- `outputs/brandmeister-at890-{timestamp}.csv` - AnyTone 890 format

### Git Hooks

A pre-commit hook runs `golangci-lint` before each commit to ensure code quality.

**Install hooks:**
```bash
./scripts/install-hooks.sh
```

**Skip hooks (emergency only):**
```bash
git commit --no-verify
```

### Automated Releases

#### BrandMeister Contacts (Auto-Release)

When you push changes to `filters/filter-brandmeister.csv` on the main branch:

1. GitHub Actions automatically runs `task generate-brandmeister-clean`
2. Generated contact files are committed to `outputs/` directory
3. A new release is created with tag `bm-{timestamp}`
4. Release includes all three formats (radioid, dm32uv, at890)

#### Binary Releases (GoReleaser)

When you push a version tag (e.g., `v1.2.3`):

1. GitHub Actions runs GoReleaser
2. Builds binaries for Linux, macOS, and Windows (AMD64 and ARM64)
3. Creates a GitHub Release with changelog
4. Includes checksums and SBOM

**Create a new release:**
```bash
# Local dry-run
 task release

# Build for current platform only
task release-local

# Create and push version tag (triggers GitHub release)
task tag-release VERSION=v1.2.3

# Or manually
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

### Makefile (Legacy)

A `Makefile` is provided for backward compatibility but is considered deprecated. Use `task` for the latest features.

```bash
make build
make test
```

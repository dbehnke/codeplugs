# Filter Files

This directory contains CSV files that specify which DMR contacts to include in generated contact lists.

## Format

Filter files can be in one of these formats:

### 1. CSV with Header (Recommended)

```csv
Radio ID,Callsign,Name
1234567,N0XXX,John Doe
7654321,N0YYY,Jane Smith
```

The following column headers are recognized:
- `Radio ID` or `Radio_ID`
- `DMR ID` or `DMR_ID`
- `id` (case insensitive)

### 2. Plain List of IDs

```csv
1234567
7654321
1111111
```

Each line contains one DMR ID.

## Usage

1. Create a new filter file in this directory (e.g., `my-contacts.csv`)
2. Add the DMR IDs you want to include
3. Commit and push to the main branch
4. The CI workflow will automatically generate a filtered contact list in the `generated/` directory

## Automation

When you push changes to filter files on the main branch, the GitHub Actions workflow will:

1. Download the latest RadioID.net contacts
2. Filter them based on your filter file
3. Commit the results to the `generated/` directory

You can also trigger the workflow manually from the Actions tab.

## Example

See `example-filter.csv` for a sample filter file.

# CSV CLI Quick Start Guide

A user-friendly CLI tool has been created to test the CSV download functionality.

## Quick Start

### Option 1: Interactive Script (Recommended)

```bash
./test-csv.sh
```

This will:
1. Build the CLI tool
2. Look for `Liked_Songs.csv` in the current directory
3. Present you with easy options to test

### Option 2: Build and Run Manually

```bash
# Build the tool
go build -o csvtest ./cmd/csvtest

# Parse CSV only (no download)
./csvtest -csv Liked_Songs.csv -parse-only

# Download first 5 tracks (quick test)
./csvtest -csv Liked_Songs.csv -end 5 -output ./test-downloads

# Download all tracks
./csvtest -csv Liked_Songs.csv -output ./downloads
```

## What Was Created

```
cmd/csvtest/
├── main.go          # CLI tool source code
└── README.md        # Detailed documentation

test-csv.sh          # Interactive test script
CSV_CLI_QUICKSTART.md  # This file
```

## Common Use Cases

### 1. Test with First Few Tracks

Perfect for testing before downloading your entire library:

```bash
./csvtest -csv Liked_Songs.csv -end 10 -output ./test
```

### 2. Download from Qobuz (Hi-Res)

```bash
./csvtest -csv Liked_Songs.csv -service qobuz -format 27
```

### 3. Download with Lyrics

```bash
./csvtest -csv Liked_Songs.csv -lyrics
```

### 4. Resume from Specific Track

If download was interrupted at track 150:

```bash
./csvtest -csv Liked_Songs.csv -start 149
```

## Available Services

- **tidal** (default) - Formats: LOSSLESS, HI_RES
- **qobuz** - Formats: 5 (MP3), 6 (FLAC CD), 7 (24-bit), 27 (Hi-Res)
- **amazon** - Format: LOSSLESS

## All Command-Line Options

```
-csv string              Path to CSV file (required)
-service string          tidal/qobuz/amazon (default: tidal)
-output string           Output directory (default: ./downloads)
-format string           Audio format (default: LOSSLESS)
-filename string         Filename template (default: title-artist)
-track-number            Include track numbers
-lyrics                  Embed lyrics
-max-cover               Embed max quality cover (default: true)
-fallback                Allow service fallback (default: true)
-first-artist            Use first artist only
-start int               Start index (0-based)
-end int                 End index (0 = all)
-parse-only              Parse without downloading
```

## Tips

1. **Always test first**: Use `-parse-only` to check your CSV
2. **Start small**: Download just a few tracks with `-end 5`
3. **Resume capability**: Use `-start` to continue interrupted downloads
4. **Check output**: Files go to `./downloads` by default
5. **Existing files**: Already downloaded files are automatically skipped

## Troubleshooting

**Build fails**:
- Make sure you're in the project root directory
- Run `go mod tidy` first

**CSV not found**:
- Use absolute path: `-csv /full/path/to/file.csv`

**Service not available**:
- Try `-fallback` to use alternative services
- Check that you have valid credentials for the service

## Getting Your Spotify CSV

1. Visit https://www.spotify.com/account/privacy/
2. Scroll to "Download your data"
3. Request your data
4. Wait for email (can take a few days)
5. Download and extract the ZIP
6. Look for `Liked_Songs.csv`

## Full Documentation

For complete details, see `cmd/csvtest/README.md`

## Testing the Implementation

The CLI tool directly calls the same backend functions that were added in the recent commit:

- `backend.ParseSpotifyCSV()` - CSV parsing
- `backend.NewSongLinkClient().CheckTrackAvailability()` - URL discovery
- `backend.NewTidalDownloader().DownloadByURL()` - Downloading
- `backend.EmbedLyricsOnlyUniversal()` - Lyrics embedding

This ensures you're testing the actual implementation that will be used in the GUI.

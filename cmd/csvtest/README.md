# SpotiFLAC CSV Downloader CLI

A command-line tool for testing the CSV import and download functionality of SpotiFLAC.

## Building

```bash
# From the project root
go build -o csvtest ./cmd/csvtest

# Or from this directory
go build -o csvtest
```

## Usage

### Basic Usage

Download all tracks from a CSV file using Tidal:

```bash
./csvtest -csv Liked_Songs.csv
```

### Parse Only (No Download)

Just parse the CSV to see what tracks are available:

```bash
./csvtest -csv Liked_Songs.csv -parse-only
```

### Download from Qobuz

```bash
./csvtest -csv Liked_Songs.csv -service qobuz -format 27
```

### Download a Specific Range

Download only tracks 1-10:

```bash
./csvtest -csv Liked_Songs.csv -start 0 -end 10
```

Resume from track 50:

```bash
./csvtest -csv Liked_Songs.csv -start 49
```

### With Lyrics

```bash
./csvtest -csv Liked_Songs.csv -lyrics
```

### Custom Output Directory

```bash
./csvtest -csv Liked_Songs.csv -output ~/Music/SpotifyImport
```

## All Options

```
  -csv string
        Path to Spotify CSV file (required)
  -service string
        Service to download from: tidal, qobuz, amazon (default "tidal")
  -output string
        Output directory (default "./downloads")
  -format string
        Audio format:
          Tidal: LOSSLESS, HI_RES
          Qobuz: 5 (MP3 320), 6 (FLAC CD), 7 (24-bit), 27 (Hi-Res)
        (default "LOSSLESS")
  -filename string
        Filename format template (default "title-artist")
        Options: title-artist, artist-title, track-title, etc.
  -track-number
        Include track numbers in filenames (default false)
  -lyrics
        Embed lyrics in downloaded files (default false)
  -max-cover
        Embed maximum quality cover art (default true)
  -fallback
        Allow fallback to alternative services (default true)
  -first-artist
        Use only the first artist name (default false)
  -start int
        Start index (0-based) (default 0)
  -end int
        End index (0 = all tracks) (default 0)
  -parse-only
        Only parse CSV without downloading (default false)
```

## Examples

### Example 1: Quick Test (First 5 Tracks)

```bash
./csvtest -csv Liked_Songs.csv -end 5 -output ./test-downloads
```

### Example 2: High Quality Qobuz Download with Lyrics

```bash
./csvtest -csv Liked_Songs.csv \
  -service qobuz \
  -format 27 \
  -lyrics \
  -output ~/Music/Qobuz
```

### Example 3: Tidal Hi-Res Download

```bash
./csvtest -csv Liked_Songs.csv \
  -service tidal \
  -format HI_RES \
  -lyrics \
  -output ~/Music/Tidal
```

### Example 4: Resume Interrupted Download

If your download was interrupted at track 123:

```bash
./csvtest -csv Liked_Songs.csv -start 122
```

## CSV Format

The tool expects a Spotify export CSV with these required columns:
- `Track URI` - Spotify track URI (e.g., spotify:track:xxxxx)
- `Track Name` - Song title
- `Album Name` - Album name
- `Artist Name(s)` - Artist names

Optional columns:
- `Release Date`
- `Duration (ms)`
- `Popularity`
- `Explicit`
- `Added At`
- `Record Label`
- `Danceability`
- `Energy`

## Getting Your Spotify CSV

1. Go to [Spotify Account Settings](https://www.spotify.com/account/privacy/)
2. Scroll to "Download your data"
3. Request your data
4. Wait for email with download link
5. Extract the ZIP file
6. Look for files like `Liked_Songs.csv` or `Playlist*.csv`

## Notes

- The tool will skip tracks that already exist (based on filename)
- Failed downloads are reported at the end with error details
- Progress is shown in real-time
- You can cancel at any time with Ctrl+C
- The tool uses the same backend as the main SpotiFLAC GUI application

## Troubleshooting

**"Track not available on [service]"**
- Try using `-fallback` flag to allow alternative services
- Some tracks may not be available on all services

**"Failed to get streaming URLs"**
- Check your internet connection
- The song.link service might be temporarily unavailable

**Large CSV files taking too long**
- Use `-start` and `-end` to process in batches
- Use `-parse-only` first to verify the CSV

## License

Same as SpotiFLAC main project.

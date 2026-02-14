# CSV Import Feature

SpotiFLAC now supports importing and downloading tracks from Spotify CSV exports.

## CSV Format

The CSV parser expects a Spotify export format with the following required columns:
- `Track URI` - Spotify track URI (e.g., `spotify:track:6hgBTpKs8Ac8a8owmUIwba`)
- `Track Name` - The name of the track
- `Album Name` - The album name
- `Artist Name(s)` - Artist names (comma-separated for multiple artists)

Optional columns that are also parsed:
- `Release Date` - Release date in YYYY-MM-DD format
- `Duration (ms)` - Track duration in milliseconds
- `Popularity` - Spotify popularity score (0-100)
- `Explicit` - Boolean indicating explicit content
- `Added At` - Timestamp when track was added
- `Record Label` - The record label
- `Danceability`, `Energy` - Spotify audio features

## How to Export from Spotify

1. Go to your Spotify account settings
2. Request your data (Account Data or Extended Streaming History)
3. Spotify will email you a download link
4. Extract the ZIP file and locate the CSV file (usually named `Liked_Songs.csv` or similar)

## Backend Functions

### 1. Parse CSV File

```go
result, err := ParseCSVFile("/path/to/Liked_Songs.csv")
```

Returns:
- `Success` - Whether parsing succeeded
- `TotalTracks` - Total tracks found in CSV
- `ValidTracks` - Number of valid tracks
- `Tracks` - Array of parsed track objects
- `Errors` - Array of parsing errors

### 2. Validate CSV File

```go
err := ValidateCSVFile("/path/to/Liked_Songs.csv")
```

Quickly checks if the CSV has the required columns without parsing all tracks.

### 3. Get CSV Statistics

```go
stats, err := GetCSVStats("/path/to/Liked_Songs.csv")
```

Returns basic statistics about the CSV file (row count, columns, headers).

### 4. Download from CSV

```go
request := CSVDownloadRequest{
    CSVFilePath:          "/path/to/Liked_Songs.csv",
    Service:              "tidal",  // or "qobuz" or "amazon"
    OutputDir:            "/path/to/output",
    AudioFormat:          "LOSSLESS",
    FilenameFormat:       "title-artist",
    TrackNumber:          true,
    EmbedLyrics:          true,
    EmbedMaxQualityCover: true,
    AllowFallback:        true,
    UseFirstArtistOnly:   false,
    StartIndex:           0,    // Start from first track
    EndIndex:             0,    // 0 means process all tracks
}

response, err := DownloadFromCSV(request)
```

Returns:
- `Success` - Overall success status
- `TotalTracks` - Total tracks in CSV
- `ProcessedTracks` - Number of tracks actually processed
- `SuccessCount` - Successfully downloaded tracks
- `FailureCount` - Failed downloads
- `SkippedCount` - Tracks that were skipped (already exist)
- `Errors` - Array of error messages
- `Message` - Summary message

## Usage Examples

### Example 1: Download All Tracks from CSV (Tidal)

```go
response, err := DownloadFromCSV(CSVDownloadRequest{
    CSVFilePath:    "/Users/username/Downloads/Liked_Songs.csv",
    Service:        "tidal",
    OutputDir:      "/Users/username/Music/Spotify Import",
    AudioFormat:    "LOSSLESS",
    FilenameFormat: "title-artist",
    EmbedLyrics:    true,
})
```

### Example 2: Download from Qobuz with Hi-Res Quality

```go
response, err := DownloadFromCSV(CSVDownloadRequest{
    CSVFilePath:    "/Users/username/Downloads/Liked_Songs.csv",
    Service:        "qobuz",
    OutputDir:      "/Users/username/Music/Spotify Import",
    AudioFormat:    "27",  // Hi-Res 24-bit
    FilenameFormat: "artist-title",
    EmbedLyrics:    true,
})
```

### Example 3: Download Specific Range of Tracks

```go
// Download tracks 1-50 only
response, err := DownloadFromCSV(CSVDownloadRequest{
    CSVFilePath:    "/Users/username/Downloads/Liked_Songs.csv",
    Service:        "tidal",
    OutputDir:      "/Users/username/Music/Spotify Import",
    AudioFormat:    "LOSSLESS",
    StartIndex:     0,   // Start at track 1 (0-indexed)
    EndIndex:       50,  // End at track 50
})
```

### Example 4: Parse CSV First, Then Download Selectively

```go
// First, parse the CSV to see what's in it
result, err := ParseCSVFile("/Users/username/Downloads/Liked_Songs.csv")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d valid tracks\n", result.ValidTracks)

// Then download from it
response, err := DownloadFromCSV(CSVDownloadRequest{
    CSVFilePath:    "/Users/username/Downloads/Liked_Songs.csv",
    Service:        "tidal",
    OutputDir:      "/Users/username/Music/Spotify Import",
    AudioFormat:    "LOSSLESS",
})

fmt.Printf("Downloaded: %d succeeded, %d skipped, %d failed\n",
    response.SuccessCount, response.SkippedCount, response.FailureCount)
```

## Features

### Automatic Spotify ID Extraction
The parser automatically extracts Spotify track IDs from the Track URI format (`spotify:track:ID`).

### Service Discovery
Uses the existing SongLink integration to find tracks on Tidal, Qobuz, or Amazon Music.

### Progress Tracking
Each download is tracked through the existing queue management system, so you can monitor progress in real-time.

### Duplicate Detection
Automatically skips tracks that have already been downloaded based on expected filename.

### Error Handling
- Continues downloading even if individual tracks fail
- Collects all errors for review after batch completion
- Provides detailed error messages for each failed track

### Batch Control
- Download all tracks or specify a range (StartIndex/EndIndex)
- Useful for resuming interrupted downloads
- Can process CSV in chunks to avoid overwhelming the system

## Audio Quality Options

### Tidal
- `LOW` - 96 kbps AAC
- `HIGH` - 320 kbps AAC
- `LOSSLESS` - FLAC 16-bit/44.1kHz (default)
- `HI_RES` - FLAC 24-bit/96kHz+ (if available)

### Qobuz
- `5` - MP3 320 kbps
- `6` - FLAC 16-bit/44.1kHz (CD Quality)
- `7` - FLAC 24-bit/96kHz
- `27` - Hi-Res FLAC 24-bit/192kHz

### Amazon Music
- `LOSSLESS` - FLAC 16-bit/44.1kHz

## Filename Format Templates

- `title-artist` - "Track Name - Artist Name.flac"
- `artist-title` - "Artist Name - Track Name.flac"
- `track-title` - "01 Track Name.flac" (with track number)
- `artist-album-title` - "Artist - Album - Track.flac"
- And more...

## Console Output

The download process provides real-time console output:

```
[1/150] Processing: One Man Circus - Elio Mei
Getting streaming URLs from song.link...
✓ Tidal URL found
Downloading from Tidal...
Downloaded: 8.45 MB (2.34 MB/s)
Embedding metadata...
✓ Downloaded successfully

[2/150] Processing: Gap Tooth Smile - Djo
✓ Tidal URL found
Downloaded: 6.82 MB (2.45 MB/s)
⊙ Skipped (already exists)

[3/150] Processing: girl, get up. (feat. SZA) - Doechii, SZA
✗ Failed: Track not available on Tidal
```

## Error Recovery

If the download process is interrupted:

1. Check the console output or error array to see which track number failed
2. Use `StartIndex` to resume from that track:

```go
response, err := DownloadFromCSV(CSVDownloadRequest{
    CSVFilePath:    "/Users/username/Downloads/Liked_Songs.csv",
    Service:        "tidal",
    OutputDir:      "/Users/username/Music/Spotify Import",
    StartIndex:     45,  // Resume from track 46 (0-indexed)
})
```

## Integration with Existing Features

The CSV import feature integrates seamlessly with existing SpotiFLAC features:
- Uses the same download queue system
- Applies the same metadata embedding
- Uses the same lyrics and cover art fetching
- Respects the same duplicate detection logic
- Works with all supported services (Tidal, Qobuz, Amazon)

## Notes

- Large CSV files (1000+ tracks) may take several hours to process
- Download speed depends on your internet connection and service API limits
- Some tracks may not be available on the selected service - use `AllowFallback: true` to try alternative services
- The CSV parser is forgiving and will skip invalid rows while continuing to process valid ones

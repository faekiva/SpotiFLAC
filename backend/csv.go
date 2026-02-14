package backend

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type CSVTrack struct {
	TrackURI     string `json:"track_uri"`
	SpotifyID    string `json:"spotify_id"`
	TrackName    string `json:"track_name"`
	AlbumName    string `json:"album_name"`
	ArtistName   string `json:"artist_name"`
	ReleaseDate  string `json:"release_date"`
	DurationMS   int    `json:"duration_ms"`
	Popularity   int    `json:"popularity"`
	Explicit     bool   `json:"explicit"`
	AddedAt      string `json:"added_at"`
	RecordLabel  string `json:"record_label"`
	Danceability float64 `json:"danceability"`
	Energy       float64 `json:"energy"`
}

type CSVImportResult struct {
	Success      bool        `json:"success"`
	TotalTracks  int         `json:"total_tracks"`
	ValidTracks  int         `json:"valid_tracks"`
	Tracks       []CSVTrack  `json:"tracks"`
	Errors       []string    `json:"errors"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// ParseSpotifyCSV parses a Spotify export CSV file
func ParseSpotifyCSV(filePath string) (*CSVImportResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Create a map of column indices
	columnMap := make(map[string]int)
	for i, header := range headers {
		// Remove BOM if present
		header = strings.TrimPrefix(header, "\ufeff")
		columnMap[header] = i
	}

	// Validate required columns
	requiredColumns := []string{"Track URI", "Track Name", "Album Name", "Artist Name(s)"}
	for _, col := range requiredColumns {
		if _, exists := columnMap[col]; !exists {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	result := &CSVImportResult{
		Success:     true,
		Tracks:      make([]CSVTrack, 0),
		Errors:      make([]string, 0),
		TotalTracks: 0,
		ValidTracks: 0,
	}

	lineNumber := 1 // Starting at 1 because we already read the header
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: %v", lineNumber, err))
			lineNumber++
			continue
		}

		lineNumber++
		result.TotalTracks++

		// Extract Track URI and Spotify ID
		trackURI := getColumnValue(record, columnMap, "Track URI")
		if trackURI == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Missing Track URI", lineNumber))
			continue
		}

		// Extract Spotify ID from URI (format: spotify:track:ID)
		spotifyID := extractSpotifyID(trackURI)
		if spotifyID == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Invalid Track URI format: %s", lineNumber, trackURI))
			continue
		}

		track := CSVTrack{
			TrackURI:    trackURI,
			SpotifyID:   spotifyID,
			TrackName:   getColumnValue(record, columnMap, "Track Name"),
			AlbumName:   getColumnValue(record, columnMap, "Album Name"),
			ArtistName:  getColumnValue(record, columnMap, "Artist Name(s)"),
			ReleaseDate: getColumnValue(record, columnMap, "Release Date"),
			AddedAt:     getColumnValue(record, columnMap, "Added At"),
			RecordLabel: getColumnValue(record, columnMap, "Record Label"),
		}

		// Parse numeric fields
		if durationStr := getColumnValue(record, columnMap, "Duration (ms)"); durationStr != "" {
			if duration, err := strconv.Atoi(durationStr); err == nil {
				track.DurationMS = duration
			}
		}

		if popularityStr := getColumnValue(record, columnMap, "Popularity"); popularityStr != "" {
			if popularity, err := strconv.Atoi(popularityStr); err == nil {
				track.Popularity = popularity
			}
		}

		// Parse boolean fields
		if explicitStr := getColumnValue(record, columnMap, "Explicit"); explicitStr != "" {
			track.Explicit = strings.ToLower(explicitStr) == "true"
		}

		// Parse float fields
		if danceabilityStr := getColumnValue(record, columnMap, "Danceability"); danceabilityStr != "" {
			if danceability, err := strconv.ParseFloat(danceabilityStr, 64); err == nil {
				track.Danceability = danceability
			}
		}

		if energyStr := getColumnValue(record, columnMap, "Energy"); energyStr != "" {
			if energy, err := strconv.ParseFloat(energyStr, 64); err == nil {
				track.Energy = energy
			}
		}

		// Validate essential fields
		if track.TrackName == "" || track.ArtistName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Line %d: Missing track name or artist name", lineNumber))
			continue
		}

		result.Tracks = append(result.Tracks, track)
		result.ValidTracks++
	}

	if result.ValidTracks == 0 {
		result.Success = false
		result.ErrorMessage = "No valid tracks found in CSV"
	}

	return result, nil
}

// getColumnValue safely gets a column value from a record
func getColumnValue(record []string, columnMap map[string]int, columnName string) string {
	if idx, exists := columnMap[columnName]; exists && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

// extractSpotifyID extracts the Spotify track ID from a Track URI
// Format: spotify:track:6hgBTpKs8Ac8a8owmUIwba -> 6hgBTpKs8Ac8a8owmUIwba
func extractSpotifyID(trackURI string) string {
	parts := strings.Split(trackURI, ":")
	if len(parts) == 3 && parts[0] == "spotify" && parts[1] == "track" {
		return parts[2]
	}
	return ""
}

// ValidateCSVFile validates that a file is a valid Spotify CSV export
func ValidateCSVFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read and validate header
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Remove BOM from first header if present
	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}

	// Check for required columns
	requiredColumns := []string{"Track URI", "Track Name", "Album Name", "Artist Name(s)"}
	headerSet := make(map[string]bool)
	for _, h := range headers {
		headerSet[h] = true
	}

	for _, required := range requiredColumns {
		if !headerSet[required] {
			return fmt.Errorf("missing required column: %s", required)
		}
	}

	return nil
}

// GetCSVStats returns statistics about a CSV file without fully parsing it
func GetCSVStats(filePath string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Count rows
	rowCount := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rowCount++
	}

	stats := map[string]interface{}{
		"total_rows": rowCount,
		"columns":    len(headers),
		"headers":    headers,
	}

	return stats, nil
}

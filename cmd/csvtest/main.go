package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"spotiflac/backend"
	"strings"
)

type Config struct {
	CSVFile            string
	Service            string
	OutputDir          string
	AudioFormat        string
	FilenameFormat     string
	TrackNumber        bool
	EmbedLyrics        bool
	EmbedMaxCover      bool
	AllowFallback      bool
	UseFirstArtistOnly bool
	StartIndex         int
	EndIndex           int
	ParseOnly          bool
}

func main() {
	config := parseFlags()

	if config.CSVFile == "" {
		fmt.Println("Error: CSV file path is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize history DB
	if err := backend.InitHistoryDB("SpotiFLAC-CLI"); err != nil {
		fmt.Printf("Warning: Failed to init history DB: %v\n", err)
	}
	defer backend.CloseHistoryDB()

	// Print configuration
	printBanner()
	printConfig(config)

	// Parse CSV
	fmt.Println("\n📄 Parsing CSV file...")
	result, err := backend.ParseSpotifyCSV(config.CSVFile)
	if err != nil {
		fmt.Printf("❌ Failed to parse CSV: %v\n", err)
		os.Exit(1)
	}

	if !result.Success {
		fmt.Printf("❌ CSV parsing failed: %s\n", result.ErrorMessage)
		os.Exit(1)
	}

	fmt.Printf("✅ Parsed successfully!\n")
	fmt.Printf("   Total tracks: %d\n", result.TotalTracks)
	fmt.Printf("   Valid tracks: %d\n", result.ValidTracks)
	if len(result.Errors) > 0 {
		fmt.Printf("   Parsing errors: %d\n", len(result.Errors))
	}

	// Show first few tracks
	fmt.Println("\n📋 First 5 tracks:")
	for i, track := range result.Tracks {
		if i >= 5 {
			break
		}
		fmt.Printf("   %d. %s - %s", i+1, track.TrackName, track.ArtistName)
		if track.AlbumName != "" {
			fmt.Printf(" (%s)", track.AlbumName)
		}
		fmt.Println()
	}

	if config.ParseOnly {
		fmt.Println("\n✅ Parse-only mode. Exiting.")
		return
	}

	// Determine range
	startIdx := config.StartIndex
	endIdx := config.EndIndex
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx <= 0 || endIdx > len(result.Tracks) {
		endIdx = len(result.Tracks)
	}

	tracksToProcess := result.Tracks[startIdx:endIdx]
	fmt.Printf("\n📦 Preparing to download %d tracks (range: %d-%d)\n", len(tracksToProcess), startIdx+1, endIdx)
	fmt.Printf("   Service: %s\n", config.Service)
	fmt.Printf("   Output: %s\n", config.OutputDir)
	fmt.Printf("   Format: %s\n", config.AudioFormat)

	// Confirm
	fmt.Print("\nContinue? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled.")
		return
	}

	// Download tracks
	fmt.Println("\n🎵 Starting downloads...\n")
	fmt.Println(strings.Repeat("=", 70))

	var successCount, failureCount, skippedCount int
	var errors []string

	for idx, track := range tracksToProcess {
		absoluteIdx := startIdx + idx + 1
		fmt.Printf("\n[%d/%d] %s - %s\n", absoluteIdx, endIdx, track.TrackName, track.ArtistName)
		fmt.Println(strings.Repeat("-", 70))

		// Download the track
		success, alreadyExists, err := downloadTrack(track, config, absoluteIdx)

		if err != nil {
			failureCount++
			errorMsg := fmt.Sprintf("Track %d (%s - %s): %v", absoluteIdx, track.TrackName, track.ArtistName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("❌ Failed: %v\n", err)
		} else if success {
			if alreadyExists {
				skippedCount++
				fmt.Printf("⊙ Skipped (already exists)\n")
			} else {
				successCount++
				fmt.Printf("✅ Downloaded successfully\n")
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("\n📊 DOWNLOAD SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Total tracks:     %d\n", endIdx)
	fmt.Printf("Processed:        %d\n", len(tracksToProcess))
	fmt.Printf("✅ Successful:     %d\n", successCount)
	fmt.Printf("⊙ Skipped:        %d\n", skippedCount)
	fmt.Printf("❌ Failed:         %d\n", failureCount)

	if len(errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, errMsg := range errors {
			fmt.Printf("   - %s\n", errMsg)
		}
	}

	fmt.Println("\n✨ Done!")
}

func downloadTrack(track backend.CSVTrack, config Config, position int) (success bool, alreadyExists bool, err error) {
	// Check if file already exists
	expectedFilename := backend.BuildExpectedFilename(
		track.TrackName,
		track.ArtistName,
		track.AlbumName,
		"",
		track.ReleaseDate,
		config.FilenameFormat,
		"",
		"",
		config.TrackNumber,
		position,
		0,
		false,
	)
	expectedPath := filepath.Join(config.OutputDir, expectedFilename)

	if fileInfo, statErr := os.Stat(expectedPath); statErr == nil && fileInfo.Size() > 100*1024 {
		return true, true, nil
	}

	// Get streaming URL from song.link
	fmt.Printf("🔗 Getting streaming URL from song.link...\n")
	client := backend.NewSongLinkClient()
	availability, err := client.CheckTrackAvailability(track.SpotifyID)
	if err != nil {
		return false, false, fmt.Errorf("failed to get streaming URLs: %w", err)
	}

	var serviceURL string
	var isAvailable bool
	switch config.Service {
	case "tidal":
		serviceURL = availability.TidalURL
		isAvailable = availability.Tidal
	case "qobuz":
		serviceURL = availability.QobuzURL
		isAvailable = availability.Qobuz
	case "amazon":
		serviceURL = availability.AmazonURL
		isAvailable = availability.Amazon
	default:
		return false, false, fmt.Errorf("unknown service: %s", config.Service)
	}

	if !isAvailable || serviceURL == "" {
		return false, false, fmt.Errorf("track not available on %s", config.Service)
	}

	fmt.Printf("✅ Found URL on %s\n", config.Service)
	fmt.Printf("⬇️  Downloading...\n")

	// Download based on service
	var filename string
	switch config.Service {
	case "tidal":
		downloader := backend.NewTidalDownloader("")
		filename, err = downloader.DownloadByURLWithFallback(
			serviceURL,
			config.OutputDir,
			config.AudioFormat,
			config.FilenameFormat,
			config.TrackNumber,
			position,
			track.TrackName,
			track.ArtistName,
			track.AlbumName,
			"",
			track.ReleaseDate,
			false,
			"",
			config.EmbedMaxCover,
			0, 0, 0, 0,
			"", "",
			fmt.Sprintf("https://open.spotify.com/track/%s", track.SpotifyID),
			config.AllowFallback,
			config.UseFirstArtistOnly,
		)

	case "qobuz":
		// Get ISRC
		isrc, _ := client.GetISRC(track.SpotifyID)
		downloader := backend.NewQobuzDownloader()
		filename, err = downloader.DownloadTrackWithISRC(
			isrc,
			track.SpotifyID,
			config.OutputDir,
			config.AudioFormat,
			config.FilenameFormat,
			config.TrackNumber,
			position,
			track.TrackName,
			track.ArtistName,
			track.AlbumName,
			"",
			track.ReleaseDate,
			false,
			"",
			config.EmbedMaxCover,
			0, 0, 0, 0,
			"", "",
			fmt.Sprintf("https://open.spotify.com/track/%s", track.SpotifyID),
			config.AllowFallback,
			config.UseFirstArtistOnly,
		)

	case "amazon":
		downloader := backend.NewAmazonDownloader()
		filename, err = downloader.DownloadByURL(
			serviceURL,
			config.OutputDir,
			config.AudioFormat,
			config.FilenameFormat,
			"", "",
			config.TrackNumber,
			position,
			track.TrackName,
			track.ArtistName,
			track.AlbumName,
			"",
			track.ReleaseDate,
			"",
			0, 0, 0,
			config.EmbedMaxCover,
			0,
			"", "",
			fmt.Sprintf("https://open.spotify.com/track/%s", track.SpotifyID),
			config.UseFirstArtistOnly,
		)

	default:
		return false, false, fmt.Errorf("unsupported service: %s", config.Service)
	}

	if err != nil {
		return false, false, err
	}

	// Check if already existed
	if strings.HasPrefix(filename, "EXISTS:") {
		return true, true, nil
	}

	// Embed lyrics if requested
	if config.EmbedLyrics && track.SpotifyID != "" {
		if strings.HasSuffix(filename, ".flac") || strings.HasSuffix(filename, ".mp3") || strings.HasSuffix(filename, ".m4a") {
			fmt.Printf("📝 Fetching lyrics...\n")
			lyricsClient := backend.NewLyricsClient()
			resp, _, err := lyricsClient.FetchLyricsAllSources(track.SpotifyID, track.TrackName, track.ArtistName, track.DurationMS/1000)
			if err == nil && resp != nil && len(resp.Lines) > 0 {
				lrc := lyricsClient.ConvertToLRC(resp, track.TrackName, track.ArtistName)
				if lrcErr := backend.EmbedLyricsOnlyUniversal(filename, lrc); lrcErr == nil {
					fmt.Printf("✅ Lyrics embedded\n")
				}
			}
		}
	}

	return true, false, nil
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.CSVFile, "csv", "", "Path to Spotify CSV file (required)")
	flag.StringVar(&config.Service, "service", "tidal", "Service to download from (tidal, qobuz, amazon)")
	flag.StringVar(&config.OutputDir, "output", "./downloads", "Output directory")
	flag.StringVar(&config.AudioFormat, "format", "LOSSLESS", "Audio format (LOSSLESS, HI_RES for Tidal; 5, 6, 7, 27 for Qobuz)")
	flag.StringVar(&config.FilenameFormat, "filename", "title-artist", "Filename format template")
	flag.BoolVar(&config.TrackNumber, "track-number", false, "Include track numbers in filenames")
	flag.BoolVar(&config.EmbedLyrics, "lyrics", false, "Embed lyrics in downloaded files")
	flag.BoolVar(&config.EmbedMaxCover, "max-cover", true, "Embed maximum quality cover art")
	flag.BoolVar(&config.AllowFallback, "fallback", true, "Allow fallback to alternative services")
	flag.BoolVar(&config.UseFirstArtistOnly, "first-artist", false, "Use only the first artist name")
	flag.IntVar(&config.StartIndex, "start", 0, "Start index (0-based)")
	flag.IntVar(&config.EndIndex, "end", 0, "End index (0 = all tracks)")
	flag.BoolVar(&config.ParseOnly, "parse-only", false, "Only parse CSV without downloading")

	flag.Parse()

	// Normalize service name to lowercase
	config.Service = strings.ToLower(config.Service)

	return config
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════════╗
║                    SpotiFLAC CSV Downloader                       ║
║                     CLI Testing Tool v1.0                         ║
╚═══════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

func printConfig(config Config) {
	fmt.Println("⚙️  Configuration:")
	fmt.Printf("   CSV File:          %s\n", config.CSVFile)
	fmt.Printf("   Service:           %s\n", config.Service)
	fmt.Printf("   Output Directory:  %s\n", config.OutputDir)
	fmt.Printf("   Audio Format:      %s\n", config.AudioFormat)
	fmt.Printf("   Filename Format:   %s\n", config.FilenameFormat)
	fmt.Printf("   Track Numbers:     %v\n", config.TrackNumber)
	fmt.Printf("   Embed Lyrics:      %v\n", config.EmbedLyrics)
	fmt.Printf("   Max Quality Cover: %v\n", config.EmbedMaxCover)
	fmt.Printf("   Allow Fallback:    %v\n", config.AllowFallback)
	fmt.Printf("   First Artist Only: %v\n", config.UseFirstArtistOnly)
	fmt.Printf("   Start Index:       %d\n", config.StartIndex)
	if config.EndIndex > 0 {
		fmt.Printf("   End Index:         %d\n", config.EndIndex)
	} else {
		fmt.Printf("   End Index:         all\n")
	}
}

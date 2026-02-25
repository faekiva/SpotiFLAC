package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	spotifyBaseURL string
	songLinkAPIURL string
)

const qobuzAppID = "798273057"

func init() {
	b, _ := base64.StdEncoding.DecodeString("aHR0cHM6Ly9vcGVuLnNwb3RpZnkuY29tL3RyYWNrLw==")
	spotifyBaseURL = string(b)
	b, _ = base64.StdEncoding.DecodeString("aHR0cHM6Ly9hcGkuc29uZy5saW5rL3YxLWFscGhhLjEvbGlua3M=")
	songLinkAPIURL = string(b)
}

type SongLinkClient struct {
	client           *http.Client
	mu               sync.Mutex
	lastAPICallTime  time.Time
	apiCallCount     int
	apiCallResetTime time.Time
}

type SongLinkResult struct {
	TidalURL  string
	AmazonURL string
	DeezerURL string
	QobuzURL  string
	ISRC      string
}

type SongLinkURLs struct {
	TidalURL  string `json:"tidal_url"`
	AmazonURL string `json:"amazon_url"`
	ISRC      string `json:"isrc"`
}

type TrackAvailability struct {
	SpotifyID string `json:"spotify_id"`
	Tidal     bool   `json:"tidal"`
	Amazon    bool   `json:"amazon"`
	Qobuz     bool   `json:"qobuz"`
	TidalURL  string `json:"tidal_url,omitempty"`
	AmazonURL string `json:"amazon_url,omitempty"`
	QobuzURL  string `json:"qobuz_url,omitempty"`
	ISRC      string `json:"isrc,omitempty"`
}

type songLinkResponse struct {
	LinksByPlatform map[string]struct {
		URL string `json:"url"`
	} `json:"linksByPlatform"`
}

func NewSongLinkClient() *SongLinkClient {
	return &SongLinkClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiCallResetTime: time.Now(),
	}
}

func (s *SongLinkClient) waitForRateLimit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.Sub(s.apiCallResetTime) >= time.Minute {
		s.apiCallCount = 0
		s.apiCallResetTime = now
	}

	if s.apiCallCount >= 9 {
		waitTime := time.Minute - now.Sub(s.apiCallResetTime)
		if waitTime > 0 {
			fmt.Printf("Rate limit reached, waiting %v...\n", waitTime.Round(time.Second))
			s.mu.Unlock()
			time.Sleep(waitTime)
			s.mu.Lock()
			s.apiCallCount = 0
			s.apiCallResetTime = time.Now()
		}
	}

	if !s.lastAPICallTime.IsZero() {
		timeSinceLastCall := time.Now().Sub(s.lastAPICallTime)
		minDelay := 7 * time.Second
		if timeSinceLastCall < minDelay {
			waitTime := minDelay - timeSinceLastCall
			fmt.Printf("Rate limiting: waiting %v...\n", waitTime.Round(time.Second))
			s.mu.Unlock()
			time.Sleep(waitTime)
			s.mu.Lock()
		}
	}
}

func buildSongLinkAPIURL(spotifyTrackID, region string) string {
	u, _ := url.Parse(songLinkAPIURL)
	q := u.Query()
	q.Set("url", spotifyBaseURL+spotifyTrackID)
	q.Set("songIfSingle", "true")
	if region != "" {
		q.Set("userCountry", region)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *SongLinkClient) doWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := 3
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = s.client.Do(req)
		if err != nil {
			return nil, err
		}

		s.mu.Lock()
		s.lastAPICallTime = time.Now()
		s.apiCallCount++
		s.mu.Unlock()

		if resp.StatusCode == 429 {
			resp.Body.Close()
			if i < maxRetries-1 {
				waitTime := 15 * time.Second
				fmt.Printf("Rate limited by API, waiting %v before retry...\n", waitTime)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("API rate limit exceeded after %d retries", maxRetries)
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries", maxRetries)
}

func (s *SongLinkClient) lookup(spotifyTrackID, region string) (*SongLinkResult, error) {
	s.waitForRateLimit()

	apiURL := buildSongLinkAPIURL(spotifyTrackID, region)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	fmt.Println("Getting streaming URLs from song.link...")

	resp, err := s.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("API returned empty response")
	}

	var songLinkResp songLinkResponse
	if err := json.Unmarshal(body, &songLinkResp); err != nil {
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return nil, fmt.Errorf("failed to decode response: %w (response: %s)", err, bodyStr)
	}

	result := &SongLinkResult{}

	if link, ok := songLinkResp.LinksByPlatform["tidal"]; ok && link.URL != "" {
		result.TidalURL = link.URL
		fmt.Printf("Found Tidal URL\n")
	}

	if link, ok := songLinkResp.LinksByPlatform["amazonMusic"]; ok && link.URL != "" {
		result.AmazonURL = link.URL
		fmt.Printf("Found Amazon URL\n")
	}

	if link, ok := songLinkResp.LinksByPlatform["deezer"]; ok && link.URL != "" {
		result.DeezerURL = link.URL
	}

	return result, nil
}

func (r *SongLinkResult) FetchISRC(client *http.Client) error {
	if r.DeezerURL == "" {
		return fmt.Errorf("no Deezer URL available to fetch ISRC")
	}

	var trackID string
	if strings.Contains(r.DeezerURL, "/track/") {
		parts := strings.Split(r.DeezerURL, "/track/")
		if len(parts) > 1 {
			trackID = strings.Split(parts[1], "?")[0]
			trackID = strings.TrimSpace(trackID)
		}
	}

	if trackID == "" {
		return fmt.Errorf("could not extract track ID from Deezer URL: %s", r.DeezerURL)
	}

	apiURL := fmt.Sprintf("https://api.deezer.com/track/%s", trackID)

	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to call Deezer API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Deezer API returned status %d", resp.StatusCode)
	}

	var deezerTrack struct {
		ID    int64  `json:"id"`
		ISRC  string `json:"isrc"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deezerTrack); err != nil {
		return fmt.Errorf("failed to decode Deezer API response: %w", err)
	}

	if deezerTrack.ISRC == "" {
		return fmt.Errorf("ISRC not found in Deezer API response for track %s", trackID)
	}

	fmt.Printf("Found ISRC from Deezer: %s (track: %s)\n", deezerTrack.ISRC, deezerTrack.Title)
	r.ISRC = deezerTrack.ISRC
	return nil
}

func (s *SongLinkClient) checkQobuzAvailability(isrc string) bool {
	qobuzAPIBase, _ := base64.StdEncoding.DecodeString("aHR0cHM6Ly93d3cucW9idXouY29tL2FwaS5qc29uLzAuMi90cmFjay9zZWFyY2g/cXVlcnk9")
	searchURL := fmt.Sprintf("%s%s&limit=1&app_id=%s", string(qobuzAPIBase), isrc, qobuzAppID)

	resp, err := s.client.Get(searchURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var searchResp struct {
		Tracks struct {
			Total int `json:"total"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return false
	}

	return searchResp.Tracks.Total > 0
}

func (s *SongLinkClient) GetAllURLsFromSpotify(spotifyTrackID string, region string) (*SongLinkURLs, error) {
	result, err := s.lookup(spotifyTrackID, region)
	if err != nil {
		return nil, err
	}

	if result.TidalURL == "" && result.AmazonURL == "" {
		return nil, fmt.Errorf("no streaming URLs found")
	}

	urls := &SongLinkURLs{
		TidalURL:  result.TidalURL,
		AmazonURL: result.AmazonURL,
	}

	if result.DeezerURL != "" {
		if err := result.FetchISRC(s.client); err == nil {
			urls.ISRC = result.ISRC
		}
	}

	return urls, nil
}

func (s *SongLinkClient) CheckTrackAvailability(spotifyTrackID string) (*TrackAvailability, error) {
	result, err := s.lookup(spotifyTrackID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	availability := &TrackAvailability{
		SpotifyID: spotifyTrackID,
	}

	if result.TidalURL != "" {
		availability.Tidal = true
		availability.TidalURL = result.TidalURL
	}

	if result.AmazonURL != "" {
		availability.Amazon = true
		availability.AmazonURL = result.AmazonURL
	}

	if result.DeezerURL != "" {
		if err := result.FetchISRC(s.client); err == nil {
			availability.ISRC = result.ISRC
			availability.Qobuz = s.checkQobuzAvailability(result.ISRC)
		}
	}

	return availability, nil
}

func (s *SongLinkClient) GetISRC(spotifyID string) (string, error) {
	result, err := s.lookup(spotifyID, "")
	if err != nil {
		return "", err
	}

	if result.DeezerURL == "" {
		return "", fmt.Errorf("deezer link not found")
	}

	if err := result.FetchISRC(s.client); err != nil {
		return "", err
	}

	return result.ISRC, nil
}

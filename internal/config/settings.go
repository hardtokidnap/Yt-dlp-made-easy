package config

import (
	"encoding/json"
	"os"
	"sync"

	"ytdlp-easy/internal/util"
)

// Settings holds all application configuration
type Settings struct {
	SchemaVersion int              `json:"SchemaVersion"`
	General       GeneralSettings  `json:"General"`
	Download      DownloadSettings `json:"Download"`
	Network       NetworkSettings  `json:"Network"`
	Auth          AuthSettings     `json:"Auth"`
	Advanced      AdvancedSettings `json:"Advanced"`
	mu            sync.RWMutex     `json:"-"`
}

type GeneralSettings struct {
	SaveFolder               string `json:"SaveFolder"`
	Theme                    string `json:"Theme"` // system, light, dark
	MaxConcurrentDownloads   int    `json:"MaxConcurrentDownloads"`
	ClipboardMonitoring      bool   `json:"ClipboardMonitoring"`
	NotificationsEnabled     bool   `json:"NotificationsEnabled"`
	CheckUpdatesOnStart      bool   `json:"CheckUpdatesOnStart"`
}

type DownloadSettings struct {
	Quality        string `json:"Quality"`
	Format         string `json:"Format"`
	AudioFormat    string `json:"AudioFormat"`
	AudioQuality   string `json:"AudioQuality"`
	EmbedThumbnail bool   `json:"EmbedThumbnail"`
	EmbedMetadata  bool   `json:"EmbedMetadata"`
	EmbedChapters  bool   `json:"EmbedChapters"`
	Sponsorblock   bool   `json:"Sponsorblock"`
}

type NetworkSettings struct {
	RateLimit string `json:"RateLimit"`
	Proxy     string `json:"Proxy"`
	Retries   int    `json:"Retries"`
}

type AuthSettings struct {
	CookiesBrowser string `json:"CookiesBrowser"`
	CookiesFile    string `json:"CookiesFile"`
	POToken        string `json:"POToken"`
	PlayerClient   string `json:"PlayerClient"` // web, mweb, web_creator, ios, android
}

type AdvancedSettings struct {
	UseNightly     bool   `json:"UseNightly"`
	OutputTemplate string `json:"OutputTemplate"`
	ExtraArgs      string `json:"ExtraArgs"`
	JSRuntime      string `json:"JSRuntime"` // auto, deno, node, bun, or path
}

// DefaultSettings returns a new Settings instance with default values
func DefaultSettings() *Settings {
	return &Settings{
		SchemaVersion: 1,
		General: GeneralSettings{
			SaveFolder:             util.DefaultDownloadFolder,
			Theme:                  "system",
			MaxConcurrentDownloads: 3,
			ClipboardMonitoring:    true,
			NotificationsEnabled:   true,
			CheckUpdatesOnStart:    true,
		},
		Download: DownloadSettings{
			Quality:        "best",
			Format:         "mp4",
			AudioFormat:    "mp3",
			AudioQuality:   "192",
			EmbedThumbnail: true,
			EmbedMetadata:  true,
			EmbedChapters:  true,
			Sponsorblock:   false,
		},
		Network: NetworkSettings{
			RateLimit: "",
			Proxy:     "",
			Retries:   10,
		},
		Auth: AuthSettings{
			CookiesBrowser: "none",
			CookiesFile:    "",
			POToken:        "",
			PlayerClient:   "default", // default, mweb, web_creator, ios, android
		},
		Advanced: AdvancedSettings{
			UseNightly:     false,
			OutputTemplate: "%(title)s.%(ext)s",
			ExtraArgs:      "",
			JSRuntime:      "auto", // auto-detect, or deno/node/bun
		},
	}
}

// Load loads settings from file, creating defaults if needed
func Load() (*Settings, error) {
	// Check if file exists
	if _, err := os.Stat(util.SettingsFile); os.IsNotExist(err) {
		// Create defaults
		s := DefaultSettings()
		if err := s.Save(); err != nil {
			return nil, err
		}
		return s, nil
	}

	// Read file
	data, err := os.ReadFile(util.SettingsFile)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	s := &Settings{}
	if err := json.Unmarshal(data, s); err != nil {
		// If parse fails, return defaults
		s = DefaultSettings()
		s.Save()
		return s, nil
	}

	// Merge with defaults to ensure all fields exist
	s.mergeDefaults()

	return s, nil
}

// Save writes settings to file
func (s *Settings) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	if err := util.EnsureAppDir(); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(util.SettingsFile, data, 0644)
}

// mergeDefaults ensures all fields have values
func (s *Settings) mergeDefaults() {
	defaults := DefaultSettings()

	// Merge missing fields (simple approach)
	if s.General.SaveFolder == "" {
		s.General.SaveFolder = defaults.General.SaveFolder
	}
	if s.General.Theme == "" {
		s.General.Theme = defaults.General.Theme
	}
	if s.General.MaxConcurrentDownloads == 0 {
		s.General.MaxConcurrentDownloads = defaults.General.MaxConcurrentDownloads
	}
	if s.Download.Quality == "" {
		s.Download.Quality = defaults.Download.Quality
	}
	if s.Download.Format == "" {
		s.Download.Format = defaults.Download.Format
	}
	if s.Download.AudioFormat == "" {
		s.Download.AudioFormat = defaults.Download.AudioFormat
	}
	if s.Download.AudioQuality == "" {
		s.Download.AudioQuality = defaults.Download.AudioQuality
	}
	if s.Advanced.OutputTemplate == "" {
		s.Advanced.OutputTemplate = defaults.Advanced.OutputTemplate
	}
	if s.Network.Retries == 0 {
		s.Network.Retries = defaults.Network.Retries
	}
}

// GetGeneral returns general settings (thread-safe)
func (s *Settings) GetGeneral() GeneralSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.General
}

// SetGeneral updates general settings (thread-safe)
func (s *Settings) SetGeneral(g GeneralSettings) error {
	s.mu.Lock()
	s.General = g
	s.mu.Unlock()
	return s.Save()
}

// Similar getters/setters for other sections...
func (s *Settings) GetDownload() DownloadSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Download
}

func (s *Settings) SetDownload(d DownloadSettings) error {
	s.mu.Lock()
	s.Download = d
	s.mu.Unlock()
	return s.Save()
}

func (s *Settings) GetNetwork() NetworkSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Network
}

func (s *Settings) GetAuth() AuthSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Auth
}

func (s *Settings) GetAdvanced() AdvancedSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Advanced
}

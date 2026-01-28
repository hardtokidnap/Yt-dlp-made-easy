package config

import (
	"encoding/json"
	"os"
	"sync"

	"ytdlp-easy/internal/util"
)

// Settings holds all application configuration
type Settings struct {
	SchemaVersion int              `json:"schema_version"`
	General       GeneralSettings  `json:"general"`
	Download      DownloadSettings `json:"download"`
	Network       NetworkSettings  `json:"network"`
	Auth          AuthSettings     `json:"auth"`
	Advanced      AdvancedSettings `json:"advanced"`
	mu            sync.RWMutex     `json:"-"`
}

type GeneralSettings struct {
	SaveFolder      string `json:"save_folder"`
	Theme           string `json:"theme"` // system, light, dark
	MaxConcurrent   int    `json:"max_concurrent"`
	ClipboardMonitor bool   `json:"clipboard_monitor"`
	Notifications   bool   `json:"notifications"`
	CheckUpdates    bool   `json:"check_updates"`
}

type DownloadSettings struct {
	Quality         string `json:"quality"`
	Format          string `json:"format"`
	AudioFormat     string `json:"audio_format"`
	AudioQuality    string `json:"audio_quality"`
	EmbedThumbnail  bool   `json:"embed_thumbnail"`
	EmbedMetadata   bool   `json:"embed_metadata"`
	EmbedChapters   bool   `json:"embed_chapters"`
	Sponsorblock    bool   `json:"sponsorblock"`
}

type NetworkSettings struct {
	RateLimit string `json:"rate_limit"`
	Proxy     string `json:"proxy"`
	Retries   int    `json:"retries"`
}

type AuthSettings struct {
	CookiesBrowser string `json:"cookies_browser"`
	CookiesFile    string `json:"cookies_file"`
	POToken        string `json:"po_token"`
}

type AdvancedSettings struct {
	UseNightly     bool   `json:"use_nightly"`
	OutputTemplate string `json:"output_template"`
	ExtraArgs      string `json:"extra_args"`
}

// DefaultSettings returns a new Settings instance with default values
func DefaultSettings() *Settings {
	return &Settings{
		SchemaVersion: 1,
		General: GeneralSettings{
			SaveFolder:      util.DefaultDownloadFolder,
			Theme:           "system",
			MaxConcurrent:   3,
			ClipboardMonitor: true,
			Notifications:   true,
			CheckUpdates:    true,
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
		},
		Advanced: AdvancedSettings{
			UseNightly:     false,
			OutputTemplate: "%(title)s.%(ext)s",
			ExtraArgs:      "",
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
	if s.General.MaxConcurrent == 0 {
		s.General.MaxConcurrent = defaults.General.MaxConcurrent
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

package cmd

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/cmd-tools/aws-commander/logger"
	"gopkg.in/yaml.v2"
)

const settingsFileName = "settings.yaml"

// SettingsStore holds the user-configurable settings, persisted to disk.
type SettingsStore struct {
	Animations bool `yaml:"animations"` // Show ASCII animations on startup and resource switching (default: false)
	mu         sync.Mutex
	path       string
}

// Settings is the global singleton for user settings.
var Settings = &SettingsStore{}

// Load reads the settings file from disk. If it doesn't exist, starts with defaults (animations: false).
func (s *SettingsStore) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := configDir()
	if dir == "" {
		return
	}

	s.path = filepath.Join(dir, settingsFileName)

	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Logger.Error().Err(err).Msg("Failed to read settings file")
		}
		return
	}

	if err := yaml.Unmarshal(data, s); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse settings file")
	}
}

// Save writes the settings to disk.
func (s *SettingsStore) Save() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		dir := configDir()
		if dir == "" {
			return
		}
		s.path = filepath.Join(dir, settingsFileName)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to create config directory")
		return
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to marshal settings")
		return
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to write settings file")
	}
}

// AnimationsEnabled returns whether ASCII animations are enabled.
func (s *SettingsStore) AnimationsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Animations
}

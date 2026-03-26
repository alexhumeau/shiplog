package collector

import (
	"encoding/json"
	"os"
	"time"
)

// State tracks the last processed commit for CLI mode.
type State struct {
	LastSHA string    `json:"last_sha"`
	LastRun time.Time `json:"last_run"`
}

// LoadState reads .shiplog-state.json. Returns nil if the file doesn't exist.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveState writes the state file after a successful run.
func SaveState(path string, sha string) error {
	state := State{
		LastSHA: sha,
		LastRun: time.Now(),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

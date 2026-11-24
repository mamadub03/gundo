package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// when gundo agent is restarted, we lose the target's (clientID) + agent communication channel (IssueNumber)
// this creates a state for gundo agent to find and use info on itself

type ClientState struct {
	ClientID    string `json:"client_id"`
	IssueNumber int    `json:"issue_number"`
}

// helper for finding and creating gundo directory
// windows: C:\Users\<Username>\AppData\Roaming,
// linux: $XDG_CONFIG_HOME -> /home/<username>/.config,
// macOS: $HOME/Library/ApplicationSupport -> /Users/<user>/Library/ApplicationSupport
func getStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve config directory: %w", err)
	}

	// create a gundo config directory filepath in memory
	gundoDir := filepath.Join(configDir, "gundo")

	if err := os.MkdirAll(gundoDir, 0700); err != nil {
		return "", fmt.Errorf("Failed to create gundo directory: %w", err)
	}
	return filepath.Join(gundoDir, "state.json"), nil
}

// load in previous state
func LoadState() (ClientState, error) {

	statePath, err := getStatePath()
	if err != nil {
		return ClientState{}, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return ClientState{}, err
	}

	var state ClientState

	if err := json.Unmarshal(data, &state); err != nil {
		return ClientState{}, fmt.Errorf("Error parsing client state: %w", err)
	}

	return state, nil
}

func SaveState(state ClientState) error {
	statePath, err := getStatePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling client state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("Error writing client state: %w", err)
	}

	return nil
}

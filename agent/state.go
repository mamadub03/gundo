package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// when gundo agent is restarted, we lose the target's (clientID) + agent communication channel (IssueNumber)
// this creates a state for gundo agent to find and use info on itself

type ClientState struct {
	ClientID    string `json:"client_id"`
	IssueNumber int    `json:"issue_number"`
	Repo        string `json:"repo,omitempty"`
	Token       string `json:"token,omitempty"`
}

type SetupPayload struct {
	Repo  string `json:"repo"`
	Token string `json:"token"`
}

// FindTempSetupFile checks for temporary configuration files in standard locations.
func FindTempSetupFile() (string, error) {
	// 1. Explicit environment override
	if envPath := strings.TrimSpace(os.Getenv("GUNDO_SETUP_FILE")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	candidates := []string{
		filepath.Join(os.TempDir(), "gundo_setup.json"),
		"gundo_setup.json",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

// ReadAndConsumeSetupFile reads the setup payload and immediately removes the temp file from disk.
func ReadAndConsumeSetupFile(path string) (SetupPayload, error) {
	data, err := os.ReadFile(path)
	// Immediately remove the setup file so credentials don't linger on disk under any circumstance
	defer os.Remove(path)

	if err != nil {
		return SetupPayload{}, fmt.Errorf("failed to read setup file %s: %w", path, err)
	}

	var payload SetupPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return SetupPayload{}, fmt.Errorf("failed to parse setup file JSON: %w", err)
	}

	payload.Repo = strings.TrimSpace(payload.Repo)
	payload.Token = strings.TrimSpace(payload.Token)

	if payload.Repo == "" || payload.Token == "" {
		return SetupPayload{}, fmt.Errorf("setup file must specify both 'repo' and 'token'")
	}

	return payload, nil
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

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return ClientState{}, os.ErrNotExist
	}

	var state ClientState

	if err := json.Unmarshal(data, &state); err != nil {
		return ClientState{}, fmt.Errorf("Error parsing client state: %w", err)
	}

	// If the state file has no client ID or issue number, treat as non-existent/empty
	if state.ClientID == "" && state.IssueNumber == 0 {
		return ClientState{}, os.ErrNotExist
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

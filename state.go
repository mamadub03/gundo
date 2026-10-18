package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ClientState struct {
	ClientID    string `json:"client_id"`
	IssueNumber int    `json:"issue_number"`
}

func getStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config directory: %w", err)
	}

	gundoDir := filepath.Join(configDir, "gundo")

	if err := os.MkdirAll(gundoDir, 0700); err != nil {
		return "", fmt.Errorf("create gundo directory: %w", err)
	}

	return filepath.Join(gundoDir, "state.json"), nil
}

func loadState() (ClientState, error) {
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
		return ClientState{}, fmt.Errorf("parse client state: %w", err)
	}

	return state, nil
}

func saveState(state ClientState) error {
	statePath, err := getStatePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal client state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("write client state: %w", err)
	}

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// when gundo is to restart, we would lose the who am I (clientID) + communication channel (IssueNumber)
// this creates a states for gundo to find and use info on itself

type ClientState struct {
	ClientID    string `json:"client_id"`
	IssueNumber int    `json:"issue_number"`
}

// helper used to figure out where the local state is stored
func getStatePath() (string, error) {

	// asks OS for user's config ddirectory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config directory: %w", err)
	}

	// join the config directory to
	gundoDir := filepath.Join(configDir, "gundo")

	// create gundoDir folder and return the filepath + folder
	if err := os.MkdirAll(gundoDir, 0700); err != nil {
		return "", fmt.Errorf("create gundo directory: %w", err)
	}
	return filepath.Join(gundoDir, "state.json"), nil
}

// allows a restarted process to find it's state by loading it in
func loadState() (ClientState, error) {

	// grabs the state path with helper
	statePath, err := getStatePath()
	if err != nil {
		return ClientState{}, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return ClientState{}, err
	}

	var state ClientState

	// decodes and adds into JSON struct (in mem due to pointer) then returns state
	if err := json.Unmarshal(data, &state); err != nil {
		return ClientState{}, fmt.Errorf("parse client state: %w", err)
	}

	return state, nil
}

// opposite of load state in case the state file gets lost/dne
func saveState(state ClientState) error {
	statePath, err := getStatePath()
	if err != nil {
		return err
	}

	// marshalls the values of state into json
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal client state: %w", err)
	}

	// writes the json to the file

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("write client state: %w", err)
	}

	return nil
}

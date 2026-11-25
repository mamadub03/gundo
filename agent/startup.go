package agent

import (
	"fmt"
	"os"
	"strings"
	"uuid"

	"gundo/shared"
)

func initializeAgentCredentials(savedState *ClientState) error {
	// 1. If state already has stored credentials, use them directly
	if savedState != nil && savedState.Token != "" && savedState.Repo != "" {
		shared.SetGitHubCredentials(savedState.Token, savedState.Repo)
		return nil
	}

	// 2. If credentials are already initialized in memory, keep them
	if shared.CurrentGitHubToken() != "" && shared.CurrentGitHubRepo() != "" {
		return nil
	}

	// 3. Check for a temporary setup file (/tmp/gundo_setup.json or ./gundo_setup.json)
	if setupPath, err := FindTempSetupFile(); err == nil {
		payload, err := ReadAndConsumeSetupFile(setupPath)
		if err != nil {
			return fmt.Errorf("error processing setup file: %w", err)
		}
		parsedRepo, err := shared.ParseGitHubRepo(payload.Repo)
		if err != nil {
			return fmt.Errorf("invalid repository in setup file: %w", err)
		}
		shared.SetGitHubCredentials(payload.Token, parsedRepo)
		fmt.Printf("[+] Ingested and removed temporary setup file: %s\n", setupPath)
		return nil
	}

	// 4. Check for environment variables
	envRepo := strings.TrimSpace(os.Getenv("GUNDO_REPO"))
	if envRepo == "" {
		envRepo = strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	}
	envToken := strings.TrimSpace(os.Getenv("GUNDO_TOKEN"))
	if envToken == "" {
		envToken = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	if envRepo != "" && envToken != "" {
		parsedRepo, err := shared.ParseGitHubRepo(envRepo)
		if err != nil {
			return fmt.Errorf("invalid repository in environment: %w", err)
		}
		shared.SetGitHubCredentials(envToken, parsedRepo)
		return nil
	}

	// 5. Fallback to shared keyring/setup initialization (interactive dev mode)
	if err := shared.InitializeGitHubToken(); err == nil {
		return nil
	}

	return fmt.Errorf("no GitHub credentials found. Provide /tmp/gundo_setup.json, set GITHUB_REPO/GITHUB_TOKEN, or run setup")
}

func Run() {
	var state ClientState

	savedState, err := LoadState()

	if err == nil {
		state = savedState

		// Clean up any stale temporary setup file if one is still present
		if setupPath, findErr := FindTempSetupFile(); findErr == nil {
			_ = os.Remove(setupPath)
		}

		if err := initializeAgentCredentials(&state); err != nil {
			fmt.Printf("Failed to initialize credentials: %v\n", err)
			return
		}

		// Backfill state if an older state file lacked repo/token
		if state.Repo == "" || state.Token == "" {
			state.Repo = shared.CurrentGitHubRepo()
			state.Token = shared.CurrentGitHubToken()
			_ = SaveState(state)
		}

		fmt.Printf("[+] Loaded existing client\n")
		fmt.Printf("[+] Client ID: %s\n", state.ClientID)
		fmt.Printf("[+] Mailbox: Issue #%d\n", state.IssueNumber)

	} else if os.IsNotExist(err) {

		fmt.Println("[+] First run detected")

		if err := initializeAgentCredentials(nil); err != nil {
			fmt.Printf("Failed to initialize credentials: %v\n", err)
			return
		}

		clientID := uuid.New().String()

		issue := shared.IssueRequest{
			Title: "Gundo Client - " + clientID,
			Body:  "Gundo Client mailbox",
		}

		createdIssue, err := shared.CreateIssue(issue)
		if err != nil {
			fmt.Printf("Failed to create mailbox: %v\n", err)
			return
		}

		state = ClientState{
			ClientID:    clientID,
			IssueNumber: createdIssue.Number,
			Repo:        shared.CurrentGitHubRepo(),
			Token:       shared.CurrentGitHubToken(),
		}

		if err := SaveState(state); err != nil {
			fmt.Printf("Failed to save client state: %v\n", err)
			return
		}

		fmt.Printf("[+] Created Client ID: %s\n", state.ClientID)
		fmt.Printf("[+] Created Mailbox: Issue #%d\n", state.IssueNumber)

	} else {
		fmt.Printf("Failed to load state: %v\n", err)
		return
	}

	client := NewClient(state.ClientID)

	// goroutine to heartbeat concurrently with poll tasks
	go client.HeartbeatLoop(state.IssueNumber)

	client.PollTasks(state.IssueNumber)
}

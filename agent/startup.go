package agent

import (
	"fmt"
	"os"
	"uuid"

	"gundo/shared"
)

func Run() {
	var state ClientState

	savedState, err := LoadState()

	if err == nil {
		state = savedState

		fmt.Printf("[+] Loaded existing client\n")
		fmt.Printf("[+] Client ID: %s\n", state.ClientID)
		fmt.Printf("[+] Mailbox: Issue #%d\n", state.IssueNumber)

	} else if os.IsNotExist(err) {

		fmt.Println("[+] First run detected")

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

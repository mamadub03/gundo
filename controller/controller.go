package controller

import (
	"fmt"
	"strings"
	"time"

	"gundo/shared"
)

// ManagedClient represents the controller's known info about a client.
type ManagedClient struct {
	ClientID    string
	IssueNumber int
	LastSeen    time.Time
	Online      bool
}

// Discovery retains only a bookmark and the latest heartbeat per mailbox,
// rather than keeping every historical comment in memory.
type discoveryMailbox struct {
	ClientID   string
	Checkpoint shared.CommentCheckpoint
	LastSeen   time.Time
}

func discoverClients(mailboxes map[int]discoveryMailbox) ([]ManagedClient, error) {
	// list all the issues representing the clients
	issues, err := shared.ListIssues()
	if err != nil {
		return nil, err
	}
	return discoverClientsFromIssues(issues, mailboxes, shared.ListCommentsSince)
}

func discoverClientsFromIssues(issues []shared.IssueResponse, mailboxes map[int]discoveryMailbox,
	list func(int, time.Time) ([]shared.IssueComment, error)) ([]ManagedClient, error) {
	var clients []ManagedClient

	const clientTitlePrefix = "Gundo Client - "
	const offlineThreshold = 6 * time.Minute

	for _, issue := range issues {
		// Ignore Issues that do not belong to Gundo clients.
		if !strings.HasPrefix(issue.Title, clientTitlePrefix) {
			continue
		}

		clientID := strings.TrimPrefix(issue.Title, clientTitlePrefix)
		mailbox := mailboxes[issue.Number]
		if mailbox.ClientID != clientID {
			mailbox = discoveryMailbox{ClientID: clientID}
		}

		managedClient := ManagedClient{
			ClientID:    clientID,
			IssueNumber: issue.Number,
			LastSeen:    mailbox.LastSeen,
			Online:      false,
		}

		comments, err := mailbox.Checkpoint.Poll(issue.Number, list)
		if err != nil {
			fmt.Printf(
				"Could not read Issue #%d: %s\n",
				issue.Number,
				consoleLabel(err.Error()),
			)

			clients = append(clients, managedClient)
			continue
		}

		// Search newest → oldest because we only care about the latest heartbeat.
		for i := len(comments) - 1; i >= 0; i-- {
			message, err := shared.ParseGundoMessage(comments[i].Body)
			if err != nil {
				continue
			}

			if message.Type != "heartbeat" {
				continue
			}

			if message.ClientID != clientID {
				continue
			}

			heartbeatTime, err := time.Parse(time.RFC3339, message.Timestamp)
			if err != nil {
				continue
			}

			managedClient.LastSeen = heartbeatTime
			break
		}

		mailbox.LastSeen = managedClient.LastSeen
		mailboxes[issue.Number] = mailbox
		managedClient.Online = !managedClient.LastSeen.IsZero() && time.Since(managedClient.LastSeen) < offlineThreshold

		clients = append(clients, managedClient)
	}

	return clients, nil
}

func waitForResult(issueNumber int, taskID string, checkpoint shared.CommentCheckpoint) (shared.GundoMessage, error) {
	return waitForResultWithPolling(issueNumber, taskID, checkpoint, shared.ListCommentsSince, time.Sleep)
}

func waitForResultWithPolling(issueNumber int, taskID string, checkpoint shared.CommentCheckpoint,
	list func(int, time.Time) ([]shared.IssueComment, error), sleep func(time.Duration)) (shared.GundoMessage, error) {
	for {
		comments, err := checkpoint.Poll(issueNumber, list)
		if err != nil {
			return shared.GundoMessage{}, err
		}

		for _, comment := range comments {
			message, err := shared.ParseGundoMessage(comment.Body)
			if err != nil {
				continue
			}

			if message.Type != "result" {
				continue
			}

			if message.TaskID != taskID {
				continue
			}

			return message, nil
		}

		sleep(5 * time.Second)
	}
}

func runCommandMenu(console *operatorConsole, client ManagedClient) {
	labels := []string{"Ping", "Get hostname", "Get operating system", "Get current user", "Custom command…", "Shutdown…", "Back to agents"}
	actions := []string{"ping", "get_hostname", "get_os", "get_user", "custom", "shutdown"}

	console.screen()
	console.agentHeading(client)
	for {
		choice, err := console.selectItem("Select an action", labels)
		if err != nil || choice == len(actions) {
			return
		}
		action := actions[choice]
		if action == "custom" {
			runCustomPrompt(console, client)
			continue
		}
		command := ""
		if action == "shutdown" {
			answer, err := console.line(fmt.Sprintf("  Shut down agent on Issue #%d? Type yes: ", client.IssueNumber))
			if err != nil {
				return
			}
			if answer != "yes" {
				continue
			}
		}
		task := shared.CreateTaskMessage(action)
		task.TargetClientID = client.ClientID
		task.Command = command
		started := time.Now()
		fmt.Fprintf(console.out, "\n  Sending %s · Issue #%d\n", labels[choice], client.IssueNumber)
		if command != "" {
			fmt.Fprintf(console.out, "  Command  %s\n", consoleLabel(command))
		}
		fmt.Fprintf(console.out, "  Task     %s\n", task.TaskID)
		createdTask, err := shared.CreateCommentWithMetadata(client.IssueNumber, task)
		if err != nil {
			console.failure(err)
			continue
		}
		fmt.Fprintln(console.out, console.style("33", "  Task posted. Waiting for result…"))
		result, err := waitForResult(client.IssueNumber, task.TaskID,
			shared.CommentCheckpoint{LastSeenCommentID: createdTask.ID, CreatedAt: createdTask.CreatedAt})
		if err != nil {
			console.failure(err)
			continue
		}
		console.result(labels[choice], task.TaskID, result, time.Since(started))
	}
}

func runCustomPrompt(console *operatorConsole, client ManagedClient) {
	fmt.Fprintf(console.out, "\n  %s\n", console.style("1;36", fmt.Sprintf("CUSTOM COMMANDS · Agent #%d", client.IssueNumber)))
	fmt.Fprintln(console.out, console.style("90", "  Type commands to run on the agent. Type 'exit' or 'quit' to return to menu."))

	prompt := fmt.Sprintf("\nAgent #%d (%s) > ", client.IssueNumber, client.ClientID)
	for {
		cmd, err := console.line(prompt)
		if err != nil {
			return
		}
		if cmd == "" {
			continue
		}
		if cmd == "exit" || cmd == "quit" {
			return
		}

		task := shared.CreateTaskMessage("custom")
		task.TargetClientID = client.ClientID
		task.Command = cmd
		started := time.Now()

		fmt.Fprintf(console.out, "  Task     %s\n", task.TaskID)
		createdTask, err := shared.CreateCommentWithMetadata(client.IssueNumber, task)
		if err != nil {
			console.failure(err)
			continue
		}
		fmt.Fprintln(console.out, console.style("33", "  Task posted. Waiting for result…"))
		result, err := waitForResult(client.IssueNumber, task.TaskID,
			shared.CommentCheckpoint{LastSeenCommentID: createdTask.ID, CreatedAt: createdTask.CreatedAt})
		if err != nil {
			console.failure(err)
			continue
		}
		console.result(cmd, task.TaskID, result, time.Since(started))
	}
}

// Run starts the interactive operator console controller.
func Run() {
	console := newOperatorConsole()
	mailboxes := make(map[int]discoveryMailbox)
	showOffline := true
	for {
		console.screen()
		fmt.Fprintln(console.out, "\n  Discovering agents…")
		clients, err := discoverClients(mailboxes)
		if err != nil {
			console.failure(err)
			choice, selectErr := console.selectItem("Discovery failed", []string{"Retry discovery", "Quit"})
			if selectErr != nil || choice == 1 {
				return
			}
			continue
		}
		firstView := true
		notice := ""
		for {
			if !firstView {
				console.screen()
			}
			firstView = false
			online := 0
			for _, client := range clients {
				if client.Online {
					online++
				}
			}
			fmt.Fprintf(console.out, "\n  AGENTS  ·  %s  ·  %s\n", console.style("32", fmt.Sprintf("%d online", online)), console.style("32", fmt.Sprintf("%d offline", len(clients)-online)))
			if notice != "" {
				fmt.Fprintln(console.out, "  "+notice)
				notice = ""
			}
			visible := visibleAgents(clients, showOffline)
			if len(visible) == 0 {
				fmt.Fprintln(console.out, "  No agents in this view. Refresh or show offline agents.")
			}
			labels := console.agentLabels(visible)
			toggle := "Show offline agents"
			if showOffline {
				toggle = "Hide offline agents"
			}
			labels = append(labels, "Refresh agents", toggle, "Quit")
			choice, err := console.selectItem("Select an agent", labels)
			if err != nil || choice == len(visible)+2 {
				return
			}
			if choice == len(visible)+1 {
				showOffline = !showOffline
				continue
			}
			if choice == len(visible) {
				break
			}
			selected := visible[choice]
			if !selected.Online {
				notice = "This agent is offline. Refresh after it starts sending heartbeats."
				continue
			}
			runCommandMenu(console, selected)
			break
		}
	}
}

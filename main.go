package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/kkirabo/c2prototype/ghapi"
	"github.com/kkirabo/c2prototype/server"
)

// c2 operator
const (
	ghOwner = "kkirabo"
	ghRepo  = "c2prototype"
)

// syntax helper
func usage() {
	fmt.Println("Usage:")
	fmt.Println("  new <session-name>       Start a new session (creates a GitHub issue)")
	fmt.Println("  resume <issue-number>    Resume an existing session")
	os.Exit(1)
}

func main() {
	// require a GitHub PAT env — never hardcode tokens
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("[!] GITHUB_TOKEN environment variable is not set")
	}

	// read cmdline arguments after [0] and quit if none are given
	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	// init github api client and add to server
	gh := ghapi.NewClient(token, ghOwner, ghRepo)
	srv := server.New(gh)

	// evaluates all the args after './program_name'
	switch args[0] {
	case "new":
		name := "C2 Session"
		if len(args) > 1 {
			name = args[1]
		}
		// catch any errors when starting a session
		if err := srv.StartSession(name); err != nil {
			log.Fatalf("[!] %s", err)
		}

	case "resume":
		if len(args) < 2 {
			fmt.Println("[!] Usage: resume <issue-number>")
			usage()
		}
		num, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("[!] Invalid issue number %q: %s", args[1], err)
		}
		if err := srv.ResumeSession(num); err != nil {
			log.Fatalf("[!] %s", err)
		}
	default: // protect against wrong arguments
		fmt.Printf("[!] Unknown command: %q\n", args[0])
		usage()
	}
}

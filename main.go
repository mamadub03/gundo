package main

import (
	"fmt"
	"os"

	"gundo/agent"
	"gundo/controller"
	"gundo/shared"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . client")
		fmt.Println("  go run . controller")
		fmt.Println("  go run . setup  (configure or replace GitHub repository and token)")
		return
	}

	switch os.Args[1] {
	case "client", "agent":
		agent.Run()

	case "controller":
		if err := shared.InitializeGitHubToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Authentication setup failed: %v\n", err)
			os.Exit(1)
		}
		controller.Run()

	case "setup":
		// Credential replacement does not load or change the client's identity file.
		if err := shared.SetupGitHubToken(); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Println("Unknown mode:", os.Args[1])
	}
}

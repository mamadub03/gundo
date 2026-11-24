package main

import (
	"fmt"
	"os"

	"gundo/controller"
	"gundo/shared"
)

func main() {
	if err := shared.InitializeGitHubToken(); err != nil {
		fmt.Fprintf(os.Stderr, "Authentication setup failed: %v\n", err)
		os.Exit(1)
	}
	controller.Run()
}

package main

import (
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
)

func main() {
	issue := IssueRequest{
		Title: "Test Issue",
		Body:  "This is a test issue created via the GitHub API.",
	}

	jsonData, err := json.Marshal(issue)

	// common pattern to use for marshalling as we cant have unused vars
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
	}

	// string() turns it into a json string
	fmt.Println(string(jsonData))

	// request type, repositiory, bytes wrapped in a way http can udnerstand
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/repos/mamadub03/gundo-test-repo/issues", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// nothing is being sent yet, just create the req object
	fmt.Println("Method:", req.Method)
	fmt.Println("URL:", req.URL.String())

	// request headers (non secret first)

	req.Header.Set("Header-Name", "Header-Value")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	fmt.Println("Headers:", req.Header)
}

type IssueRequest struct {
	// using JSON tags to use lowercase keys when struct becomes json
	Title       string `json:"title"`
	Body        string `json:"body"`
}
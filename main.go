package main

import (
	"fmt"
	// "net/http"
	// "bytes"
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

}

type IssueRequest struct {
	// using JSON tags to use lowercase keys when struct becomes json
	Title       string `json:"title"`
	Body        string `json:"body"`
}
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {

	issue := IssueRequest{
		Title: "A new C2 Test",
		Body:  "This is an issue created by go >:) for the 2nd time",
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

	token := os.Getenv("GITHUB_TOKEN")

	if token == "" {
		fmt.Println("Error: GITHUB_TOKEN environment variable is not set.")
		return
	}

	// token from env var
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	fmt.Println("Status:", resp.Status)

	// we need to close the response body after we are done with it to free up resources
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {

		var gitHubError GitHubErrorResponse

		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)

		// github error could fail at decode so add this check
		if err != nil {
			fmt.Println("Error decoding GitHub error response:", err)
			return
		}

		fmt.Println("Github request failed: ", resp.Status)
		fmt.Println("Error message from GitHub:", gitHubError.Message)

		return
	}
	// when we recieve an HTTP response, we only get the body and resp headers
	// we need to decode the body into JSON to get the values
	// then we can add it to the struct we created to hold the values we want to use

	// decoder object that is connected to resp.Body
	decoder := json.NewDecoder(resp.Body)
	var createdIssue IssueResponse

	// reads the json and fills the values
	err = decoder.Decode(&createdIssue)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	fmt.Println("Created Issue Number:", createdIssue.Number)
	fmt.Println("Created Issue Title:", createdIssue.Title)
	fmt.Println("Created Issue URL:", createdIssue.HTMLURL)

}

type IssueRequest struct {
	// using JSON tags to use lowercase keys when struct becomes json
	Title string `json:"title"`
	Body  string `json:"body"`
}

type IssueResponse struct {
	// we dont need to use all the fields, just the ones we want to use
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

type GitHubErrorResponse struct {
	Message string `json:"message"`
}

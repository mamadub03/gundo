package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func createIssue(issue IssueRequest) (IssueResponse, error) {

	// turn the issue struct to JSON
	jsonData, err := json.Marshal(issue)
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error marshalling JSON: %w", err)
	}

	fmt.Println(string(jsonData))

	// create a new HTTP requqest to the API endpoint
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/repos/mamadub03/gundo-test-repo/issues", bytes.NewBuffer(jsonData))
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// set the necessary headers for the request
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return IssueResponse{}, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// setting up the client to send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 201 failure check handler
	if resp.StatusCode != http.StatusCreated {
		var gitHubError GitHubErrorResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)
		if err != nil {
			return IssueResponse{}, fmt.Errorf("error decoding GitHub error response: %w", err)
		}
		return IssueResponse{}, fmt.Errorf("GitHub request failed: %s, message: %s", resp.Status, gitHubError.Message)
	}

	// create a decoder to read the response body and decode it into the IssueResponse struct
	var createdIssue IssueResponse
	decoder := json.NewDecoder(resp.Body)

	// decode the JSON response into the createdIssue struct (points to the struct addy)
	err = decoder.Decode(&createdIssue)
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error decoding JSON: %w", err)
	}

	return createdIssue, nil
}

func getIssue(issueNumber int) (IssueResponse, error) {

	url := fmt.Sprintf("https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d", issueNumber)

	// create a new HTTP request to the API endpoint
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// set the necessary headers for the request
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return IssueResponse{}, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// setting up the client to send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return IssueResponse{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 200 failure check handler
	if resp.StatusCode != http.StatusOK {
		var gitHubError GitHubErrorResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)
		if err != nil {
			return IssueResponse{}, fmt.Errorf("error decoding GitHub error response: %w", err)
		}
		return IssueResponse{}, fmt.Errorf("GitHub request failed: %s, message: %s", resp.Status, gitHubError.Message)
	}

	// create a decoder to read the response body and decode it into the IssueResponse struct
	var issue IssueResponse
	decoder := json.NewDecoder(resp.Body)

	// decode the JSON response into the issue struct (points to the struct addy)
	err = decoder.Decode(&issue)

	if err != nil {
		return IssueResponse{}, fmt.Errorf("error decoding JSON: %w", err)
	}

	return issue, nil

}

func main() {

	issue := IssueRequest{
		Title: "tes t etststtas da",
		Body:  "This is an issue nebw by go >:) for the 2nd time",
	}

	createdIssue, err := createIssue(issue)
	if err != nil {
		fmt.Printf("Error creating issue: %v\n", err)
		return
	}

	retrievedIssue, err := getIssue(createdIssue.Number)

	if err != nil {
		fmt.Printf("Error getting issue: %v\n", err)
		return
	}

	fmt.Printf("Issue created successfully! Number: %d, Title: %s, URL: %s\n", createdIssue.Number, createdIssue.Title, createdIssue.HTMLURL)
	fmt.Printf("Retrieved issue! Number: %d, Title: %s, URL: %s\n", retrievedIssue.Number, retrievedIssue.Title, retrievedIssue.HTMLURL)

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

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

func listComments(issueNumber int) ([]IssueComment, error) {

	url := fmt.Sprintf(
		"https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d/comments?per_page=100",
		issueNumber,
	)

	// create a new HTTP request to the API endpoint
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// set the necessary headers for the request
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// setting up the client to send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 200 failure check handler
	if resp.StatusCode != http.StatusOK {
		var gitHubError GitHubErrorResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)
		if err != nil {
			return nil, fmt.Errorf("error decoding GitHub error response: %w", err)
		}
		return nil, fmt.Errorf("GitHub request failed: %s, message: %s", resp.Status, gitHubError.Message)
	}

	// create a decoder to read the response body and decode it into a slice of IssueComment structs
	var comments []IssueComment
	decoder := json.NewDecoder(resp.Body)

	// decode the JSON response into the comments slice
	// comments --> [0]: ID, Body -->[1]: ID, Body -->[2]: ID, Body....
	// can get comments[0].ID or body etc

	err = decoder.Decode(&comments)

	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return comments, nil

}

func createComment(issueNumber int, message GundoMessage) error {

	// FLOW: take GundoMessage --> marshal to JSON --> create comment request struct --> marshal to JSON --> send POST request to GitHub API

	// marshal GundoMessage to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %w", err)
	}

	// wrap the JSON data into a CommentRequest struct
	commentRequest := CommentRequest{
		Body: string(jsonData),
	}

	// marshal the CommentRequest struct to JSON to be sent in the request body
	commentJSON, err := json.Marshal(commentRequest)
	if err != nil {
		return fmt.Errorf("error marshalling comment JSON: %w", err)
	}

	// post process to put comment on the issue number
	url := fmt.Sprintf("https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d/comments", issueNumber)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(commentJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// setting up the client to send the request and get the response to ensure the comment was created successfully
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var gitHubError GitHubErrorResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)
		if err != nil {
			return fmt.Errorf("error decoding GitHub error response: %w", err)
		}
		return fmt.Errorf("GitHub request failed: %s, message: %s", resp.Status, gitHubError.Message)
	}

	return nil
}

func listIssues() ([]IssueResponse, error) {

	// grab issues that are available and represent each agent

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not set")
	}

	url := "https://api.github.com/repos/mamadub03/gundo-test-repo/issues?state=open&per_page=100"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gundo")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error listing issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var githubErr GitHubErrorResponse

		if err := json.NewDecoder(resp.Body).Decode(&githubErr); err != nil {
			return nil, fmt.Errorf(
				"GitHub returned status %d",
				resp.StatusCode,
			)
		}

		return nil, fmt.Errorf(
			"GitHub error: %s",
			githubErr.Message,
		)
	}

	var issues []IssueResponse

	// decode results on the response body into issues struct if not give erro

	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("error decoding issues: %w", err)
	}

	return issues, nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"uuid"
)

func generateTaskID() string {
	// generate a new UUID
	return uuid.New().String()
}

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

	url := fmt.Sprintf("https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d/comments", issueNumber)

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

func pollComments(issueNumber int) {

	// keeing track of the last seen comment
	var lastSeenCommentID int64

	// grab list of comments on an issue number
	comments, err := listComments(issueNumber)
	if err != nil {
		fmt.Printf("Error listing comments: %v\n", err)
		return
	}

	// make sure there are comments to set the baseline
	if len(comments) > 0 {
		lastSeenCommentID = comments[len(comments)-1].ID // set the last seen comment to the last comment in the list
	}
	for {

		comments, err = listComments(issueNumber)

		if err != nil {
			fmt.Printf("Error listing comments: %v\n", err)
			return
		}

		// 1. checks for new comments to update comments slice
		// 2. if the comment id is greater than the las seen comment, new comment is detected
		// 3. sets that new comment as the last seen comment and prints it out
		// IMPORTANT 4. this loop can get very slow so this can be an optimization point later.
		for _, comment := range comments {
			if comment.ID > lastSeenCommentID {
				fmt.Printf("New comment detected! ID: %d, Body: %s\n", comment.ID, comment.Body)
				lastSeenCommentID = comment.ID
			}
		}

		time.Sleep(10 * time.Second) // Poll every 10 seconds
	}
}

func createTaskMessage(action string) GundoMessage {

	// create a new GundoMessage struct with the provided action and a generated task ID
	message := GundoMessage{
		Type:      "task",
		TaskID:    generateTaskID(),
		Action:    action,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	return message
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

	pollComments(17)
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

type IssueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

type GundoMessage struct {

	// struct for the gundo message that will be sent to the github issue comment
	// omitemtpy is used to not include JSON when marshaling if the field is empty
	Type      string `json:"type"`
	TaskID    string `json:"task_id"`
	Action    string `json:"action,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

type CommentRequest struct {
	// struct for the comment request that will be sent to the github issue comment
	Body string `json:"body"`
}

type GitHubErrorResponse struct {
	Message string `json:"message"`
}

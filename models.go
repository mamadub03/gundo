package main

import "time"

// represents the data gundo sends to github when creating an issue
// go converts into json when used
type IssueRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// represents the data gundo recieves back when an issue was created and extracts number, title, and url
// go recieves the json and breaks it into a struct data type
type IssueResponse struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

// recieves the issue comment with comment info/metadata
type IssueComment struct {
	// keeps track of what has been processed
	ID   int64  `json:"id"`
	// message of comment (will contain gundo message)
	// gundo message still a string and parsed later
	Body string `json:"body"`
	// Use GitHub's creation time for retrieval, not the message author's clock.
	CreatedAt time.Time `json:"created_at"`
}

// structured communication language for gundo
type GundoMessage struct {

	// type of message, either "task", "result", or "heartbeat"
	Type   string `json:"type"`

	// unique identifier for the task, used to match results to tasks
	TaskID string `json:"task_id,omitempty"`

	// used in going from controller to client
	TargetClientID string `json:"target_client_id,omitempty"`

	// used in going from client to controller
	ClientID string `json:"client_id,omitempty"`

	Action string `json:"action,omitempty"`
	Input  string `json:"input,omitempty"`

	//using command to have cleaner field for custom commands
	Command string `json:"command,omitempty"`

	Output string `json:"output,omitempty"`
	Status string `json:"status,omitempty"`

	OS            string `json:"os,omitempty"`
	Version       string `json:"version,omitempty"`
	// when the agent was created
	StartedAt     string `json:"started_at,omitempty"`

	// not implemented at the moment
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`

	// timestamp of when the message was created, in RFC3339 format
	Timestamp string `json:"timestamp"`
}

type CommentRequest struct {
	Body string `json:"body"`
}

type GitHubErrorResponse struct {
	Message string `json:"message"`
}

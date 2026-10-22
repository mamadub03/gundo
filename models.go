package main

type IssueRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type IssueResponse struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}

type IssueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// let's create a matrix on who does what
// even better later we can seperate the types of messages as it is growing

type GundoMessage struct {
	Type   string `json:"type"`
	TaskID string `json:"task_id,omitempty"`

	// used in going from controller to client
	TargetClientID string `json:"target_client_id,omitempty"`

	// used in going from client to controller
	ClientID string `json:"client_id,omitempty"`

	Action string `json:"action,omitempty"`
	Input  string `json:"input,omitempty"`

	Output string `json:"output,omitempty"`
	Status string `json:"status,omitempty"`

	OS            string `json:"os,omitempty"`
	Version       string `json:"version,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	UptimeSeconds int64  `json:"uptime_seconds,omitempty"`

	Timestamp string `json:"timestamp"`
}

type CommentRequest struct {
	Body string `json:"body"`
}

type GitHubErrorResponse struct {
	Message string `json:"message"`
}

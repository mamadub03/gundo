package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
	return listCommentsSince(issueNumber, time.Time{})
}

// A zero boundary keeps full-history reads available for the initial baseline.
func listCommentsSince(issueNumber int, since time.Time) ([]IssueComment, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	return listCommentsSinceWithClient(issueNumber, &http.Client{}, token, since)
}

// The injected client lets tests exercise real pagination with in-memory HTTP
// responses, without reading credentials or contacting GitHub.
func listCommentsWithClient(issueNumber int, client *http.Client, token string) ([]IssueComment, error) {
	return listCommentsSinceWithClient(issueNumber, client, token, time.Time{})
}

func listCommentsSinceWithClient(issueNumber int, client *http.Client, token string, since time.Time) ([]IssueComment, error) {
	nextURL := fmt.Sprintf(
		"https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d/comments?per_page=100",
		issueNumber,
	)
	if !since.IsZero() {
		// Filter on GitHub before downloading pages. Keep this boundary fixed for
		// the entire read, following the returned next links just as in Phase 1.
		nextURL += "&since=" + url.QueryEscape(since.UTC().Format(time.RFC3339))
	}
	var comments []IssueComment
	visited := make(map[string]bool)
	for nextURL != "" {
		// A broken next link must produce an error, not an endless request loop.
		if visited[nextURL] {
			return nil, fmt.Errorf("comment pagination repeated a page URL")
		}
		visited[nextURL] = true

		page, headers, err := fetchCommentPage(client, token, nextURL)
		if err != nil {
			// Discard earlier pages: callers must never treat partial history as complete.
			return nil, err
		}
		// GitHub returns ascending IDs; appending pages preserves caller ordering.
		comments = append(comments, page...)
		nextURL, err = nextCommentPage(nextURL, headers.Values("Link"))
		if err != nil {
			return nil, err
		}
	}
	return comments, nil
}

// Keep each response's lifetime inside one call so its body is closed before
// requesting the next page, rather than deferring every close until the end.
func fetchCommentPage(client *http.Client, token, pageURL string) ([]IssueComment, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating comment page request: %w", err)
	}
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching comment page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var gitHubError GitHubErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&gitHubError); err != nil {
			return nil, nil, fmt.Errorf("GitHub comment page returned status %d: %w", resp.StatusCode, err)
		}
		return nil, nil, fmt.Errorf("GitHub comment page returned status %d: %s", resp.StatusCode, gitHubError.Message)
	}
	var comments []IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, nil, fmt.Errorf("error decoding comment page: %w", err)
	}
	return comments, resp.Header, nil
}

// Missing Link/next is normal, even for a full 100-comment page. Follow the
// server's links instead of guessing page numbers from the number of comments.
func nextCommentPage(pageURL string, headers []string) (string, error) {
	current, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid current comment page URL: %w", err)
	}
	var next string
	for _, header := range headers {
		links, err := splitCommentLinks(header)
		if err != nil {
			return "", err
		}
		for _, link := range links {
			target, params, ok := strings.Cut(strings.TrimPrefix(link, "<"), ">")
			if !strings.HasPrefix(link, "<") || !ok || target == "" {
				return "", fmt.Errorf("malformed comment pagination link")
			}
			// Link parameters use the same quoted key=value form as MIME parameters.
			_, attributes, err := mime.ParseMediaType("application/link" + params)
			if err != nil || attributes["rel"] == "" {
				return "", fmt.Errorf("malformed comment pagination parameters")
			}
			for _, relation := range strings.Fields(attributes["rel"]) {
				if relation != "next" {
					continue
				}
				if next != "" {
					return "", fmt.Errorf("multiple next links in comment pagination")
				}
				reference, err := url.Parse(target)
				if err != nil {
					return "", fmt.Errorf("invalid next comment page URL: %w", err)
				}
				// url.Parse leaves query escapes unchecked; reject malformed cursors too.
				if _, err := url.ParseQuery(reference.RawQuery); err != nil {
					return "", fmt.Errorf("invalid next comment page query: %w", err)
				}
				resolved := current.ResolveReference(reference)
				// Every page carries authorization; never forward it to another origin.
				if resolved.Scheme != "https" || resolved.Host != current.Host || resolved.User != nil || resolved.Fragment != "" {
					return "", fmt.Errorf("next comment page URL must remain on the GitHub API origin")
				}
				next = resolved.String()
			}
		}
	}
	return next, nil
}

// Commas separate links, except inside <URLs> or quoted parameter values.
func splitCommentLinks(header string) ([]string, error) {
	var links []string
	start := 0
	inURL, quoted, escaped := false, false, false
	for i, char := range header {
		switch {
		case escaped:
			escaped = false
		case quoted && char == '\\':
			escaped = true
		case !inURL && char == '"':
			quoted = !quoted
		case !quoted && char == '<':
			if inURL {
				return nil, fmt.Errorf("malformed comment pagination link")
			}
			inURL = true
		case !quoted && char == '>':
			if !inURL {
				return nil, fmt.Errorf("malformed comment pagination link")
			}
			inURL = false
		case !inURL && !quoted && char == ',':
			links = append(links, strings.TrimSpace(header[start:i]))
			start = i + 1
		}
	}
	if inURL || quoted || escaped {
		return nil, fmt.Errorf("unterminated comment pagination link")
	}
	links = append(links, strings.TrimSpace(header[start:]))
	return links, nil
}

func createComment(issueNumber int, message GundoMessage) error {
	_, err := createCommentWithMetadata(issueNumber, message)
	return err
}

// Retain the created task's transport bookmark before waiting for its result.
// Heartbeat/result callers can continue using the error-only createComment API.
func createCommentWithMetadata(issueNumber int, message GundoMessage) (IssueComment, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return IssueComment{}, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	return createCommentWithClient(issueNumber, message, &http.Client{}, token)
}

func createCommentWithClient(issueNumber int, message GundoMessage, client *http.Client, token string) (IssueComment, error) {

	// FLOW: take GundoMessage --> marshal to JSON --> create comment request struct --> marshal to JSON --> send POST request to GitHub API

	// marshal GundoMessage to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return IssueComment{}, fmt.Errorf("error marshalling JSON: %w", err)
	}

	// wrap the JSON data into a CommentRequest struct
	commentRequest := CommentRequest{
		Body: string(jsonData),
	}

	// marshal the CommentRequest struct to JSON to be sent in the request body
	commentJSON, err := json.Marshal(commentRequest)
	if err != nil {
		return IssueComment{}, fmt.Errorf("error marshalling comment JSON: %w", err)
	}

	// post process to put comment on the issue number
	url := fmt.Sprintf("https://api.github.com/repos/mamadub03/gundo-test-repo/issues/%d/comments", issueNumber)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(commentJSON))
	if err != nil {
		return IssueComment{}, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "gundo")

	req.Header.Set("Authorization", "Bearer "+token)

	// setting up the client to send the request and get the response to ensure the comment was created successfully
	resp, err := client.Do(req)
	if err != nil {
		return IssueComment{}, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var gitHubError GitHubErrorResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&gitHubError)
		if err != nil {
			return IssueComment{}, fmt.Errorf("error decoding GitHub error response: %w", err)
		}
		return IssueComment{}, fmt.Errorf("GitHub request failed: %s, message: %s", resp.Status, gitHubError.Message)
	}

	var created IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return IssueComment{}, fmt.Errorf("comment was created but its response could not be decoded: %w", err)
	}
	if created.ID <= 0 || created.CreatedAt.IsZero() {
		return IssueComment{}, fmt.Errorf("comment was created but its ID or created_at is missing")
	}
	return created, nil
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

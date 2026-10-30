package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDiscoveryRetainsHeartbeatAndIndependentBookmarks(t *testing.T) {
	stamp := time.Now().UTC().Truncate(time.Second)
	heartbeat := func(client string, timestamp time.Time) string {
		return fmt.Sprintf(`{"type":"heartbeat","client_id":%q,"timestamp":%q}`, client, timestamp.Format(time.RFC3339))
	}
	issues := []IssueResponse{
		{Number: 22, Title: "Gundo Client - client-1"},
		{Number: 23, Title: "Gundo Client - client-2"},
		{Number: 24, Title: "Unrelated issue"},
	}
	api := &commentAPI{t: t, comments: map[int][]IssueComment{
		22: {{ID: 10, CreatedAt: stamp, Body: heartbeat("client-1", stamp)}},
		23: {{ID: 20, CreatedAt: stamp.Add(-time.Minute), Body: heartbeat("client-2", stamp.Add(-time.Minute))}},
	}}
	cache := make(map[int]discoveryMailbox)
	clients, err := discoverClientsFromIssues(issues, cache, api.list)
	if err != nil || len(clients) != 2 || !clients[0].Online || !clients[1].Online || len(api.reads) != 2 {
		t.Fatalf("initial discovery: %+v, %v", clients, err)
	}
	if !api.reads[0].since.IsZero() || !api.reads[1].since.IsZero() {
		t.Fatal("first discovery must bootstrap each mailbox")
	}
	api.comments[22] = append(api.comments[22], IssueComment{ID: 11, CreatedAt: stamp.Add(time.Second), Body: `{"type":"result"}`})
	api.reads = nil
	clients, err = discoverClientsFromIssues(issues, cache, api.list)
	if err != nil || !clients[0].Online || !clients[0].LastSeen.Equal(stamp) || !clients[1].Online {
		t.Fatalf("a non-heartbeat refresh lost liveness: %+v, %v", clients, err)
	}
	if len(api.reads) != 2 || !api.reads[0].since.Equal(stamp.Add(-time.Second)) || !api.reads[1].since.Equal(stamp.Add(-time.Minute-time.Second)) ||
		cache[22].Checkpoint.LastSeenCommentID != 11 || cache[23].Checkpoint.LastSeenCommentID != 20 {
		t.Fatal("mailboxes did not retain independent progress")
	}
	// Simulate an old cached heartbeat without sleeping for six minutes.
	mailbox := cache[22]
	mailbox.LastSeen = stamp.Add(-7 * time.Minute)
	cache[22] = mailbox
	clients, err = discoverClientsFromIssues(issues, cache, api.list)
	if err != nil || clients[0].Online || !clients[1].Online {
		t.Fatalf("heartbeat age was not recalculated: %+v, %v", clients, err)
	}
	before := cache[22]
	api.failPage = 1
	if _, err := discoverClientsFromIssues(issues, cache, api.list); err != nil || cache[22] != before {
		t.Fatal("failed discovery changed the cached bookmark")
	}
	api.failPage = 0
	api.comments[22] = append(api.comments[22], IssueComment{ID: 12, CreatedAt: stamp.Add(2 * time.Second), Body: heartbeat("client-1", stamp)})
	clients, err = discoverClientsFromIssues(issues, cache, api.list)
	if err != nil || !clients[0].Online || cache[22].Checkpoint.LastSeenCommentID != 12 {
		t.Fatalf("discovery did not recover: %+v, %v", clients, err)
	}
}

func TestPostedTaskBookmarkFindsImmediateResult(t *testing.T) {
	stamp := pollingTestTime()
	task := GundoMessage{Type: "task", TaskID: "task-1", TargetClientID: "client-1", Action: "ping"}
	posts := 0
	client := &http.Client{Transport: commentTestTransport(func(req *http.Request) (*http.Response, error) {
		posts++
		if req.Method != http.MethodPost || req.Header.Get("Authorization") != "Bearer dummy-test-token" {
			t.Fatal("incorrect task POST")
		}
		var envelope CommentRequest
		if err := json.NewDecoder(req.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		message, err := parseGundoMessage(envelope.Body)
		if err != nil || message.TaskID != task.TaskID || message.TargetClientID != task.TargetClientID {
			t.Fatalf("task payload changed: %+v, %v", message, err)
		}
		return commentResponse(201, `{"id":100,"created_at":"2026-10-30T12:00:00Z"}`), nil
	})}
	created, err := createCommentWithClient(22, task, client, "dummy-test-token")
	if err != nil || created.ID != 100 || !created.CreatedAt.Equal(stamp) || posts != 1 {
		t.Fatalf("task metadata was not retained: %+v, %v", created, err)
	}
	checkpoint := commentCheckpoint{LastSeenCommentID: created.ID, CreatedAt: created.CreatedAt}
	api := &commentAPI{t: t, comments: map[int][]IssueComment{22: {
		{ID: 99, CreatedAt: stamp.Add(-time.Minute), Body: `{"type":"result","task_id":"task-1","output":"old"}`},
		{ID: 100, CreatedAt: stamp, Body: `{"type":"task","task_id":"task-1"}`},
		// The result already exists before the first GET and shares the task's second.
		{ID: 101, CreatedAt: stamp, Body: `{"type":"result","task_id":"task-1","output":"pong"}`},
	}}}
	result, err := waitForResultWithPolling(22, task.TaskID, checkpoint, api.list, func(time.Duration) {
		t.Fatal("an immediate result should not require another polling iteration")
	})
	if err != nil || result.Output != "pong" || len(api.reads) != 1 || !api.reads[0].since.Equal(stamp.Add(-time.Second)) {
		t.Fatalf("immediate result was missed: %+v, %v", result, err)
	}
	// The waiter received its own checkpoint, leaving discovery/other readers alone.
	if checkpoint.LastSeenCommentID != 100 {
		t.Fatal("result wait changed another consumer's bookmark")
	}
}

func TestWaitForResultAdvancesAcrossIrrelevantComments(t *testing.T) {
	stamp := pollingTestTime()
	checkpoint := commentCheckpoint{LastSeenCommentID: 10, CreatedAt: stamp}
	api := &commentAPI{t: t, comments: map[int][]IssueComment{22: {
		{ID: 11, CreatedAt: stamp.Add(time.Second), Body: `{"type":"heartbeat"}`},
		{ID: 12, CreatedAt: stamp.Add(2 * time.Second), Body: "not JSON"},
		{ID: 13, CreatedAt: stamp.Add(3 * time.Second), Body: `{"type":"result","task_id":"other"}`},
	}}}
	sleeps := 0
	result, err := waitForResultWithPolling(22, "wanted", checkpoint, api.list, func(delay time.Duration) {
		sleeps++
		if sleeps > 1 || delay != 5*time.Second {
			t.Fatal("unexpected result-wait timing")
		}
		api.comments[22] = append(api.comments[22], IssueComment{ID: 14, CreatedAt: stamp.Add(3 * time.Second), Body: `{"type":"result","task_id":"wanted","output":"found"}`})
	})
	if err != nil || result.Output != "found" || sleeps != 1 || len(api.reads) != 2 || !api.reads[1].since.Equal(stamp.Add(2*time.Second)) {
		t.Fatalf("wait did not advance incrementally: %+v, %v", result, err)
	}
	api.failPage = 1
	if _, err := waitForResultWithPolling(22, "wanted", checkpoint, api.list, func(time.Duration) { t.Fatal("slept after failed read") }); err == nil {
		t.Fatal("result wait hid a retrieval failure")
	}
}

func TestCreateCommentMetadataFailuresDoNotRetry(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "malformed success body", status: 201, body: "broken"},
		{name: "missing creation timestamp", status: 201, body: `{"id":1}`},
		{name: "missing ID", status: 201, body: `{"created_at":"2026-10-30T12:00:00Z"}`},
		{name: "HTTP failure", status: 403, body: `{"message":"forbidden"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: commentTestTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return commentResponse(test.status, test.body), nil
			})}
			_, err := createCommentWithClient(22, GundoMessage{Type: "task"}, client, "dummy-test-token")
			if err == nil || calls != 1 {
				t.Fatalf("error = %v, POST count = %d", err, calls)
			}
			if test.status == 201 && !strings.Contains(err.Error(), "was created") {
				t.Fatal("a malformed success response should explain that the POST succeeded")
			}
		})
	}
}

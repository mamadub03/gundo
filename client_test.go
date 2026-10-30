package main

import (
	"testing"
	"time"
)

func TestPollTasksOnceUsesBookmarkAndFiltersMessages(t *testing.T) {
	stamp := pollingTestTime()
	client := newClient("client-1")
	var checkpoint commentCheckpoint
	// The startup read establishes progress but deliberately does not run tasks.
	history := []IssueComment{{ID: 100, CreatedAt: stamp, Body: `{"type":"task","task_id":"old","target_client_id":"client-1","action":"ping"}`}}
	if _, err := checkpoint.accept(history); err != nil {
		t.Fatal(err)
	}
	api := &commentAPI{t: t, comments: map[int][]IssueComment{22: append(history,
		IssueComment{ID: 101, CreatedAt: stamp, Body: `{"type":"heartbeat","client_id":"client-1"}`},
		IssueComment{ID: 102, CreatedAt: stamp, Body: `{"type":"result","task_id":"another-task"}`},
		IssueComment{ID: 103, CreatedAt: stamp, Body: `{"type":"task","task_id":"other-target","target_client_id":"client-2","action":"ping"}`},
		IssueComment{ID: 104, CreatedAt: stamp, Body: "not JSON"},
		IssueComment{ID: 105, CreatedAt: stamp, Body: `{"type":"task","task_id":"new-task","target_client_id":"client-1","action":"ping"}`},
	)}}
	var posted []GundoMessage
	post := func(issue int, message GundoMessage) error {
		if issue != 22 {
			t.Fatalf("posted to the wrong mailbox: %d", issue)
		}
		posted = append(posted, message)
		return nil
	}
	if err := client.pollTasksOnce(22, &checkpoint, api.list, post); err != nil {
		t.Fatal(err)
	}
	if len(posted) != 1 || posted[0].TaskID != "new-task" || posted[0].Output != "pong" || posted[0].ClientID != "client-1" {
		t.Fatalf("incorrect task execution/results: %+v", posted)
	}
	if checkpoint.LastSeenCommentID != 105 || !api.reads[0].since.Equal(stamp.Add(-time.Second)) {
		t.Fatal("wrong bookmark or request boundary")
	}
	if err := client.pollTasksOnce(22, &checkpoint, api.list, post); err != nil || len(posted) != 1 {
		t.Fatalf("overlap reprocessed a task: %+v, %v", posted, err)
	}
	// Even a batch containing only non-tasks must advance the bookmark.
	api.comments[22] = append(api.comments[22], IssueComment{ID: 106, CreatedAt: stamp.Add(time.Second), Body: `{"type":"result"}`})
	if err := client.pollTasksOnce(22, &checkpoint, api.list, post); err != nil || checkpoint.LastSeenCommentID != 106 || len(posted) != 1 {
		t.Fatalf("non-task progress failed: %+v, %v", checkpoint, err)
	}
}

func TestPollTasksOnceDoesNotExecutePartialBatch(t *testing.T) {
	stamp := pollingTestTime()
	checkpoint := commentCheckpoint{LastSeenCommentID: 1, CreatedAt: stamp}
	before := checkpoint
	api := &commentAPI{t: t, failPage: 2, comments: make(map[int][]IssueComment)}
	for i := 2; i <= 102; i++ {
		api.comments[22] = append(api.comments[22], IssueComment{ID: int64(i), CreatedAt: stamp, Body: `{"type":"task","task_id":"task","target_client_id":"client-1","action":"ping"}`})
	}
	posted := 0
	err := newClient("client-1").pollTasksOnce(22, &checkpoint, api.list, func(int, GundoMessage) error {
		posted++
		return nil
	})
	if err == nil || posted != 0 || checkpoint != before {
		t.Fatalf("partial fetch processed tasks: %d posts, %+v, %v", posted, checkpoint, err)
	}
}

// func TestHandlePing(t *testing.T) {

// 	client, err := newClient("test-client")
// 	if err != nil {
// 		t.Fatalf("failed to create client: %v", err)
// 	}

// 	task := GundoMessage{
// 		Type:   "task",
// 		TaskID: "task-123",
// 		Action: "ping",
// 	}

// 	result := client.handleTask(task)

// 	if result.Type != "result" {
// 		t.Errorf("expected result type, got %s", result.Type)
// 	}

// 	if result.TaskID != task.TaskID {
// 		t.Errorf("expected task ID %s, got %s", task.TaskID, result.TaskID)
// 	}

// 	if result.ClientID != client.ID {
// 		t.Errorf("expected client ID %s, got %s", client.ID, result.ClientID)
// 	}

// 	if result.Status != "success" {
// 		t.Errorf("expected success status, got %s", result.Status)
// 	}

// 	if result.Output != "pong" {
// 		t.Errorf("expected pong, got %s", result.Output)
// 	}
// }

// func TestHandleGetHostname(t *testing.T) {
// 	client, err := newClient("test-client")
// 	if err != nil {
// 		t.Fatalf("failed to create client: %v", err)
// 	}

// 	task := GundoMessage{
// 		Type:   "task",
// 		TaskID: "task-hostname",
// 		Action: "get_hostname",
// 	}

// 	result := client.handleTask(task)

// 	expected, err := os.Hostname()
// 	if err != nil {
// 		t.Fatalf("could not get expected hostname: %v", err)
// 	}

// 	if result.Output != expected {
// 		t.Errorf("expected hostname %s, got %s", expected, result.Output)
// 	}
// }

// func TestHandleGetOS(t *testing.T) {
// 	client, err := newClient("test-client")
// 	if err != nil {
// 		t.Fatalf("failed to create client: %v", err)
// 	}
// 	task := GundoMessage{
// 		Type:   "task",
// 		TaskID: "task-os",
// 		Action: "get_os",
// 	}

// 	result := client.handleTask(task)

// 	if result.Output != runtime.GOOS {
// 		t.Errorf("expected OS %s, got %s", runtime.GOOS, result.Output)
// 	}
// }

// func TestHandleGetUser(t *testing.T) {
// 	client, err := newClient("test-client")
// 	if err != nil {
// 		t.Fatalf("failed to create client: %v", err)
// 	}

// 	task := GundoMessage{
// 		Type:   "task",
// 		TaskID: "task-user",
// 		Action: "get_user",
// 	}

// 	result := client.handleTask(task)

// 	expected, err := user.Current()
// 	if err != nil {
// 		t.Fatalf("could not get expected user: %v", err)
// 	}

// 	if result.Output != expected.Username {
// 		t.Errorf("expected user %s, got %s", expected.Username, result.Output)
// 	}
// }

// func TestHandleUnsupportedAction(t *testing.T) {
// 	client := newClient("test-client")

// 	task := GundoMessage{
// 		Type:   "task",
// 		TaskID: "task-bad",
// 		Action: "do_something_random",
// 	}

// 	result := client.handleTask(task)

// 	if result.Status != "error" {
// 		t.Errorf("expected error status, got %s", result.Status)
// 	}

// 	if result.Output != "unsupported action" {
// 		t.Errorf("expected unsupported action message, got %s", result.Output)
// 	}
// }

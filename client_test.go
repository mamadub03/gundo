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

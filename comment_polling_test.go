package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type commentRead struct {
	issue int
	page  int
	since time.Time
}

// This in-memory API models append-only mailboxes. The production code must
// supply since, follow Link, decode JSON, and filter overlapping IDs itself.
type commentAPI struct {
	t        *testing.T
	comments map[int][]IssueComment
	reads    []commentRead
	failPage int
	onRead   func(commentRead)
}

func commentResponse(status int, data string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       &commentTestBody{Reader: strings.NewReader(data)},
	}
}

func (api *commentAPI) list(issue int, since time.Time) ([]IssueComment, error) {
	client := &http.Client{Transport: commentTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Host != "api.github.com" || path.Base(req.URL.Path) != "comments" {
			api.t.Fatalf("unexpected mocked request: %s %s", req.Method, req.URL)
		}
		query := req.URL.Query()
		if query.Get("per_page") != "100" {
			api.t.Fatal("page size was not preserved")
		}
		issue, err := strconv.Atoi(path.Base(path.Dir(req.URL.Path)))
		if err != nil {
			api.t.Fatal(err)
		}
		read := commentRead{issue: issue, page: 1}
		if value := query.Get("page"); value != "" {
			read.page, err = strconv.Atoi(value)
			if err != nil {
				api.t.Fatal(err)
			}
		}
		if value := query.Get("since"); value != "" {
			read.since, err = time.Parse(time.RFC3339, value)
			if err != nil {
				api.t.Fatal(err)
			}
		}
		api.reads = append(api.reads, read)
		if api.onRead != nil {
			api.onRead(read)
		}
		if read.page == api.failPage {
			return commentResponse(503, `{"message":"simulated page failure"}`), nil
		}
		filtered := make([]IssueComment, 0)
		for _, comment := range api.comments[issue] {
			if read.since.IsZero() || comment.CreatedAt.After(read.since) {
				filtered = append(filtered, comment)
			}
		}
		start := min((read.page-1)*100, len(filtered))
		end := min(start+100, len(filtered))
		data, err := json.Marshal(filtered[start:end])
		if err != nil {
			api.t.Fatal(err)
		}
		response := commentResponse(200, string(data))
		if end < len(filtered) {
			nextURL := *req.URL
			query.Set("page", strconv.Itoa(read.page+1))
			nextURL.RawQuery = query.Encode()
			response.Header.Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL.String()))
		}
		return response, nil
	})}
	return listCommentsSinceWithClient(issue, client, "dummy-test-token", since)
}

func pollingTestTime() time.Time {
	return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
}

func TestCommentCheckpointOverlapAndSameTimestamp(t *testing.T) {
	stamp := pollingTestTime()
	checkpoint := commentCheckpoint{LastSeenCommentID: 100, CreatedAt: stamp}
	api := &commentAPI{t: t, comments: map[int][]IssueComment{22: {
		{ID: 99, CreatedAt: stamp.Add(-time.Second)},
		{ID: 100, CreatedAt: stamp},
		{ID: 101, CreatedAt: stamp}, // A later ID can share the bookmarked second.
		{ID: 102, CreatedAt: stamp.Add(time.Second)},
	}}}
	got, err := checkpoint.poll(22, api.list)
	if err != nil || len(got) != 2 || got[0].ID != 101 || got[1].ID != 102 {
		t.Fatalf("new comments = %+v, error = %v", got, err)
	}
	if !api.reads[0].since.Equal(stamp.Add(-time.Second)) || checkpoint.LastSeenCommentID != 102 {
		t.Fatalf("wrong boundary or bookmark: %+v, %+v", api.reads, checkpoint)
	}
	before := checkpoint
	got, err = checkpoint.poll(22, api.list)
	if err != nil || len(got) != 0 || checkpoint != before || len(api.reads) != 2 {
		t.Fatalf("unchanged poll moved bookmark or returned overlap: %+v, %v", got, err)
	}
	// Even if an already-seen comment is returned with a changed body, skip its
	// ID. It must not change our creation-time bookmark or cause task execution.
	got, err = checkpoint.accept([]IssueComment{{ID: 100, Body: "changed"}})
	if err != nil || len(got) != 0 || checkpoint != before {
		t.Fatal("an old ID changed progress")
	}
}

func TestCommentCheckpointEmptyMailboxAndMissingTimestamp(t *testing.T) {
	var checkpoint commentCheckpoint
	api := &commentAPI{t: t, comments: make(map[int][]IssueComment)}
	if got, err := checkpoint.poll(22, api.list); err != nil || len(got) != 0 || !checkpoint.since().IsZero() {
		t.Fatalf("empty mailbox: %+v, %v", got, err)
	}
	api.comments[22] = []IssueComment{{ID: 1, CreatedAt: pollingTestTime()}}
	if got, err := checkpoint.poll(22, api.list); err != nil || len(got) != 1 || checkpoint.LastSeenCommentID != 1 {
		t.Fatalf("first comment: %+v, %v", got, err)
	}
	before := checkpoint
	got, err := checkpoint.accept([]IssueComment{
		{ID: 2, CreatedAt: pollingTestTime().Add(time.Second)},
		{ID: 3},
	})
	if err == nil || got != nil || checkpoint != before {
		t.Fatal("missing metadata must reject the whole batch without advancing")
	}
}

func TestIncrementalPaginationAndRecovery(t *testing.T) {
	for _, count := range []int{100, 101, 250, 1000} {
		t.Run(fmt.Sprintf("%d_new_comments", count), func(t *testing.T) {
			stamp := pollingTestTime()
			checkpoint := commentCheckpoint{LastSeenCommentID: 10, CreatedAt: stamp}
			api := &commentAPI{t: t, comments: map[int][]IssueComment{22: {{ID: 10, CreatedAt: stamp}}}}
			for i := 1; i <= count; i++ {
				api.comments[22] = append(api.comments[22], IssueComment{ID: int64(10 + i), CreatedAt: stamp.Add(time.Duration(i) * time.Second)})
			}
			// A failed later page must not consume even the successful first page.
			before := checkpoint
			api.failPage = 2
			if got, err := checkpoint.poll(22, api.list); err == nil || got != nil || checkpoint != before {
				t.Fatal("partial read advanced the checkpoint")
			}
			api.failPage = 0
			api.reads = nil
			got, err := checkpoint.poll(22, api.list)
			if err != nil || len(got) != count || checkpoint.LastSeenCommentID != int64(10+count) {
				t.Fatalf("recovery returned %d new comments: %v", len(got), err)
			}
			// The overlapping bookmark itself uses one result slot.
			if len(api.reads) != (count+1+99)/100 {
				t.Fatalf("unexpected page count: %d", len(api.reads))
			}
			for i, read := range api.reads {
				if !read.since.Equal(stamp.Add(-time.Second)) || read.page != i+1 {
					t.Fatal("query boundary or comment ordering changed during pagination")
				}
			}
			for i, comment := range got {
				if comment.ID != int64(11+i) {
					t.Fatal("new comments were reordered or lost")
				}
			}
		})
	}
}

func TestIncrementalPollingRequestSavings(t *testing.T) {
	stamp := pollingTestTime()
	api := &commentAPI{t: t, comments: make(map[int][]IssueComment)}
	for i := 1; i <= 1000; i++ {
		api.comments[22] = append(api.comments[22], IssueComment{ID: int64(i), CreatedAt: stamp.Add(time.Duration(i) * time.Second)})
	}
	var checkpoint commentCheckpoint
	// Bootstrap reads the existing history once, as the client does on startup.
	if _, err := checkpoint.poll(22, api.list); err != nil || len(api.reads) != 10 {
		t.Fatalf("baseline requests = %d, error = %v", len(api.reads), err)
	}
	api.comments[22] = append(api.comments[22], IssueComment{ID: 1001, CreatedAt: stamp.Add(1001 * time.Second)})
	api.reads = nil
	if got, err := checkpoint.poll(22, api.list); err != nil || len(got) != 1 || len(api.reads) != 1 {
		t.Fatalf("incremental read: %d new comments, %d requests, %v", len(got), len(api.reads), err)
	}
	api.reads = nil
	if got, err := checkpoint.poll(22, api.list); err != nil || len(got) != 0 || len(api.reads) != 1 {
		t.Fatalf("quiet read: %d new comments, %d requests, %v", len(got), len(api.reads), err)
	}
}

func TestCommentsArrivingDuringPagination(t *testing.T) {
	stamp := pollingTestTime()
	checkpoint := commentCheckpoint{LastSeenCommentID: 1, CreatedAt: stamp}
	api := &commentAPI{t: t, comments: make(map[int][]IssueComment)}
	for i := 1; i <= 101; i++ {
		api.comments[22] = append(api.comments[22], IssueComment{ID: int64(i), CreatedAt: stamp.Add(time.Duration(i-1) * time.Second)})
	}
	api.onRead = func(read commentRead) {
		if read.page == 2 {
			api.comments[22] = append(api.comments[22], IssueComment{ID: 102, CreatedAt: stamp.Add(101 * time.Second)})
		}
	}
	got, err := checkpoint.poll(22, api.list)
	if err != nil || len(got) != 101 || checkpoint.LastSeenCommentID != 102 {
		t.Fatalf("arrival during pagination was lost: %d comments, %v", len(got), err)
	}
	api.onRead = nil
	api.comments[22] = append(api.comments[22], IssueComment{ID: 103, CreatedAt: stamp.Add(101 * time.Second)})
	got, err = checkpoint.poll(22, api.list)
	if err != nil || len(got) != 1 || got[0].ID != 103 {
		t.Fatalf("arrival after pagination in the same second was lost: %+v, %v", got, err)
	}
}

func TestCheckpointFailureKeepsProgress(t *testing.T) {
	checkpoint := commentCheckpoint{LastSeenCommentID: 5, CreatedAt: pollingTestTime()}
	before := checkpoint
	got, err := checkpoint.poll(22, func(int, time.Time) ([]IssueComment, error) {
		return []IssueComment{{ID: 6, CreatedAt: pollingTestTime()}}, fmt.Errorf("failed read")
	})
	if err == nil || got != nil || !reflect.DeepEqual(checkpoint, before) {
		t.Fatal("a failed fetch must not consume partial results")
	}
}

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestConsoleAgentOrderingAndLabels(t *testing.T) {
	clients := []ManagedClient{
		{ClientID: "offline", IssueNumber: 1},
		{ClientID: "same-prefix-client-b", IssueNumber: 9, Online: true},
		{ClientID: "same-prefix-client-a", IssueNumber: 2, Online: true},
	}
	original := append([]ManagedClient(nil), clients...)
	visible := visibleAgents(clients, false)
	if len(visible) != 2 || visible[0].IssueNumber != 2 || visible[1].IssueNumber != 9 {
		t.Fatal("online selection mapping incorrect")
	}
	all := visibleAgents(clients, true)
	if len(all) != 3 || all[2].Online || !reflect.DeepEqual(original, clients) {
		t.Fatal("sorting changed discovery snapshot")
	}
	c := &operatorConsole{}
	labels := c.agentLabels(visible)
	if labels[0] == labels[1] || !strings.Contains(labels[0], "#2") {
		t.Fatal("short IDs lost mailbox identity")
	}
}

func TestConsolePlainSelection(t *testing.T) {
	for _, tc := range []struct {
		input  string
		want   int
		failed bool
	}{
		{"invalid\n9\n2\n", 1, false}, {"0\n", -1, true}, {"", -1, true},
	} {
		var out bytes.Buffer
		c := &operatorConsole{reader: bufio.NewReader(strings.NewReader(tc.input)), out: &out}
		got, err := c.selectItem("Choose", []string{"First", "Second"})
		if got != tc.want || (err != nil) != tc.failed {
			t.Fatalf("selection = %d, %v", got, err)
		}
		if strings.Contains(out.String(), "\x1b") {
			t.Fatal("plain fallback emitted terminal escapes")
		}
	}
}

func TestConsoleSanitizesRemoteText(t *testing.T) {
	input := "hello\x1b[2J\x1b]0;fake-title\x07\r\u202e\nworld\t!"
	got := consoleText(input)
	if got != "hello\nworld\t!" {
		t.Fatalf("unexpected sanitized text: %q", got)
	}
	if consoleLabel(input) != "hello world !" {
		t.Fatal("multiline text escaped into menu labels")
	}
	var out bytes.Buffer
	c := &operatorConsole{out: &out}
	c.result("Ping", "task-1", GundoMessage{Status: "error", Output: input}, time.Second)
	if strings.ContainsAny(out.String(), "\x1b\r\u202e") || !strings.Contains(out.String(), "error") {
		t.Fatal("result rendering lost status or emitted controls")
	}
}

func TestConsoleCancelDoesNotPost(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = commentTestTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled selection made a request")
		return nil, nil
	})
	for _, input := range []string{"6\nno\n7\n", "5\n\n7\n", ""} {
		c := &operatorConsole{reader: bufio.NewReader(strings.NewReader(input)), out: io.Discard}
		runCommandMenu(c, ManagedClient{Online: true, IssueNumber: 22})
	}
}

func TestConsoleTaskResultFlow(t *testing.T) {
	oldTransport, oldToken := http.DefaultTransport, githubToken
	t.Cleanup(func() { http.DefaultTransport, githubToken = oldTransport, oldToken })
	githubToken = "dummy-test-token"
	var task GundoMessage
	calls := 0
	http.DefaultTransport = commentTestTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		status, body := 200, ""
		switch calls {
		case 1:
			if req.Method != "POST" {
				t.Fatal("expected task POST")
			}
			var envelope CommentRequest
			if err := json.NewDecoder(req.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(envelope.Body), &task); err != nil {
				t.Fatal(err)
			}
			if task.Action != "ping" || task.TargetClientID != "client-1" {
				t.Fatal("menu selected wrong action or target")
			}
			status, body = 201, `{"id":100,"created_at":"2026-11-05T12:00:00Z"}`
		case 2:
			if req.Method != "GET" || req.URL.Query().Get("since") != "2026-11-05T11:59:59Z" {
				t.Fatal("result wait lost task checkpoint")
			}
			result, _ := json.Marshal(GundoMessage{Type: "result", TaskID: task.TaskID, Status: "success", Output: "pong"})
			body = fmt.Sprintf(`[{"id":101,"created_at":"2026-11-05T12:00:01Z","body":%q}]`, result)
		default:
			t.Fatal("unexpected request")
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	var out bytes.Buffer
	c := &operatorConsole{reader: bufio.NewReader(strings.NewReader("1\n7\n")), out: &out}
	runCommandMenu(c, ManagedClient{ClientID: "client-1", IssueNumber: 22, Online: true})
	if strings.Count(out.String(), "OPERATOR CONSOLE") != 1 || strings.Contains(out.String(), "Press Enter") {
		t.Fatal("session was redrawn or paused between actions")
	}
	if calls != 2 || !strings.Contains(out.String(), "Task posted") || !strings.Contains(out.String(), "pong") || !strings.Contains(out.String(), task.TaskID) {
		t.Fatal("task/result display incomplete")
	}
}

func TestConsoleScreen(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		var out bytes.Buffer
		c := &operatorConsole{out: &out, interactive: interactive}
		c.screen()
		if strings.Contains(out.String(), "\x1b[2J\x1b[H") != interactive {
			t.Fatal("screen clearing must be limited to interactive terminals")
		}
	}
}

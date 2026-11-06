package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// RoundTrip replaces the network boundary only; the production pagination,
// request headers, JSON decoding, and error handling still run normally.
type commentTestTransport func(*http.Request) (*http.Response, error)

func (transport commentTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

type commentTestBody struct {
	io.Reader
	closed bool
}

func (body *commentTestBody) Close() error {
	body.closed = true
	return nil
}

const commentTestURL = "https://api.github.com/repos/mamadub03/gundo-test-repo/issues/22/comments?per_page=100"

func TestListCommentsPagination(t *testing.T) {
	for _, count := range []int{0, 1, 30, 99, 100, 101, 250, 251} {
		t.Run(fmt.Sprintf("%d_comments", count), func(t *testing.T) {
			want := make([]IssueComment, count)
			for i := range want {
				// IDs need to stay ordered, but are not necessarily consecutive.
				want[i] = IssueComment{ID: int64(1000 + 3*i), Body: fmt.Sprintf("comment %d", i)}
			}
			if count >= 250 {
				// Relevant messages beyond page one must reach the existing callers.
				want[100].Body = `{"type":"task","task_id":"task-1","target_client_id":"client-1","action":"ping"}`
				want[150].Body = `{"type":"heartbeat","client_id":"client-1","timestamp":"2026-10-30T12:00:00Z"}`
				want[200].Body = `{"type":"result","task_id":"task-1","client_id":"client-1","output":"pong"}`
			}

			requests := 0
			var bodies []*commentTestBody
			pageURL := func(page int) string {
				if page == 1 {
					return commentTestURL
				}
				// A different API path proves the supplied Link is followed literally.
				return fmt.Sprintf("https://api.github.com/repositories/123/issues/22/comments?per_page=100&page=%d", page)
			}
			client := &http.Client{Transport: commentTestTransport(func(req *http.Request) (*http.Response, error) {
				for _, body := range bodies {
					if !body.closed {
						t.Fatal("previous response body is still open")
					}
				}
				requests++
				if got := req.URL.String(); got != pageURL(requests) {
					t.Fatalf("request URL = %q, want %q", got, pageURL(requests))
				}
				if req.Method != http.MethodGet || req.Header.Get("Authorization") != "Bearer dummy-test-token" ||
					req.Header.Get("User-Agent") != "gundo" || req.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
					t.Fatal("method or headers were not preserved on every page")
				}
				start := (requests - 1) * 100
				if start > count || (count > 0 && start == count) {
					t.Fatal("requested a page after the final response")
				}
				end := min(start+100, count)
				data, err := json.Marshal(want[start:end])
				if err != nil {
					t.Fatal(err)
				}
				headers := make(http.Header)
				if end < count {
					// Other relations can precede next, and headers can have multiple values.
					headers.Add("Link", fmt.Sprintf(`<%s>; rel="last"`, pageURL((count+99)/100)))
					headers.Add("Link", fmt.Sprintf(`<%s>; rel="next"`, pageURL(requests+1)))
				}
				body := &commentTestBody{Reader: strings.NewReader(string(data))}
				bodies = append(bodies, body)
				return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: body}, nil
			})}

			got, err := listCommentsWithClient(22, client, "dummy-test-token")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != count || (count > 0 && !reflect.DeepEqual(got, want)) {
				t.Fatalf("returned comments differ from the %d ordered fixture comments", count)
			}
			if requests != max(1, (count+99)/100) {
				t.Fatalf("made %d requests for %d comments", requests, count)
			}
			for _, body := range bodies {
				if !body.closed {
					t.Fatal("final response body was not closed")
				}
			}
			if count >= 250 {
				for index, messageType := range map[int]string{100: "task", 150: "heartbeat", 200: "result"} {
					message, err := parseGundoMessage(got[index].Body)
					if err != nil || message.Type != messageType {
						t.Fatalf("missing %s message beyond page one: %v", messageType, err)
					}
				}
			}
		})
	}
}

func TestNextCommentPage(t *testing.T) {
	page2 := commentTestURL + "&page=2"
	for _, test := range []struct {
		name    string
		headers []string
		want    string
		wantErr bool
	}{
		{name: "missing header"},
		{name: "no next relation", headers: []string{`<?page=1>; rel="prev"`}},
		{name: "next among other links", headers: []string{`<?page=3>; rel="last", <` + page2 + `>; rel="next", <?page=1>; rel="prev"`}, want: page2},
		{name: "relative next URL", headers: []string{`<?per_page=100&page=2>; rel="next"`}, want: page2},
		{name: "quoted comma and relation list", headers: []string{`<` + page2 + `>; title="page, two"; rel="next last"`}, want: page2},
		{name: "comma in URL", headers: []string{`<?cursor=a,b>; rel="next"`}, want: "https://api.github.com/repos/mamadub03/gundo-test-repo/issues/22/comments?cursor=a,b"},
		{name: "empty header", headers: []string{""}, wantErr: true},
		{name: "missing brackets", headers: []string{page2 + `; rel="next"`}, wantErr: true},
		{name: "missing closing bracket", headers: []string{`<` + page2 + `; rel="next"`}, wantErr: true},
		{name: "missing relation", headers: []string{`<` + page2 + `>`}, wantErr: true},
		{name: "unclosed quote", headers: []string{`<` + page2 + `>; rel="next`}, wantErr: true},
		{name: "bad parameter", headers: []string{`<` + page2 + `>; rel="next"; broken`}, wantErr: true},
		{name: "empty URL", headers: []string{`<>; rel="next"`}, wantErr: true},
		{name: "invalid URL escape", headers: []string{`<?page=%zz>; rel="next"`}, wantErr: true},
		{name: "duplicate next", headers: []string{`<` + page2 + `>; rel="next"`, `<` + page2 + `>; rel="next"`}, wantErr: true},
		{name: "other host", headers: []string{`<https://example.invalid/comments>; rel="next"`}, wantErr: true},
		{name: "insecure scheme", headers: []string{`<http://api.github.com/comments>; rel="next"`}, wantErr: true},
		{name: "URL credentials", headers: []string{`<https://user@api.github.com/comments>; rel="next"`}, wantErr: true},
		{name: "fragment", headers: []string{`<?page=2#fragment>; rel="next"`}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := nextCommentPage(commentTestURL, test.headers)
			if (err != nil) != test.wantErr || got != test.want {
				t.Fatalf("nextCommentPage = (%q, %v), want (%q, error=%v)", got, err, test.want, test.wantErr)
			}
		})
	}
}

func TestListCommentsPaginationFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		body       string
		link       string
		networkErr bool
		wantError  string
	}{
		{name: "HTTP failure on page two", status: 503, body: `{"message":"unavailable"}`, wantError: "status 503"},
		{name: "non-JSON HTTP failure", status: 403, body: "forbidden", wantError: "status 403"},
		{name: "invalid JSON on page two", status: 200, body: "broken", wantError: "decoding comment page"},
		{name: "network failure on page two", networkErr: true, wantError: "simulated connection failure"},
		{name: "malformed Link on page two", status: 200, body: `[{"id":2,"body":"second"}]`, link: "broken", wantError: "pagination"},
		{name: "repeated next page", status: 200, body: `[]`, link: `<?per_page=100&page=2>; rel="next"`, wantError: "repeated"},
		{name: "cycle to first page", status: 200, body: `[]`, link: `<` + commentTestURL + `>; rel="next"`, wantError: "repeated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			var bodies []*commentTestBody
			client := &http.Client{Transport: commentTestTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				if requests > 2 {
					t.Fatal("unexpected extra request or retry")
				}
				status, data, link := 200, `[{"id":1,"body":"first"}]`, `<?per_page=100&page=2>; rel="next"`
				if requests == 2 {
					if test.networkErr {
						return nil, fmt.Errorf("simulated connection failure")
					}
					status, data, link = test.status, test.body, test.link
				}
				headers := make(http.Header)
				if link != "" {
					headers.Set("Link", link)
				}
				body := &commentTestBody{Reader: strings.NewReader(data)}
				bodies = append(bodies, body)
				return &http.Response{StatusCode: status, Header: headers, Body: body}, nil
			})}
			got, err := listCommentsWithClient(22, client, "dummy-test-token")
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want containing %q", err, test.wantError)
			}
			if got != nil {
				t.Fatalf("returned partial results after a failed page: %+v", got)
			}
			if requests != 2 {
				t.Fatalf("made %d requests, want 2", requests)
			}
			for _, body := range bodies {
				if !body.closed {
					t.Fatal("response body not closed on failure")
				}
			}
		})
	}
}

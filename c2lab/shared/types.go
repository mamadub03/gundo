package shared

// Task is what the controller sends to the agent through GitHub.
type Task struct {
    TaskID  string `json:"task_id"`
    AgentID string `json:"agent_id"`
    Action  string `json:"action"`
}

// Result is what the agent sends back to the controller through GitHub.
type Result struct {
    TaskID string `json:"task_id"`
    Output string `json:"output"`
    Status string `json:"status"` // "success" or "error"
}
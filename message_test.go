package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseTaskMessage(t *testing.T) {
	body := `{
		"type":"task",
		"task_id":"abc123",
		"action":"get_hostname",
		"timestamp":"2026-10-10T20:00:00Z"
	}`

	message, err := parseGundoMessage(body)
	if err != nil {
		t.Fatalf("expected valid Gundo message, got error: %v", err)
	}

	if message.Type != "task" {
		t.Errorf("expected type task, got %s", message.Type)
	}

	if message.TaskID != "abc123" {
		t.Errorf("expected task ID abc123, got %s", message.TaskID)
	}

	if message.Action != "get_hostname" {
		t.Errorf("expected action get_hostname, got %s", message.Action)
	}
}

func TestParseResultMessage(t *testing.T) {
	body := `{
		"type":"result",
		"task_id":"abc123",
		"output":"WEB-01",
		"timestamp":"2026-10-10T20:01:00Z"
	}`

	message, err := parseGundoMessage(body)
	if err != nil {
		t.Fatalf("expected valid result message, got error: %v", err)
	}

	if message.Type != "result" {
		t.Errorf("expected type result, got %s", message.Type)
	}

	if message.TaskID != "abc123" {
		t.Errorf("expected task ID abc123, got %s", message.TaskID)
	}

	if message.Output != "WEB-01" {
		t.Errorf("expected output WEB-01, got %s", message.Output)
	}

	if message.Action != "" {
		t.Errorf("expected empty action on result, got %s", message.Action)
	}
}

func TestParseInvalidMessage(t *testing.T) {
	body := `yooooooo`

	_, err := parseGundoMessage(body)

	if err == nil {
		t.Fatal("expected invalid JSON to return an error")
	}
}

func TestCreateTaskMessage(t *testing.T) {
	message := createTaskMessage("get_hostname")

	if message.Type != "task" {
		t.Errorf("expected type task, got %s", message.Type)
	}

	if message.TaskID == "" {
		t.Error("expected task ID to be generated")
	}

	if message.Action != "get_hostname" {
		t.Errorf("expected action get_hostname, got %s", message.Action)
	}

	if message.Output != "" {
		t.Errorf("expected output to be empty, got %s", message.Output)
	}

	if message.Timestamp == "" {
		t.Error("expected timestamp to be generated")
	}

	_, err := time.Parse(time.RFC3339, message.Timestamp)
	if err != nil {
		t.Errorf("timestamp is not valid RFC3339: %v", err)
	}
}

func TestGenerateTaskIDUnique(t *testing.T) {
	first := generateTaskID()
	second := generateTaskID()

	if first == "" || second == "" {
		t.Fatal("expected generated task IDs to not be empty")
	}

	if first == second {
		t.Errorf("expected unique task IDs, both were %s", first)
	}
}

func TestTaskMessageOmitsOutput(t *testing.T) {
	message := createTaskMessage("get_hostname")

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("error marshaling task: %v", err)
	}

	var decoded map[string]interface{}

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("error decoding JSON for test: %v", err)
	}

	if _, exists := decoded["output"]; exists {
		t.Error("expected output field to be omitted from task JSON")
	}
}

func TestRoundTripTaskMessage(t *testing.T) {
	original := createTaskMessage("get_hostname")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("error marshaling original message: %v", err)
	}

	parsed, err := parseGundoMessage(string(data))
	if err != nil {
		t.Fatalf("error parsing marshaled message: %v", err)
	}

	if parsed.Type != original.Type {
		t.Errorf("type changed during round trip")
	}

	if parsed.TaskID != original.TaskID {
		t.Errorf("task ID changed during round trip")
	}

	if parsed.Action != original.Action {
		t.Errorf("action changed during round trip")
	}

	if parsed.Timestamp != original.Timestamp {
		t.Errorf("timestamp changed during round trip")
	}
}

package main

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

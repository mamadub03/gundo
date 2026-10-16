package main

import "fmt"

func main() {

	fmt.Println("=== Gundo Controller Demo ===")

	// 1. Create agent mailbox
	issue := IssueRequest{
		Title: "Gundo Agent Mailbox",
		Body:  "Mailbox used for Gundo controller/agent communication.",
	}

	createdIssue, err := createIssue(issue)
	if err != nil {
		fmt.Printf("Error creating issue: %v\n", err)
		return
	}

	fmt.Printf(
		"[+] Mailbox created: Issue #%d\n",
		createdIssue.Number,
	)

	// 2. Create structured Gundo task
	task := createTaskMessage("get_hostname")

	fmt.Printf(
		"[+] Task created: %s\n",
		task.TaskID,
	)

	// 3. Send task through GitHub comment transport
	err = createComment(createdIssue.Number, task)
	if err != nil {
		fmt.Printf("Error creating task comment: %v\n", err)
		return
	}

	fmt.Println("[+] Task posted to GitHub")

	// 4. Read comments back from GitHub
	comments, err := listComments(createdIssue.Number)
	if err != nil {
		fmt.Printf("Error listing comments: %v\n", err)
		return
	}

	if len(comments) == 0 {
		fmt.Println("No comments returned from GitHub.")
		return
	}

	// 5. Grab newest comment
	latestComment := comments[len(comments)-1]

	fmt.Printf(
		"[+] Received GitHub comment ID: %d\n",
		latestComment.ID,
	)

	// 6. Convert GitHub comment body back into Go
	message, err := parseGundoMessage(latestComment.Body)
	if err != nil {
		fmt.Printf("Error parsing Gundo message: %v\n", err)
		return
	}

	// poll not used in this itertion as this is just for one round trip sake

	// 7. Prove round trip
	fmt.Println("\n=== Parsed Gundo Message ===")
	fmt.Printf("Type:      %s\n", message.Type)
	fmt.Printf("Task ID:   %s\n", message.TaskID)
	fmt.Printf("Action:    %s\n", message.Action)
	fmt.Printf("Timestamp: %s\n", message.Timestamp)

	fmt.Println("\n[+] Controller round trip successful")
}

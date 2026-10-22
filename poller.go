package main

import (
	"fmt"
	"time"
)

func pollComments(issueNumber int) {

	// keeing track of the last seen comment
	var lastSeenCommentID int64

	// grab list of comments on an issue number
	comments, err := listComments(issueNumber)
	if err != nil {
		fmt.Printf("Error listing comments: %v\n", err)
		return
	}

	// make sure there are comments to set the baseline
	if len(comments) > 0 {
		lastSeenCommentID = comments[len(comments)-1].ID // set the last seen comment to the last comment in the list
	}
	for {

		comments, err = listComments(issueNumber)

		if err != nil {
			fmt.Printf("Error listing comments: %v\n", err)
			return
		}

		// 1. checks for new comments to update comments slice
		// 2. if the comment id is greater than the las seen comment, new comment is detected
		// 3. sets that new comment as the last seen comment and prints it out
		// 4. if the comment is not a valid GundoMessage, it will still be marked as seen and will not be processed again

		// IMPORTANT: this loop can get very slow so this can be an optimization point later.
		// O(m * n) - m represents each poll and n represents each comment in the pull
		// can be n^2 if new comment on each poll.

		for _, comment := range comments {
			if comment.ID > lastSeenCommentID {

				// structured version that go can understand and parse and be connected to frontend in some way
				// but thats for later :)

				message, err := parseGundoMessage(comment.Body)
				if err != nil {
					fmt.Printf("Error parsing GundoMessage: %v\n", err)
				} else {
					fmt.Printf("Parsed GundoMessage: Type: %s, TaskID: %s, Action: %s, Output: %s, Timestamp: %s\n",
						message.Type,
						message.TaskID,
						message.Action,
						message.Output,
						message.Timestamp)
				}
				// mark the comment as seen whether valid or not
				lastSeenCommentID = comment.ID
			}
		}

		time.Sleep(10 * time.Second) // Poll every 10 seconds
	}
}

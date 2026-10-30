package main

import (
	"fmt"
	"time"
)

// Each consumer owns its own bookmark. It is intentionally in memory: client
// restarts still establish a fresh baseline without replaying historical tasks.
type commentCheckpoint struct {
	LastSeenCommentID int64
	CreatedAt         time.Time
}

func (checkpoint commentCheckpoint) since() time.Time {
	if checkpoint.CreatedAt.IsZero() {
		// An empty mailbox has no bookmark yet; an unfiltered read is still cheap.
		return time.Time{}
	}
	// GitHub's since filter is strict. Overlap by one second so another comment
	// with the same timestamp remains visible; the ID check removes repeats.
	return checkpoint.CreatedAt.Add(-time.Second)
}

func (checkpoint *commentCheckpoint) poll(issueNumber int, list func(int, time.Time) ([]IssueComment, error)) ([]IssueComment, error) {
	comments, err := list(issueNumber, checkpoint.since())
	if err != nil {
		// list returns only after every page succeeds. Failed reads keep the bookmark.
		return nil, err
	}
	return checkpoint.accept(comments)
}

// accept filters overlap and validates the whole batch before committing any
// progress. Its returned comments are new transport messages, including non-tasks.
func (checkpoint *commentCheckpoint) accept(comments []IssueComment) ([]IssueComment, error) {
	next := *checkpoint
	var unseen []IssueComment
	for _, comment := range comments {
		if comment.ID <= next.LastSeenCommentID {
			continue
		}
		if comment.CreatedAt.IsZero() {
			return nil, fmt.Errorf("GitHub comment %d is missing created_at", comment.ID)
		}
		unseen = append(unseen, comment)
		next.LastSeenCommentID = comment.ID
		next.CreatedAt = comment.CreatedAt
	}
	*checkpoint = next
	return unseen, nil
}

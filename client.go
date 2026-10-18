package main

import (
	"time"
	"uuid"
)

// fixed client version rep
const clientVersion = "0.1.0"

// in memory state that payload/process is running. It gives us a place to handle client specific behavior

type Client struct {
	ID        string
	Version   string
	StartedAt time.Time
	// set of task ID already handled to avoid duplicate execution
	Processed map[string]struct{}
}

// function to create a new client struct in memory and returns its address.
// that way we can actually modify the values in memory rather than a copy
func newClient() *Client {
	return &Client{
		ID:        uuid.New().String(),
		Version:   clientVersion,
		StartedAt: time.Now().UTC(),
		Processed: make(map[string]struct{}),
	}
}

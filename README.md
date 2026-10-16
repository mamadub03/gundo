# Gundo

Gundo is a Go-based controller/agent platform built for authorized
collegiate red-team and cyber defense competitions.

## Architecture

Controller
    ↓
GitHub Issues / Comments
    ↑
Agent

GitHub Issues act as persistent agent mailboxes, while comments carry
structured task and result messages.

## Project Structure

main.go       - controller entry point
github.go     - GitHub transport
messages.go   - Gundo message protocol
models.go     - shared data models
poller.go     - comment polling logic
agent/        - agent implementation
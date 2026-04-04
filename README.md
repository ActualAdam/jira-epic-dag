# jira-epic-dag

A Go TUI app using Bubble Tea that connects to Jira Cloud and displays epic dependency graphs.

## Prerequisites

- **Go** 1.21+ (`brew install go`)
- **Jira API token** — generate one at https://id.atlassian.com/manage-profile/security/api-tokens

## Configuration

Set these environment variables:

```sh
export JIRA_HOST=https://yoursite.atlassian.net
export JIRA_EMAIL=you@example.com
export JIRA_API_TOKEN=your-api-token
```

## Run

```sh
go run ./cmd/jira-epic-dag
```

## Build & Verify

```sh
go build ./...
go vet ./...
```

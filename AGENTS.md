# AGENTS.md — RingRouter Project Conventions

## Overview

RingRouter is a self-deployed LLM API gateway. It routes requests to multiple AI providers through a unified OpenAI-compatible API.

## Tech Stack

- **Backend**: Go 1.27+, net/http, GORM v2
- **Database**: SQLite, MySQL, PostgreSQL (all three supported)
- **Frontend**: React + TypeScript + Vite + Tailwind CSS + shadcn/ui (planned)

## Architecture

```
internal/
├── config/      — Configuration loading
├── database/    — Database connection & migration
├── handler/     — HTTP request handlers
├── middleware/   — Auth, logging, CORS
├── model/       — GORM data models
├── provider/    — LLM provider adapters
└── router/      — HTTP routing
```

## Rules

- Follow `AGENTS.md` at project root for general coding standards
- Go code: `gofmt` + `go vet` before commit
- Database: always use GORM parameterized queries, never raw SQL concatenation
- New providers: implement `provider.Provider` interface
- All external input must be validated

## Build

```bash
make build    # Build binary
make run      # Build and run
make test     # Run tests
make lint     # Run go vet
```
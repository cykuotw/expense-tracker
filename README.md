# Expense Tracker

A Splitwise-like fullstack app for tracking shared expenses, balances, groups, and invitations.

## Stack

- Backend: Go, Gin, PostgreSQL
- Frontend: React, TypeScript, Vite, Tailwind CSS, DaisyUI
- Auth: cookie-based auth, refresh tokens, Google OAuth

## Local Run

### Backend

Requirements:
- Go
- PostgreSQL
- migration tool if you want to create new migrations

Common commands:

```bash
make build
make run
```

Database migration commands:

```bash
make migrate-up
make migrate-down
make migrate-step n=1
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Production build:

```bash
cd frontend
npm run build
```

## Config

The app uses environment variables for:
- backend listen address
- frontend/API origins
- PostgreSQL connection
- JWT and refresh token settings
- Google OAuth client settings
- cookie and CORS behavior

For local development, frontend and backend should use matching origins consistently, for example all `localhost` instead of mixing `localhost` and `127.0.0.1`.

## Current Auth / Browser Setup

- cookie-based auth with access and refresh tokens
- CSRF protection for mutating browser requests
- split-origin friendly config for frontend and API
- Google OAuth callback handled by the backend

## Deployment Direction

Current deployment plan is:
- frontend on S3 + CloudFront
- backend on EC2 behind nginx
- PostgreSQL on a separate EC2 instance
- GitHub Actions for CI/CD

HTTPS is expected to terminate at nginx for the backend.

## AI Agent Workflow

This repo uses a lightweight task-note workflow for AI-assisted changes:
- review `.agent/tasks/todo.md` before starting implementation work
- keep active task notes under `.agent/tasks/todo/`
- move completed task notes to `.agent/tasks/done/` only when the user explicitly confirms the task is done
- keep the task index files aligned with the task note files when task-tracking changes are made

Use this workflow to preserve context between agent sessions and make in-flight work easier to audit.
Refer to `AGENTS.md` for the full repository-specific agent instructions and constraints.

## Roadmap

Planned follow-up work includes:
- tracing and monitoring logs
- ISO-based multi-currency support
- OCR support for receipts
- frontend migration from DaisyUI to shadcn/ui
- deployment automation and backup hardening

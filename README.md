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

See the example environment files for the supported configuration:

- `backend/.env.example`

Use the backend example env file as the source of truth for local setup and backend deployment configuration. Frontend API routing is configured through `public/runtime-config.js` in local development and deploy-generated `dist/runtime-config.js` in deployed builds.

## Deployment

Deployment automation lives under `deployment/`.

- `make deploy`: deploy backend and frontend
- `make deploy all`: apply infra, deploy backend, deploy frontend, then deploy edge
- `make deploy infra`: apply Terraform-managed infrastructure only
- `make deploy backend`: build and release the backend only
- `make deploy frontend`: build and publish frontend assets only
- `make deploy edge`: build and release nginx/TLS edge only

See `deployment/README.md` for the deployment contract and command layout.

## AI Agent Workflow

This repo uses a lightweight task-note workflow for AI-assisted changes:

- review `.agents/tasks/todo.md` before starting implementation work
- keep active task notes under `.agents/tasks/todo/`
- move completed task notes to `.agents/tasks/done/` only when the user explicitly confirms the task is done
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

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

Requires Node 22.13.1 or a compatible Node 22 release, plus pnpm 10.23.0.
Both tool versions are declared in `frontend/package.json`; `.node-version` is
provided for compatible local version managers.

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run dev
```

Production build:

```bash
cd frontend
pnpm run build
```

## Config

See the example environment files for the supported configuration:

- `backend/.env.example`

Use the backend example env file as the source of truth for local setup and backend deployment configuration. Frontend API routing is configured through `public/runtime-config.js` in local development and deploy-generated `dist/runtime-config.js` in deployed builds.

## Deployment

Deployment automation lives under `deployment/`.

### Primary: serverless

The primary deployment target is the budget-oriented serverless path. Run the
following once to create the external deployment configuration template:

```bash
make deploy ACTION=init
```

Common serverless commands:

- `make deploy`: automatically select the next safe deployment action.
- `make deploy ACTION=plan`: review the planned serverless changes.
- `make deploy ACTION=deploy`: apply a serverless deployment.
- `make deploy ACTION=update SCOPE=migrations|backend|frontend|all`: update a specific serverless scope.
- `make deploy ACTION=status`: inspect serverless deployment status.
- `make deploy ACTION=destroy`: destroy the serverless deployment.

`deployment/serverful/` is retained solely as archived reference code. It is
not a supported deployment path; use the serverless `make deploy` commands
above for every environment.

See `deployment/README.md` for the deployment contract and command layout.

## AI Agent Workflow

This repo uses a lightweight task-note workflow for AI-assisted changes:

- review `.agents/tasks/todo.md` before starting implementation work
- keep active task notes under `.agents/tasks/todo/`
- move completed task notes to `.agents/tasks/done/` only when the user explicitly confirms the task is done
- keep the task index files aligned with the task note files when task-tracking changes are made

Use this workflow to preserve context between agent sessions and make in-flight work easier to audit.
Refer to `AGENTS.md` for the full repository-specific agent instructions and constraints.

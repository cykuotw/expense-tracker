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

## Roadmap

Planned follow-up work includes:
- tracing and monitoring logs
- ISO-based multi-currency support
- OCR support for receipts
- deployment automation and backup hardening

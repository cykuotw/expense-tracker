# System Architecture

Expense Tracker is a modular-monolith application for shared groups, expenses,
balances, and invitation-based registration. This document is a durable map of
component ownership and trust boundaries. It describes the primary target
architecture, not a statement about a live environment.

For product setup and commands, begin with the [README](README.md). For
deployment procedures and configuration, use the deployment documents linked
below as the source of truth.

## Deployment: Unified Serverless

The unified serverless deployment is the only supported deployment path. It
keeps the request-serving backend small and gives the deployment workflow
explicit control over its database-connection budget.

```text
Browser
  |
  +-- static application --> CloudFront --> S3
  |
  +-- API requests ------> API Gateway --> Worker Lambda
                                             |
                                             v
                                        VPC security groups
                                             |
                                             v
                                      PostgreSQL on EC2

Deployment workflow --> Bootstrap Lambda --> migrations and bootstrap work
```

- **Frontend:** React, TypeScript, Vite, Tailwind CSS, and DaisyUI produce the
  static application published through the frontend distribution.
- **Worker Lambda:** runs the Go/Gin HTTP application. API Gateway is the
  public API entry point; the Worker is activated only after deployment
  configuration is published.
- **Bootstrap Lambda:** performs explicit migration and bootstrap work before
  Worker releases that depend on it. Migrations are never a request-time
  responsibility.
- **Database:** PostgreSQL is stateful infrastructure on its own EC2 host.
  The Worker and Bootstrap functions connect through the VPC and security-group
  boundary; it is not a public application API.
- **Deployment owner:** `deployment/serverless/` owns the unified serverless
  implementation, its deployment sequencing, and its Terraform state.

The canonical operator guide is
[deployment/serverless/README.md](deployment/serverless/README.md). It owns
the detailed deployment, update, verification, resume, and destroy behavior.

## Retained Serverful Reference Code

The repository retains `deployment/serverful/` as archived implementation and
infrastructure reference code only. It is not a backup deployment path and
must not be used to provision, update, or destroy environments. All supported
deployments, including frontend publication, use `deployment/serverless/`.

Use [deployment/README.md](deployment/README.md) and
[deployment/serverless/README.md](deployment/serverless/README.md) for the
supported operator workflow.

## Application Boundaries

```text
frontend/src
  pages, components, contexts, hooks, API client
             |
             v
backend/internal/tracker
  application assembly and route groups
             |
             v
backend/services/<domain>
  HTTP handlers, request extraction/validation, domain operations, stores
             |
             v
PostgreSQL
```

- The frontend owns presentation, local form state, and user-facing feedback.
  It calls the backend through the shared API client; it must not be treated as
  the authorization or accounting boundary.
- The frontend's PWA worker may cache only public static shell assets. API,
  authentication, CSRF, and other credential-bearing requests remain
  network-only; offline mode never exposes cached expense or account data.
  The serverless publisher serves versioned assets as immutable and entry-point
  files (including the worker and runtime configuration) with `no-cache`.
- `backend/internal/tracker` assembles the Gin application and its public,
  authenticated, and administrator route groups.
- `backend/services/` is organized by domain, including auth, users, groups,
  expenses, invitations, and administration. A domain's handlers own HTTP
  concerns; stores own persistence; multi-step business operations belong at
  the domain boundary rather than in the frontend.
- `backend/types` contains shared contracts and models. Domain-only types
  should stay close to their owning domain unless sharing is genuinely needed.

## Frontend Browser Compatibility

Browser behavior is part of the frontend quality boundary. For every frontend
change, assess whether its behavior can vary by browser or platform. When it
can, retain focused automated coverage and manually verify the affected flow
in Safari before release: iOS Safari for mobile behavior and macOS Safari when
the desktop interaction differs.

This includes, but is not limited to, custom controls, forms, focus and blur,
touch and gestures, software-keyboard behavior, viewport and safe-area layout,
scrolling, date/number/locale formatting, media or file APIs, PWA behavior,
and CSS features with browser-dependent rendering. Chromium verification alone
is not sufficient for such changes. Verify the user-visible behavior, not only
that the page renders: opening and dismissal, selection and submission,
keyboard and touch operation, navigation, state updates, and responsive
layout as applicable.

## Authentication and Authorization

- Local password and Google sign-in create application sessions using access
  and refresh tokens. Browser clients use cookie-based authentication; the
  backend also supports bearer-token extraction where required by its request
  boundary.
- Protected routes derive the authenticated actor from the access token. The
  request body is not authority for a user, group, or expense relationship.
- Authorization belongs in backend domain handling and uses persisted resource
  and group relationships. Frontend route guards improve navigation only; they
  do not grant access.
- State-changing browser requests are protected by the backend's CSRF and
  trusted-origin policy.

See the auth and middleware packages under `backend/services/` for the
implemented request flow. API route and response details should be documented
by the API contract rather than duplicated here.

## Expense Domain Model and Invariants

- A group contains members and expenses. An expense has descriptive and
  monetary data, may have item rows, and has ledger rows that describe who
  lent and borrowed each share.
- Balances are derived from unsettled, non-deleted expense ledgers. Expense
  creation, editing, deletion, and settlement are accounting-affecting
  operations and must keep their related writes consistent.
- The backend, not the browser, is responsible for validating trusted actor
  identity, resource membership, currency and amount rules, and the final
  consistency of split amounts and derived balances.
- Soft deletion removes an expense from normal balance and list calculations;
  it is not the same as a permanent purge.
- Dates and timestamps, exact monetary representation, mutation atomicity,
  idempotency, and audit history are cross-cutting correctness boundaries.
  They must be changed deliberately with schema, API, and frontend behavior
  kept compatible.

This is an ownership map, not an API specification. When a change alters a
domain rule, update its domain tests and API contract as well as this document
when the architectural boundary changes.

## Configuration, Secrets, and Operations

- Human-edited deployment configuration and credentials stay outside version
  control. The serverless deployer creates protected temporary projections
  rather than storing runtime secrets in Terraform state or repository files.
- Deployment and repository Python tooling must run through `uv` (for example,
  `uv run python ...`); the system `python3` version is not a supported
  execution environment.
- Runtime configuration, database credentials, signing keys, invitation
  secrets, cookies, and external-provider credentials must never be added to
  source code, public documentation, logs, or client-side storage.
- PostgreSQL is durable state. The supported deployment uses daily encrypted
  logical PostgreSQL dumps in a private S3 bucket with bounded retention and a
  separate, on-demand isolated restore verification. This is backup and
  recovery coverage, not high availability or failover.

Refer to the relevant deployment guide for configuration ownership and
operational procedures. Do not copy secret-bearing examples or environment
values into this document.

## Keeping This Document Current

Update `ARCHITECTURE.md` in the same change when any of these change:

- the supported deployment topology;
- public request-routing, authentication, authorization, or data trust
  boundaries;
- a domain's ownership or persistence model;
- the source-of-truth location for deployment or operational documentation.

Keep it concise. Link to canonical detailed documentation instead of repeating
commands, environment settings, resource identifiers, or implementation
history.

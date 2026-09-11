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

EventBridge --> Push Sender Lambda (outside VPC) --> public Web Push providers
                    |
                    +-- IAM Invoke --> Delivery Lambda (VPC) --> PostgreSQL
```

- **Frontend:** React, TypeScript, Vite, Tailwind CSS, and shadcn/Radix components produce the
  static application published through the frontend distribution.
- **Worker Lambda:** runs the Go/Gin HTTP application. API Gateway is the
  public API entry point; the Worker is activated only after deployment
  configuration is published.
- **Bootstrap Lambda:** performs explicit migration and bootstrap work before
  Worker releases that depend on it. Migrations are never a request-time
  responsibility.
- **Notifications:** a scheduled Sender outside the VPC invokes the private
  Delivery Lambda to claim work and acknowledge results. Delivery owns database
  access; Sender owns Web Push transport and VAPID signing. Neither a NAT Gateway
  nor a paid VPC endpoint is required. IAM permits Sender to invoke only Delivery;
  neither function exposes an HTTP endpoint. PostgreSQL leases prevent concurrent
  claims, and expired leases permit recovery after interrupted sends. Delivery is
  at-least-once: a provider may accept a push before its acknowledgement is lost.
  See [notification component](backend/services/notification/README.md).
- **Database:** PostgreSQL is stateful infrastructure on its own EC2 host.
  The Worker and Bootstrap functions connect through the VPC and security-group
  boundary; it is not a public application API. Operator host inspection uses
  one Terraform-managed EC2 Instance Connect Endpoint with a dedicated,
  SSH-only security-group path; it does not make the database public.
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
  files with `no-cache`. In particular, the generated `service-worker.js` must
  remain in the publisher's mutable-file set; serving it with immutable caching
  can leave installed clients on a stale application shell and prevent the reload
  prompt from discovering new releases. Any change to the generated worker
  filename must update the publisher and its cache-header regression test in the
  same change.
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

Automated WebKit coverage does not replace the required manual Safari check.

## Authentication and Authorization

- Local password and Google sign-in create application sessions using access
  and refresh tokens. Browser clients use cookie-based authentication; the
  backend also supports bearer-token extraction where required by its request
  boundary.
- PostgreSQL stores only refresh-token hashes. Rotation atomically consumes a
  predecessor and creates one successor in the same token family; verified
  reuse revokes that family, while unrelated sessions remain active.
- Invitation links carry an opaque one-time secret only in their URL fragment.
  The browser exchanges it in a CSRF-protected request for a short-lived,
  HttpOnly registration-session cookie; PostgreSQL retains hashes of both the
  invitation secret and temporary registration session, never their raw values.
- Protected routes derive the authenticated actor from the access token. The
  request body is not authority for a user, group, or expense relationship.
- Authorization belongs in backend domain handling and uses persisted resource
  and group relationships. Frontend route guards improve navigation only; they
  do not grant access.
- State-changing browser requests are protected by the backend's CSRF and
  trusted-origin policy.

See the auth and middleware packages under `backend/services/` for the
implemented request flow. A generated OpenAPI contract is not yet the maintained
source of truth. For current HTTP behavior, inspect route assembly in
`backend/internal/tracker`, domain handlers and tests in `backend/services/`,
shared types in `backend/types`, and their consumers in `frontend/src/types`
and `frontend/src/lib/api.ts`. Do not duplicate endpoint details here.

## Expense Domain Model and Invariants

- A group contains members and expenses. An expense has descriptive and
  monetary data, may have item rows, and has ledger rows that describe who
  lent and borrowed each share.
- Each expense stores one canonical allocation mode and one allocation row per
  selected participant. The allocation rows preserve user-entered source values
  for equal, exact-amount, percentage, and equal-plus-adjustment modes; selected
  zero-value participants remain explicit rows rather than being inferred from
  ledger amounts.
- Balances are derived from unsettled, non-deleted expense ledgers. Expense
  creation, editing, deletion, and settlement are accounting-affecting
  operations and must keep their related writes consistent.
- The backend, not the browser, is responsible for validating trusted actor
  identity, resource membership, currency and amount rules, and the final
  consistency of split amounts and derived balances. Create and update requests
  submit the allocation configuration, while the backend deterministically
  derives final ledger shares using the currency precision and stable
  participant ordering. Allocation parsing, validation, and ledger derivation
  live in `backend/services/expense/allocation`; HTTP routes coordinate the
  request and persistence boundaries. Allocation, ledger reconciliation, and
  balance updates share the expense mutation transaction.
- Soft deletion removes an expense from normal balance and list calculations;
  it is not the same as a permanent purge.
- An expense occurrence is a calendar day, stored as `expense.occurred_on
  DATE` and exchanged as `occurredOn` in strict `YYYY-MM-DD` form. Browser
  code must retain and format its numeric components directly rather than
  parsing it as a JavaScript `Date` or treating it as midnight in a time zone.
- Audit, expiry, and revocation values are instants. Non-user temporal columns
  use PostgreSQL `TIMESTAMPTZ`; backend writes and API values are normalized
  to UTC. The users table retains its existing timestamp contract.
- During the occurrence-date compatibility window, expense responses retain
  legacy `expenseTime`, and legacy null `occurred_on` rows derive a read
  fallback from the UTC day of that value. New clients use `occurredOn` as the
  sole occurrence-date source; legacy fields are removed only in a later
  contract cleanup.
- Dates and timestamps, exact monetary representation, mutation atomicity,
  idempotency, and audit history are cross-cutting correctness boundaries.
  They must be changed deliberately with schema, API, and frontend behavior
  kept compatible.

This is an ownership map, not an API specification. When a change alters a
domain rule, update its domain tests and affected request/response types and
consumers together. Once a maintained API specification is introduced, update
it in the same change. Update this document when the architectural boundary
changes; establishing a contract-generation system is a separate change.

## Configuration, Secrets, and Operations

- Human-edited deployment configuration and credentials stay outside version
  control. The serverless deployer creates protected temporary projections
  rather than storing runtime secrets in Terraform state or repository files.
- Runtime configuration, database credentials, signing keys, invitation
  secrets, cookies, and external-provider credentials must never be added to
  source code, public documentation, logs, or client-side storage.
- Request-serving release binaries validate their complete security-relevant
  configuration before constructing a database connection. The serverless
  Worker is compiled with an authoritative release marker; deployment-time
  checks remain defense in depth rather than the application security boundary.
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

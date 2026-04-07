# Agent Rules

## Security

- Do not read or access `.env`, `.env.*`, `.tfvars`, `secrets/**`, or `**/*.pem` files.
- If access to any sensitive file is required, ask the user first.

## Development process

- First read `.agent/tasks/todo.md`.
- You make the plan before the code change, and ask for permission before any code changes.
- Update `.agent/tasks/todo.md` with the plan so other agents can follow it.
- Active task notes live in `.agent/tasks/todo/`; completed task notes live in `.agent/tasks/done/`.
- Task notes in `.agent/tasks/todo/` and `.agent/tasks/done/` must use YAML frontmatter.
- Frontmatter fields and allowed values are defined in `.agent/tasks/task-metadata.md`.
- Required frontmatter keys are `status`, `priority`, `type`, and `kind`; `area` is optional.
- `status` must match the directory: use `todo` under `.agent/tasks/todo/` and `done` under `.agent/tasks/done/`.
- Keep `.agent/tasks/todo.md` and `.agent/tasks/done.md` as plain indexes; task metadata belongs in the task files, not in the indexes.
- Do not encode priority in task filenames; use a stable descriptive slug and keep priority only in frontmatter.
- Prefer task notes to use the sections `Goals`, `Scope`, and `Acceptance Criteria` when they help clarify the work.
- For multi-phase task series, use a shared `Series` label in each note and add a `Phases` section in the overview note when useful.
- Keep `.agent/tasks/todo.md` and `.agent/tasks/done.md` in sync with the task note files.
- When a task is completed, move its note from `.agent/tasks/todo/` to `.agent/tasks/done/`, remove it from `.agent/tasks/todo.md`, and add it to `.agent/tasks/done.md`.
- Do not mark a task done or move it to `.agent/tasks/done/` unless the user explicitly tells you to.
- A plan is not required for skill installation or task-tracking housekeeping such as updating todo/done indexes, moving task notes, or keeping those files aligned.
- Do not use Git commands unless the user explicitly asks for the specific Git command to run.

## Shared skills

- Shared repo-local skills live under `.agent/skills`; tracked installs are recorded in `skills-lock.json`.
- When installing a shared skill for this workspace, prefer `.agent/skills`.
- Keep `skills-lock.json` in sync when adding or updating shared skills.
- Check `.agent/skills` before assuming a required skill is missing.

## Default Response Length

Unless the user explicitly asks for detail, default to concise answers.

Rules:

- Start with the direct answer. No preamble.
- Short to medium by default. When uncertain, choose short.
- Expand only if: (1) the user asks, (2) the task is genuinely complex, or (3) brevity would reduce precision, safety, or usefulness.
- Do not expose internal reasoning unless it materially helps.
- Do not add background sections unless necessary.
- Do not add summaries or closing remarks unless they add necessary clarity.
- Prefer direct, dense, low-filler answers.

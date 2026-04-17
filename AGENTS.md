# Agent Rules

## Security

- Do not read or access `.env`, `.env.*`, `.tfvars`, `secrets/**`, or `**/*.pem` files.
- If access to any sensitive file is required, ask the user first.

## Development process

- First read `.agents/tasks/todo.md`.
- You make the plan before the code change, and ask for permission before any code changes.
- Update `.agents/tasks/active-plan.md` with the active implementation plan so other agents can follow the current attempt.
- Keep `.agents/tasks/active-plan.md` for transient execution state only; durable task requirements belong in task note files.
- Remove or clear `.agents/tasks/active-plan.md` before the final response when the implementation attempt is complete or abandoned.
- Active task notes live in `.agents/tasks/todo/`; completed task notes live in `.agents/tasks/done/`.
- Task notes in `.agents/tasks/todo/` and `.agents/tasks/done/` must use YAML frontmatter.
- Frontmatter fields and allowed values are defined in `.agents/tasks/task-metadata.md`.
- Required frontmatter keys are `status`, `priority`, `type`, and `kind`; `area` is optional.
- `status` must match the directory: use `todo` under `.agents/tasks/todo/` and `done` under `.agents/tasks/done/`.
- Keep `.agents/tasks/todo.md` and `.agents/tasks/done.md` as plain indexes; task metadata belongs in the task files, not in the indexes.
- Do not put active implementation plans in `.agents/tasks/todo.md` or `.agents/tasks/done.md`; keep those files as plain indexes.
- Do not encode priority in task filenames; use a stable descriptive slug and keep priority only in frontmatter.
- Prefer task notes to use the sections `Goals`, `Scope`, and `Acceptance Criteria` when they help clarify the work.
- For multi-phase task series, use a shared `Series` label in each note and add a `Phases` section in the overview note when useful.
- Keep `.agents/tasks/todo.md` and `.agents/tasks/done.md` in sync with the task note files.
- When a task is completed, move its note from `.agents/tasks/todo/` to `.agents/tasks/done/`, remove it from `.agents/tasks/todo.md`, and add it to `.agents/tasks/done.md`.
- Do not mark a task done or move it to `.agents/tasks/done/` unless the user explicitly tells you to.
- A plan is not required for skill installation or task-tracking housekeeping such as updating todo/done indexes, moving task notes, or keeping those files aligned.
- Git read-only commands are allowed without extra permission when they are used to inspect repo state, diffs, history, or tracked files.
- Allowed read-only Git commands include: `git status`, `git diff`, `git diff --cached`, `git show`, `git log`, `git branch --show-current`, `git ls-files`, `git grep`, and `git blame`.
- Do not run Git commands that modify the worktree, index, refs, remotes, stash, tags, or repository metadata unless the user explicitly asks for that exact operation.
- Prohibited write or state-changing Git commands include: `git add`, `git commit`, `git checkout`, `git switch`, `git restore`, `git reset`, `git clean`, `git merge`, `git rebase`, `git cherry-pick`, `git revert`, `git stash`, `git tag`, `git pull`, `git fetch`, and `git push`.
- If a Git command is not clearly read-only, ask the user before running it.

## Shared skills

- Shared repo-local skills live under `.agents/skills`; tracked installs are recorded in `skills-lock.json`.
- When installing a shared skill for this workspace, prefer `.agents/skills`.
- Keep `skills-lock.json` in sync when adding or updating shared skills.
- Check `.agents/skills` before assuming a required skill is missing.

## Subagents

- When the user explicitly asks to launch, use, delegate to, or parallelize work with a subagent, you may launch a single-use subagent for the requested task.
- You may also use a single-use subagent when the user has explicitly allowed subagents for work that benefits from an isolated context window, such as code review, focused testing or verification, large document summarization, or narrow codebase exploration.
- Keep each launched subagent narrowly scoped, with a clear question or responsibility.
- Do not use subagents as persistent memory, long-lived task owners, or background workers beyond their assigned task.
- When a launched subagent finishes, is no longer needed, or the attempt is abandoned, clean it up before the final response.

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

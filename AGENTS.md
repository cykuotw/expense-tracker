# Agent Rules

## Security

- Do not read or access `.env`, `.env.*`, `.tfvars`, `secrets/**`, `**/*.pem`, `**/runtime-env*.json`, or `**/worker-runtime*.json` files. This includes runtime configuration example files.
- If access to any sensitive file is required, ask the user first.
- Before reading review diffs, list changed paths and exclude the restricted files above. Read only permitted paths, and report any resulting coverage gap.

## Development process

- First read `.agents/tasks/todo.md`.
- Before architecture, deployment, authentication, authorization, expense-domain, or other cross-cutting changes, read `ARCHITECTURE.md`.
- For frontend changes, read the Frontend Browser Compatibility section of `ARCHITECTURE.md`.
- Make a plan before code changes and obtain permission before implementing it. A short plan is sufficient for a small, localized change. Approval covers implementation, relevant checks, and fixes within the approved scope; ask again only for scope expansion or separately restricted operations.
- A plan is not required for skill installation or task-tracking housekeeping such as updating todo/done indexes, moving task notes, or keeping those files aligned.
- Update `.agents/tasks/active-plan.md` with the active implementation plan so other agents can follow the current attempt.
- Keep `.agents/tasks/active-plan.md` for transient execution state only; durable task requirements belong in task note files.
- Remove or clear `.agents/tasks/active-plan.md` before the final response when the implementation attempt is complete or abandoned.
- If the user limits writable files and excludes the plan file, keep the plan and handoff in the conversation. Do not modify an existing plan outside the approved scope.
- Verify the requested behavior with checks appropriate to the change. For documentation-only changes, check references, consistency, and the diff; for behavior changes, run relevant tests and affected build or lint checks. Fix failures within scope, and report any remaining failure or unavailable check with its cause and next step. Do not describe unverified behavior as verified or release-ready, or claim speed or model-quality improvements without measurements.
- If Safari verification required by `ARCHITECTURE.md` is unavailable, record the affected flow, platform, automated checks performed, and remaining manual steps in the handoff. Implementation may be ready for review, but browser verification remains incomplete and must be completed before release.
- When blocked, preserve the remaining work and resumption steps in the active plan, or in the conversation when file writes are restricted. Implementation completion does not authorize task archival or release.
- Deployment and repository Python tooling, including skill scripts, must run through `uv` (for example, `uv run python ...`); do not substitute the system `python3` environment.

## Task tracking (read when creating or updating task notes)

- Active task notes live in `.agents/tasks/todo/`; completed task notes live in `.agents/tasks/done/`.
- Read `.agents/tasks/task-metadata.md` for required YAML frontmatter, allowed values, directory/state mapping (including blocked tasks), and stable filename conventions.
- Keep `.agents/tasks/todo.md` and `.agents/tasks/done.md` as plain indexes, synchronized with task notes. Metadata and durable requirements belong in task notes; active implementation plans belong in the active plan file.
- Prefer task notes to use the sections `Goals`, `Scope`, and `Acceptance Criteria` when they help clarify the work.
- For multi-phase task series, use a shared `Series` label in each note and add a `Phases` section in the overview note when useful.
- Only when the user explicitly tells you to mark a task done, update its status, move its note from `.agents/tasks/todo/` to `.agents/tasks/done/`, and update both indexes.

## Git permissions

- Git read-only commands are allowed without extra permission when they are used to inspect repo state, diffs, history, or tracked files.
- Allowed read-only Git commands include: `git status`, `git diff`, `git diff --cached`, `git show`, `git log`, `git branch --show-current`, `git ls-files`, `git grep`, and `git blame`.
- Do not run Git commands that modify the worktree, index, refs, remotes, stash, tags, or repository metadata unless the user explicitly asks for that exact operation.
- Prohibited write or state-changing Git commands include: `git add`, `git commit`, `git checkout`, `git switch`, `git restore`, `git reset`, `git clean`, `git merge`, `git rebase`, `git cherry-pick`, `git revert`, `git stash`, `git tag`, `git pull`, `git fetch`, and `git push`.
- If a Git command is not clearly read-only, ask the user before running it.

## Shared skills

- Use the appropriate workspace skill when a task matches its documented scope, following the selection and compatibility rules below. Apply it before implementation; a relevant skill may also be used while creating or refining a task note.
- Shared repo-local skills live under `.agents/skills`; tracked installs are recorded in `skills-lock.json`.
- Install shared skills for this workspace in `.agents/skills`, not a personal/global skill directory.
- Keep `skills-lock.json` in sync when adding or updating shared skills.
- Check `.agents/skills` before assuming a required skill is missing.
- Treat upstream skills as maintained dependencies. Do not edit, replace, update, or remove them or their lock entries without explicit authorization for that operation. Check the GitHub source recorded in `skills-lock.json` when an upstream update is requested; report differences before proposing installation.
- Apply skills to the workspace stack documented in `ARCHITECTURE.md`; generic examples or another project's assumptions do not redefine it. Use `frontend/package.json` and `frontend/components.json` for installed tooling and component configuration.
- Choose skills by the changed concern: `shadcn` for its components, `tailwind-design-system` for shared Tailwind tokens and patterns, `ui-ux-pro-max` for visual/interaction decisions, and `vercel-react-best-practices` for React implementation. Use `design` for creative deliverables and route to its relevant specialized skill. Add complementary guidance only when its concern is affected; a local UI fix does not require a new design system.
- For reviews, establish whether the request covers a commit range, staged changes, unstaged changes, or a combination before selecting the diff. Use `code-review` for general review and `find-bugs` for focused bug discovery; neither authorizes fixes or additional security-scan workflows.
- Resolve bundled script and reference paths from the installed skill directory and verify they exist; keep the working directory appropriate to the script's project inputs. Reuse valid scripts. If a required dependency, CLI, or browser capability is unavailable, report the limitation, continue independent authorized work, and ask before installing or modifying anything outside scope. Do not claim checks succeeded or silently bypass a required check.

## Public documentation

- Keep public README files and documentation focused on product, architecture, and operator usage. Do not expose internal task tracking, phase labels, agent workflows, task status, private filesystem paths, credentials, account/resource identifiers, or other private operational details.

## Personal preferences

- If `.agents/preference.md` exists, read it and follow it for user-specific communication preferences.
- These preferences are personal/session behavior only; they do not override project security, development, task-tracking, or tooling rules.

## Subagents

- When the user explicitly asks to launch, use, delegate to, or parallelize work with a subagent, you may launch a single-use subagent for the requested task.
- You may also use a single-use subagent when the user has explicitly allowed subagents for work that benefits from an isolated context window, such as code review, focused testing or verification, large document summarization, or narrow codebase exploration.
- Keep each launched subagent narrowly scoped, with a clear question or responsibility.
- Do not use subagents as persistent memory, long-lived task owners, or background workers beyond their assigned task.
- When a launched subagent finishes, is no longer needed, or the attempt is abandoned, clean it up before the final response.

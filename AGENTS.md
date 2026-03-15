# Agent Rules

## Security

- Do not read or access `.env`, `.env.*`, `secrets/**`, or `**/*.pem` files.
- If access to any sensitive file is required, ask the user first.

## Development process

- First read the `.agent/tasks/todo.md` to see what is on the list
- You make the plan before the code change, and ask for permission before any code changes.
- The plan should be updated in the `.agent/tasks/todo.md` in order to keep track between multiple agents.
- Active task notes live under `.agent/tasks/todo/` and completed task notes live under `.agent/tasks/done/`.
- Keep `.agent/tasks/todo.md` and `.agent/tasks/done.md` in sync with the task note files.
- When a task is completed, move its task note from `.agent/tasks/todo/` to `.agent/tasks/done/`, remove it from `.agent/tasks/todo.md`, and add it to `.agent/tasks/done.md`.
- A plan is not required for task-tracking housekeeping changes such as updating the todo or done indexes, moving task notes between those folders, or keeping these tracking files aligned.
- You do not have the permission of using all the Git related command. Unless the user will explicitly ask you to do so with the very specific git command that you supposed to do, you should not do any Git command at all.

## Shared skills

- Shared repo-local skills live under `.agent/skills`.
- Tracked shared skills for this repo are listed in `SKILLS.md`.
- When installing a new skill for this workspace, prefer `.agent/skills` so other agents working in this repo can access it.
- Update `SKILLS.md` when adding or updating a shared repo skill.
- Check `.agent/skills` before assuming a required skill is missing.

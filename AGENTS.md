# Agent Rules

## Security

- Do not read or access `.env`, `.env.*`, `secrets/**`, or `**/*.pem` files.
- If access to any sensitive file is required, ask the user first.

## Development process

- First read the .agent/todo.md to see what is on the list
- You make the plan before the code change, and ask for permission before any code changes.
- The plan should be updated in the .agent/todo.md in order to keep the track between multiple agents.
- You do not have the permission of using all the Git related command. Unless the user will explicitly ask you to do so with the very specific git command that you supposed to do, you should not do any Git command at all.

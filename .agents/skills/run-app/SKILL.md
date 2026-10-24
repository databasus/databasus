---
name: run-app
description: Start, reuse and stop the local Databasus app, and check its pages in the browser through the chrome-devtools MCP server. Use when a task needs to see or click through the web interface, to check a UI change, or to launch the app locally.
---

# Run the app and check it in the browser

The browser server is registered for Claude Code as `chrome-devtools`. It opens a fresh browser profile in every session, so sign in at the start of each one. Whether the browser is headless or visible is the developer's choice in `.env`; leave `BROWSER_HEADLESS` as it is.

The ports to check and open are the ones the `links` process prints in [`process-compose.yaml`](../../../process-compose.yaml). The account, the seeded databases and their sources are listed in the package comment of the [seed command](../../../backend/cmd/seed_dev/doc.go).

## Before opening a page

Check both ports first, and act on what you find:

- **Both answer, and the app is from your worktree:** use it and leave it running when you hand the task over. The app is yours when the process listening on the backend port has its working directory inside your worktree; read it with `ss -ltnp` and `/proc/<pid>/cwd`.
- **Both answer, and the app is from another worktree:** leave it running, skip the browser check and tell the user it was skipped because the running app serves another worktree's code.
- **Only one answers:** start nothing, skip the browser check and tell the user which port is held.
- **Neither answers:** start `mise run dev -t=false` at the repository root in the background, and wait until the backend port accepts connections and the seed process has finished.

## Working with the app

- Sign in with the seeded account.
- When the app of your worktree lacks the seeded account or a seeded database, run `go run ./cmd/seed_dev` in `backend/` instead of creating them by hand. It creates only what is missing.
- A browser check adds to the tests the task names and never replaces them.

## When you are done

Stop a run you started yourself before handing the task over. Send SIGINT to the `process-compose` process whose working directory is your worktree root, wait for it to exit, and confirm that both ports are free. A run killed with SIGKILL leaves processes and containers behind, so never use it. If a port stays held, tell the user which one.

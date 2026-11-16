---
description: Run sovctl, the KubeUI dashboard's command-line twin (clusters, VMs, images, registry, mesh, connections, idle timer, AI agents)
argument-hint: "[command] [args]  e.g. status | timer set 15m | mesh overview | ai chat \"how many nodes?\""
allowed-tools: Bash(bin/sovctl:*), Bash(make sovctl), Bash(make -s sovctl)
---

Run the project's `sovctl` CLI for the user and report the result.

Arguments: `$ARGUMENTS`

1. If `bin/sovctl` does not exist or `ui/backend/cmd/sovctl/` is newer than it, build it first with `make -s sovctl`.
2. With no arguments, run `bin/sovctl status`. Otherwise run `bin/sovctl $ARGUMENTS` from the repo root.
   If the user described what they want in words instead of giving a command, translate it using
   `bin/sovctl help` and `bin/sovctl help <command>`, and say which command you ran.
3. Show the output as is (it is already a table), then add one or two sentences only if something needs
   attention: a red state, an error, a stuck deletion, or a failed probe.
4. If it cannot reach the backend, tell the user to start the UI (`make ui`) or set `SOVCTL_SERVER`.

Commands that change or destroy things need the user's explicit go-ahead **in this conversation** before you
add `-y` or run them: `cluster deploy`, `cluster delete`, `cluster istio`, `timer set` / `timer reset`,
`registry rm`, `mesh failover`, `connections scale`, `ai approve`, and `ai chat --yes`. If the user's
arguments already contain the command and they typed it themselves, that is the go-ahead, except that you
still never add `-y` on your own. Do not run `sovctl shell` (it needs an interactive terminal).

Use `-o json` when you need to read values programmatically rather than show a table.

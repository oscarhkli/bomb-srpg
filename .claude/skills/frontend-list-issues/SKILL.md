---
name: frontend-list-issues
description: Scan docs/frontend/*-log.md files for open (non-Solved) known issues and report them grouped by spec, without reading the full body of any log file. Use when the user asks what known issues are still open across frontend specs, check the known issues, find any potentially solveable issues, or when another frontend-* skill needs a cross-spec issue summary before proceeding.
allowed-tools: Bash
model: "haiku"
---

# Frontend List Issues

`frontend-sdd` writes one `{spec}-log.md` per spec to track implementation issues found during the build (see its "Known Issue Handling" section). As the number of specs grows, so does the number of these logs, and there's no single place to see what's still open. This skill answers that question on demand by grepping, not by maintaining a separate index file that would itself need upkeep.

## Why grep, not Read

Log files only grow — issues accumulate over a spec's lifetime and old logs stick around even after every issue in them is solved. Reading each log's full body to check its status would mean re-reading the same resolved issues over and over as more specs pile up. Grep sidesteps this: only the matching lines (an issue's title and its `Status:` line) enter context, never the surrounding prose or already-solved entries. Do the scan with `grep`/`rg` via Bash — do not `Read` the log files directly.

## Status vocabulary: blacklist, don't allowlist

Log entries end with a line like `**Status: Solved.**` or `**Status: Deferred.**` (see `frontend-sdd`'s "Known Issue Handling" section and `references/example-log.md` for the exact format). Today `Solved` is the only terminal status in use anywhere in this repo. Match against that blacklist, not an allowlist of open statuses — future specs may introduce new open-ish statuses (`Blocked`, `Needs-Repro`, etc.) and those should show up as open by default without this skill needing an edit. Only a status that's explicitly terminal (currently just `Solved`) should be excluded.

## Workflow

1. Find the logs:
   ```bash
   find docs/frontend -name '*-log.md' 2>/dev/null
   ```
   If this returns nothing, report "No log files exist yet — no open issues to report" and stop.

2. Pull each issue's title line together with its status line, then filter out terminal statuses:
   ```bash
   grep -n -B1 -H '\*\*Status:' docs/frontend/*-log.md 2>/dev/null | grep -v 'Status: Solved'
   ```
   - The first `grep` grabs every `**Status: ...**` line plus the line right before it (the numbered issue title, e.g. `3. **Bombs don't chain-explode diagonally.** ...`).
   - The second `grep -v` drops any pair whose status line contains a terminal status. Today that's just `Solved` — extend this filter only if a new terminal status is deliberately introduced, not for new open statuses.

3. Group the remaining lines by source file. Derive each spec's name from its log filename by stripping the `-log.md` suffix (e.g. `docs/frontend/p3-spec002-match-log.md` → `p3-spec002-match`).

4. Report, grouped by spec, something like:
   ```
   ## p3-spec002-match (docs/frontend/p3-spec002-match-log.md)
   - Bombs don't chain-explode diagonally. — Status: Deferred. No test changes made yet...
   - ...

   ## p3-spec001-match (docs/frontend/p3-spec001-match-log.md)
   - ...
   ```
   If every log's issues turn out to be `Solved` (the filtered grep produces no output at all), report "No open issues — all known issues across frontend specs are resolved."

Do not edit any `-log.md` file or spec file as part of this skill — it only reads and reports.

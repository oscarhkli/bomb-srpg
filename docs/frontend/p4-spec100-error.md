---
title: "Phase 4.X: Error & Warning Reporting Taxonomy"
---

# Phase 4.X: Error & Warning Reporting Taxonomy

> Warning: Unaudited Spec drafted by Agent while discussion.

## Context

`ErrorPanel` and `console.*` are both in use today, but which a given failure goes to is decided ad hoc per call site rather than by a rule. Two concrete gaps motivate this spec:

- Player-facing strings like `'Invalid bombPlaced event received from server'` (`MatchScene.ts`) describe a client/server desync — not something a player can act on — yet are shown verbatim in `ErrorPanel`.
- `MatchSettingsScene.ts`'s `getCatalog()` failure path discards the caught error entirely (`.catch(() => ...)`); an unexpected failure there leaves no trace anywhere, not even `console.error`.

The backend already has the taxonomy this spec needs: `mapError()` (`server/server_manager.go`) classifies every engine/server error into an HTTP status, and the split is already domain-vs-infra — 409 is a game-rule rejection written to be read by a player, 400/401/404 mean a stale session or client bug, and 500 is pre-sanitized to the literal string `"internal error"` (never leaks Go internals). The frontend's job is to route by that signal, not re-derive intent from message text.

`ConfirmDialog` was audited as part of scoping this spec and found to be already correctly scoped — every call site (`MatchScene.ts`) is a genuine yes/no confirmation (Resolve/Reset/Surrender), never error text. It is out of scope below.

## Goal

- Define a severity taxonomy for every error/warning currently surfaced by the client.
- Introduce one routing point every catch site goes through, so logging and player-messaging are decided together instead of independently per call site.
- Guarantee: nothing reaches `ErrorPanel` that a player cannot act on, and nothing that reaches the player skips `console.error`.

## Non-Goal

- Third-party telemetry (Sentry-like remote error tracking) — the dev diagnostic sink below is a local, dev-only file write, not a hosted service.
- Changing `mapError()` or the server's HTTP status taxonomy — it's already correct; this spec only consumes it.
- `ConfirmDialog` changes — confirmed out of scope by the audit above.
- Removing/gating `console.log` debug leftovers (`MatchScene.ts:885`, `boardRenderer.ts:195,204`) — these are dev-time noise, not part of the error/warning taxonomy; worth a separate cleanup pass.

## Error Taxonomy

| Tier | Meaning | Channel | Example |
| --- | --- | --- | --- |
| **Domain** | Server rejected a legal-looking action for a game-rule reason (`ApiError.status === 409`) | `ErrorPanel`, server text as-is | unit already moved, out of bomb range, match already ended |
| **Session** | Client is out of sync with server state (`status` 400/401/404) | `ErrorPanel`, generic session message + `console.warn` with the real status/text | stale token, room not found, malformed request body |
| **Infra** | Unexpected failure: 500, network drop, JSON parse failure, malformed/unrecognized event shape from the server | `ErrorPanel` generic fallback (never raw text) + `console.error` with full detail | `Failed to fetch`, `"internal error"`, `'Invalid bombPlaced event received from server'` |
| **Diagnostic** | Renderer/animation desync that degrades gracefully — no player action needed, no player message | `console.warn` only | missing stage texture, unhandled resolve-turn event type, bomb graphics missing on resolve |
| **Operational** | Best-effort background operation whose failure doesn't block the user flow | `console.error` only, flow continues | `deleteMatch()` failing during scene teardown |

## Routing Contract

One function, called from every catch site instead of `showError(...)`/bare `console.error(...)`:

```
reportError(err: unknown, context: { op: string; detail?: unknown }): string
```

- Always logs `err` (and `context.detail`, e.g. the offending event payload) to `console.error` or `console.warn` per tier.
- Returns the string the caller should hand to `ErrorPanel.show(...)` — server text verbatim for Domain, a generic message for Session/Infra.
- Classifies by `err instanceof ApiError ? err.status : undefined`; anything not an `ApiError` (network/parse failure, thrown `TypeError`) is Infra.
- Diagnostic and Operational sites don't call this — they stay direct `console.warn`/`console.error` calls, since no player message is ever produced.

## Dev Diagnostic Sink

Motivating case: [p4-spec011](p4-spec011-match.md)'s CPU stale-snapshot bug was diagnosed from a screenshot of `ErrorPanel`, because that was the only artifact reachable outside the browser. `console.error`/`console.warn` output isn't copy-pasteable in every environment this gets played in, and the payload (`context.detail` — the offending event, the `gameStateSnapshot`) never left the browser at all. This section makes that payload reachable outside the browser.

- Every `console.error`/`console.warn` call this spec produces — both from `reportError()` (Domain/Session/Infra) and the direct Diagnostic/Operational call sites — also goes through one sink function, `logDiagnostic(tier, op, detail)`. It keeps calling `console.*` exactly as before; it additionally fire-and-forgets a `POST` of `{ tier, op, message, detail, timestamp }` to a new dev-only server route, `POST /debug/log`.
- `POST /debug/log` (new, flat under `/server` per the existing package rule) does nothing but append the received JSON as one line to `server/.debug/client-log.jsonl`, no auth, no response body beyond 204. It exists only so the log is a file on disk instead of a browser console — not a queryable store, not a dashboard.
- Both ends are dev-only: the client only calls the sink when built in dev mode (Vite's `import.meta.env.DEV`), and the server route is registered only when the server is started with the existing dev flag/build (never reachable from a production build). A failed `POST` (route missing, network drop) is swallowed — the sink must never itself throw or show `ErrorPanel`.
- `context.detail` on `reportError()` and the payload passed to the existing Diagnostic/Operational `console.warn`/`console.error` sites must carry enough to reproduce: the offending event and, where one is in scope, the current `gameStateSnapshot`. This was already true for most sites migrated below; the migration table's "Current" column is where to check whether `detail` needs to be added, not just relocated.

## Call Site Migration

Existing sites this spec's implementation must re-route, per the audit:

| File:line | Current | New tier |
| --- | --- | --- |
| `MatchScene.ts:271,481,678,710,752` | `showError(err instanceof Error ? err.message : String(err))` | Domain/Session/Infra via `reportError` |
| `MatchScene.ts:401,415,515,546,782` | `showError('Invalid ... event received from server')` | Infra — generic player message, `console.error` gets the event payload |
| `MatchScene.ts:394` | `showError('CPU turn did not complete')` | Infra |
| `MatchScene.ts:179,197,583` | `showError('Failed to load match state/config')` | Infra (currently no `console.error` alongside — gap) |
| `MatchScene.ts:599` | `showError('Match config is still loading, please try again shortly')` | Domain-equivalent (actionable) — keep as-is, no `reportError` needed |
| `startMatch.ts:51` | `` errorPanel.show(`Failed to create match: ${result.message}`) `` | Domain/Infra via `reportError` (currently no logging at all — gap) |
| `MatchSettingsScene.ts:89-93` | `.catch(() => errorPanel.show('Failed to load match catalog'))` | Infra — **currently swallows `err` entirely, must log it** |
| `MatchScene.ts:301,818` | `console.error(...)` only | Diagnostic/Operational — already correct, no change |
| `boardRenderer.ts:75,118`, `resolveTurnPlayer.ts:125,235` | `console.warn(...)` | Diagnostic — already correct, no change |

## Acceptance Criteria

1. Given `submitTurnCommand` rejects with a 409, when the catch handler runs, then `console.error` logs the full `ApiError` and `ErrorPanel` shows the server's message verbatim.
2. Given a `bombPlaced` event with an unrecognized shape, when handled, then `console.error` logs the event payload and `ErrorPanel` shows a generic message — never the raw `'Invalid ... event received from server'` string.
3. Given `getCatalog()` rejects for any reason, when the catch handler runs, then `console.error` is always called — no catch block silently discards `err`.
4. Given any 500, network, or parse failure, when shown to the player, then `ErrorPanel` never displays the raw error text or stack.
5. Given a Diagnostic or Operational failure (missing texture, unhandled event type, best-effort `deleteMatch()`), when it occurs, then no `ErrorPanel` message is shown and the flow continues unaffected.
6. Given any tier logs via `console.error`/`console.warn` in a dev build, when it fires, then `server/.debug/client-log.jsonl` gains one matching line with `tier`, `op`, `message`, `detail`, and `timestamp` — readable from disk without touching the browser.
7. Given a production build, when any error/warning fires, then no request reaches `/debug/log` and the route itself is not registered server-side.
8. Given `/debug/log` is unreachable (route missing, network drop), when the sink fires, then no exception escapes it and `ErrorPanel`/the rest of the flow is unaffected.

## Log

Implementation issues found during the build (non-spec gaps) will be tracked in `p4-spec-100-error-log.md` once work starts.

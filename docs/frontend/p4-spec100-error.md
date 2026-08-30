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

- A remote log sink (Sentry-like telemetry) — speculative infra, not requested.
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

## Log

Implementation issues found during the build (non-spec gaps) will be tracked in `p4-spec-100-error-log.md` once work starts.

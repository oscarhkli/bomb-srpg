---
title: "Log: p3-spec001-match"
---

# Known Issues

Found via manual code walkthrough after implementation, not by unit tests. These are issues in the client-layer implementation and its test coverage, not gaps in `p3-spec001-match.md` —- logged here for traceability since that spec is what surfaced them.

1. **`api.ts`'s room/token guard was conflated.** The original `requireRoom()` threw unless *both* `roomId` and `token` were set, but the backend (`server/http_handlers.go`) only requires `Authorization` on action endpoints (`turn-commands`, `start-turn`, `reset`, `resolve`, `surrender`). Read/setup endpoints (`createMatch`, `getMatchState`, `getMatchConfig`, `getVictoryResult`, `getAllowedTiles`) need only a room id. Fixed by splitting into independent `requireRoomId()` / `requireToken()` checks — the latter only invoked via `authHeaders()`, used solely by the actually-authenticated calls.
   **Status: Solved.**
2. **`DevBootScene` never called `initRoom()`.** It captured `{ id: roomId }` from `createMatchRoom()` but passed straight into `createMatch()` without registering it via `initRoom(roomId)` first, so the (pre-fix) guard threw immediately on first real run. Fixed by calling `initRoom(roomId)` right after `createMatchRoom()` resolves.
   **Status: Solved.**
3. **Blind spot in `api.test.ts`:** its `beforeEach` unconditionally stubs both `initRoom()` and `initToken()`, so no existing unit test could have caught either issue above — they only surfaced via manual browser verification. Worth keeping in mind when adding tests for future client-layer code: assert against the *minimum* required setup for a given call, not the maximum.
   **Status: Deferred.** No test changes made yet to enforce minimum-required setup.
4. **No integration test exercises real frontend→backend calls.** `MatchScene.test.ts` and `api.test.ts` both mock `../engine/api`/`fetch` entirely, so nothing verifies that `types/api.ts`'s hand-written TS types actually match the Go server's real JSON responses — this class of gap is exactly what let issues 1–2 above slip through `make web-test`. Deferred to a dedicated future task (e.g. once turn submission lands and more of the API surface is in real use); `make run-server` + `make web-dev` manual verification is the interim substitute.
   **Status: Deferred.**
5. **No headless-browser tooling for automated visual verification.** PR #24's manual test-plan step (confirm `MatchScene` terrain colors + camera centering) had to be eyeballed by hand in a real browser — this environment has no `chromium-cli`/Playwright set up, so an agent can't screenshot-verify the canvas itself. If this recurs often, consider adding Playwright as a dev dependency in `web/` (`npm install -D playwright`) to let future verification be scripted and screenshotted instead of manual.
   **Status: Deferred.** No tooling installed yet.

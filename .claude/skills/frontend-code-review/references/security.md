# Client-side security review

This app has no server-rendered HTML — Phaser renders to `<canvas>`, and the frontend only talks to the Go backend via JSON over `web/src/engine/api.ts`. That rules out classic server-side XSS (there's no template engine injecting user input into HTML on the server). But client-side JS can still create real DOM-injection and data-handling risks. Calibrate severity to what's actually reachable in this app — don't invent OWASP-top-10 findings that need a code path this codebase doesn't have.

## Things to flag

- **Any `innerHTML`, `outerHTML`, `insertAdjacentHTML`, or `document.write`.** Phaser apps rarely need raw DOM manipulation (text goes through Phaser `Text` GameObjects, which render to canvas and are not an injection vector). If a diff touches the DOM directly — e.g. to render an overlay, error message, or debug panel — and interpolates any value that ultimately originates from user input or the backend API response into HTML via these APIs, that's a real XSS path. Prefer `textContent` or Phaser's own text rendering; flag anything that doesn't.
- **`eval`, `new Function(...)`, or `setTimeout`/`setInterval` given a string instead of a function.** These execute arbitrary strings as code. There's essentially no legitimate reason for this in a Phaser/TS app; treat any occurrence as high severity and ask why it's there.
- **Unsafe URL construction for `fetch`/navigation.** `web/src/engine/api.ts` builds request URLs — check whether any path or query segment is built by string-concatenating a value that came from user input or server data without validation, which could let a crafted value redirect a request somewhere unintended (open redirect / SSRF-adjacent, though limited by same-origin/CORS in a browser). Also check that API base URLs aren't accidentally hardcoded to something other than the intended backend in a way a diff introduces.
- **Secrets or tokens in client code.** Anything that looks like an API key, auth token, or credential hardcoded into TS source, `.env` files bundled by Vite (only `VITE_`-prefixed vars ship to the client — check nothing sensitive is prefixed that way), or written to `localStorage`/`sessionStorage`/cookies without thinking about XSS exposure (anything in those stores is readable by any script that runs on the page). This app currently has no auth, so this mostly matters as a forward-looking check if a diff starts adding one.
- **`postMessage` without an origin check.** If any diff adds cross-window/iframe messaging, sending or receiving without validating `event.origin` lets any embedding page inject or read messages.
- **Logging sensitive data to the console or to any telemetry call.** Low severity here (no auth yet) but worth a mention if a diff logs full API responses or request bodies that could later contain something sensitive.
- **Dependency risk.** If a diff adds a new npm dependency, a quick sanity check (is it maintained, does it need DOM/`eval`-like capabilities it shouldn't) is worth a low-severity note — not a full supply-chain audit, just don't wave through an odd new dependency silently.

## What NOT to flag

- Don't invent server-side injection findings (SQLi, template injection) — this skill only reviews `web/`, and there's no server-rendered HTML or SQL here.
- Don't flag Phaser `Text` GameObject content as an XSS risk — it renders to canvas, not the DOM, and isn't interpreted as HTML/JS.
- Don't flag CORS as a frontend problem — CORS policy is set server-side (Go backend); a frontend `fetch` call itself isn't a CORS vulnerability.

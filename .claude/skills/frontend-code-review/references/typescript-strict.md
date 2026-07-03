# TypeScript strict-mode review

This repo's `tsconfig.json` already has `strict`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitReturns`, and `noFallthroughCasesInSwitch` on, and ESLint runs `flat/recommended-type-checked` + `flat/stylistic-type-checked`. That means `tsc`/`eslint` already catch most naive type errors — **don't re-report anything the compiler or linter would already flag** (you can't run them here, but if a pattern is a plain compile error under this config, it's not a useful finding). Focus on the things that compile cleanly but defeat the point of strict mode:

## Things to flag

- **`any` (explicit or via untyped third-party return)** — the project's own instructions (root `CLAUDE.md`) say `any` is disallowed. Any explicit `any`, or a value silently typed `any` because it came from an untyped call, is a finding.
- **`as` casts, especially `as unknown as X`** — a cast is the developer overriding the type checker. Ask: does the runtime value actually match the asserted type? A cast from an API response (`web/src/types/api.ts` mirrors Go JSON) that isn't validated is a real risk if the backend contract changes silently.
- **Non-null assertions (`!`)** — e.g. `element!`, `array[i]!`. With `noUncheckedIndexedAccess` on, indexing already returns `T | undefined`; a `!` right after silently reintroduces the crash the config was turned on to prevent. Check whether the un-null-ness is actually guaranteed nearby (a prior `if` guard) or just assumed.
- **`@ts-ignore` / `@ts-expect-error`** — always worth a line, since it's explicitly suppressing the type system. `@ts-expect-error` is less bad (it fails if the error disappears) but still flag what it's hiding.
- **Optional-property handling under `exactOptionalPropertyTypes`** — this flag means `{ x?: string }` is *not* the same as `{ x: string | undefined }`. Code that does `obj.x = undefined` to "clear" an optional field will fail to compile — but code that works around this by loosening a type (adding `| undefined` broadly, or widening to `any`) defeats the purpose. Flag workarounds, not the compiler errors themselves.
- **Structural typing surprises** — TS types are structural, not nominal. Two differently-named types with the same shape are interchangeable, which can let a value of the wrong *semantic* type (e.g. a `BombID` shape used where a `UnitID` is expected) pass silently if `engine/codecs.go`'s ID encoding isn't mirrored with distinct branded types on the TS side. Check `web/src/types/api.ts` usage sites for this.
- **Floating promises** — ESLint has `no-floating-promises` as `warn` (not `error`), so it won't block a build. An un-awaited async call in a Phaser scene method (e.g. an API call inside `create()` or an input handler) that isn't awaited or `.catch()`-handled can cause silent failures or race conditions with scene teardown. Worth a finding when the promise's rejection would matter (e.g. a network call to the Go backend).
- **Type import hygiene** — `consistent-type-imports` is already enforced by ESLint; don't re-flag it, but do flag if a *value* is imported through a `type`-only import path in a way that would break at runtime (rare, but possible with re-exports).

## Phaser-specific typing note

Phaser 4's own types are large and sometimes force `any` at integration boundaries (e.g. certain plugin or texture APIs). Distinguish between "the project code chose `any`" (flag it) vs. "Phaser's own d.ts forces a cast at a well-contained boundary" (lower severity, note it but don't treat it as equally bad).

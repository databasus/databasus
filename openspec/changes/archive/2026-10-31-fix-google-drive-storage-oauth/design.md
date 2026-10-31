## Context

See `proposal.md` for motivation and scope. Current evidence:

- `frontend/src/features/storages/ui/edit/storages/EditGoogleDriveStorageComponent.tsx:20` serializes client settings into `state` and opens a new window.
- `frontend/src/pages/storage-oauth/ui/StorageOAuthPage.tsx:15` exchanges the code in the browser; line 91 trusts a JSON draft from state and line 107 can leave a failed callback loading indefinitely.
- `backend/internal/modules/storages/internal/httpapi/google_drive_settings.go:15` omits the client secret and token from responses. Reconnection therefore cannot reuse a secret by reading it into the browser.
- `frontend/src/features/storages/lib/isStorageSettingsFilled.ts:11` already distinguishes saved storages from new drafts, and the existing storage API supports retaining omitted secrets.
- `frontend/browser/HttpFixture.ts:43` installs a boundary on one page and records JSON bodies. Popup navigation and the form-encoded Google token request need a narrow fixture extension or explicit scenario-owned routing.

Root `AGENTS.md`, `frontend/AGENTS.md`, and the frontend ownership rules in `docs/architecture.md#frontend` constrain placement, public imports, local lifecycle state and feedback. The design and localization documents govern the changed controls and six dictionaries.

## Goals / Non-Goals

**Goals:** remove credentials from navigation, bind callbacks to one live editor, keep authorization in the form that requested it and use the existing test/save API.

**Non-Goals:** see the proposal. In particular, the browser continues to hold user-entered secrets and receive Google tokens in memory. This is not the server-owned confidential-client design recommended for a long-lived integration.

## Decisions

### 1. One in-memory attempt owned by the editor

Use **storage authorization attempt**, with canonical code name `StorageAuthorizationAttempt`, for one single-use, time-limited in-memory record binding a Google Drive response to the editor, account session, workspace, client settings and redirect address that initiated storage authorization. Add this short definition to `docs/glossary.md` in the implementation change. It is distinct from the existing `OAuthSignInAttempt`, which authorizes account sign-in rather than storage access; it introduces no API resource.

Keep the attempt and submitted client settings in a focused Google Drive authorization unit within the current storage feature. The editor supplies workspace, storage/new-editor identity, the originating `SessionSnapshot` and client-settings revision. A new editor instance has a distinct lifetime even when both forms are creating an unsaved storage. The attempt contains 32 cryptographically random bytes encoded as 64 lowercase hex characters, creation time, the fixed redirect URI and the captured context. Use `crypto.getRandomValues`, which does not add a secure-context-only hashing dependency to existing HTTP installations.

Only one attempt is active per editor. Start a ten-minute expiry timer and expose explicit cancel/restart. Client ID/secret changes, session replacement, workspace change, unmount or a new attempt cancel the old attempt and abort or ignore its exchange. Name and presentation changes do not invalidate it. Consume the attempt synchronously before awaiting exchange; retain a separate request revision so obsolete completions cannot publish tokens or feedback. StrictMode cleanup releases listeners and timers without making setup permanently unusable.

Rejected alternatives: browser storage would persist credentials or require reconstructing a secret-bearing draft; a module-global attempt would couple independent editors; server attempts add endpoints, persistence and cleanup for a provider planned for removal.

### 2. A single same-origin BroadcastChannel relay

Open an attempt-specific channel, named with a fixed interaction prefix and its random state, before synchronously opening Google from the user's click. Use only `BroadcastChannel`; do not add an opener or storage-event fallback. Its browser-defined same-origin/storage-key restriction avoids depending on `window.opener` surviving provider navigation. Treat the channel name as correlation, not permission: still validate message shape, origin, exact state, TTL and the current session/editor/settings context.

The callback accepts only the random-state format and exactly one nonempty code or provider error. Capture those values once in page-owned state that survives effect setup replay, then clear the current query using `history.replaceState` before relay or callback-owned requests. Old JSON-state URLs get restart guidance and are never decoded as drafts. Forward only a discriminated code/refusal message, never credentials or user-provided error descriptions. The editor sends a matching acknowledgment when it accepts the message; the callback can close after acknowledgment or show completion if closing is disallowed. If no acknowledgment arrives within five seconds, show restart guidance. Release the callback channel/timer on completion or unmount. Repeat setup can relay again, but the initiating editor's consumed attempt prevents a second exchange.

If channel creation or popup opening fails, release the attempt immediately and show a translated error. If the popup is abandoned, the user can cancel; expiry bounds waiting without a popup-closure polling loop. Loss of an opener reference alone is not treated as cancellation.

Rejected alternatives: `postMessage` requires a surviving opener and window identity; a public shared channel admits unrelated messages and complicates correlation; adding two transports expands the repair and its tests.

### 3. Preserve direct exchange and existing persistence

The initiating editor sends one form-encoded request to the fixed `https://oauth2.googleapis.com/token` endpoint with its captured client ID, client secret, code and exact redirect URI. It never uses the authenticated Databasus transport for Google, so no application session token reaches Google. Keep the existing Drive scope, offline access and consent request. Translate failures locally; do not log response bodies or show Google's raw error payload. Accept a token response only when its required access/refresh fields are usable and its attempt is still current.

Apply the token to the current Google settings only if the captured client-settings revision is still current; retain newer name input. Show authorization status instead of the raw token. The current editor then performs the existing explicit connection test and create/update. Authorization never directly invokes save. A saved storage can be renamed/tested without receiving or re-entering its existing secrets. Explicit reconnection asks for the client secret again; cancellation or failure retains its existing saved credentials and any previous usable in-memory draft token.

Rejected alternatives: a backend proxy or attempt API is outside the requested minimal repair; returning tokens through the popup broadens credential handling; automatically saving from the callback bypasses the caller's selection and draft lifecycle.

### 4. Keep the patch local before the domain migration

Keep authorization orchestration in `features/storages` with a focused public relay entry point for `pages/storage-oauth`; keep rendering and route concerns on the page. Remove the callback's `EditStorageComponent` dependency and delete `StorageOauthDto`. The later resource-screen change moves the whole interaction to `features/edit-storage` without reimplementing its protocol. Do not change sign-in OAuth or introduce a universal OAuth feature.

## Risks / Trade-offs

- Browser-held secrets remain visible to code running in that origin and the browser user. Memory-only storage and narrow payloads repair URL/persistence exposure; they do not provide server-side confidentiality.
- Google token endpoint CORS and Google client configuration remain external prerequisites of the existing direct exchange. Synthetic tests prove local orchestration, not live provider acceptance. A provider/CORS refusal is a bounded translated failure; it does not trigger an unplanned backend fallback.
- Reloading the initiating tab loses authorization context. Require a fresh attempt; never persist secrets to make recovery seamless.
- A severed opener must not break completion. Browser tests remove the opener relationship and still exercise channel delivery.
- The browser fixture currently misses popup first requests. Install context-level finite routes before opening the popup, preserve unexpected-request checks and parse form-encoded request bodies explicitly. No catch-all external allowance or real Google account is permitted.

## Migration Plan

Deploy frontend changes together; no stored-data or API migration is required. Keep the existing registered redirect address. Old in-flight callbacks are rejected. Update the architecture's storage callback description, add the distinct storage authorization attempt to the glossary and update the working milestone notes that still assume server-owned attempts. Rolling back the frontend does not change stored credentials, but reintroduces the URL exposure, so prefer a forward fix.

## References

- [Google authorization flow](https://developers.google.com/identity/protocols/oauth2/web-server): code exchange, offline authorization and state validation. The retained browser exchange is a deliberate deployment trade-off, not a claim of server-flow equivalence.
- [HTML same-origin broadcasting](https://html.spec.whatwg.org/multipage/web-messaging.html#broadcasting-to-other-browsing-contexts): channel scope and lifecycle.

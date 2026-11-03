## Context

The findings were re-checked against the current code; the review's line numbers are partly out of date.

- **Sign-in message.** `frontend/src/pages/auth-callback/ui/AuthCallbackPage.tsx:53-57` translates an `ApiError` and shows `app.oauthCallback.invalidProvider` for anything else. `frontend/src/features/oauth-sign-in/model/OAuthCallback.ts:38-40` and `frontend/src/features/oauth-sign-in/model/OAuthSignInAttempt.ts:32-46` throw plain errors for a provider refusal, a missing code and a missing or invalid attempt. Network and server failures are already an `ApiError` (`frontend/src/shared/api/apiClient.ts:47`, `frontend/src/shared/api/requireData.ts:31-34`). The keys `codeNotFound` and `stateMissing` have no callers. A failure to create the attempt at sign-in start is a separate path with its own message (`frontend/src/features/oauth-sign-in/ui/OAuthSignInButtons.tsx:25-27,52`) and does not change.
- **Selected tab.** `frontend/src/widgets/app-shell/ui/SidebarComponent.tsx:95` uses `bg-hover`, lines 101 and 153 always render `tab.icon`, and `selectedIcon` from `frontend/src/app/ui/ApplicationScreen.tsx:84-144` is never read. v3.64.0 used `bg-blue-600` with the white icon. That class is `#155dfc`, the seed `colorPrimary` in `frontend/src/shared/theme/getAntdTheme.ts:9`. The light theme resolves `colorPrimary` to that value, and the dark algorithm resolves it to `#1552d9`, so the dark selected background is slightly darker than in v3.64.0.
- **New-version link.** `frontend/src/app/ui/ApplicationScreen.tsx:263-273` renders `↗` with an `aria-label` and no tooltip, inside a version block with `hidden md:block`, so narrow viewports do not show it. v3.64.0 wrapped the link with a green dot in a `Tooltip` (`frontend/src/widgets/main/MainScreenComponent.tsx:374-383` at that tag).
- **Database search.** `frontend/src/pages/databases/ui/DatabasesComponent.tsx:267-272` is a plain `<input>` with only a placeholder. `frontend/src/pages/notifiers/ui/NotifiersComponent.tsx:150-155` is the labelled pattern.
- **Language switch.** `frontend/src/shared/i18n/LocaleProvider.tsx` changes the language without remounting the tree. The token effect in `frontend/src/pages/databases/ui/physical-backups/usePhysicalRestoreToken.ts:25-44` depends on the target subject, and the load effect in `frontend/src/pages/databases/ui/verification/EditVerificationConfigComponent.tsx:101-127` depends on the database, the load attempt and the session guard.
- **Restore tokens.** `backend/internal/modules/backups/internal/physical/restorestream/service.go:52-70` issues an independent token without revoking earlier ones, and refuses the request while the user's restore stream is held (lines 110-118).
- **`window.alert`.** Production code has no call, and every check found is `.not.toHaveBeenCalled()`.

Contrast values below come from AntD design tokens resolved with this project's seed colors through `theme.getDesignToken`. No browser was opened to obtain them.

| Pair | Light | Dark |
|---|---|---|
| `bg-hover` against the menu background | 1.09:1 | 1.23:1 |
| White on `colorPrimary` | 5.25:1 | 6.48:1 |
| White on `colorPrimaryHover` | 3.63:1 | 4.26:1 |
| White on `colorLink`, the `bg-accent` utility | 5.25:1 | 2.32:1 |
| `colorPrimary` against the desktop sidebar, `colorBgContainer` | 5.25:1 | 2.84:1 |
| `colorPrimary` against the mobile drawer, `colorBgElevated` | 5.25:1 | 2.54:1 |

The approach follows the root `AGENTS.md` rules on naming, comments and translated copy, and the `frontend/AGENTS.md` rules on AntD primitives, `@ant-design/icons`, component structure, design and localization. It does not change the architecture in `docs/architecture.md`: the widget keeps its place, and no new theme role appears.

## Goals / Non-Goals

**Goals:**

- Tell the user what to do after an unfinished sign-in.
- Make the selected section visible in both themes and both layouts, and announce it to assistive technology.
- Restore the explanation of the new-version link.
- Give the database search field an accessible name.
- Pin language-switch behavior for the three surfaces the earlier work left untested.

**Non-Goals:**

- Distinguishing sign-in failure causes in the message, or changing the sign-in start failure message.
- Raising the contrast of unselected tab icons, which were the same asset in v3.64.0.
- Showing the new-version link on narrow viewports.
- Removing `window.alert` checks from files this change does not otherwise touch.

## Decisions

### One message for every failure the callback finds before the exchange

`AuthCallbackPage` shows a new key, `app.oauthCallback.notCompleted`, for anything that is not an `ApiError`. The keys `invalidProvider`, `codeNotFound` and `stateMissing` are deleted from all six dictionaries, and `OAuthCallback.ts` does not change.

Rejected: restoring `codeNotFound` and `stateMissing`. They describe protocol details, give the user no action, and leave a refusal and a lost attempt needing further messages. Rejected: passing a failure kind out of `OAuthCallback` for per-cause messages, because every cause leads to the same action.

### AntD `Button` for sidebar entries instead of new theme roles

Both `SidebarComponent.tsx` layouts render each entry as AntD `Button`, `type="primary"` when selected and `type="text"` otherwise, with Tailwind for size and alignment, and render `selectedIcon` on the selected entry. The primary button already resolves `colorPrimary` and `colorTextLightSolid` from the theme, so no new role, `index.css` utility or documentation entry is needed.

Rejected: new `--app-primary` and `--app-on-primary` roles on native buttons, which reproduce two tokens the primary button already resolves and add a role with one consumer. Rejected: `bg-accent`, which leaves the white icon at 2.32:1 in the dark theme. Rejected: reading `theme.useToken()` inside the sidebar, which moves theme values into an inline style. Rejected: deleting `selectedIcon` and keeping `bg-hover`, which keeps 1.09:1 and fixes only assistive technology.

### `aria-current="page"` and empty `alt` on tab icons

Both sidebar layouts set `aria-current="page"` on the selected button only, passed through `Button`. Tab icons get `alt=""`, because `aria-label` already names each button and `docs/frontend-design.md` asks for an empty alternative on decorative images.

Rejected: `role="tab"` with `aria-selected`, because these buttons open whole sections and have no tab panel relationship. Rejected: dropping `aria-label` in favor of the image text, because the mobile entry already has a visible label and both layouts should name the button in the same way.

### Tooltip around the link, dot inside it

AntD `Tooltip` wraps the `<a>`, with `trigger={['hover', 'focus']}` and the existing `app.navigation.newVersionAvailable` copy. The link keeps its address and `aria-label`, and contains a small `bg-success` dot. The link must be the Tooltip's direct child because keyboard focus lands on the link, and the version block keeps `hidden md:block`.

Rejected: a Tooltip inside the link around the dot, which never receives keyboard focus. Rejected: keeping `↗` and only adding a tooltip, which leaves the unexplained character visible until hover. Rejected: the v3.64.0 hover-only tooltip, because a keyboard user would not get the explanation visually. Rejected: adding the link to the mobile menu, which is a new feature outside this repair.

### Automated browser check for the selected tab

A scenario in `frontend/browser/scenarios/appShell.spec.ts` runs in both themes at 769 and 320 CSS pixels. With the pointer moved away from the entry, it asserts `aria-current`, the white icon, at least 4.5:1 between white and the computed background, and at least 4.5:1 between the computed label color and the background on the mobile entry. It also activates another section and asserts that these move.

Rejected: a manual browser check, because `openspec/config.yaml` keeps manual browser clicking out of tasks. Rejected: a happy-dom assertion on class names, which cannot prove resolved colors. Screen tests still pin `aria-current`, because they are the faster layer for that behavior.

### Language tests are regression guards

The three language tests live beside their surfaces: `PhysicalRestoreComponent`, `EditVerificationConfigComponent` and `EditPhysicalBackupConfigSection`. Each uses the real locale provider and an in-test button that calls `setLocale`, as `DatabaseComponent.test.tsx:31-35` does. The restore test also counts token requests. A repeated request would replace a command the user may have copied, leave an extra live token until it expires, and be refused during the user's restore stream, which would replace the commands with an error.

From the code above these tests are expected to pass on the current code, so they do not start red. The sign-in, sidebar, new-version link and search tests reproduce defects and must fail first. If a language test fails, it has found a real defect, and the cause is fixed in this change.

Rejected: covering the three surfaces through one `DatabaseComponent` scenario, which would not reach the restore screen or the verification editor directly.

### Remove `window.alert` checks only in touched tests

Checks go from `DatabasesComponent.test.tsx` and `EditVerificationConfigComponent.test.tsx`, the two existing test files with such checks that this change edits, as the task requested. A test left without a meaningful assertion gets one for its real outcome.

Rejected: a sweep across all 19 files, which adds a large mechanical diff unrelated to the defects. It can be done separately. Rejected: keeping all checks, which the task did not ask for; the trade-off is recorded under risks so the user can reverse it.

### Browser checks wait for the state they measure

Three browser scenarios failed intermittently under machine load, each because a check ran during an animation:

- `frontend/browser/enlargeText.ts` returned while `.ant-select-selection-item` was still animating `font-size`, so `assertAccountReflow` usually measured the 14px text and passed. The settled state overflowed. `enlargeText` now waits for CSS transitions in the region. It waits for transitions only, because a spinner's infinite animation never finishes.
- `frontend/browser/openWorkspaceScreen.ts` did not wait for the closing drawer, which returns focus to the menu button when its animation ends and took focus from a control focused earlier. The helper now waits for that focus on the narrow layout.
- The accessibility scan in `frontend/browser/scenarios/smtp.spec.ts` read a validation message while it faded in. The scenario now waits for animations before scanning.

Rejected: retrying the checks, which would keep hiding the overflow. Rejected: asserting at the first passing moment with a poll, for the same reason.

### The restore deletion race test holds its restore open

`TestDeleteDatabaseRacingRestoreStart` treated an accepted restore next to a successful deletion as a failure. `openspec/specs/databases/spec.md` refuses deletion only while a restore is in progress, and the failed run's log showed the small restore completing before the deletion ran, so the production code was right. The test now restores into the blocked target from `newBlockedTargetSpec`, which keeps an accepted restore running through the deletion and makes the original assertion hold in every ordering.

Rejected: also accepting 204 with 202, because such a test could no longer catch a deletion beside a running restore.

### Selects that wrap their value

The settled measurement exposed two real defects at 320 CSS pixels with enlarged text: the storage type `Select` at `frontend/src/features/edit-storage/ui/edit/EditStorageComponent.tsx` kept 250px beside its logo and overflowed its row, and the workspace `Select` at `frontend/src/widgets/app-shell/ui/WorkspaceSelectionComponent.tsx` clipped its value. Both use `WRAPPING_SELECT_CLASS_NAME` from `frontend/src/shared/ui`, and the storage type row may now shrink with its logo kept whole. This follows the `frontend-application` requirement that translated content wraps without clipping, so no delta is needed.

Rejected: accepting an ellipsis in the check, because it hides the selected storage type from the user. Rejected: a wrapper component, because only presentation differs and the AntD `Select` API stays in use.

## Risks / Trade-offs

- **The dark selected background is low against its surroundings:** 2.84:1 on the desktop sidebar and 2.54:1 in the mobile drawer, below the 3:1 commonly asked of interface shapes. Both values are accepted. The icon and label on it reach 6.48:1, and the current-page state is exposed separately, so the selected state stays identifiable. A lighter blue would lower the contrast of the white icon and label.
- **Hovering the selected entry lowers contrast** to 3.63:1 in the light theme and 4.26:1 in the dark theme, because `Button` uses `colorPrimaryHover`. This is the state of every primary button in the application and lasts only while the pointer is over the entry, so it is accepted. The requirement states its minimum for the state without the pointer.
- **Unselected icons stay pale in the light theme** at about 1.56:1. This predates the change and is listed out of scope. The stronger selected state does not make it worse.
- **Removed `window.alert` checks were a small guard.** The main spec requires no browser-native dialog on database screens, and these two files stop checking it directly. The other 17 files keep their checks, and production code has no call today. The user can drop this step without affecting anything else in the change.
- **Visual references.** The stored references in `frontend/browser/references` capture the SMTP editor element only, so the sidebar change should not affect them. The browser run in the change verification confirms this.
- **A generic message hides the cause from support.** The current text does not name the real cause either, because it reports an invalid provider for every such failure. The callback already removes the provider's `error` parameter from the address, and this change does not show it. The trade-off is accepted.

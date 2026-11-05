# Frontend Development

## Setup

Install dependencies:

```bash
pnpm install
```

## Development

To run the development server:

```bash
pnpm dev
```

## Build

To build the project for production:

```bash
pnpm build
```

This will compile TypeScript and create an optimized production build.

## Code Quality

### Linting

To run every frontend check without changing files:

```bash
pnpm lint
```

### Formatting

To format code using oxfmt:

```bash
pnpm format
```

This will automatically format all TypeScript, JavaScript, JSON, CSS, and Markdown files.

The lint command checks formatting without changing files. To watch tests, use `pnpm exec vitest`; to preview a build, use `pnpm exec vite preview`. The [testing guide](../docs/testing.md#commands) lists browser checks and the command that updates visual references.

## Workspace resource screens

Workspace settings, storage and notifier lists/details live in their `src/pages` slices. Reusable editors live in `features/edit-storage` and `features/edit-notifier`. `pages/databases` owns the database list, details, settings and creation wizard, the logical and physical backup lists with their config editors, the restore screens, WAL stream monitoring and the verification config and verification screens. It composes those editors for database transfer, notifier selection and backup-storage configuration. `pages/instance-settings` owns the mail server, the audit log and the verification agent screens. `features/database-connection` holds the connection form, which the page also uses for the restore target. Workspace creation belongs to app-local UI.

The [architecture](../docs/architecture.md#frontend) describes ownership and asynchronous acceptance. The [testing guide](../docs/testing.md) describes production browser UI checks. The real application is tested from outside by the [browser end-to-end suite](../docs/testing.md#browser-end-to-end-suite) in `e2e/browser`.

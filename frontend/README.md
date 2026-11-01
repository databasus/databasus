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

To check for linting and type errors:

```bash
pnpm lint
pnpm typecheck
```

### Formatting

To format code using oxfmt:

```bash
pnpm format
```

This will automatically format all TypeScript, JavaScript, JSON, CSS, and Markdown files.

To check formatting without changing files:

```bash
pnpm format:check
```

## Workspace resource screens

Workspace settings, storage and notifier lists/details live in their `src/pages` slices. Reusable editors live in `features/edit-storage` and `features/edit-notifier`. `pages/databases` owns the database list, details, settings and creation wizard, and composes those editors for database transfer, notifier selection and backup-storage configuration. `features/databases` keeps the connection forms that restore also uses. Workspace creation belongs to app-local UI.

The [architecture](../docs/architecture.md#frontend) describes ownership and asynchronous acceptance. The [testing guide](../docs/testing.md) describes production browser UI checks and the isolated real-API workspace/storage scenario.

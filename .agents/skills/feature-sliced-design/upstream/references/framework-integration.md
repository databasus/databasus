# Framework Integration

How to set up FSD within specific frameworks. Covers directory placement,
routing integration, and framework-specific path alias configuration.

## General principle

Place FSD layers inside `src/` to avoid naming conflicts with framework
directories. The FSD `app/` and `pages/` layers are **not** the same as
framework directories with the same names (e.g., Next.js `app/`).

Examples here import through `@/` pointing at `src/`. Whether that is one
root alias or one entry per layer is a tooling choice, not an FSD rule,
and it changes no layer semantics; how the resolver is wired differs by
framework, which is what each section below shows. Configure an entry
only for a layer the project actually has.

The FSD layers inside `src/` keep the standard shape described in
`references/layer-structure.md`. Each section below shows only what the
framework adds or renames around them, not the layers' own internals.

The Next.js, Nuxt, and Astro sections follow the official tech guides on
fsd.how. React Router and Vite are this skill's own additions with no
official guide behind them. SvelteKit and Electron have official guides
and are not covered here; read those directly.

## Next.js

FSD works with both the App Router and the Pages Router. Next.js uses the
`app/` and `pages/` folder names for its own routing. Those names collide
with the FSD `app/` and `pages/` layers. Rename the FSD layers to `_app/`
and `_pages/` (with the underscore prefix). Do this even if you only use
one router. Keep the Next.js routing folders at the project root so `src/`
holds only FSD code. The FSD linter (Steiger) expects this naming.

### Projects on the previously recommended pattern

An earlier version of this guide recommended a different layout. It kept the
Next.js `app`/`pages` folders at the root and added an empty root `pages/`
placeholder. The `src/app`/`src/pages` layers were not prefixed. Projects set
up that way keep working. The empty `pages/` placeholder can break the build on
Next.js 13.5 and later. That is why the prefix is now the default. Use
`_app`/`_pages` for new projects. Move a project off the old pattern when you
can.

### App Router

Route files in `app/` re-export from the FSD `_pages/` layer.

#### Directory structure

```text
my-nextjs-project/
  app/                     ← Next.js App Router (routing only)
    layout.tsx
    page.tsx
    profile/
      page.tsx
    api/
      get-example/
        route.ts
  src/
    _app/                  ← FSD app layer
      providers/
        index.tsx          ← All providers (QueryClient, theme, etc.)
      styles/
        globals.css
      api-routes/          ← Route Handler implementations (see below)
        index.ts
        get-example-data.ts
    _pages/                ← FSD pages layer
      home/                ← slice; segments per layer-structure.md
      profile/
    widgets/               ← FSD widgets layer (when needed)
    features/              ← FSD features layer (when needed)
    entities/              ← FSD entities layer (when needed)
    shared/                ← FSD shared layer
      db/                  ← Database queries (see below)
```

#### Wiring Next.js routes to FSD pages

```typescript
// app/layout.tsx
import { Providers } from '@/_app/providers';
import '@/_app/styles/globals.css';

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body><Providers>{children}</Providers></body>
    </html>
  );
}

// app/example/page.tsx: re-export the FSD page (component + metadata)
export { ExamplePage as default, metadata } from '@/_pages/example';
```

Keep route files free of logic. Re-export the page component plus
whatever route exports the framework needs and the FSD page provides,
such as `metadata` or `generateMetadata` when the page has one.

### Pages Router

The Pages Router uses `pages/` at the project root. Each route file should
re-export the corresponding page module from the FSD `_pages/` layer.

```text
my-nextjs-project/
  pages/                   ← Next.js Pages Router (routing only)
    _app.tsx
    api/example.ts         ← API route re-export
    example/index.tsx
  src/
    _app/
      custom-app/          ← Custom App component
      api-routes/          ← Route Handler implementations
    _pages/
      example/
        ui/example.tsx
        index.ts
```

```typescript
// pages/example/index.tsx
export { Example as default } from '@/_pages/example';

// pages/_app.tsx: re-export the custom App from src/_app/custom-app
export { App as default } from '@/_app/custom-app';
```

The custom App implementation lives in `src/_app/custom-app/` and exposes
only what the framework entry file imports. App has no slices, so treat
this as a segment boundary inside the layer, not a slice.

### Proxy and instrumentation

`proxy.ts` and `instrumentation.ts` sit at the framework boundary, not
inside the FSD layers. Next.js looks for them in the project root, or in
`src/` when the Next.js app itself uses `src/`, at the same level as its
`app/` or `pages/` folder. Keep them there and out of `src/_app/`.

Next.js 16 deprecated the `middleware` convention and renamed it `proxy`.
Check the `next` version in `package.json` before modifying `middleware.ts`
or `proxy.ts`. Use `proxy.ts` for Next.js 16+ and `middleware.ts` for 15 or
earlier. Earlier versions do not load `proxy.ts`, so renaming `middleware.ts`
to `proxy.ts` disables middleware without an error, including auth checks.
Only rename an existing `middleware.ts` as part of a Next.js 16 upgrade.

With the layout above, the Next.js `app/` and `pages/` folders sit at the
project root, so both files sit there too. The `src/` option applies to
projects that keep Next.js routing under `src/`, and never means `src/_app/`.

> **Where this comes from.** The official Next.js guide on fsd.how says
> both files must be in the project root and that Next.js will not find
> them under `src/`. That matched earlier Next.js versions and is no
> longer what the framework documents, so this section follows the
> framework.

### Route Handlers (API routes)

Use a dedicated `api-routes` segment in the FSD `_app/` layer
(`src/_app/api-routes/`) to host the actual request handlers. The Next.js
`app/api/*/route.ts` (App Router) or `pages/api/*.ts` (Pages Router) files
become thin re-exports.

**App Router:**

```typescript
// src/_app/api-routes/get-example-data.ts
import { getExamplesList } from '@/shared/db';

export const getExampleData = () => {
  try {
    const examplesList = getExamplesList();
    return Response.json({ examplesList });
  } catch {
    return Response.json(null, {
      status: 500,
      statusText: 'Ouch, something went wrong',
    });
  }
};

// src/_app/api-routes/index.ts
export { getExampleData } from './get-example-data';

// app/api/example/route.ts
export { getExampleData as GET } from '@/_app/api-routes';
```

**Pages Router:**

```typescript
// src/_app/api-routes/get-example-data.ts
import type { NextApiRequest, NextApiResponse } from 'next';

const config = { api: { bodyParser: { sizeLimit: '1mb' } }, maxDuration: 5 };
const handler = (req: NextApiRequest, res: NextApiResponse) =>
  res.status(200).json({ message: 'Hello from FSD' });

export const getExampleData = { config, handler } as const;

// pages/api/example.ts
import { getExampleData } from '@/_app/api-routes';
export const config = getExampleData.config;
export default getExampleData.handler;
```

Keep Route Handlers as framework-facing adapters and delegate domain rules
to the FSD boundary that owns them. FSD is primarily a frontend
methodology. If `api-routes` grows to many endpoints, consider moving the
backend to a separate package in a monorepo.

### Database access

Place database queries in a `db` segment in `shared/` (`src/shared/db/`).
Co-locate caching and revalidation logic with the queries themselves.

Plain data access is all that goes there. Rule 4-5 does not relax because
code runs on the server: domain rules and use-case orchestration stay with
the slice that owns them.

### Path aliases

```json
// tsconfig.json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/_app/*": ["src/_app/*"],
      "@/_pages/*": ["src/_pages/*"],
      "@/shared/*": ["src/shared/*"]
    }
  }
}
```

Next.js reads `tsconfig.json` paths automatically. No `next.config.js`
alias configuration is needed.

### Server and client public APIs

In the Next.js App Router, a single slice can contain both client-usable modules
and server-only modules.

Keep `index.ts` free of server-only exports, such as Server Components or
data-access functions that import `server-only`. When a Client Component imports
the slice, those exports can enter the client module graph and cause build
errors.

Split only when this boundary is required. Put server-only exports in
`index.server.ts`.

## Nuxt (v3-compatible layout)

Nuxt keeps file routing in `pages/` at the project root, the name FSD
reserves for the pages layer. The official Nuxt guide resolves this by
moving Nuxt's routing folder inside the FSD `app` layer and giving `src/`
a single `@` alias. This section mirrors that guide.

The shape below is a Nuxt 3 project, which is what the official guide
covers. Nuxt 4 changed the default `srcDir` to `app/` and moves `pages/`,
`layouts/`, and `components/` under it, so its framework `app/` collides
with the FSD app layer the way Next.js does. No official FSD guide covers
that layout yet. A Nuxt 4 project can keep this shape, which Nuxt still
auto-detects, and follow this section. Nuxt 3 itself reached end of life
on 31 July 2026, so read the heading as the layout, not the version to
start on.

### Directory structure

```text
my-nuxt-project/
  nuxt.config.ts
  src/
    app/                   ← FSD app layer
      routes/              ← Nuxt file routing (dir.pages)
        index.vue
      layouts/             ← Nuxt layouts (dir.layouts)
    pages/                 ← FSD pages layer
      home/
        ui/home-page.vue
        index.ts
    shared/
```

### nuxt.config.ts

```typescript
// nuxt.config.ts
export default defineNuxtConfig({
  alias: {
    "@": "../src",
  },
  dir: {
    pages: "./src/app/routes",
    layouts: "./src/app/layouts",
  },
});
```

### Wiring Nuxt routes to FSD pages

```vue
<!-- src/app/routes/index.vue -->
<script setup>
import { HomePage } from "@/pages/home";
</script>

<template>
  <HomePage />
</template>
```

The official guide also shows config-based routing through
`app/router.options.ts` instead of file routing. Either way, route
definitions live in the `app` layer and page slices in `pages`.

## Vite + React

### Directory structure

```text
my-vite-project/
  src/
    app/                   ← FSD app layer
      providers/
      router.tsx
      styles/
      main.tsx             ← Entry point
    pages/
    shared/
  index.html
  vite.config.ts
  tsconfig.json
```

### Path aliases

Mirror the standard `tsconfig.json` mapping in `vite.config.ts` so the
Vite resolver agrees with TypeScript:

```typescript
// vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@/app": resolve(__dirname, "src/app"),
      "@/pages": resolve(__dirname, "src/pages"),
      "@/shared": resolve(__dirname, "src/shared"),
    },
  },
});
```

Create React App is no longer maintained. A project still on it follows
this section; the only difference is that the aliases go into a `craco`
config instead of `vite.config.ts`. Migrate to Vite when you can.

## React Router (framework mode)

React Router v7 in framework mode owns an `app/` directory for its root
layout, route config, and route modules. That name collides with the FSD
app layer, so apply the same fix as for Next.js: React Router's `app/`
stays at the project root, FSD lives in `src/`, and the FSD app layer is
renamed to `_app/`. Only `app/` collides, so `pages/` keeps its name.

Library mode (`createBrowserRouter` inside a plain Vite app) has no
framework directory. The Vite + React section applies as is, with the
router defined in the FSD app layer.

### Directory structure

```text
my-router-project/
  app/                     ← React Router (routing only)
    root.tsx               ← Root layout, mounts providers from @/_app
    routes.ts              ← Route config
    routes/
      home.tsx             ← Thin wrapper around @/pages/home
      product.tsx
  src/
    _app/                  ← FSD app layer
      providers/
      styles/
    pages/
      home/
      product/
        ui/ProductPage.tsx
        api/fetch-product.ts
        index.ts
    shared/
  react-router.config.ts
  vite.config.ts
  tsconfig.json
```

### Wiring routes to FSD pages

Route modules stay thin. They own the framework-required route exports and
delegate the application work to what the FSD page's public API exposes.

```typescript
// app/routes.ts
import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("products/:id", "routes/product.tsx"),
] satisfies RouteConfig;
```

```typescript
// app/routes/product.tsx
import type { Route } from "./+types/product";
import { ProductPage, fetchProduct } from "@/pages/product";

export const loader = ({ params }: Route.LoaderArgs) =>
  fetchProduct(params.id);

export default function ProductRoute({ loaderData }: Route.ComponentProps) {
  return <ProductPage product={loaderData} />;
}
```

**Framework requirement:** `loader` and the default component are Route
Module exports, so both live in the route file, and React Router hands
the component its generated `Route.ComponentProps`.

**This skill's recommendation:** let the generated types stop there. The
wrapper makes one call into the page's `api` segment or `shared/api` and
passes plain props down, so the FSD page stays framework-independent.

### Path aliases

Keep the standard `tsconfig.json` mapping, with `@/_app` pointing at
`src/_app`, and let `vite-tsconfig-paths` feed it to the React Router
Vite plugin:

```typescript
// vite.config.ts
import { defineConfig } from "vite";
import { reactRouter } from "@react-router/dev/vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [reactRouter(), tsconfigPaths()],
});
```

## Astro

Astro uses `src/pages/` for file-based routing, which collides with the FSD
`pages/` layer. Move the FSD pages layer to `src/_pages/` (with the
underscore prefix) and reserve `src/pages/` for Astro routes.

### Directory structure

```text
my-astro-project/
  src/
    pages/                 ← Astro routing (thin entry points)
      404.astro
      index.astro
    _pages/                ← FSD pages layer
      home/
        ui/HomePage.astro
        index.ts
    features/              ← when needed
    entities/              ← when needed
    widgets/               ← existing projects that keep the layer
    shared/
```

### Wiring Astro routes to FSD pages

The Astro route file imports and renders the FSD page, nothing else:

```astro
---
// src/pages/index.astro
import { HomePage } from '@/_pages/home';
---
<HomePage />
```

### Path aliases (tsconfig.json)

The official FSD Astro guide uses a single `@/*` alias pointing at `src/*`
rather than one alias per layer, and this section follows it:

```json
{
  "extends": "astro/tsconfigs/strict",
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

Imports then reference the layer path directly: `@/_pages/home`,
`@/shared/ui`, `@/entities/user`.

### Working with integrations

Some Astro integrations (for example, Starlight) use content collections
that expect content in fixed folders such as `src/content/docs/`. If the
integration does not allow the path to be changed, leave it as-is. The
content folder lives alongside FSD layers without collision:

```text
src/
  _pages/                  ← FSD pages layer
  content/                 ← Integration content (Starlight, etc.)
    docs/
      getting-started.md
  shared/                  ← FSD shared layer
```

Let the integration handle its own routing and rendering, while FSD layers
manage application-specific code.

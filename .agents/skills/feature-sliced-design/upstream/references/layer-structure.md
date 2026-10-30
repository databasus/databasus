# Layer Structure Reference

Detailed folder structures, code examples, and naming conventions for each
FSD layer. Use this reference when creating, reviewing, or reorganizing
project structure.

## App layer

App-wide initialization: providers, routing, global styles, entry point.
Organized by segments only, no slices.

The methodology does not formally standardize App segment names. The
common convention list (`ui`, `api`, `model`, `lib`, `config`) applies to
all layers but is rarely a good fit here. In practice, projects use names
that describe purpose: `routes`, `store`, `styles`, `providers`,
`entrypoint`, etc. Choose names that match your stack (for example,
`providers` for React/Vue provider components that wrap Redux,
QueryClient, or theme contexts):

```text
app/
  routes/          ← Route configuration (or router.tsx for single file)
  store/           ← Global state store (Redux configureStore, Zustand root)
  styles/          ← Global CSS, reset, theme variables
  providers/       ← Provider components (Redux Provider, QueryClientProvider)
  entrypoint.tsx   ← Application entry point (main.tsx, index.tsx)
```

A smaller project may collapse some of these into single files:

```text
app/
  router.tsx       ← Route configuration
  store.ts         ← Store configuration
  styles/
    global.css
  providers.tsx    ← All providers in one file
  index.tsx        ← Entry point
```

```typescript
// app/router.tsx
import { HomePage } from '@/pages/home';
import { ProfilePage } from '@/pages/profile';

export const router = createBrowserRouter([
  { path: '/', element: <HomePage /> },
  { path: '/profile/:id', element: <ProfilePage /> },
]);
```

**Belongs in app:** Global providers (Redux store, QueryClient, theme),
routing setup, global styles, error boundaries, analytics initialization.

**Does not belong:** Feature-specific code, business logic, page-level UI.

## Pages layer

Route-level composition. In v2.1, pages **own substantial logic**: they are
not thin wrappers. In early project stages, most code lives here.

```text
pages/
  home/
    ui/
      HomePage.tsx
      HeroSection.tsx
      FeaturesGrid.tsx
    model/
      home-data.ts          ← Page-specific state + logic
    api/
      fetch-home-data.ts    ← Page-specific API calls
    index.ts
  profile/
    ui/
      ProfilePage.tsx
      ProfileForm.tsx
      ProfileStats.tsx
    model/
      profile.ts            ← Profile state + validation logic
    api/
      update-profile.ts
      fetch-profile.ts
    index.ts
```

**Belongs in pages:** Page-specific UI, forms, validation, data fetching,
state management, business logic, API integrations. Even code that looks
reusable stays here if it is simpler to keep local.

**Does not belong:** Code that is already reused across multiple pages,
has a stable focused responsibility, and must have one shared home.
Extract when all three hold, not when reuse alone appears.

### Page layout patterns

A typical page composes features and entities from lower layers, plus its own
local UI components:

```typescript
// pages/product-detail/ui/ProductDetailPage.tsx
import { AddToCart } from '@/features/add-to-cart';
import { ProductCard } from '@/entities/product';
import { PageHeader } from './PageHeader'; // local to this page

export const ProductDetailPage = ({ productId }) => {
  const product = useProductDetail(productId); // local hook in this page

  return (
    <>
      <PageHeader />
      <ProductCard data={product} />
      <AddToCart productId={productId} />
      <RelatedProducts products={product.related} /> {/* local component */}
    </>
  );
};
```

For pages that only need shared + page-local code (no extracted layers):

```typescript
// pages/about/ui/AboutPage.tsx
import { Card } from '@/shared/ui/Card';
import { TeamSection } from './TeamSection';  // local to this page
import { MissionStatement } from './MissionStatement';

export const AboutPage = () => (
  <main>
    <MissionStatement />
    <Card><TeamSection /></Card>
  </main>
);
```

## Widgets layer (discouraged)

Widgets are a layer for placing reusable UI blocks. They can be composed from
multiple UI elements into a meaningful section of a screen and then used in
upper layers such as Pages or App.

> **The official layer reference discourages using the Widgets layer**, and
> this skill follows it. The reasoning, and where each kind of UI block
> goes instead, is in `SKILL.md`, Section 1.

One edge case: multiple flows from the Features layer may need to be composed
together, the kind of case that previously would have been placed in Widgets.
In most cases this can be resolved by taking a different approach to
composition. The parent (`pages` or `app`) imports the features and connects
them, which is Strategy C in `references/cross-import-patterns.md`. Still,
there may be edge cases that are genuinely hard to resolve. When that happens,
document the situation in
[feature-sliced/skills#7](https://github.com/feature-sliced/skills/issues/7).

Discouraging the layer does not mean removing it entirely. It means
recommending against actively adopting it. Projects already using widgets can
keep using them as before, and the standard slice/segment and public API rules
apply just as on any other layer:

```text
widgets/
  header/
    ui/
      Header.tsx
      Navigation.tsx
      UserMenu.tsx
    model/
      header.ts              ← Widget state
    api/
      fetch-notifications.ts
    index.ts
```

**If you still use widgets:** Navigation bars, sidebars, dashboards, and
footers are the typical examples. Simple UI primitives belong in `shared/ui/`,
and single-use page sections stay in the page.

## Where should layouts be placed?

Layout components often need to compose data handling, state management,
access control and user actions that are shared across multiple routes.

In React Router nested child routes may share a common URL path such as
`/users`, `/users/:id` and `/users/:id/settings`. Instead of repeating the
same handling in each page you can use the router's nesting capabilities to
apply a common layout and route-level logic in one place.

The location of a layout should be determined based on its **scope and
responsibility** rather than its structural complexity.

- Layouts responsible for the entire application or routing structure should
  be placed in `app`.
- Layouts specific to a particular page or route group should be placed in
  `pages`.
- Layout UI that is reusable without business context can be placed in
  `shared/ui`.
- Layouts centered around a specific user action or user flow and reused
  across multiple pages can be implemented in the corresponding `features`
  slice.

A layout in `shared` that directly imports from `features`, `entities` or
`pages` violates the layer import rule. Modules in `app` and `pages` can
import modules from lower layers to compose a screen.

> A module can only import modules from layers below the layer it belongs to.

Before extracting a layout into a separate module consider the following:

- Is this layout actually reused across multiple routes?
- Is it specific to a particular page or route structure?
- Is the layout itself the reusable unit or is it only the user action used
  within the layout?

A layout used by only a small number of pages and tied to a particular screen
structure may be simpler to define directly in the corresponding `page` or
route configuration.

1. **Configure a route layout in the App layer**
   You can group multiple routes with a common URL path using the router's
   nesting capabilities and assign a single layout in `app`.
   A layout located in `app` can compose modules from `pages`, `features`,
   `entities` and `shared` without violating the layer import rule.

2. **Pass feature UI through render props or slots**
   In React you can use the render props pattern. In Vue you can use slots.
   In this approach the layout in `shared` provides only the common UI
   structure while the required feature UI is passed from `app` or `pages`.
   This allows the layout to compose the required screen without directly
   depending on a specific feature.

3. **Define it directly in a page**
   A layout used only by a specific page can be defined directly in the
   corresponding `page` without introducing a separate abstraction.
   When there is little duplicated code and the layout is unlikely to change
   frequently there is no need to extract it into a shared module.

## Features layer

Independent, reusable user interactions. Create a feature when an
interaction is already reused across consumers, has a focused
responsibility, and needs one shared implementation. A second consumer on
its own does not require one (the extraction rule in `SKILL.md`).

```text
features/
  auth/                     ← Signing in
    ui/
      LoginForm.tsx
    model/
      auth.ts               ← Session state + logic
    api/
      login.ts
    index.ts
  register/                 ← Signing up, a separate use case
    ui/
      RegisterForm.tsx
    model/
      register.ts
    api/
      register.ts
    index.ts
  add-to-cart/
    ui/
      AddToCartButton.tsx
    model/
      cart.ts
    index.ts
  like-post/
    ui/
      LikeButton.tsx
    model/
      like.ts
    api/
      toggle-like.ts
    index.ts
```

**Feature composition**: features consume entities and are composed in
higher layers:

```typescript
// pages/feed/ui/PostCard.tsx  (composition lives in the page that uses it)
import { UserAvatar } from '@/entities/user';
import { LikeButton } from '@/features/like-post';
import { CommentButton } from '@/features/comment-create';

export const PostCard = ({ post }) => (
  <article>
    <UserAvatar userId={post.authorId} />
    <h2>{post.title}</h2>
    <p>{post.content}</p>
    <div>
      <LikeButton postId={post.id} />
      <CommentButton postId={post.id} />
    </div>
  </article>
);
```

## Entities layer

Reusable business domain models. Create an entity when domain logic or
state is already reused across consumers, has a focused responsibility,
and needs one authoritative home (the extraction rule in `SKILL.md`).
**Starting without this layer is completely valid.**

```text
// Minimal entity: model only (most common form)
entities/user/
  model/
    user.ts                  ← Types + domain logic
  index.ts

// Entity with UI (use with caution)
// Caution: adding UI to entities increases cross-import risk.
// Other entities may want to import this UI, leading to @x dependencies.
// Entity UI should only be imported from higher layers (features, pages,
// app), never from other entities.
entities/product/
  model/
    product.ts
  ui/
    ProductCard.tsx
  index.ts
```

## Shared layer structure

Infrastructure with no business logic. Organized by segments only (no slices).
Segments may import from each other.

```text
shared/
  ui/                ← UI kit: Button, Input, Modal, Card
  lib/               ← Utilities: formatDate, debounce, classnames
  api/               ← API client, route constants, CRUD helpers, base types
  auth/              ← Auth tokens, login utilities, session management
  config/            ← Environment variables, app settings
```

There is no `assets/` segment here. Assets live with the code that uses
them, and a shared presentation asset goes to `shared/ui/` with the
component that owns it (`references/asset-handling.md`).

The official API requests guide groups request functions under
`shared/api/endpoints/` and re-exports them from `shared/api/index.ts`.
The examples in this skill also show flat domain-named files (`client.ts`,
`product.ts`) and per-controller folders (`example/get-example.ts`); treat
those as variations of the same idea, not competing standards. Whatever the
internal shape, consumers import from the segment index, or from a
component folder's own index where Rule 4-2 allows one.

```typescript
// shared/ui/Button/Button.tsx
export const Button = ({ children, onClick, variant = 'primary' }) => (
  <button className={`btn btn-${variant}`} onClick={onClick}>
    {children}
  </button>
);

// shared/ui/Button/index.ts
export { Button } from './Button';
export type { ButtonProps } from './Button';
```

Shared **may** contain application-aware code: route constants, API
endpoints, branding assets, and transport types such as `ProductDTO`.
It must **never** hold the business rules an entity or feature owns, nor
import from those layers.

For asset placement specifically (images, icons, fonts, PDFs), see
`references/asset-handling.md`.

## Segments

A segment groups related code within a slice (or within App/Shared). The
standard segments cover the most common technical purposes:

- **`ui`**: UI display (components, date formatters, styles).
- **`api`**: backend interactions (request functions, data types, mappers).
- **`model`**: data model (schemas, interfaces, stores, business logic).
- **`lib`**: library code that other modules in this slice need.
- **`config`**: configuration files and feature flags.

Custom segments are allowed when needed (for example, `routes` and `i18n`
in the Shared layer, or `auth` for token storage when split out from
`shared/api`).

### Group by what it is *for*, not by what it *is*

Segment names describe **purpose**, not the kind of code they hold. This
is the desegmentation principle:

```text
// BAD: grouping by technical kind (what the code is)
shared/
  components/         ← What kind of components?
  hooks/              ← Which feature do they serve?
  types/              ← Which domain do they describe?
  utils/              ← Utility for what?
  helpers/            ← Same problem
  actions/            ← Redux actions for what?

// GOOD: grouping by purpose (what the code is for)
shared/
  ui/                 ← For displaying UI
  api/                ← For talking to the backend
  lib/                ← For library code that supports the slice
  config/             ← For configuration
```

A segment named `types/` cannot answer "types for what?" without inspecting
the contents. A segment named `model/` says: this is the data model.
Inside `model/`, files are named by domain (`user.ts`, `order.ts`), not by
technical role.

This rule applies everywhere: in `shared/`, in slices, and when designing
new custom segments.

## Naming conventions

### Domain-based file naming

Within a segment, name files after what they are for, the concern or
domain they serve, not after their technical mechanism:

```text
// BAD: technical-role naming mixes domains
model/types.ts          ← Which types? User? Order?
model/utils.ts
api/endpoints.ts        ← Every domain's requests in one file
model/selectors.ts

// GOOD: domain-based naming, each file owns one domain
model/user.ts           ← User types + logic + store
model/order.ts          ← Order types + logic + store
api/fetch-profile.ts    ← Clear what this API does
model/todo.ts           ← Redux slice + selectors + thunks
```

The fault in `api/endpoints.ts` is the single file, not the word. An
`endpoints/` directory holding one file per domain is the shape the
official API requests guide uses.

### Single-concern segments

If a segment contains only one domain concern, the filename may match the
slice name:

```text
features/auth/
  model/
    auth.ts          ← Single concern, matches slice name
```

### Index files as public API

Every slice must have an `index.ts` that re-exports its public interface:

```typescript
// entities/user/index.ts
export { UserAvatar } from "./ui/UserAvatar";
export { useUser, type User } from "./model/user";
```

## Slice groups

A **slice group** is a folder that contains related slices on the same
layer, used purely to make the structure easier to navigate as the number
of slices grows. A slice group is **not** a slice itself: it has no
segments (`model/`, `ui/`, `api/`), no public API (`index.ts`), and no
shared code. Slice isolation rules apply unchanged inside a group: sibling
slices in the same group cannot import from each other.

Slice groups are optional. Use them only when the layer has grown large
enough that a flat structure becomes hard to scan and there is an obvious
grouping criterion.

### When to use

- Several slices share the same business context and are scattered across
  the layer.
- The slice names clearly suggest they belong to the same topic.
- The layer has grown to the point where it is hard to scan at a glance.

### When NOT to use

- Names alone are enough for quick navigation.
- There is no natural grouping criterion.
- The group would hold too few slices to make the layer easier to scan.

### Example: grouping payment-related entities

```text
entities/
  payment/                  ← Slice group (no public API)
    invoice/                ← Slice
      model/
      ui/
      index.ts
    receipt/                ← Slice (model/, ui/, index.ts)
    transaction/            ← Slice (model/, ui/, index.ts)
  user/                     ← Slice (not in any group)
  product/                  ← Slice
```

Imports go through the full path:

```typescript
import { Invoice } from "@/entities/payment/invoice";
import { Receipt } from "@/entities/payment/receipt";
```

The same pattern applies to the Pages layer. For example, grouping
`pages/order/{list,detail,create}` when there are multiple pages on the same
topic such as list, detail, create, and edit. This is one possible example
and does not represent the default structure for the Pages layer.

### Features: use with caution

Slice groups can be applied to Features, but features often span multiple
entities and lack a natural grouping criterion. A group like
`features/cart/` tends to attract everything cart-related (DTOs, mappers,
helpers) until it stops being a navigation aid and starts acting as the
home for the entire cart domain, which weakens the principle that
features are split by use case. Before grouping features, check that the
group contains only feature slices and that the grouping earns its keep in
navigation.

### Anti-patterns

- **Do not put `index.ts` on the group folder.** That promotes the group
  to a slice and breaks the layer's contract.
- **Do not put shared `utils.ts`, `constants.ts`, or `types.ts` files
  inside the group.** A slice group has no shared code. Move reusable
  infrastructure to `shared/`. If the layer is `entities` and the shared
  logic is genuinely domain logic, consider whether the boundaries are
  too granular and the slices should be merged into one isolated entity
  (see `references/excessive-entities.md`). The `@x` notation does not
  apply to slice groups. It is a cross-import surface between entity
  slices, not a sharing mechanism for siblings within a group.
- **Do not relax slice isolation inside the group.** Grouping is
  navigation; it creates no sharing boundary. Siblings that need to depend
  on each other are resolved the same way as ungrouped slices, through
  `references/cross-import-patterns.md`, not with a `_common/` file.

## Path aliases

Configure path aliases so imports follow the `@/layer/slice` pattern. Add
an entry only for a layer the project actually has; an alias for a layer
that does not exist invites someone to fill it.

```json
// tsconfig.json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/app/*": ["src/app/*"],
      "@/pages/*": ["src/pages/*"],
      "@/shared/*": ["src/shared/*"]
    }
  }
}
```

That is the three-layer project from Section 5-3. Add `@/features/*`,
`@/entities/*`, or `@/widgets/*` when those layers appear, not before.

For framework-specific alias configuration (Vite, Next.js, Nuxt, Astro),
see `references/framework-integration.md`.

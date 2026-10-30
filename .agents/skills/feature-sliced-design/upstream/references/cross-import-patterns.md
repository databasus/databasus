# Cross-Import Resolution Patterns

How to resolve cross-imports between slices on the same layer. Rule 4-3
disallows them by default, so the first move is always to remove the
dependency by changing a boundary or a composition. Strategies A to C do
that. Strategy D and `@x` are the documented exceptions for a dependency
that cannot reasonably be removed, not a second way of working.

Treat a cross-import as a code smell rather than an impossible state:
some are deliberate, and those should be explicit and rare.

## What is a cross-import?

A cross-import is an import between different slices within the same layer.
For example:

- importing `features/apply-coupon` from `features/add-to-cart`
- importing `widgets/sidebar` from `widgets/header`

The `shared` and `app` layers do not have slices, so imports within those
layers are not cross-imports.

## Why is this a code smell?

Cross-imports blur domain boundaries and introduce implicit dependencies.
Four concrete problems:

1. **Unclear ownership and responsibility.** When `cart` imports from
   `product`, it becomes unclear which slice owns the shared logic. A
   change to `product`'s public contract now forces a change in `cart`,
   and a deep import couples `cart` to `product`'s internals as well.
   This makes bugs harder to localize and code harder to reason about.
2. **Reduced isolation and testability.** A core benefit of sliced
   architecture is that a slice can be read, changed, and tested with
   little knowledge of its siblings. Cross-imports break that. Testing
   `cart` now requires setting up `product`, and a change in one slice
   can fail tests in another.
3. **Increased cognitive load.** Working on `cart` now requires accounting
   for how `product` is structured. As cross-imports accumulate, tracing
   the impact of a change requires following more code across slice
   boundaries.
4. **Path to circular dependencies.** Cross-imports often start as one-way
   dependencies but evolve into bidirectional ones (A imports B, B imports
   A). This locks slices together and makes refactoring increasingly costly.

## Entities layer: prefer boundary merge over @x

Cross-imports in `entities` are usually caused by splitting entities too
granularly. Before reaching for `@x`, consider whether the boundaries should
be merged instead.

The `@x` notation is available as a dedicated cross-import surface for
`entities`, but it should be treated as a **last resort**, a **necessary
compromise**, not a recommended approach. Think of `@x` as an explicit
gateway for unavoidable domain references, not a general-purpose reuse
mechanism. Overuse locks entity boundaries together and makes refactoring
more costly over time.

### How @x works (when boundary merge is genuinely impossible)

Each entity exposes a special `@x/` directory containing files named after
the consuming entity. This makes the cross-import explicit and auditable.

**Direction rule:** in the path `entities/A/@x/B`, **A is the producer and
B is the consumer**. Read it as "A crossed with B": the file `A/@x/B.ts`
is the public API that A exposes specifically for B. So in the example
below, `entities/user/@x/order.ts` is what `user` exposes to `order`, and
`order` imports from it.

```text
entities/
  user/
    @x/
      order.ts          ← Exposed specifically for the order entity
    model/
      user.ts
    index.ts
  order/
    model/
      order-summary.ts  ← Imports from user/@x/order
    index.ts
```

```typescript
// entities/user/@x/order.ts: exposes only what order needs
export { getUserDisplayName } from "../model/user";

// entities/order/model/order-summary.ts
import { getUserDisplayName } from "@/entities/user/@x/order";
```

### Rules when using @x

1. Document why `@x` is needed and why merging boundaries does not apply.
2. Review periodically. Requirements change and `@x` may become unnecessary.
3. Minimize the surface area of `@x` exports.
4. Only between entities. Features and widgets should use Strategy C or D
   below, not `@x`.

## Features and widgets: four strategies

In `features` and `widgets`, multiple strategies are available depending on
project context. Cross-imports here are not always forbidden; they are
dependencies that should be deliberate. The four strategies below are
listed in preferred order, but each fits different situations.

### Strategy A: slice merge

If two slices are not truly independent and always change together, merge
them into a single larger slice.

```text
// Before: two features that always change together
features/edit-profile/
features/edit-profile-privacy/

// After: one cohesive feature
features/edit-profile/
  ui/
    EditProfileForm.tsx
    PrivacyFields.tsx
  model/
    edit-profile.ts
    privacy.ts
  index.ts
```

If two slices keep cross-importing each other and effectively move as one
unit, they are likely one feature in practice. Merging is often the simpler
and cleaner choice.

### Strategy B: move a shared domain responsibility into an entity

If multiple features share a domain rule or domain state, move that
responsibility into the entity that owns it. Key principles:

- What moves down is an established domain responsibility, not feature UI
  or user-flow orchestration.
- Interaction-specific UI and workflow logic stay in `features`.
- Features import that responsibility through the entity's public API.

For example, if `features/add-to-cart` and `features/buy-now` both need
the rule for whether a product can be purchased, that rule belongs to the
product domain, while each button remains its own user action.

```text
entities/
  product/
    model/
      can-purchase.ts     ← the shared domain rule
    index.ts

features/
  add-to-cart/
    ui/AddToCartButton.tsx
    model/add-to-cart.ts  ← imports canPurchase from @/entities/product
    index.ts
  buy-now/
    ui/BuyNowButton.tsx
    model/buy-now.ts      ← imports the same canPurchase
    index.ts
```

### Strategy C: compose from an upper layer (IoC)

When several lower-layer modules have to take part in one composition,
assemble them in a layer above all of them. A page can import multiple
features and entities to compose a screen, and components can be passed
through props or children where that helps. By default one feature does
not import another; Strategy D below is the documented exception when
that dependency cannot be removed.

Instead of connecting slices within the same layer via cross-imports,
compose them at a higher level (`pages` or `app`). The upper layer assembles
and connects the slices; the slices themselves do not know about each other.

Common Inversion of Control techniques:

- **Render props (React)**: pass components or render functions as props.
- **Slots (Vue)**: use named slots to inject content from parent components.
- **Dependency injection**: pass dependencies through props or context.

#### Basic composition (React)

```typescript
// features/follow-user/index.ts
export { FollowButton } from "./ui/FollowButton";

// features/report-user/index.ts
export { ReportUserButton } from "./ui/ReportUserButton";

// pages/profile/ui/ProfilePage.tsx
import { FollowButton } from "@/features/follow-user";
import { ReportUserButton } from "@/features/report-user";

export const ProfilePage = ({ userId }) => (
  <div>
    <FollowButton userId={userId} />
    <ReportUserButton userId={userId} />
  </div>
);
```

Following and reporting are two user actions that happen to sit on one
screen. Neither knows the other exists; the page puts them there.

#### Render props (React)

When one feature's UI has to place another feature's control inside it,
use a render prop to invert the dependency:

```typescript
// features/manage-wishlist/ui/WishlistItems.tsx
// RemoveButton is this feature's own ui/, not a cross-import.
interface WishlistItemsProps {
  items: WishlistItem[];
  renderAddToCart?: (productId: string) => React.ReactNode;
}

export const WishlistItems = ({ items, renderAddToCart }: WishlistItemsProps) => (
  <ul>
    {items.map((item) => (
      <li key={item.id}>
        <span>{item.title}</span>
        <RemoveButton itemId={item.id} />
        {renderAddToCart?.(item.productId)}
      </li>
    ))}
  </ul>
);

// pages/wishlist/ui/WishlistPage.tsx
import { WishlistItems } from "@/features/manage-wishlist";
import { AddToCartButton } from "@/features/add-to-cart";

export const WishlistPage = () => (
  <WishlistItems
    items={items}
    renderAddToCart={(productId) => <AddToCartButton productId={productId} />}
  />
);
```

`manage-wishlist` never imports `add-to-cart`. It leaves a hole per row,
and the page fills it.

#### Slots (Vue)

Vue's slot system provides a natural way to compose features without
cross-imports:

```vue
<!-- features/manage-wishlist/ui/WishlistItems.vue -->
<script setup lang="ts">
defineProps<{ items: WishlistItem[] }>();
</script>

<template>
  <ul>
    <li v-for="item in items" :key="item.id">
      <span>{{ item.title }}</span>
      <slot name="add-to-cart" :productId="item.productId" />
    </li>
  </ul>
</template>

<!-- pages/wishlist/ui/WishlistPage.vue -->
<script setup lang="ts">
import { WishlistItems } from "@/features/manage-wishlist";
import { AddToCartButton } from "@/features/add-to-cart";
</script>

<template>
  <WishlistItems :items="items">
    <template #add-to-cart="{ productId }">
      <AddToCartButton :productId="productId" />
    </template>
  </WishlistItems>
</template>
```

### Strategy D: cross-feature reuse only via Public API

If strategies A through C do not fit and cross-feature reuse is
genuinely unavoidable, allow it only through an explicit Public API
(exported hooks or UI components). Do not access another slice's
`store`, `model`, or internal implementation.

Unlike strategies A through C, which aim to eliminate cross-imports,
this strategy accepts them while minimizing risk through strict
boundaries.

```typescript
// features/auth/index.ts
export { useAuth } from "./model/use-auth";
export { AuthButton } from "./ui/AuthButton";

// features/edit-profile/ui/ProfileMenu.tsx
import { useAuth, AuthButton } from "@/features/auth";

export const ProfileMenu = () => {
  const { user } = useAuth();
  if (!user) return <AuthButton />;
  return <div>{user.name}</div>;
};
```

The boundary holds: `features/edit-profile` cannot import from
`@/features/auth/model/internal/*`. Only what `features/auth` explicitly
exposes through `index.ts` is reachable.

The `@x` notation is for the entities layer only. Features and widgets use
strategies A through D above; their access path is the standard public API
(`index.ts`), not a dedicated cross-import surface.

## When to treat a cross-import as a problem

After reviewing these strategies, the question is: when is a cross-import
acceptable to keep, and when should it be treated as a code smell and
refactored?

Common warning signs:

- Directly depending on another slice's `store`, `model`, or business logic
- Deep imports into another slice's internal files (bypassing the public API)
- Bidirectional dependencies (A imports B, and B imports A)
- Changes in one slice frequently breaking another slice
- Flows that should be composed in `pages` or `app`, but are forced into
  cross-imports within the same layer

When these signals appear, treat the cross-import as a code smell and apply
one of the strategies above.

## Strictness depends on project context

The strictness of cross-import enforcement depends on the project:

- In **early-stage products** with heavy experimentation, allowing some
  cross-imports may be a pragmatic speed trade-off.
- In **long-lived or regulated systems** (fintech, large-scale services),
  stricter boundaries pay off in maintainability and stability.

Cross-imports are not an absolute prohibition. They are dependencies that
are generally best avoided, but sometimes used intentionally. If a
cross-import is introduced:

- Treat it as a deliberate architectural choice.
- Document the reasoning in code (a comment explaining why other
  strategies do not apply).
- Revisit it periodically as the system evolves; if requirements change,
  the cross-import may no longer be needed.

## Decision flow for AI agents

```text
Two slices on the same layer need to share code.
  │
  ├─ ENTITIES layer?
  │   ├─ Are they one cohesive domain boundary?
  │   │   └─ YES → Merge. Stop.
  │   ├─ Does either entity really need to know the other, or can a
  │   │  page or feature hold both?
  │   │   └─ Compose above them. Stop.
  │   ├─ Is the shared part business-neutral infrastructure?
  │   │   └─ YES → Move that part to shared/. Stop.
  │   └─ Boundaries must stay separate and the domain dependency is real?
  │       └─ Use @x as last resort. Document why merge is not possible.
  │
  └─ FEATURES or WIDGETS layer?
      ├─ Strategy A: Do they always change together?
      │   └─ YES → Merge slices.
      │
      ├─ Strategy B: Is the shared part domain-only logic?
      │   └─ YES → Push down to entities. Keep UI in features.
      │
      ├─ Strategy C: Can the connection be assembled by a higher layer?
      │   └─ YES → Compose in pages or app via render props, slots, or DI.
      │
      └─ Strategy D: Is reuse genuinely unavoidable and the access surface
                     limited to a Public API?
          └─ YES → Allow, but only through index.ts. Never reach into
                   model/, store/, or internal files. Do not use @x in
                   features or widgets.
```

## Anti-patterns

- **Reaching for `@x` in features or widgets.** `@x` is for entities only.
  Use Strategy C (compose) or D (Public API) instead.
- **Treating `@x` as a clean solution.** It is a compromise. If you find
  yourself adding multiple `@x` files between the same entities, the
  boundaries are probably wrong. Merge them.
- **Bypassing the Public API to access internals.** Even when Strategy D is
  in use, importing from `@/features/auth/model/internal/*` defeats the
  purpose. Restrict yourself to what `index.ts` exports.
- **Bidirectional cross-imports.** A imports B and B imports A says the
  boundaries or the composition are wrong. Re-check whether the slices are
  one boundary, whether the shared part belongs lower, or whether the
  composition should move up.

## See also

- `references/excessive-entities.md`: prevent the conditions that lead to
  entity-layer cross-imports in the first place.
- `references/layer-structure.md`: layer rules and import directions.

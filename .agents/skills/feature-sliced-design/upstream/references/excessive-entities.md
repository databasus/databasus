# Excessive Entities

How to keep the `entities` layer clean and avoid over-extracting business
logic into entities. Excessive entities cause ambiguity (what code belongs
where), coupling, and constant import dilemmas as code scatters across
sibling entities.

An entity is not where every reusable business concept goes. It is a
stable, low-level domain boundary that several consumers genuinely have
to share.

## Why this matters

The `entities` layer is one of the lower layers and is widely accessible.
Every layer except `shared` can import from it. That global nature means
changes to `entities` propagate widely, so the boundaries need care up
front to avoid costly refactors. Adding an entity is cheap; removing one
after many consumers depend on it is expensive.

## How to keep entities clean

### 0. Consider having no entities layer

An FSD application without an `entities` layer is still FSD. Skipping the
layer simplifies the architecture and keeps it available for future scaling.

**Thin clients** (where the backend handles most data processing and the
client mostly exchanges data) usually do not need an entities layer.
**Thick clients** (significant client-side business logic) are better
candidates for entities.

The classification is not strictly binary. Different parts of the same
application may behave as thick or thin clients.

```text
// Thin client without entities layer (still valid FSD)
src/
  app/
  pages/
    dashboard/
    profile/
  shared/
    api/
    ui/
```

### 1. Avoid preemptive slicing

FSD v2.1 encourages **deferred decomposition** of slices. Place code in the
`model` segment of the consuming page (widget, feature) first. Move it to
`entities` later, when business requirements stabilize and reuse is
confirmed across multiple consumers.

The later code moves to `entities`, the less dangerous the refactor. Code
in `entities` can affect every higher-layer slice that imports it.

```text
// Iteration 1: code lives where it is used
pages/profile/
  model/
    profile-validation.ts    ← page-specific for now

// Iteration 2 (once several pages must share one copy of the rule):
entities/profile/
  model/
    profile-validation.ts    ← moved once the rule needs one home
```

### 2. Avoid unnecessary entities

Do not create an entity for every piece of business logic. Use types from
`shared/api` and place logic in the `model` segment of the current slice.
For genuinely reusable business logic, use the `model` segment within an
entity slice while the transport types stay in `shared/api`. That means
the API shapes, `OrderDto` here; a type that exists because of a business
rule can belong to the entity model once the entity does.

```text
shared/
  api/
    endpoints/
      order.ts              ← OrderDto type and request functions

entities/
  order/
    model/
      apply-discount.ts     ← Business logic that uses OrderDto
    index.ts
```

The DTO lives in `shared/api/endpoints/order.ts`. Once an `order`
boundary has been earned, reusable rules that operate on it (calculating
discounts, applying promotions) live in `entities/order/model/`. Until
then they stay in the consuming slice's `model/`. Do not mirror every API
endpoint with a corresponding entity.

### 3. Exclude CRUD operations from entities

CRUD operations involve boilerplate code without significant business
logic. Putting them in `entities` clutters the layer and obscures the code
that genuinely matters. Where it goes instead is the request placement
rule's answer, not a fixed path: a single consumer keeps it in that
slice's `api/`, and plain resource access shared across consumers lands
in `shared/api` (`references/auth-and-api.md`):

```text
shared/
  api/
    client.ts
    endpoints/
      order.ts          ← getOrder, createOrder, updateOrder, deleteOrder
      products.ts       ← Standard CRUD for products
      cart.ts           ← Standard CRUD for cart
    index.ts
```

For complex CRUD with atomic updates, rollbacks, or transactions, evaluate
whether the operation carries business rules. Complexity alone does not
send it to `entities`: decide by what owns the rule. A checkout or a
cancellation is a use case and belongs to a feature or the page that runs
it; a rule the domain owns belongs to the entity; anything else stays in
`shared/api`.

### 4. Store authentication data in shared

Prefer `shared/auth` (or `shared/api`) over a `user` entity for tokens and
session DTOs. They are specific to authentication, rarely reused outside
it, and wrapping a login response in a `user` entity tends to pull the
entity into `@x` chains. A `user` entity earns its place when user-domain
responsibilities hold a stable boundary outside the login flow, such as
profile identity read across several product contexts (avatars in
comments, names in posts). An entities layer that already exists is not
itself a reason, and neither is profile reuse on its own. Tokens and the
session stay in `shared/auth` unless an established entity genuinely owns
that state.

Both folder shapes, when to split `shared/auth` from `shared/api`, and the
three ways to expose the token to the API client are in
`references/auth-and-api.md`.

### 5. Minimize cross-imports

FSD permits cross-imports between entities via `@x`, but they introduce
technical issues including circular dependencies. Design entities within
**isolated business contexts** so cross-imports become unnecessary.

**Non-isolated context (avoid):**

```text
entities/
  order/
    @x/
    model/
  order-item/
    @x/
    model/
  order-customer-info/
    @x/
    model/
```

Three sibling entities all referencing each other through `@x`. This is a
sign that the boundaries are wrong.

**Isolated context (preferred):**

```text
entities/
  order-info/
    model/
      order-info.ts    ← order, items, and customer info together
    index.ts
```

One entity encapsulates the related logic, so there is no `@x` file and
no way for the sibling slices to form a cycle.

The general rule: when several entities have `@x` dependencies on each
other, treat that as a signal to merge the boundaries, not as something to
manage.

## Decision tree for AI agents

```text
A new piece of domain-related code or state needs a home.
  │
  ├─ Is the project a thin client?
  │   └─ YES → Strong signal to start without entities. Keep reading
  │            the branches below rather than stopping here.
  │
  ├─ Is the logic used in only one place right now?
  │   └─ YES → Keep in the consuming slice's model/. Defer extraction.
  │
  ├─ Is it a CRUD operation without business meaning?
  │   ├─ One consumer → that slice's api/ segment
  │   └─ Shared across consumers → shared/api/endpoints/<resource>.ts
  │
  ├─ Is it auth data (tokens, session, login DTOs)?
  │   ├─ Does an established user or session entity already own this
  │   │  state, rather than merely existing?
  │   │   └─ YES → that entity's model/
  │   └─ Otherwise → shared/auth/ (the default).
  │       An entities layer existing is not a reason to move it.
  │       Avoid placing in a page, widget, or single feature slice.
  │
  ├─ Is it just a TypeScript type for an API response?
  │   └─ YES → shared/api/. No entity needed for types alone.
  │
  └─ Is it reused now, stable enough to name, and required to stay
     consistent across its consumers?
      └─ YES → Create entities/<name>/model/.
               Verify the boundary is isolated and does not require @x
               to communicate with sibling entities.
```

## Anti-patterns

- **Creating entities preemptively.** Wait for reuse that is real, has a
  reason to change of its own, and needs one authoritative copy.
- **Mirroring every API endpoint with an entity.** API endpoints belong in
  `shared/api`. Entities exist for business logic, not for paralleling the
  backend structure.
- **Creating a `user` entity *only* to wrap a login response.** A `user`
  entity is justified when a stable user-domain responsibility is shared
  across non-auth flows (avatars in comments, names in posts). Until then
  `shared/auth` is simpler. The official Auth guide accepts a token store
  in Shared or in an entities slice for the current user or session
  (`references/auth-and-api.md`); what it rules out is a page, widget, or
  single feature slice.
- **Splitting one domain into many entities (`order`, `order-item`,
  `order-customer-info`).** This produces `@x` chains. Merge into a single
  isolated context (`order-info` or `order`).
- **Putting CRUD wrappers in entities.** They clutter the layer. Place
  them by the request placement rule: with their consumer while there is
  one, in `shared/api/endpoints/` once several slices call them.

## See also

- `references/cross-import-patterns.md`: how to handle cross-imports when
  they appear, and why `@x` is a last resort.
- `references/layer-structure.md`: layer responsibilities and the entities
  segment shape.

# Authentication, Types, and API Requests

Concrete code patterns for authentication, type definitions, and API
request handling within FSD structure. State management patterns (Redux,
TanStack Query) are in `references/state-management.md`. Code samples are
React; the placement rules are framework-agnostic.

## Authentication

Auth is one of the most common sources of confusion in FSD. The key question
is: what goes in `shared/`, what goes in `features/` or `pages/`?

### Auth data: `shared/auth/` or `shared/api/`

Credential storage, authentication-session plumbing, and the API-client
helpers around them are **infrastructure**, not business logic. Keep them
in shared. The login flow itself, its form state, validation and error
handling, is a user action and does not come with them:

```typescript
// shared/auth/token.ts
const TOKEN_KEY = "auth_token";
export const getToken = () => localStorage.getItem(TOKEN_KEY);
export const setToken = (t: string) => localStorage.setItem(TOKEN_KEY, t);
export const clearToken = () => localStorage.removeItem(TOKEN_KEY);

// shared/auth/session.ts
export interface Session { userId: string; email: string; role: "admin" | "user" }
// useSession depends on the auth provider (React Context, Zustand, etc.)
export const useSession = (): Session | null => { /* ... */ };
```

The `shared/auth/index.ts` re-exports from these files following the
standard public API pattern.

Which of the two: the token can sit in `shared/api` next to the client,
where every request function can reach it directly. When token management
grows past that (refresh, expiry, invalidation), the official Auth guide
separates the responsibilities: requests and the API client stay in
`shared/api`, and the token store with its management logic moves to
`shared/auth`.

### Auth UI: pages (single use) or features (multi-use)

Place the login form in the slice that consumes it. Single-use (only on the
login page) goes in `pages/login/`; multi-use (dedicated page + modal login)
goes in `features/auth/`:

```text
pages/login/                     ← Single-use
  ui/{LoginPage,LoginForm}.tsx
  model/login.ts                 ← Form state, validation
  api/login.ts                   ← POST /auth/login
  index.ts

features/auth/                   ← Multi-use: signing in
  ui/LoginForm.tsx
  model/auth.ts
  api/login.ts
  index.ts
features/register/               ← Signing up, its own use case
  ui/RegisterForm.tsx
  model/register.ts
  api/register.ts
  index.ts
```

### Dialog for login

If you need a login dialog that can be reused across multiple pages, you can
implement it as a **feature** responsible for the login user action and flow.
A login dialog typically includes logic such as form state management, input
validation, authentication requests and error handling. These responsibilities
belong in the `features` layer because they handle user actions and flows.

```text
features/
  auth/
    ui/LoginDialog.tsx
    model/
    api/
    index.ts
```

When multiple pages need the login dialog, they can import and use the feature
from each page or from the route configuration in `app`.

A component responsible only for the common dialog UI and basic interactions
can be placed in `shared/ui`. This component should not include login-specific
logic such as authentication requests, input validation or authentication
state management.

The UI and logic required for login should be managed in `features/auth`.
When necessary, `LoginDialog` can be implemented by composing the dialog
component from `shared/ui`.

```text
shared/
  ui/
    modal/
      Modal.tsx
      index.ts
features/
  auth/
    ui/LoginDialog.tsx
    model/
    api/
    index.ts
```

### When to use shared/auth vs a user entity

The official Auth guide presents two valid storage locations: **In Shared**
(`shared/auth` or `shared/api`) and **In Entities** (a `user` entity).
**In Pages/Widgets** is not recommended.

`shared/auth` is the default for tokens, refresh and expiry handling, and
the rest of the authentication session. Keep them there unless something
else already owns that state.

A `user` or `session` entity may own auth state, token included, when that
entity is an established boundary that genuinely owns it. Ownership is what
decides. An entities layer existing, a `user` entity existing, and profile
data being reused are not reasons to move credentials.

```text
// Path A: shared/auth (simpler default)
shared/auth/session.ts         ← userId, email, role, token

// Path B: an established user-domain boundary that owns this state
entities/user/
  model/
    current-user.ts            ← Current authenticated user + token
    user.ts                    ← Generic user type
  api/get-current-user.ts
  index.ts
```

For the entity approach, the API client in `shared/api` cannot import from
`entities/`. The official guide describes three solutions: pass the token
manually, expose it through a context with the key kept in `shared/api`,
or inject the token into the API client when the entity store updates.

A `user` entity created **only** to wrap a login response is premature.
`references/excessive-entities.md` explains what that costs.

### In Pages/Widgets (not recommended)

It is not recommended to place the token store in `pages` or in a specific
`features` slice.

Tokens are not state that belongs only to a specific page or a single user
action. They are application-wide state used by multiple authenticated API
requests and user flows.
For example, if the token store is placed in `features/auth`, another feature
cannot directly import it. Different feature slices on the same layer should
remain independent from one another.

Similarly, if the token store is placed in `pages`, modules on lower layers
cannot access it. This makes it difficult to reuse the token store in
authenticated API requests or other user flows.
Place the token store in `shared` or in an `entities` slice representing the
current user or session, according to the criteria described above.

### Logout and token invalidation

Most applications do not provide a separate page exclusively for logout.
Instead, logout functionality is made available wherever it is needed, such as
in a header, settings screen, or user menu.

Logout generally consists of the following steps.

1. Send an authenticated logout request to the backend.
   For example, `POST /logout`.
2. Reset the token store.
   Remove both the access token and refresh token.
3. Reset the current user information and authentication state when necessary.
4. Navigate to the login page or another screen when necessary.

The location of the logout request should be determined by the project's API
organization and the scope in which the request is reused.
If all API endpoints are managed in `shared/api`, authentication-related
requests such as login, logout, and token refresh can be placed together.

```text
shared/
  api/
    client.ts
    endpoints/
      login.ts
      logout.ts
      refresh-token.ts
    index.ts
```

If the logout request is used only as part of a specific logout flow, it can
be placed in the `api` segment of `features/logout`.
If logout is reused across multiple screens and represents an independent user
flow that includes token cleanup, user state cleanup, and error handling,
the flow can be extracted into `features/logout`.

Navigation is the exception. `features/logout` must not import the router
from `app/`, which would be an upward import (Rule 4-1). Let the page or
route configuration navigate after the action resolves, or pass the
navigation callback into the feature:

```typescript
// pages/settings/ui/SettingsPage.tsx
const logout = useLogout();

const onLogout = async () => {
  await logout();
  navigate("/login");
};
```

```text
features/
  logout/
    api/logout.ts
    ui/LogoutButton.tsx
    index.ts
```

If logout requires its own state or reusable processing logic, a `model`
segment can be added. There is no need to create unused segments in advance
for a simple logout feature.

`features/logout` coordinates tasks such as sending the logout request and
resetting the token store as a single user flow. The token store itself and
the token management logic should remain in the previously selected location
under `shared` or `entities`.

On the other hand, if the logout logic is simple and used in only one or two
places, it does not necessarily need to be extracted into a separate feature.
It can be composed directly in the page or route configuration where it is
used.

> Slice names should be based on user actions and flows rather than the UI
> location where they are displayed. Therefore, even when logout is triggered
> from a header, `features/logout` is more appropriate than `features/header`
> when the behavior is extracted as an independent user flow.

### Automatic logout

The token store and current user state should be reset when the client's
authentication state can no longer be maintained, such as in the following
cases:

- The user requests to log out.
- The refresh token has expired or is invalid, causing the token refresh
  request to be rejected.

If the authentication state is not reset, the UI may appear as though the user
is still logged in while authenticated API requests continue to fail.
Even if the logout request fails, the client can still reset the token store
and current user state. However, the server-side session or refresh token may
not have been invalidated, so the backend authentication policy should also be
taken into account.

> If tokens are managed in an entity representing the current user or session,
> the token reset logic can be placed in the slice's `model` segment. If tokens
> are managed on the Shared layer, they can be separated into a module
> responsible for authentication, such as `shared/auth`.

The refresh failure is usually detected by the API client in `shared/api`,
which cannot reach an entity to clear its state: that would be an upward
import. Report the failure upward instead, through the callback, event, or
context the official guide already uses to hand the token down, and let the
layer that owns the state do the resetting. The same wiring that gets the
token into the client carries the failure back out.

## Type definitions

### Where to define types

The location of type definitions follows the same rules as any other code:

| Type scope | Location |
| --- | --- |
| API response/request shapes shared across the app | Domain-named files in `shared/api/` (e.g., `shared/api/product.ts`) |
| Types for a specific entity's domain model | `entities/<name>/model/<name>.ts` |
| Types used only within one page | `pages/<name>/model/<name>.ts` |
| Types used only within one feature | `features/<name>/model/<name>.ts` |
| Generic utility types (e.g., `Nullable<T>`) | Purpose-named files in `shared/lib/` (e.g., `shared/lib/nullable.ts`) |

Per Rule 4-4 (domain-based file naming), avoid grouping all types in
`types.ts` or `utils.ts`. A file named `types.ts` cannot answer "types
for what?" without inspection; a file named `product.ts` can.

### Example: API types in shared

```typescript
// shared/api/product.ts: raw API response shapes
export interface ProductDTO {
  id: string;
  name: string;
  price: number;
  category: string;
  createdAt: string;
}
```

### Example: domain types in entities

```typescript
// entities/product/model/product.ts: the shape the domain works with
import type { ProductDTO } from "@/shared/api";

export interface Product {
  id: string;
  name: string;
  price: number;
  listPrice: number;
  inStock: boolean;
}

export const fromProductDTO = (dto: ProductDTO): Product => ({
  id: dto.id,
  name: dto.name,
  price: dto.price,
  listPrice: dto.listPrice,
  inStock: dto.stock > 0,
});

// a rule, not a stored flag: it follows the price rather than a snapshot
export const isOnSale = (product: Product) =>
  product.price < product.listPrice && product.inStock;
```

**Key principle:** Raw API shapes go in `shared/api/`. A domain model stays
with its current consumer until an entity boundary has been earned; once it
has, the types and rules that entity owns live in its `model/`. If you only
need the raw shape, do not create an entity just for types.

## API request handling

### Basic pattern: API calls in the consuming slice

```typescript
// pages/product-detail/api/fetch-product.ts
import { apiClient, type ProductDTO } from "@/shared/api";

export const fetchProduct = (id: string): Promise<ProductDTO> =>
  apiClient.get(`/products/${id}`).then((r) => r.data);
```

### Shared API client setup

```typescript
// shared/api/client.ts
import axios from "axios";
import { getToken } from "@/shared/auth";

export const apiClient = axios.create({ baseURL: import.meta.env.VITE_API_URL });

apiClient.interceptors.request.use((config) => {
  const token = getToken();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});
```

The `shared/api/index.ts` re-exports from these files, so consumers import
`apiClient` and the DTO types from `@/shared/api` rather than reaching into
`client.ts` or `product.ts` (Rule 4-2).

### CRUD helpers in shared

```typescript
// shared/api/create-crud-api.ts
import { apiClient } from "./client";

export const createCrudApi = <T>(resource: string) => ({
  getAll: () => apiClient.get<T[]>(`/${resource}`).then((r) => r.data),
  getById: (id: string) => apiClient.get<T>(`/${resource}/${id}`).then((r) => r.data),
  create: (data: Partial<T>) => apiClient.post<T>(`/${resource}`, data).then((r) => r.data),
  update: (id: string, data: Partial<T>) => apiClient.put<T>(`/${resource}/${id}`, data).then((r) => r.data),
  remove: (id: string) => apiClient.delete(`/${resource}/${id}`),
});

// Usage: export const productsApi = createCrudApi<ProductDTO>("products");
```

### Request placement rule

Two questions decide where a request function goes. Ask them in order.

**Question 1: does one consumer own this request, or is it genuinely
shared?**

One consumer means the request stays with that consumer. This is the
pages-first rule applied to `api/` segments.

- Data fetching for a single page (e.g., dashboard stats) →
  `pages/<name>/api/`
- An action owned by a single feature (e.g., `toggleLike`) →
  `features/<name>/api/`

Genuinely shared between consumers: continue to question 2.

**Question 2: does the request carry domain rules?**

Domain rules are permission checks, status transitions, derived
calculations, or a model the frontend composes from several responses.
Knowing a resource's URL and response shape is not a domain rule.

- No domain rules → `shared/api/`. Plain resource access is
  infrastructure no matter how many slices call it. Generic CRUD belongs
  here; build it from `shared/api/create-crud-api.ts`.
- Domain rules → `entities/<name>/api/`, once the boundary is stable.

A `getUserById` that only wraps `GET /users/:id` stays in `shared/api/`
even when every page calls it. If it resolves the caller's permissions
first, ask who owns that rule: an established user domain puts it in
`entities/user/api/`, while a rule that only one screen applies stays with
that page or feature.

> **Where this comes from.** The official API requests guide defaults
> request functions to `shared/api` or the consuming slice's `api`
> segment, and warns against placing API calls in `entities` prematurely.
> Question 2 follows that advice. Existing code placed under an earlier
> reading is still not a violation: a request function already sitting in
> `entities/<name>/api/` without domain rules keeps working. Relocate it
> when you are already changing that slice, or when the entity has no
> other reason to exist. Do not sweep a repository to move these
> functions.

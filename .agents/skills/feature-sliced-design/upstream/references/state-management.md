# State Management

Concrete code patterns for Redux and TanStack Query (React Query) within
FSD structure. Authentication, type, and API request patterns are in
`references/auth-and-api.md`. Code samples are React; the placement
rules are framework-agnostic.

## State management: Redux

FSD has no Redux guide of its own. The placement rule below is Step 6 of
the official `from-custom` migration guide, and the only official Redux
code is in the business entities guide. The rest of this section is
Rules 4-1, 4-2, and 4-4 applied to Redux Toolkit.

### Where a Redux slice belongs

**Redux does not decide the layer; ownership does.** Work out which slice
owns the state with Section 2 of `SKILL.md`, then put the Redux code in
that slice's `model/` segment. A `todo` noun does not make an entity, and
a `toggle-todo` verb does not make a feature.

Once a boundary has been earned, Step 6 gives two destinations: a stable
reusable business-domain responsibility may move to Entities, a stable
reusable user-interaction boundary may move to Features. Both assume the
slice is already reused across pages. A slice used by a single page stays
in that page's `model/` segment.

### Business-entity slice in entities

The request is plain resource access, so it lives in `shared/api` with
its DTO (Request placement rule in `references/auth-and-api.md`). The
entity imports it; `model/` holds only the Redux wiring. The entity uses
the transport type as it is here; convert to a separate domain type only
when a business rule needs a shape the backend does not send.

```typescript
// shared/api/todo.ts
import { apiClient } from "./client";

export interface TodoDto { id: string; title: string; completed: boolean }

export const getTodos = (): Promise<TodoDto[]> =>
  apiClient.get("/todos").then((r) => r.data);
```

```typescript
// entities/todo/model/todo.ts
import { createSlice, createAsyncThunk } from "@reduxjs/toolkit";
import { getTodos, type TodoDto } from "@/shared/api";

interface TodoState { items: TodoDto[]; loading: boolean }

export const fetchTodos = createAsyncThunk("todos/fetch", getTodos);

const todoSlice = createSlice({
  name: "todos",
  initialState: { items: [], loading: false } as TodoState,
  reducers: {
    setCompleted: (state, { payload }: { payload: { id: string; completed: boolean } }) => {
      const todo = state.items.find((t) => t.id === payload.id);
      if (todo) todo.completed = payload.completed;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchTodos.pending, (state) => { state.loading = true; })
      .addCase(fetchTodos.fulfilled, (state, action) => {
        state.items = action.payload;
        state.loading = false;
      });
  },
});

export const { setCompleted } = todoSlice.actions;
export const selectTodos = (state: { todos: TodoState }) => state.todos.items;
export const todoReducer = todoSlice.reducer;
```

A thunk that still carries its own request belongs in the `api` segment
(`references/migration-guide.md`, Part 2 Step 5). Once the request lives
in `shared/api`, the thunk is only Redux wiring and stays in `model/`
next to the reducer that handles it, which is the shape of the official
business entities guide.

The selector takes only the state it reads, not `RootState`. `RootState` is
declared in `app/`, so an entity importing it would depend on a higher layer
(Rule 4-1). The trade-off is that the selector no longer type-checks against
the whole store. Type it at the `app/` layer when you need that guarantee.

The slice's public API re-exports what consumers need:

```typescript
// entities/todo/index.ts
export { todoReducer, selectTodos, setCompleted, fetchTodos } from "./model/todo";
```

**Key:** Do not split Redux code by Redux mechanism into `reducers.ts`,
`selectors.ts`, and `thunks.ts`. That is the technical-role naming Rule
4-4 rules out. Keep a reducer, its selectors, and its thunks together in
one domain-named file, and when the model outgrows it, split by domain
concern (`todo.ts`, `todo-filter.ts`) rather than by mechanism.

### User-action slice in features

Assuming the action is already reused across pages and has earned a
feature boundary, it consumes the entity through the entity's public API
and exposes its own hook through the feature's:

```typescript
// features/toggle-todo/model/use-toggle-todo.ts
import { useDispatch } from "react-redux";
import { setCompleted } from "@/entities/todo";

export const useToggleTodo = () => {
  const dispatch = useDispatch();
  return (id: string, current: boolean) =>
    dispatch(setCompleted({ id, completed: !current }));
};
```

### Registering slices in app

```typescript
// app/providers/store.ts
import { configureStore } from "@reduxjs/toolkit";
import { todoReducer } from "@/entities/todo";
import { userReducer } from "@/entities/user";

export const store = configureStore({
  reducer: {
    todos: todoReducer,
    user: userReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
```

The store imports each slice's reducer through its public API
(`index.ts`), never reaching into `model/` directly (Rule 4-2). Do not
let individual slices create their own stores.

## State management: TanStack Query (React Query)

This section follows the official React Query guide. Guidance applies to
`@tanstack/react-query` v5 (formerly React Query). The package name is
`@tanstack/react-query`.

### Where to store query keys

Three placements are valid. Choose by project size and by which slice owns
the request, not by which folders happen to exist.

**Option 1: Flat in `shared/api/queries/`** (small projects, few endpoints):

```text
shared/api/
  queries/
    example.ts
    another-example.ts
  index.ts          ← export { exampleQueries } from './queries/example';
```

**Option 2: Per controller in `shared/api/<controller>/`** (many endpoints):

```text
shared/api/example/
  index.ts          ← export { exampleQueries } from './example.query';
  example.query.ts  ← Query factory: keys + functions
  get-example.ts
  create-example.ts
  update-example.ts
  delete-example.ts
```

**Option 3: Per entity in `entities/<entity>/api/`** when the request
carries domain rules, the entity boundary already exists, and each request
corresponds to a single entity. Generic CRUD and plain resource access
stay in `shared/api` however many slices call them (`auth-and-api.md`,
request placement rule, Question 2). When entities reference each other,
see `references/cross-import-patterns.md` for `@x` as a last resort.

> **Where this comes from.** Two official guides differ here. The React
> Query guide calls the per-entity split the cleanest option once a
> project has entities, and shows CRUD files inside `entities/*/api/`.
> The excessive-entities guide excludes CRUD from entities and keeps it
> in `shared/api/endpoints/`. This skill follows the second, and
> Question 2 is how it decides: an existing entities folder is not by
> itself a reason to move a request into it.

### Where to store mutations

Mixing mutations with queries is not recommended. Two patterns are
accepted:

1. **A mutation hook in the `api/` segment near the place of use.** Use
   `setQueryData` for cache updates:

   ```typescript
   // src/pages/example/api/use-update-example.ts
   export const useUpdateExample = () => {
     const queryClient = useQueryClient();
     return useMutation({
       mutationFn: ({ id, newTitle }) => apiClient.patch(`/posts/${id}`, { title: newTitle }).then((r) => r.data),
       onSuccess: (newPost, { id }) => queryClient.setQueryData(POST_QUERIES.detail({ id }).queryKey, newPost),
     });
   };
   ```

2. **A `mutationFn` defined in `shared/` or `entities/`** and called from
   `useMutation` in the component.

### Query factory pattern

A query factory is an object whose values return query keys. Each key is
wrapped in `queryOptions`, a built-in helper from `@tanstack/react-query` v5
that lets you share `queryKey` and `queryFn` between `useQuery`,
`useSuspenseQuery`, `prefetchQuery`, `setQueryData`, and similar APIs
without rewriting them:

```typescript
// src/shared/api/post/post.queries.ts
import { queryOptions } from "@tanstack/react-query";
import { getPosts, getDetailPost, type DetailPostQuery } from "./get-posts";

export const POST_QUERIES = {
  all: () => ["posts"],
  lists: () => [...POST_QUERIES.all(), "list"],
  list: (page: number, limit: number) => queryOptions({
    queryKey: [...POST_QUERIES.lists(), page, limit],
    queryFn: () => getPosts(page, limit),
    placeholderData: (prev) => prev,
  }),
  detail: (query?: DetailPostQuery) => queryOptions({
    queryKey: [...POST_QUERIES.all(), "detail", query?.id],
    queryFn: () => getDetailPost({ id: query?.id }),
  }),
};
```

Consume with `useQuery(POST_QUERIES.detail({ id }))`. For pagination,
`placeholderData: prev => prev` prevents UI flicker when navigating pages.

**Benefits of a query factory:** the keys and query definitions for a
domain are reachable through one object, so consumers share them instead
of rebuilding keys. Refetching and cache updates become a one-line call
(`queryClient.invalidateQueries({ queryKey: POST_QUERIES.all() })`). The
request functions themselves stay in their own files; the factory wires
them to keys.

### Infinite scroll

Use `infiniteQueryOptions` with `initialPageParam` and `getNextPageParam`.
Add the infinite key to the same factory shown above:

```typescript
import { infiniteQueryOptions } from "@tanstack/react-query";

// Inside POST_QUERIES:
infinite: (limit: number) => infiniteQueryOptions({
  queryKey: [...POST_QUERIES.lists(), "infinite", limit],
  queryFn: ({ pageParam }) => getPosts(pageParam, limit),
  initialPageParam: 0,
  getNextPageParam: (lastPage) => lastPage.skip + lastPage.limit < lastPage.total ? lastPage.skip / lastPage.limit + 1 : undefined,
}),
```

Consume with `useInfiniteQuery` and flatten via `data?.pages.flatMap(...)`.

### Suspense mode

`queryOptions` and `useSuspenseQuery` are compatible, and the factory does
not change. Components use `useSuspenseQuery` instead of `useQuery` and skip
`isLoading` entirely. Wrap interested subtrees with an `ErrorBoundary` +
`Suspense` provider in the App layer:

```tsx
// src/app/providers/suspense-provider.tsx
import { Suspense } from "react";
import { ErrorBoundary } from "react-error-boundary";

export const SuspenseProvider = ({ children }) => (
  <ErrorBoundary fallback={<div>Something went wrong</div>}>
    <Suspense fallback={<div>Loading...</div>}>{children}</Suspense>
  </ErrorBoundary>
);
```

### Reading mutation state with useMutationState

`useMutationState` lets any component read the state of a mutation without
passing props, useful for global save indicators. Store mutation keys next
to the query factory:

```typescript
// src/shared/api/post/post.queries.ts
export const POST_MUTATIONS = {
  updateTitle: () => ["post", "update-title"],
  create: () => ["post", "create"],
};
```

Tag the mutation with `mutationKey`, then read its state from any component:

```tsx
// src/features/update-post/api/use-update-post-title.ts
export const useUpdatePostTitle = () =>
  useMutation({
    mutationKey: POST_MUTATIONS.updateTitle(),
    mutationFn: ({ id, newTitle }) => apiClient.patch(`/posts/${id}`, { title: newTitle }),
  });

// src/app/ui/save-indicator.tsx   (app-wide; page-local if one page)
import { useMutationState } from "@tanstack/react-query";
import { POST_MUTATIONS } from "@/shared/api/post";

export const SaveIndicator = () => {
  const isPending = useMutationState({
    filters: { mutationKey: POST_MUTATIONS.updateTitle(), status: "pending" },
    select: (m) => m.state.status,
  }).length > 0;
  return isPending && <span>Saving...</span>;
};
```

### QueryProvider in the app layer

```tsx
// src/app/providers/query-provider.tsx
import { QueryClient, QueryClientProvider, MutationCache, QueryCache } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { toast } from "sonner";

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: (e) => toast.error(e.message) }),
  mutationCache: new MutationCache({ onError: (e) => toast.error(e.message) }),
  defaultOptions: { queries: { staleTime: 5 * 60 * 1000, gcTime: 5 * 60 * 1000 } },
});

export const QueryProvider = ({ children }) => (
  <QueryClientProvider client={queryClient}>
    {children}
    <ReactQueryDevtools />
  </QueryClientProvider>
);
```

`QueryCache.onError` and `MutationCache.onError` give one place to wire up
global toast notifications instead of repeating error handling on every hook.

### Code generation

The official guide notes that OpenAPI/Swagger generators are less flexible
than the hand-written factory above. Whichever you pick, generated clients
are transport code: keep them in `@/shared/api/`, or in a separate
generated package. Do not move generated endpoints into entities because
they happen to name business resources.

### Custom API client

Standardize base URL, headers, and JSON handling in a single class in
`shared/api/`:

```typescript
// src/shared/api/api-client.ts
export class ApiClient {
  #baseUrl: string;
  constructor(url: string) { this.#baseUrl = url; }

  async #handle<T>(response: Response): Promise<T> {
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  }

  get = <T>(path: string) => fetch(`${this.#baseUrl}${path}`).then((r) => this.#handle<T>(r));
  // post, put, delete follow the same pattern with method/headers/body.
}

export const apiClient = new ApiClient(API_URL);
```

**Key principle:** Place query and mutation hooks with the slice that owns
the responsibility. Page-specific queries stay in the page. Plain resource
access and generic CRUD stay in `shared/api/`. Reach for
`entities/<name>/api/` only when an established entity boundary owns that
request; an Entities layer existing is not a placement rule.

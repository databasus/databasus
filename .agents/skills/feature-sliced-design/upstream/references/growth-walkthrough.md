# Growth Walkthrough

One small shop through four snapshots, showing which moments earn a layer
and which do not. Read it when starting a project, or when deciding
whether entities are needed yet. Each snapshot gives the tree, what
changed in the product, and which rule from `SKILL.md` decided the
response. Every decision here comes from `SKILL.md`; this file adds no
placement rules of its own. Not every product change earns a layer: a
layer is earned by a stable responsibility that needs one home, not by a
count of how many places use something.

## Snapshot 0: two pages, three layers

The shop has a home page with a product list and a product detail page.
The detail page shows an "on sale" badge.

```text
src/
  app/
    providers/
    router.tsx
    styles/
  pages/
    home/
      ui/HomePage.tsx
      ui/ProductCard.tsx        ← list card, used only here
      api/fetch-products.ts     ← one consumer: this page
      index.ts
    product/
      ui/ProductPage.tsx
      ui/SaleBadge.tsx          ← badge, used only here
      model/is-on-sale.ts       ← the rule: price < listPrice
      api/fetch-product.ts      ← one consumer: this page
      index.ts
  shared/
    api/
      client.ts
      product.ts                ← ProductDTO, read by both pages
      index.ts
    ui/
      Button/
      Card/
```

**What is absent on purpose.** No `entities/`, no `features/`, no
`widgets/`. Each request has one consumer, so it sits in that page's
`api/` segment: Question 1 of the request placement rule is asked before
Question 2 (`references/auth-and-api.md`). `ProductDTO` is in
`shared/api` because both pages read the same transport shape. The sale
rule sits in the product page because only that page applies it (Step 1).
This is complete, valid FSD (Section 5-3).

## Snapshot 1: a third page reuses product data, no layer appears

A search page is added. It fetches products, shows them as cards, and
marks the ones on sale.

```text
  pages/
    home/
      index.ts                  ← api/fetch-products.ts moved out
    search/                     ← new slice
      ui/SearchPage.tsx
      ui/ProductCard.tsx        ← a second card, copied from home
      model/is-on-sale.ts       ← a second copy of the rule, from product
      index.ts
  shared/
    api/
      product.ts                ← ProductDTO, and now fetchProducts
```

**The request moved down, and no layer opened.** `fetchProducts` has two
consumers now, so Question 1 stops keeping it in the home page.
Question 2 asks whether it carries domain rules, and a URL with a
response shape is not a rule, so it lands in `shared/api` next to the
DTO. Note where it did not land: `entities/product/api` was never a
candidate, and the official API requests guide warns against putting
requests there prematurely. Moving code down a layer is not the same as
opening one.

`fetchProduct` did not move. The detail page is still its only consumer.

**The second card is a copy, not an extraction.** Two `ProductCard`
files look like a signal for `entities/product/ui`. They are not, yet.
The search card shows a match snippet and the home card does not, so they
are not the same code, and they will keep drifting apart for their own
reasons. Step 1 covers this case directly: used in two pages but the
duplication is manageable, so separate copies are valid. Extracting now
would force two cards that want to differ into one component that has to
serve both.

**The rule is copied too.** The search card marks sale items, so
`is-on-sale.ts` is copied out of the product page. Keeping both copies
local is still cheaper than committing to a shared boundary before the
two consumers are required to agree, and so far nothing requires it. A
second copy is not a boundary on its own. What turns one into a boundary
is the subject of Snapshot 2.

## Snapshot 2: a rule diverges, `entities/product` appears

Marketing changes what "on sale" means: the price must be below the list
price *and* the item must be in stock. The product page is updated. The
copy in the search page, made in Snapshot 1, is not. Search now marks
items on sale that the detail page says are not.

This is the signal. The two copies are the same rule, they must agree,
and they no longer do. Check the extraction rule (`SKILL.md`, Section 1):

1. The same code is used in multiple places right now. Yes, two pages.
2. It has a reason to change that is independent of any one consumer.
   Yes: the rule changes when marketing changes it, not when either page
   changes.
3. The boundary has a focused responsibility. Yes: "is this product on
   sale" and nothing else.

All three hold, so the rule gets one home (Step 4).

```text
  entities/                     ← new layer
    product/
      model/is-on-sale.ts       ← replaces both copies; the one home
      index.ts
  pages/
    product/
      ui/SaleBadge.tsx          ← stays; now calls isOnSale from
                                   @/entities/product
    search/
      ui/ProductCard.tsx        ← stays; calls the same isOnSale
  shared/
    api/
      product.ts                ← ProductDTO stays here
```

**What did not move.** The transport type `ProductDTO` stays in
`shared/api`. The official excessive-entities guide moves the logic into
the entity's `model` and leaves the API shape where it was (Section 5-2,
item 3). A transport type does not follow business logic into an entity
just because that logic reads it. The badge and the cards stay in their
pages: they are UI, and Section 6 warns against adding UI to entities
until there is a reason. The entity is one file and an index. That is
enough.

## Snapshot 3: an action is reused, `features/add-to-cart` appears

Between Snapshots 2 and 3 the product page gains an "Add to cart" button,
with its request and an optimistic cart update in the page's `api` and
`model` segments. Search results now need the same action.

Step 3 asks whether this is a complete user action, used in multiple
places, with a stable boundary. The button, the request, and the cart
update form one action; two pages use it; adding to the cart means the
same thing from either page. Extract it.

The request goes with the feature because it is the add-to-cart use case
itself, not a generic cart CRUD wrapper. A plain reusable cart request
would have stayed in `shared/api` however many slices called it.

```text
  features/                     ← new layer
    add-to-cart/
      ui/AddToCartButton.tsx
      api/add-to-cart.ts        ← the use case itself, not cart CRUD
      model/add-to-cart.ts      ← pending and optimistic state for it
      index.ts
  pages/
    product/
      ui/ProductPage.tsx        ← renders <AddToCartButton />
    search/
      ui/ProductCard.tsx        ← renders <AddToCartButton />
```

**What did not appear.** A `cart` entity. The state here exists only to
run the add-to-cart interaction and carries no cart-domain responsibility
anyone else could reuse, so Step 4 keeps it inside the feature. A cart a
checkout page and an order summary both had to agree on would be a
different question. A `widgets/` layer. Nothing in four snapshots needed
one, and the callout in Section 1 says not to reach for it.

## What the walkthrough shows

| Moment | Trigger | Response | Rule |
| --- | --- | --- | --- |
| 0 | Two pages | `app/`, `pages/`, `shared/` | Section 5-3 |
| 1 | Third page reads product data | No layer; `fetchProducts` moves to `shared/api` | Question 1, Question 2 |
| 2 | Same rule, two copies, one stale | `entities/product/model` | The extraction rule, Step 4 |
| 3 | Same complete action on two pages | `features/add-to-cart` | Step 3 |

Reuse alone opened no layer. A domain rule that had to stay consistent
across its consumers earned `entities`; a complete user action with one
shared behavior earned `features`. Everything else stayed where it was
used, and one request moved down without earning anything.

Once a layer exists, `references/layer-structure.md` shows the full shape
of its slices and segments. This file only shows the moment it appears.

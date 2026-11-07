# Combobox API Reference

## Table of Contents
- [useCombobox hook](#usecombobox-hook)
- [Combobox (root)](#combobox-root)
- [Sub-components](#sub-components)
- [CSS variables & Styles API](#css-variables--styles-api)
- [Default styles and overriding them](#default-styles-and-overriding-them)

---

## useCombobox hook

```tsx
const combobox = useCombobox(options?: UseComboboxOptions);
```

**Options:**
```ts
interface UseComboboxOptions {
  defaultOpened?: boolean;
  opened?: boolean;
  onOpenedChange?: (opened: boolean) => void;
  onDropdownClose?: (eventSource: 'keyboard' | 'mouse' | 'unknown') => void;
  onDropdownOpen?: (eventSource: 'keyboard' | 'mouse' | 'unknown') => void;
  // onDropdownOpen runs after the dropdown is mounted: option elements can be queried in it
  loop?: boolean;           // Default: true — keyboard nav wraps at boundaries
  scrollBehavior?: ScrollBehavior; // Default: 'instant'
}
```

**Returned store:**
```ts
interface ComboboxStore {
  // Dropdown state
  dropdownOpened: boolean;
  openDropdown(eventSource?: 'keyboard' | 'mouse' | 'unknown'): void;
  closeDropdown(eventSource?: 'keyboard' | 'mouse' | 'unknown'): void;
  toggleDropdown(eventSource?: 'keyboard' | 'mouse' | 'unknown'): void;

  // Option keyboard navigation
  selectedOptionIndex: number;
  getSelectedOptionIndex(): number;       // -1 when nothing is selected
  selectOption(index: number): void;
  selectActiveOption(): string | null;   // highlight the option with `active` prop (first option if none)
  selectFirstOption(): string | null;    // highlight the first option that is not disabled
  selectNextOption(): string | null;
  selectPreviousOption(): string | null;
  resetSelectedOption(): void;
  clickSelectedOption(): void;
  // Re-syncs the internal highlight index with the DOM. It runs in a timeout, after the next render,
  // so it can be called right before the state update that changes the list.
  // No argument ('selected'): keep the index on the highlighted option, or reset it if that option is gone.
  // 'active': move the index to the active option WITHOUT highlighting it; with scrollIntoView it scrolls to it.
  updateSelectedOptionIndex(
    target?: 'active' | 'selected' | number,
    options?: { scrollIntoView?: boolean }
  ): void;

  // Programmatic focus
  searchRef: React.RefObject<HTMLInputElement | null>;  // Combobox.Search input
  targetRef: React.RefObject<HTMLElement | null>;
  focusSearchInput(): void;
  focusTarget(): void;
}
```

"Selected" option means the option highlighted by keyboard navigation (`data-combobox-selected`),
not the option that holds the current value.

### useVirtualizedCombobox

Store for lists where only some options are in the DOM. It takes the same open/close options as
`useCombobox` plus:

```ts
useVirtualizedCombobox({
  totalOptionsCount: number;                    // number of options after filtering
  getOptionId: (index: number) => string | null; // id of the option element, for aria-activedescendant
  selectedOptionIndex: number;                  // highlighted index, kept in your state
  setSelectedOptionIndex: (index: number) => void; // called by keyboard navigation
  onSelectedOptionSubmit: (index: number) => void; // Enter on the highlighted option
  activeOptionIndex?: number;                   // index of the option that holds the value
  isOptionDisabled?: (index: number) => boolean;
})
```

The store only moves the index. You render the highlight (`selected` prop on `Combobox.Option`),
scroll the list (in `setSelectedOptionIndex`) and handle clicks (`onClick` on the option).
DOM-based helpers (`selectActiveOption`, `selectFirstOption`, `updateSelectedOptionIndex`) do not
apply: set the index in your state instead. Full example in patterns.md.

---

## Combobox (root)

```tsx
<Combobox
  store={combobox}              // required — ComboboxStore from useCombobox()
  onOptionSubmit={fn}           // (value: string, optionProps) => void
  size="sm"                     // MantineSize | string, default: 'sm'
  dropdownPadding={4}           // any CSS padding value, 4px when not set
  resetSelectionOnOptionHover   // boolean — hovering an option removes the keyboard highlight
  disabled                      // boolean — Popover prop, the dropdown cannot be opened
  readOnly                      // boolean — blocks keyboard interactions on the target only;
                                // option clicks still call onOptionSubmit, and your own
                                // toggleDropdown() in onClick still opens: guard it yourself
  floatingHeight="viewport"     // dropdown fills the available viewport height, disables flip
  // + all Popover props (position, offset, width, withinPortal, etc.)
/>
```

---

## Sub-components

### Combobox.Target
```tsx
<Combobox.Target
  targetType="input"          // 'button' | 'input', default: 'input'
  withKeyboardNavigation      // boolean, default: true
  withAriaAttributes          // boolean, default: true
  withExpandedAttribute       // boolean, default: false — adds role="combobox" and aria-expanded
  autoComplete="off"          // string
  refProp="ref"               // prop name used to pass the ref to the child
>
  {/* single child — the trigger element */}
</Combobox.Target>
```

While the dropdown is open the target element has the `data-expanded` attribute (unless `withAriaAttributes={false}`): use it to style the open state.

Use `targetType="button"` when the trigger is a button: Space and Enter open the dropdown.
With the default `input` type they do not.

### Combobox.DropdownTarget
Marks the element used for dropdown positioning when separate from the keyboard events target. Used together with `Combobox.EventsTarget` in multi-select/pills patterns.

### Combobox.EventsTarget
Receives keyboard events for dropdown navigation. Used alongside `Combobox.DropdownTarget` when the typing input is nested inside the trigger (e.g. inside `PillsInput`).

### Combobox.Dropdown
```tsx
<Combobox.Dropdown hidden={false}>
  {/* dropdown content */}
</Combobox.Dropdown>
```

Use `hidden` to hide the dropdown without unmounting it, for example when there are no options to show or the search query is empty. Prefer it to conditionally opening the store.

### Combobox.Options
```tsx
<Combobox.Options labelledBy="some-label-id">
  {/* Combobox.Option or Combobox.Group elements */}
</Combobox.Options>
```

### Combobox.Option
```tsx
<Combobox.Option
  value="react"       // string | number | boolean | bigint, required
  active={false}      // marks the option that holds the current value: sets data-combobox-active,
                      // used by selectActiveOption(). Has no styles by default — render a check icon
                      // or style [data-combobox-active] yourself
  selected={false}    // sets data-combobox-selected (keyboard highlight) manually
  disabled={false}
>
  React
</Combobox.Option>
```

### Combobox.Search
Built-in search input for the dropdown, wired to keyboard navigation. Render it before `Combobox.Options`. Focus it with `combobox.focusSearchInput()` in `onDropdownOpen`.

```tsx
<Combobox.Search
  value={search}
  onChange={(e) => setSearch(e.currentTarget.value)}
  placeholder="Search..."
  withAriaAttributes      // boolean, default: true
  withKeyboardNavigation  // boolean, default: true
  // + all Input props
/>
```

### Combobox.Empty
```tsx
<Combobox.Empty>Nothing found</Combobox.Empty>
```

### Combobox.Group
`label` accepts any React node. A group with no options inside is hidden.

```tsx
<Combobox.Group label="Frontend">
  <Combobox.Option value="react">React</Combobox.Option>
</Combobox.Group>
```

### Combobox.Header / Combobox.Footer
```tsx
<Combobox.Header>Custom header</Combobox.Header>
<Combobox.Footer>Custom footer</Combobox.Footer>
```

### Combobox.Chevron
```tsx
<Combobox.Chevron size="sm" error={error} color="blue" />
```

### Combobox.ClearButton
```tsx
<Combobox.ClearButton onClear={() => setValue(null)} />
```

Rendered with `tabIndex={-1}` and `aria-hidden`: it is a mouse-only shortcut for input triggers, where keyboard users clear the value by deleting the text. For a button trigger use `<CloseButton aria-label="Clear" />` in the `rightSection` instead, so that it is reachable by keyboard.

### Combobox.HiddenInput
```tsx
<Combobox.HiddenInput
  value={value}           // primitive, array of primitives, or null
  valuesDivider=","       // string, default: ','
  name="myField"
  form="myForm"
/>
```

---

## CSS variables & Styles API

### CSS variables (set on the `dropdown` element)
| Variable | Description |
|---|---|
| `--combobox-option-fz` | Option font size (driven by `size` prop) |
| `--combobox-option-padding` | Option padding (driven by `size` prop) |
| `--combobox-padding` | Dropdown padding (driven by `dropdownPadding`, default 4px) |
| `--combobox-floating-options-max-height` | Available height for options when `floatingHeight="viewport"` is set |

### Styles API selectors
| Selector | Element |
|---|---|
| `dropdown` | Dropdown container |
| `options` | Options listbox |
| `option` | Individual option |
| `search` | Search input |
| `empty` | Nothing found message |
| `header` | Dropdown header |
| `footer` | Dropdown footer |
| `group` | Group container |
| `groupLabel` | Group label text |

### Option data attributes
| Attribute | Meaning |
|---|---|
| `data-combobox-selected` | Option highlighted by keyboard navigation (styled by default) |
| `data-combobox-active` | Option with `active` prop (no default styles) |
| `data-combobox-disabled` | Disabled option |

---

## Default styles and overriding them

Pass `classNames` to `Combobox` to style any selector from the table above with a CSS module:
`<Combobox classNames={{ dropdown: classes.dropdown, option: classes.option }} />`.

Default styles use `:where()` selectors with zero specificity, so a plain class rule always wins.

| Element | Default styles |
|---|---|
| `dropdown` | `padding: var(--combobox-padding)` (4px). Border, background, radius and shadow come from Popover: set them with the `radius` and `shadow` props or in CSS |
| `option` | `padding: var(--combobox-option-padding)`, `font-size: var(--combobox-option-fz)`, `border-radius: var(--mantine-radius-default)`, transparent background, pointer cursor |
| `option[data-combobox-selected]` | `background-color: var(--mantine-primary-color-filled)`, white text |
| `option[data-combobox-disabled]` | `opacity: 0.35`, `cursor: not-allowed` |
| `option:hover` (not selected, not disabled) | light: `var(--mantine-color-gray-0)`, dark: `var(--mantine-color-dark-7)` |
| `option[data-combobox-active]` | no styles |
| `header`, `footer` | option padding and font size, negative inline margins that cancel the dropdown padding, 1px border on the side facing the options |
| `groupLabel` | dimmed, 85% of option font size, a line after the text |
| `empty` | centered, dimmed, option padding |

`size` (`xs` to `xl`) sets `--combobox-option-fz` to the theme font size and `--combobox-option-padding`
to `4px 8px`, `6px 10px`, `8px 12px`, `10px 16px`, `14px 20px`. Both variables are inherited, so
content inside an option can use them.

Popover props used most with a custom look: `width` (`'target'` by default, or a number),
`position` (`'bottom-start'`), `offset` (gap in px), `radius`, `shadow`,
`transitionProps={{ transition: 'pop-top-left', duration: 150 }}`, `withArrow`.

---
name: mantine-combobox
description: >
  Build custom dropdown/select/autocomplete/multiselect components using Mantine's Combobox
  primitives. Use this skill when: (1) creating a new custom select-like component with
  Combobox primitives, (2) building a searchable dropdown, (3) implementing a multi-select
  or tags input variant, (4) customizing option rendering, (5) adding custom filtering logic,
  or (6) any task involving useCombobox, Combobox.Target, Combobox.Option, or Combobox.Dropdown.
---

# Mantine Combobox Skill

Written for Mantine 9.x.

## Overview

`Combobox` provides low-level primitives for building any select-like UI. The built-in
`Select`, `MultiSelect`, `Autocomplete`, and `TagsInput` components are all built on top of it.

## Check for a ready-made component first

Build with `Combobox` primitives only when none of these fit:

| Need | Use |
|---|---|
| Standard select, multi-select, autocomplete or tags input | `Select`, `MultiSelect`, `Autocomplete`, `TagsInput` (all support `renderOption`) |
| Options dropdown attached to a button or any other element, no input | `ComboboxPopover` (data-driven: `data`, `value`, `onChange`, `searchable`, `multiple`) |
| Hierarchical options | `TreeSelect` (see below) |
| More than a few hundred options | `Combobox` with `useVirtualizedCombobox`, see the virtualized pattern |

```tsx
<ComboboxPopover data={['React', 'Angular', 'Vue']} value={value} onChange={setValue}>
  <ComboboxPopover.Target>
    <Button variant="default">{value || 'Select framework'}</Button>
  </ComboboxPopover.Target>
</ComboboxPopover>
```

`ComboboxPopover` opens on target click by itself, marks the selected option with a check icon and
handles keyboard navigation. Useful props: `searchable`, `multiple` (value becomes `string[]`, the
dropdown stays open while picking), `allowDeselect={false}` (one option is always selected),
`nothingFoundMessage`, `renderOption`, `limit`, `maxDropdownHeight`, `name` (hidden input for forms),
`comboboxProps={{ width: 220, position: 'bottom-start' }}`. `data` supports `disabled` items and groups
like `Select`: `[{ group: 'Size', items: [{ value: 'xs', label: 'XS' }] }]`. It cannot render custom dropdown content (tabs, header, footer, create actions): use
`Combobox` primitives for that.

`TreeSelect` takes nested `data` (`{ value, label, children? }[]`, unique values) and renders an
input with an expandable tree in the dropdown. Props include `mode` (single, multiple, or checkbox
with parent-child cascade), `searchable`, `clearable`, `renderNode`. Build the tree from `Combobox`
primitives instead when you need a different layout: only leaves selectable, a flat list of search
results with full paths, or a button trigger. In that case render every visible row as a
`Combobox.Option` and decide in `onOptionSubmit` whether the row selects or expands.

## Core Workflow

### 1. Create the store

```tsx
const combobox = useCombobox({
  onDropdownClose: () => combobox.resetSelectedOption(),
  onDropdownOpen: () => combobox.selectFirstOption(),
});
```

### 2. Render structure

```tsx
<Combobox store={combobox} onOptionSubmit={handleSubmit}>
  <Combobox.Target targetType="button">
    <InputBase
      component="button"
      type="button"
      pointer
      rightSection={<Combobox.Chevron />}
      onClick={() => combobox.toggleDropdown()}
    >
      {value || <Input.Placeholder>Pick value</Input.Placeholder>}
    </InputBase>
  </Combobox.Target>
  <Combobox.Dropdown>
    <Combobox.Options>
      {options.map((item) => (
        <Combobox.Option value={item} key={item}>{item}</Combobox.Option>
      ))}
    </Combobox.Options>
  </Combobox.Dropdown>
</Combobox>
```

### 3. Handle submit

```tsx
const handleSubmit = (val: string) => {
  setValue(val);
  combobox.closeDropdown();
};
```

## Target Types

| Scenario | Use |
|---|---|
| Button trigger (no text input) | `<Combobox.Target targetType="button">` |
| Input trigger | `<Combobox.Target>` (default) |
| Pills + separate input (multi-select) | `<Combobox.DropdownTarget>` + `<Combobox.EventsTarget>` |

## What Combobox does for you

- The target handles keyboard: ArrowUp/ArrowDown move the highlight and skip disabled options (and
  open the dropdown when it is closed), Enter submits the highlighted option, Escape closes the
  dropdown. With `targetType="button"`, Space and Enter also open it. Tab is not handled.
- Enter does nothing when no option is highlighted, so a free-text input can handle it itself.
- Nothing is highlighted until the user presses an arrow key, unless you call `selectFirstOption()`
  or `selectActiveOption()`.
- Clicking an option does not move focus away from the input, and a disabled option never calls
  `onOptionSubmit`.
- `Combobox.Option` forwards `data-*` attributes and event handlers to its element.
- `Combobox.Search` handles the same keys as the target. Your own `onKeyDown` on the target child or
  on `Combobox.Search` is called first, then the built-in handling runs, so extra keys (PageUp,
  ArrowRight) can be added there. To take the keyboard over completely (a textarea, where arrows
  must move the caret), set `withKeyboardNavigation={false}` on `Combobox.Target` and call
  `selectNextOption`, `selectPreviousOption` and `clickSelectedOption` yourself.
- Defaults: the dropdown is as wide as the target (`width="target"`), rendered in a portal, kept
  mounted while closed, opens without animation, and hovering an option does not change the
  keyboard highlight.
- The dropdown renders in a portal above modals and drawers and is not clipped by them. Inside a
  `Modal`, Escape closes only the dropdown; a second Escape closes the modal. No extra code is needed.
- An option does not have to select a value: `onOptionSubmit` can toggle, expand a row or run an
  action, and decides itself whether to close the dropdown.
- Other interactive content in the dropdown (buttons in `Combobox.Header`, a Retry link) is not
  handled: add `onMouseDown={(event) => event.preventDefault()}` to it so the input keeps focus and
  an `onBlur` handler does not close the dropdown.

Two words that are easy to mix up:

- **selected** option: the one highlighted by keyboard navigation (`data-combobox-selected`).
- **active** option: the one that holds the current value, marked with the `active` prop
  (`data-combobox-active`). It has no styles by default: render a check icon.

## References

Read both before building anything beyond the basic select above:

- **[`references/patterns.md`](references/patterns.md)** — complete examples: searchable select (input trigger), free-text input with suggestions, suggestions for a textarea (mentions), custom styles with a CSS module, virtualized list for thousands of options, custom form input built on Combobox, search inside the dropdown, multi-select with pills, creatable option, async search with loading and error states, groups, custom option rendering, highlighting the current value on open, clear button, form integration, dropdown that fits the viewport
- **[`references/api.md`](references/api.md)** — `useCombobox` options and every store method with what it does, default styles and how to override them, `useVirtualizedCombobox`, all sub-component props, CSS variables, Styles API selectors

## Looking things up

If the references do not cover what you need, do not guess:

- If the Mantine MCP server (`@mantine/mcp-server`) is connected, use `search_docs` and `get_item_doc`
- Otherwise fetch `https://mantine.dev/llms.txt` and open the Combobox page. More than 50 complete examples are at `https://mantine.dev/combobox/`


---
name: mantine-form
description: >
  Build forms using @mantine/form. Use this skill when: (1) setting up a form with useForm,
  (2) adding validation rules, schema validation (zod, valibot, arktype) or async validation,
  (3) working with nested object or array fields, (4) sharing form state across components
  with createFormContext, (5) choosing between controlled and uncontrolled mode, (6) reading
  form values during render with form.useWatchValue, (7) using the standalone useField hook, or
  (8) any task involving useForm, getInputProps, onSubmit, insertListItem, or form validation.
---

# Mantine Form Skill

Written for Mantine 9.x.

## Core Workflow

### 1. Set up the form

Uncontrolled mode is the recommended mode for all forms. The default is `'controlled'`, so set it explicitly.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: {
    email: '',
    age: 0,
  },
  validate: {
    email: isEmail('Invalid email'),
    age: isInRange({ min: 18 }, 'Must be at least 18'),
  },
});
```

### 2. Wire inputs with getInputProps

In uncontrolled mode every input needs `key={form.key('path')}`. Without it the input does not
update after `form.setFieldValue`, `form.setValues` or `form.reset`.

```tsx
<TextInput label="Email" key={form.key('email')} {...form.getInputProps('email')} />
<NumberInput label="Age" key={form.key('age')} {...form.getInputProps('age')} />
```

Checkboxes and switches need `{ type: 'checkbox' }`:

```tsx
<Checkbox
  label="I agree"
  key={form.key('agreed')}
  {...form.getInputProps('agreed', { type: 'checkbox' })}
/>
```

`Select`, `NumberInput`, `Radio.Group`, `Checkbox.Group` and other Mantine inputs with a `value` /
`onChange` pair work with plain `getInputProps` and a `key`:

```tsx
<Radio.Group label="Delivery" key={form.key('delivery')} {...form.getInputProps('delivery')}>
  <Radio value="standard" label="Standard" />
  <Radio value="express" label="Express" />
</Radio.Group>
```

For standalone radios without `Radio.Group`, use `form.getInputProps('color', { type: 'radio', value: 'red' })`
and add `key={form.key('color')}` to each radio.

Components without an `error` prop (`Slider`, `RangeSlider`, `Rating`, `SegmentedControl`, `Chip.Group`)
need `{ withError: false }`, and you render the error yourself:

```tsx
<Slider key={form.key('stock')} {...form.getInputProps('stock', { withError: false })} />
{form.errors.stock && <Text c="red" size="sm">{form.errors.stock}</Text>}
```

### 3. Handle submission

```tsx
<form onSubmit={form.onSubmit((values) => console.log(values))}>
  ...
  <Button type="submit">Submit</Button>
</form>
```

`onSubmit` only calls the handler when validation passes. If the handler returns a promise,
`form.submitting` is `true` until it settles. To handle failures:

```tsx
form.onSubmit(
  (values, event) => save(values),
  (errors, values, event) => console.log('Validation failed', errors) // { 'user.email': 'Invalid email' }
);
```

## Validation

### Rules object (most common)

```tsx
validate: {
  name: isNotEmpty('Required'),
  email: isEmail('Invalid email'),
  password: hasLength({ min: 8 }, 'Min 8 chars'),
  confirmPassword: matchesField('password', 'Passwords do not match'),
}
```

### Schema (zod, valibot, arktype and other Standard Schema libraries)

```tsx
import { z } from 'zod/v4';
import { schemaResolver, useForm } from '@mantine/form';

const schema = z.object({
  email: z.email({ error: 'Invalid email' }),
  age: z.number().min(18, { error: 'Must be at least 18' }),
});

const form = useForm({
  mode: 'uncontrolled',
  initialValues: { email: '', age: 0 },
  validate: schemaResolver(schema, { sync: true }),
});
```

`schemaResolver` is built in: no resolver package is needed. Pass `{ sync: true }` for
synchronous schemas so that `form.validate()` returns a plain result instead of a `Promise`.

### Function (for cross-field logic)

```tsx
validate: (values) => ({
  endDate: values.endDate < values.startDate ? 'End must be after start' : null,
});
```

### When to validate

By default fields are validated on submit only.

```tsx
validateInputOnChange: true,            // also validate every field when it changes
validateInputOnChange: ['email'],       // only the listed fields
validateInputOnBlur: ['email'],         // same options, on blur
validateInputOnBlur: [`members.${FORM_INDEX}.email`], // list items, FORM_INDEX is exported from @mantine/form
```

With validation on blur, an error that appears when the user presses the submit button shifts the
layout, and the click can miss the button. Keep the submit button in a place that does not move
when errors appear (for example a footer with fixed position), or validate on change instead.

## Modes

|                                  | `'uncontrolled'` (recommended)      | `'controlled'` (default) |
| -------------------------------- | ----------------------------------- | ------------------------ |
| Values storage                   | Ref                                 | React state              |
| `form.values`                    | Not updated, use `form.getValues()` | Updated on every change  |
| Re-renders on value change       | No                                  | Yes                      |
| Input props                      | `defaultValue` + `onChange`         | `value` + `onChange`     |
| `key={form.key(path)}` on inputs | Required                            | Not needed               |

## What rerenders in uncontrolled mode

- Typing in an input does not rerender the form.
- `form.getValues()` during render is not updated by typing. To show or hide part of the form
  based on a value, use `form.useWatchValue(path)`. Nested paths work: `form.useWatchValue('members.0.role')`.
- `setFieldValue`, `setValues`, `insertListItem`, `removeListItem`, `reorderListItem`, `reset` and
  `initialize` do rerender, so rendering a list from `form.getValues().items.map(...)` is correct.
- `form.errors`, `form.isDirty()`, `form.isDirty('path')`, `form.isTouched()`, `form.submitting` and
  `form.validating` can be used during render in both modes: the form rerenders when they change.

```tsx
const shipsInternationally = form.useWatchValue('shipsInternationally');
```

## References

For anything beyond a basic form, read both references before writing code:

- **[`references/patterns.md`](references/patterns.md)** — read for: reusing one form for different records (edit dialogs), focusing the first invalid field, more than one submit button, changing a value while the user types, nested objects, array fields, lists inside lists and `formRootRule`, async validation (rules and async schemas), conditional fields, conditional validation, multi-step forms, loading initial values from a server, saving and setting a new baseline, controlling a form from outside its component, custom inputs, form context across components, `transformValues`, standalone `useField`, server errors after submission
- **[`references/api.md`](references/api.md)** — read for everything else. It is the only place that lists every `useForm` option and return member (`watch`, `onValuesChange`, `enhanceGetInputProps`, `resetField`, `clearFieldError`, `touchTrigger` and others), plus `useField`, `createFormContext`, `createFormActions`, `schemaResolver`, built-in validators and types

## Looking things up

If the references do not cover what you need, do not guess:

- If the Mantine MCP server (`@mantine/mcp-server`) is connected, use `search_docs` and `get_item_doc`
- Otherwise fetch `https://mantine.dev/llms.txt` and open the linked form pages

# @mantine/form Patterns

## Table of Contents
- [Basic form with validation](#basic-form-with-validation)
- [Nested object fields](#nested-object-fields)
- [Array / list fields](#array--list-fields)
- [Async validation](#async-validation)
- [Conditional fields](#conditional-fields)
- [Conditional validation](#conditional-validation)
- [Multi-step form](#multi-step-form)
- [Loading initial values](#loading-initial-values)
- [Saving and new baseline](#saving-and-new-baseline)
- [Reusing one form for different records](#reusing-one-form-for-different-records)
- [Focusing the first invalid field](#focusing-the-first-invalid-field)
- [Several submit buttons](#several-submit-buttons)
- [Changing a value while the user types](#changing-a-value-while-the-user-types)
- [Setting sibling fields from one field](#setting-sibling-fields-from-one-field)
- [Controlling a form from outside](#controlling-a-form-from-outside)
- [Custom inputs](#custom-inputs)
- [Form context across components](#form-context-across-components)
- [transformValues](#transformvalues)
- [Uncontrolled mode](#uncontrolled-mode)
- [Standalone useField](#standalone-usefield)
- [Server errors after submission](#server-errors-after-submission)

---

## Basic form with validation

```tsx
import { useForm, isEmail, isNotEmpty, hasLength } from '@mantine/form';

function BasicForm() {
  const form = useForm({
    mode: 'uncontrolled',
    initialValues: { name: '', email: '', password: '' },
    validate: {
      name: isNotEmpty('Name is required'),
      email: isEmail('Invalid email'),
      password: hasLength({ min: 8 }, 'Password must be at least 8 characters'),
    },
  });

  return (
    <form onSubmit={form.onSubmit((values) => console.log(values))}>
      <TextInput label="Name" key={form.key('name')} {...form.getInputProps('name')} />
      <TextInput label="Email" key={form.key('email')} {...form.getInputProps('email')} />
      <PasswordInput label="Password" key={form.key('password')} {...form.getInputProps('password')} />
      <Button type="submit">Submit</Button>
    </form>
  );
}
```

---

## Nested object fields

Use dot notation to address nested fields.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: {
    user: {
      name: '',
      address: {
        city: '',
        zip: '',
      },
    },
  },
  validate: {
    user: {
      name: isNotEmpty('Required'),
      address: {
        city: isNotEmpty('City is required'),
        zip: matches(/^\d{5}$/, 'Must be 5 digits'),
      },
    },
  },
});

// Access nested fields with dot notation
<TextInput key={form.key('user.name')} {...form.getInputProps('user.name')} />
<TextInput key={form.key('user.address.city')} {...form.getInputProps('user.address.city')} />
<TextInput key={form.key('user.address.zip')} {...form.getInputProps('user.address.zip')} />
```

---

## Array / list fields

Give every item a stable `key` value (`randomId` from `@mantine/hooks`) and use it as the React key
of the row. Use `formRootRule` to validate the list itself next to its items.

```tsx
import { formRootRule, isNotEmpty, useForm } from '@mantine/form';
import { randomId } from '@mantine/hooks';

const form = useForm({
  mode: 'uncontrolled',
  initialValues: {
    employees: [{ name: '', role: '', key: randomId() }],
  },
  validate: {
    employees: {
      [formRootRule]: isNotEmpty('At least one employee is required'),
      name: isNotEmpty('Name required'),
      role: isNotEmpty('Role required'),
    },
  },
});

// Render the list
const fields = form.getValues().employees.map((item, index) => (
  <Group key={item.key}>
    <TextInput
      placeholder="Name"
      key={form.key(`employees.${index}.name`)}
      {...form.getInputProps(`employees.${index}.name`)}
    />
    <TextInput
      placeholder="Role"
      key={form.key(`employees.${index}.role`)}
      {...form.getInputProps(`employees.${index}.role`)}
    />
    <ActionIcon aria-label="Remove employee" onClick={() => form.removeListItem('employees', index)}>
      <IconTrash />
    </ActionIcon>
  </Group>
));

return (
  <form onSubmit={form.onSubmit((values) => console.log(values))}>
    {fields}
    {form.errors.employees && <Text c="red" size="sm">{form.errors.employees}</Text>}
    <Button
      onClick={() => form.insertListItem('employees', { name: '', role: '', key: randomId() })}
    >
      Add employee
    </Button>
    <Button type="submit">Submit</Button>
  </form>
);
```

**List methods:**
```tsx
form.insertListItem('employees', { name: '', role: '', key: randomId() });     // append
form.insertListItem('employees', { name: '', role: '', key: randomId() }, 0);  // prepend
form.removeListItem('employees', index);
form.reorderListItem('employees', { from: 2, to: 0 });
form.replaceListItem('employees', index, { name: 'New', role: 'Dev', key: randomId() });
```

List methods rerender the component in both modes, and field errors move with their rows.

**Rules that depend on other items** get the item path as the third argument (`'employees.1.email'`):

```tsx
validate: {
  employees: {
    email: (value, values, path) => {
      const index = Number(path.split('.')[1]);
      const isDuplicate = values.employees.some((item, i) => i < index && item.email === value);
      return isDuplicate ? 'Duplicate email' : null;
    },
  },
},
```

---

## Async validation

Return a `Promise` from any validator. Use the provided `AbortSignal` to avoid stale results.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: { username: '' },
  validate: {
    username: async (value, _values, _path, signal) => {
      if (!value) return 'Username is required';
      const response = await fetch(`/api/check-username?q=${value}`, { signal });
      if (signal?.aborted) return null;
      const { taken } = await response.json();
      return taken ? 'Username is already taken' : null;
    },
  },
  validateInputOnChange: ['username'],
  validateDebounce: 500,   // debounce on-change validation
});
```

A schema can be async too. Use `schemaResolver(schema)` without `{ sync: true }`:

```tsx
const schema = z.object({
  discountCode: z.string().refine(async (code) => code === '' || (await checkCode(code)), {
    error: 'Unknown discount code',
  }),
});

const form = useForm({
  mode: 'uncontrolled',
  initialValues: { discountCode: '' },
  validate: schemaResolver(schema),
});

// Validate one field and act on the result, for example on blur
const { hasError } = await form.validateField('discountCode');
```

`form.submitting` is `true` during async validation on submit as well, so `loading={form.submitting}`
on the submit button covers the whole submit. `form.validating` is `true` while any async validation runs, `form.isValidating('username')` checks one field.
With async rules, `form.validate()` and `form.isValid()` return a `Promise`.

---

## Conditional fields

Use `form.useWatchValue` to read a value during render. `form.getValues()` does not rerender the
component in uncontrolled mode.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: { hasCompany: false, companyName: '' },
});

const hasCompany = form.useWatchValue('hasCompany');

<Checkbox
  label="I represent a company"
  key={form.key('hasCompany')}
  {...form.getInputProps('hasCompany', { type: 'checkbox' })}
/>
{hasCompany && (
  <TextInput
    label="Company name"
    key={form.key('companyName')}
    {...form.getInputProps('companyName')}
  />
)}
```

---

## Conditional validation

Hidden fields keep their values and are still validated. Make the rule depend on the value that
controls visibility.

```tsx
// Rules object: the second argument is all form values
validate: {
  companyName: (value, values) =>
    values.hasCompany && value.trim().length === 0 ? 'Company name is required' : null,
}

// zod: add issues with the field path, the path becomes the error key ('billing.zip')
const schema = z
  .object({ sameAsShipping: z.boolean(), billing: z.object({ zip: z.string() }) })
  .superRefine((values, ctx) => {
    if (!values.sameAsShipping && !/^\d{5}$/.test(values.billing.zip)) {
      ctx.addIssue({ code: 'custom', path: ['billing', 'zip'], message: 'ZIP must be 5 digits' });
    }
  });
```

---

## Multi-step form

Validate only the fields of the current step with `validateField`. It sets the field error and
returns `{ hasError }`. Await the results: with async rules they are promises.

```tsx
const stepFields = [
  ['username', 'password'],
  ['fullName', 'country'],
] as const;

const handleNext = async () => {
  const results = await Promise.all(stepFields[active].map((path) => form.validateField(path)));
  if (results.every((result) => !result.hasError)) {
    setActive((current) => current + 1);
  }
};
```

Values of unmounted steps are kept in the form. Give each step component access to the form with
[form context](#form-context-across-components).

---

## Loading initial values

Call `form.initialize` when the data arrives. It sets both values and initial values, so the form
is not dirty afterwards and `form.reset()` returns to the loaded values. It works once.

```tsx
const form = useForm({ mode: 'uncontrolled', initialValues: { name: '', country: '' } });

useEffect(() => {
  if (query.data) {
    form.initialize(query.data);
  }
}, [query.data]);

<Button disabled={!form.isDirty()} onClick={form.reset}>Discard changes</Button>
```

---

## Saving and new baseline

After a successful save, make the saved values the new initial values so that `form.isDirty()` is
`false` and `form.reset()` returns to them:

```tsx
const handleSubmit = async (values: typeof form.values) => {
  await save(values);
  form.setInitialValues(values);
  form.resetDirty(values);
};
```

Per-field state: `form.isDirty('email')` during render, `form.resetField('email')` to undo one field.

---

## Reusing one form for different records

`initialize` works once. For a form that stays mounted and is opened for different records (an
edit dialog over a table), set new initial values and reset. This replaces values and clears
errors, touched and dirty state from the previous record:

```tsx
const openFor = (user: UserValues) => {
  form.setInitialValues(user);
  form.reset();
  setOpened(true);
};

<Button type="submit" disabled={!form.isDirty()}>Save</Button>
```

`form.isDirty()` compares with the values the form was opened with, so it is `false` again when
a change is reverted.

---

## Focusing the first invalid field

Use the second argument of `form.onSubmit` and `form.getInputNode`. Errors are keyed by path in no
particular order, so keep your own list of fields in visual order:

```tsx
const FIELDS_IN_ORDER = ['firstName', 'lastName', 'email', 'address.city'] as const;

const handleErrors = (errors: FormErrors) => {
  const firstInvalid = FIELDS_IN_ORDER.find((path) => errors[path]);
  if (firstInvalid) {
    form.getInputNode(firstInvalid)?.focus();
  }
};

<form onSubmit={form.onSubmit(handleSubmit, handleErrors)}>
```

Call the same function after `form.setErrors(serverErrors)` to focus the first field rejected by a server.

---

## Several submit buttons

For a second action that validates less ("Save draft"), validate the fields it needs and read the
values yourself instead of going through `form.onSubmit`:

```tsx
const handleSaveDraft = () => {
  const { hasError } = form.validateField('name');
  if (!hasError) {
    saveDraft(form.getTransformedValues());
  }
};

<Button type="submit">Publish</Button>
<Button variant="default" onClick={handleSaveDraft}>Save draft</Button>
```

---

## Changing a value while the user types

To transform what the user types (uppercase, strip characters), control that one input with
`useWatchValue` and `setFieldValue`. Do not add `key` to it: `setFieldValue` changes the key, which
would remount the input and lose focus.

```tsx
const code = form.useWatchValue('warehouseCode');

<PinInput
  value={code}
  onChange={(value) => form.setFieldValue('warehouseCode', value.toUpperCase())}
  error={!!form.errors.warehouseCode}
/>
```

---

## Setting sibling fields from one field

Spread `getInputProps`, then override `onChange`, call the original and set the other fields:

```tsx
const productProps = form.getInputProps(`lines.${index}.productId`);

<Select
  data={products}
  key={form.key(`lines.${index}.productId`)}
  {...productProps}
  onChange={(value) => {
    productProps.onChange(value);
    const product = catalogue.find((item) => item.id === value);
    if (product) {
      form.setFieldValue(`lines.${index}.unitPrice`, product.price);
    }
  }}
/>
```

`setFieldValue` updates the other input (its `key` changes) and clears its error.

---

## Controlling a form from outside

A toolbar or another component that is not a child of the form can control it with
`createFormActions`. The form opts in with `name`.

```tsx
// settings-form.ts
export const settingsFormActions = createFormActions<SettingsValues>('settings-form');

// SettingsForm.tsx
const form = useForm({ name: 'settings-form', mode: 'uncontrolled', initialValues });
<form id="settings-form" onSubmit={form.onSubmit(handleSubmit)}>...</form>

// Toolbar.tsx, rendered anywhere on the page
<Button onClick={() => settingsFormActions.reset()}>Reset</Button>
<Button onClick={() => settingsFormActions.setValues(demoValues)}>Load demo data</Button>
<Button onClick={() => settingsFormActions.clearErrors()}>Clear errors</Button>
<Button type="submit" form="settings-form">Save</Button>
```

To change every input at once (for example disable them while saving), use `enhanceGetInputProps`:

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues,
  enhanceGetInputProps: ({ form }) => ({ disabled: form.submitting }),
});
```

---

## Custom inputs

A component works with `form.getInputProps` when it accepts `value`, `defaultValue`, `onChange`,
`error`, `onFocus` and `onBlur`. `onChange` can be called with a raw value. Use `useUncontrolled`
from `@mantine/hooks` so that it works in both form modes; `key={form.key(path)}` remounts it on
`form.reset()`.

```tsx
import { ActionIcon, Group, Input, Text } from '@mantine/core';
import { useUncontrolled } from '@mantine/hooks';

interface StepperInputProps {
  label?: React.ReactNode;
  error?: React.ReactNode;
  value?: number;
  defaultValue?: number;
  onChange?: (value: number) => void;
  onFocus?: React.FocusEventHandler;
  onBlur?: React.FocusEventHandler;
}

function StepperInput({ label, error, value, defaultValue, onChange, onFocus, onBlur }: StepperInputProps) {
  const [current, setCurrent] = useUncontrolled({ value, defaultValue, finalValue: 1, onChange });

  return (
    <Input.Wrapper label={label} error={error} labelElement="div" onFocus={onFocus} onBlur={onBlur}>
      <Group gap="xs">
        <ActionIcon aria-label="Decrease" onClick={() => setCurrent(current - 1)}>−</ActionIcon>
        <Text>{current}</Text>
        <ActionIcon aria-label="Increase" onClick={() => setCurrent(current + 1)}>+</ActionIcon>
      </Group>
    </Input.Wrapper>
  );
}

<StepperInput label="Guests" key={form.key('guests')} {...form.getInputProps('guests')} />
```

- `getInputProps` also returns `data-path`. Spread the remaining props onto the focusable element
  of the input if `form.getInputNode(path)?.focus()` should work for it.
- `onBlur` on a wrapper fires whenever focus moves between elements inside it. For an input made of
  several focusable parts (a trigger and a dropdown search), call `onBlur` only when
  `event.relatedTarget` is outside the component.
- A field can hold an object (`{ country, number }`). A rule on that field receives the object and
  its error is stored at the field path (`form.errors.phone`). Update it as a whole with
  `onChange({ ...value, number })`.
- `required` on a Mantine input adds the native attribute, which makes the browser block submit
  before `form.onSubmit` runs. Use `withAsterisk` for the visual mark and validate in the form.

---

## Form context across components

Share one form instance across a component tree without prop drilling.

```tsx
// 1. Create typed context once
import { createFormContext, isNotEmpty } from '@mantine/form';

interface ProfileValues {
  bio: string;
  website: string;
}

const [FormProvider, useFormContext, useProfileForm] = createFormContext<ProfileValues>();
// With transformValues: createFormContext<ProfileValues, TransformedProfileValues>()

// 2. Wrap your form tree with FormProvider
function ProfileForm() {
  const form = useProfileForm({
    mode: 'uncontrolled',
    initialValues: { bio: '', website: '' },
    validate: {
      bio: isNotEmpty('Bio is required'),
    },
  });

  return (
    <FormProvider form={form}>
      <form onSubmit={form.onSubmit((values) => save(values))}>
        <BioField />
        <WebsiteField />
        <Button type="submit">Save</Button>
      </form>
    </FormProvider>
  );
}

// 3. Access form in any child — no prop drilling
function BioField() {
  const form = useFormContext();
  return <Textarea label="Bio" key={form.key('bio')} {...form.getInputProps('bio')} />;
}

function WebsiteField() {
  const form = useFormContext();
  return <TextInput label="Website" key={form.key('website')} {...form.getInputProps('website')} />;
}
```

---

## transformValues

Shape the values before they reach `onSubmit`. The transform is applied transparently — `onSubmit` receives `TransformedValues`, not `Values`.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: {
    price: '',        // stored as string in input
    tags: 'a, b, c', // stored as comma-separated string
  },
  transformValues: (values) => ({
    price: Number(values.price),
    tags: values.tags.split(',').map((t) => t.trim()),
  }),
});

// handler receives { price: number, tags: string[] }
form.onSubmit((values) => console.log(values));

// Type for a handler declared separately
type Payload = TransformedValues<typeof form>; // import type { TransformedValues } from '@mantine/form'
const handleSubmit = (values: Payload) => save(values);
```

Validation runs on the raw values, the transform is applied after it passes.

---

## Uncontrolled mode

The recommended mode for all forms. Values are stored in a ref, so typing does not rerender the form.

```tsx
const form = useForm({
  mode: 'uncontrolled',
  initialValues: { name: '', email: '' },
  validate: { email: isEmail('Invalid email') },
});

// Every input needs key={form.key(path)}: inputs receive defaultValue, and the key is what
// updates them after form.setFieldValue, form.setValues and form.reset
<TextInput label="Name" key={form.key('name')} {...form.getInputProps('name')} />
<TextInput label="Email" key={form.key('email')} {...form.getInputProps('email')} />

// Read current values in event handlers:
const current = form.getValues();
```

- `form.values` is not updated in uncontrolled mode. Use `form.getValues()` in handlers and `form.useWatchValue(path)` during render.
- `form.errors`, `form.submitting` and `form.validating` are React state in both modes and can be used during render.

---

## Standalone useField

Manage a single field without a full form — useful for isolated inputs or custom field components.

```tsx
import { useField, isEmail } from '@mantine/form';

function EmailField() {
  const field = useField({
    initialValue: '',
    validate: isEmail('Invalid email'),
    validateOnBlur: true,
  });

  return (
    <TextInput label="Email" {...field.getInputProps()} />
  );
}
```

---

## Server errors after submission

Set server-side errors on fields after a failed API call. Keys are dot paths, so list items work
too (`'lines.2.quantity'`). A server error is cleared when the user changes that field. For an
error that does not belong to a field, keep it in your own state.

```tsx
const form = useForm({ mode: 'uncontrolled', initialValues: { email: '', password: '' } });

const handleSubmit = async (values: typeof form.values) => {
  try {
    await login(values);
  } catch (error) {
    if (error instanceof ApiValidationError) {
      // Map server field errors onto form: { email: 'Already registered' }
      form.setErrors(error.fields);
    } else {
      form.setFieldError('password', 'Invalid email or password');
    }
  }
};

<form onSubmit={form.onSubmit(handleSubmit)}>
  <TextInput label="Email" key={form.key('email')} {...form.getInputProps('email')} />
  <PasswordInput label="Password" key={form.key('password')} {...form.getInputProps('password')} />
  <Button type="submit" loading={form.submitting}>Sign in</Button>
</form>
```

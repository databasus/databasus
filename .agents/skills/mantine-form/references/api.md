# @mantine/form API Reference

## Table of Contents
- [useForm options](#useform-options)
- [useForm return value](#useform-return-value)
- [useField](#usefield-hook)
- [createFormContext](#createformcontext)
- [createFormActions](#createformactions)
- [schemaResolver](#schemaresolver)
- [Built-in validators](#built-in-validators)
- [Validation rule format](#validation-rule-format)
- [Key types](#key-types)

---

## useForm options

`Values` is the type of form values, `TransformedValues` is the type returned by `transformValues`
(what the `onSubmit` handler receives). Both are inferred from `initialValues` and `transformValues`,
so explicit generics are rarely needed.

```ts
useForm<Values, TransformedValues>({
  mode?,
  initialValues?,
  initialErrors?,
  initialTouched?,
  initialDirty?,
  validate?,
  validateInputOnChange?,
  validateInputOnBlur?,
  clearInputErrorOnChange?,
  transformValues?,
  onValuesChange?,
  enhanceGetInputProps?,
  onSubmitPreventDefault?,
  touchTrigger?,
  cascadeUpdates?,
  validateDebounce?,
  resolveValidationError?,
  name?,
})
```

| Option | Type | Default | Description |
|---|---|---|---|
| `mode` | `'controlled' \| 'uncontrolled'` | `'controlled'` | Controlled: values in React state, re-render on every change. Uncontrolled (recommended): values in a ref, no re-render on value change. |
| `initialValues` | `Values` | — | Starting values for all fields |
| `initialErrors` | `FormErrors` | `{}` | Starting error messages |
| `initialTouched` | `FormStatus` | `{}` | Starting touched flags |
| `initialDirty` | `FormStatus` | `{}` | Starting dirty flags |
| `validate` | `FormRulesRecord<Values> \| (values) => FormErrors \| Promise<FormErrors>` | — | Rules object, function, or `schemaResolver(schema)` |
| `validateInputOnChange` | `boolean \| string[]` | `false` | Validate on change (all or named fields). Use `FORM_INDEX` for list items: `` `jobs.${FORM_INDEX}.title` `` |
| `validateInputOnBlur` | `boolean \| string[]` | `false` | Validate on blur (all or named fields), supports `FORM_INDEX` |
| `clearInputErrorOnChange` | `boolean` | `true` | Clear field error when its value changes. Applies to errors set with `setErrors` and `setFieldError` as well |
| `transformValues` | `(values: Values) => TransformedValues` | identity | Transform values before they reach `onSubmit` handler |
| `onValuesChange` | `(values, previous) => void` | — | Called whenever any value changes |
| `enhanceGetInputProps` | `({ inputProps, field, options, form }) => object \| void` | — | Merge extra props into every `getInputProps` call, for example `({ form }) => ({ disabled: form.submitting })` |
| `onSubmitPreventDefault` | `'always' \| 'never' \| 'validation-failed'` | `'always'` | When to call `event.preventDefault()` |
| `touchTrigger` | `'focus' \| 'change'` | `'change'` | When a field becomes touched |
| `cascadeUpdates` | `boolean` | `false` | `form.watch` subscribers of child paths are also called when a parent path is set |
| `validateDebounce` | `number` | `0` | Debounce on-change and on-blur field validation (ms). Does not apply to `form.validate()` and `onSubmit` |
| `resolveValidationError` | `(error: unknown) => ReactNode` | message extractor | Transform raw validation errors to display values |
| `name` | `string` | — | Form name for `createFormActions` event bus |

---

## useForm return value

### Values

| Member | Type | Description |
|---|---|---|
| `values` | `Values` | Current form values. Not updated in uncontrolled mode, use `getValues()` |
| `errors` | `FormErrors` | Current errors keyed by dot path (`'members.1.email'`). React state in both modes. Contains only fields that have an error: cleared errors are removed, not set to `null` |
| `submitting` | `boolean` | True from the moment the form is submitted until validation (including async) and the promise returned by the `onSubmit` handler have settled |
| `validating` | `boolean` | True while async validation started by `validate`, `validateField`, submit, or on-change/on-blur validation is running. Not set by `isValid()` |
| `initialized` | `boolean` | True after `initialize()` has been called |

### Getting & setting values

| Method | Signature | Description |
|---|---|---|
| `getValues` | `() => Values` | Get current values snapshot |
| `getInitialValues` | `() => Values` | Get initial values snapshot |
| `setValues` | `(values: Partial<Values> \| (prev) => Partial<Values>) => void` | Merge partial values into form state. Does not clear errors, call `clearErrors()` if needed |
| `setFieldValue` | `(path, value \| updater) => void` | Set a single field by dot-path |
| `setInitialValues` | `(values: Values) => void` | Update the initial values that `reset` returns to. Does not change dirty state, call `resetDirty(values)` as well |
| `initialize` | `(values: Values) => void` | Set values and initial values, mark the form as initialized. The form is not dirty afterwards. Works once, later calls are ignored. Use it for values loaded from a server |
| `reset` | `() => void` | Reset to the current initial values (the ones set by `initialize` or `setInitialValues`, if called), clear errors/touched/dirty |
| `resetField` | `(path) => void` | Reset a single field to its initial value. In 9.6 and earlier `isDirty(path)` stays `true` afterwards |

### Errors

| Method | Signature | Description |
|---|---|---|
| `setErrors` | `(errors: FormErrors \| (prev) => FormErrors) => void` | Replace all errors. Keys are dot paths, list items included: `{ 'lines.2.quantity': 'Not enough stock' }`. `null`, `undefined` and `false` values are dropped |
| `setFieldError` | `(path, error: ReactNode) => void` | Set one field's error |
| `clearFieldError` | `(path) => void` | Clear one field's error |
| `clearErrors` | `() => void` | Clear all errors |

### Dirty & touched

| Method | Signature | Description |
|---|---|---|
| `isDirty` | `(path?) => boolean` | True if field (or any field) differs from initial value |
| `isTouched` | `(path?) => boolean` | True if field (or any field) has been interacted with. With `touchTrigger: 'focus'` a field is touched as soon as it is focused |
| `getDirty` | `() => FormStatus` | All dirty flags |
| `getTouched` | `() => FormStatus` | All touched flags |
| `setDirty` | `(status: FormStatus \| (prev) => FormStatus) => void` | Overwrite dirty flags |
| `setTouched` | `(status: FormStatus \| (prev) => FormStatus) => void` | Overwrite touched flags |
| `resetDirty` | `(values?) => void` | Reset dirty tracking (optionally to new baseline) |
| `resetTouched` | `() => void` | Reset all touched flags |

### Array fields

| Method | Signature | Description |
|---|---|---|
| `insertListItem` | `(path, item, index?) => void` | Insert item at index (appends if omitted) |
| `removeListItem` | `(path, index) => void` | Remove item at index |
| `reorderListItem` | `(path, { from, to }) => void` | Move item from one index to another |
| `replaceListItem` | `(path, index, item) => void` | Replace item at index |

### Validation

| Method | Signature | Description |
|---|---|---|
| `validate` | `() => FormValidationResult \| Promise<...>` | Validate all fields, returns result |
| `validateField` | `(path) => FormFieldValidationResult \| Promise<...>` | Validate one field now (not debounced), set its error, return `{ hasError, error }` |
| `isValid` | `(path?) => boolean \| Promise<boolean>` | True if form/field has no errors |
| `isValidating` | `(path?) => boolean` | True if async validation is pending |
| `setSubmitting` | `(value: boolean) => void` | Manually control `submitting` flag |

`validate`, `validateField` and `isValid` return plain results when all rules are synchronous and a `Promise` when a rule is async or `schemaResolver` is used without `{ sync: true }`. TypeScript infers which one from the rules passed to `useForm`.

Before 9.7, the `useForm` returned by `createFormContext` loses this inference: the methods are typed as synchronous even when a rule is async. At runtime they still return a `Promise`, so `await` the result anyway. The form returned by `useFormContext()` is typed as synchronous in all versions unless the rules type is passed as the third generic of `createFormContext`.

### Submission

| Method | Signature | Description |
|---|---|---|
| `onSubmit` | `(handler, onError?) => FormEventHandler` | Returns event handler; calls `handler(values, event)` only when valid, otherwise `onError(errors, values, event)` |
| `onReset` | `FormEventHandler` | Pass to `<form onReset>` to reset on native reset |
| `getTransformedValues` | `(values?) => TransformedValues` | Apply `transformValues` to given or current values |

### Input binding

| Method | Signature | Description |
|---|---|---|
| `getInputProps` | `(path, options?) => object` | Returns `{ value, onChange, error, onFocus, onBlur }` to spread on an input (`defaultValue` instead of `value` in uncontrolled mode). `onChange` accepts a change event or a raw value |
| `getInputNode` | `(path) => HTMLElement \| null` | Get the DOM node of the input that received `getInputProps(path)` (found by its `data-path` attribute), for example to focus it |
| `key` | `(path) => string` | React `key` for the input. Required on every input in uncontrolled mode |

**`getInputProps` options:**
```ts
{
  type?: 'input' | 'checkbox' | 'radio' // 'checkbox' and 'radio' use checked/defaultChecked instead of value
  value?: string                // required for type: 'radio' — the value of this radio option
  withError?: boolean           // default true for type: 'input' — include error prop
  withFocus?: boolean           // default true (false for type: 'radio') — include onFocus for touched tracking
}
```

### Watch

| Method | Signature | Description |
|---|---|---|
| `watch` | `(path, subscriber) => void` | Subscribe to a field's changes with a callback, does not rerender. Uses `useEffect` inside: call it at the top level of the component like a hook. For an array path it also fires on nested changes and list operations |
| `useWatchValue` | `(path) => value` | Hook: returns the field value and rerenders the component when it changes. Works in both modes. For values derived from a whole list (totals, counts), subscribe with `watch('lines', ...)` instead |

`watch` and `useWatchValue` can be called in any component that has the `form` object, including child components that receive it through props or context.

`watch` subscriber receives `{ value, previousValue, touched, dirty }`.

---

## useField hook

Standalone single-field hook — no full form needed.

```ts
const field = useField({
  initialValue,
  validate?,
  validateOnChange?,    // boolean, default false
  validateOnBlur?,      // boolean, default false
  clearErrorOnChange?,  // boolean, default true
  initialError?,
  initialTouched?,      // boolean, default false
  onValueChange?,
  type?,                // 'input' | 'checkbox' | 'radio', default 'input'
  mode?,                // 'controlled' | 'uncontrolled', default 'controlled'
  resolveValidationError?,
})
```

**Return value:**
```ts
{
  getInputProps(options?) // spread on input
  getValue()             // current value
  setValue(value)        // set value
  reset()               // reset to initial state
  validate()            // returns Promise<ReactNode | void>
  isValidating          // boolean
  error                 // ReactNode
  setError(error)
  isTouched()           // boolean
  isDirty()             // boolean
  resetTouched()
  key                   // number — add to the input as `key` in uncontrolled mode
}
```

---

## createFormContext

Shares a `useForm` instance across multiple components via React context.

```ts
const [FormProvider, useFormContext, useForm] = createFormContext<Values>();

// With transformValues, pass the transformed type as the second generic
const [FormProvider, useFormContext, useForm] = createFormContext<Values, TransformedValues>();
```

Returns a tuple:
- **`FormProvider`** — `<FormProvider form={form}>` wraps the subtree
- **`useFormContext`** — access the form anywhere inside the provider (throws if used outside)
- **`useForm`** — same as the package-level `useForm` but typed to `Values`

---

## createFormActions

Event-bus API for controlling a named form from outside its React tree.

```ts
const actions = createFormActions<Values>('my-form-name');
```

The form must be created with the same `name`:
```ts
const form = useForm({ name: 'my-form-name', initialValues: {...} });
```

The form must be mounted for actions to have an effect. Actions rerender the form the same way the form's own methods do. There is no `submit` or `resetField` action: to submit from outside, give the `<form>` an `id` and use `<Button type="submit" form="that-id">`.

**Available actions** (all dispatch custom DOM events):
`setFieldValue`, `setValues`, `setInitialValues`, `setErrors`, `setFieldError`,
`clearFieldError`, `clearErrors`, `reset`, `validate`, `validateField`,
`insertListItem`, `removeListItem`, `reorderListItem`, `setDirty`, `setTouched`,
`resetDirty`, `resetTouched`

---

## schemaResolver

Validates the form with any [Standard Schema](https://standardschema.dev/) library (zod 4, valibot, arktype). No resolver package is needed.

```ts
import { schemaResolver } from '@mantine/form';

validate: schemaResolver(schema)                 // validate() returns a Promise
validate: schemaResolver(schema, { sync: true }) // validate() returns a plain result, for synchronous schemas
```

Error keys are the issue paths joined with dots (`user.address.city`, `employees.0.name`).

---

## Built-in validators

All validators return a rule function `(value) => ReactNode | null` to pass to `validate`; `matchesField` also reads the second `values` argument. To use a validator inside your own rule, call it with the value: `isEmail('Invalid email')(value)`.

`isEmail`, `isUrl` and `matches` reject an empty string. For an optional field, check for empty yourself: `(value) => (value === '' ? null : isUrl('Invalid URL')(value))`.

| Validator | Signature | Validates |
|---|---|---|
| `isNotEmpty(error?)` | — | Not null, undefined, false, empty string, or empty array |
| `isEmail(error?)` | — | Valid email format |
| `isUrl(error?)` or `isUrl(options, error?)` | `options: { protocols?: string[], allowLocalhost?: boolean }` | Valid URL (default: http/https, no localhost) |
| `matches(regexp, error?)` | — | String matches regex |
| `hasLength(payload, error?)` | `payload: number \| { min?, max? }` | String/array length (exact or range) |
| `isInRange({ min?, max? }, error?)` | — | Number within range |
| `matchesField(fieldName, error?)` | — | Value equals another field's value |
| `isNotEmptyHTML(error?)` | — | HTML string is not empty after stripping tags |
| `isJSONString(error?)` | — | String is valid JSON |
| `isOneOf(values[], error?)` | — | Value is one of allowed values |

All `error` arguments are optional `ReactNode`. When omitted, the error is `true`: the input gets invalid styles without a message.

---

## Validation rule format

```ts
// Rules object — keys match form field names, nested objects follow the values shape
validate: {
  name: isNotEmpty(),
  address: {
    street: isNotEmpty(),
    zip: matches(/^\d{5}$/, 'Invalid ZIP'),
  },
  // List of objects: keys are the item fields, the rule runs for every item.
  // formRootRule validates the array itself; its error is at form.errors.employees
  employees: {
    [formRootRule]: isNotEmpty('At least one employee is required'),
    name: isNotEmpty('Name is required'),
    // List inside a list item: rules nest the same way.
    // Errors: 'employees.0.skills' (root rule), 'employees.0.skills.1.title' (item)
    skills: {
      [formRootRule]: (skills) => (skills.length < 2 ? 'Add at least 2 skills' : null), // typed correctly from 9.7, see below
      title: isNotEmpty('Skill is required'),
    },
  },
}

// Function — receives full values, returns errors object
validate: (values) => ({
  endDate: values.endDate < values.startDate ? 'End must be after start' : null,
})

// Async — return a Promise from any rule
validate: {
  username: async (value, values, path, signal) => {
    const taken = await checkUsername(value, { signal });
    if (signal?.aborted) return null;
    return taken ? 'Username is taken' : null;
  }
}
```

In 9.6 and earlier the `formRootRule` of a list nested in a list item is typed with the item instead of the array (the runtime value is the array): read the array from the second argument, `values.employees[index].skills`, with the index taken from the path.

Rules are read on every render, so they can use props and state of the component directly.

Validators receive `(value, allValues, fieldPath, abortSignal?)`. For list items `fieldPath` includes the index: `'employees.1.name'`. The `abortSignal` is provided for async rules — check `signal.aborted` before applying stale results.

---

## Key types

```ts
type FormErrors = Record<string, ReactNode>
type FormStatus = Record<string, boolean>

interface FormValidationResult {
  hasErrors: boolean
  errors: FormErrors
}

interface FormFieldValidationResult {
  hasError: boolean
  error: ReactNode
}

type FormRulesRecord<Values>   // shape of the `validate` rules object
type UseFormReturnType<Values, TransformedValues = Values> // type of the `form` object, for props

// Path utilities (for typed field paths)
type LooseKeys<T>              // union of all dot-paths into T
type TransformedValues<typeof form> // type that onSubmit handler receives after transformValues
```

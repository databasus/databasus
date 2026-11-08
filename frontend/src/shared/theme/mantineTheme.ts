/* eslint-disable i18next/no-literal-string -- Theme tokens, CSS variables and component configuration contain technical values only. */
import {
  ActionIcon,
  Alert,
  Button,
  Checkbox,
  Combobox,
  createTheme,
  Drawer,
  Input,
  InputWrapper,
  Menu,
  Modal,
  Notification,
  PasswordInput,
  Popover,
  Radio,
  Switch,
  Tabs,
  Tooltip,
  virtualColor,
  type CSSVariablesResolver,
} from '@mantine/core';
import { DateInput, DateTimePicker, InlineDateTimePicker, TimePicker } from '@mantine/dates';
import { Notifications } from '@mantine/notifications';

import classes from '@/shared/theme/controls.module.css';

const calendarClassNames = {
  calendarHeaderLevel: classes.calendarHeaderLevel,
  day: classes.calendarDay,
};

const controlHeights: Record<string, string> = {
  xs: '1.5rem',
  sm: '2rem',
  md: '2.5rem',
};

export const mantineTheme = createTheme({
  // eslint-disable-next-line i18next/no-literal-string -- CSS font-family stack
  fontFamily: 'Jost, system-ui, -apple-system, "Segoe UI", "Noto Sans CJK SC", sans-serif',
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, Consolas, monospace',
  fontSizes: { xs: '0.875rem', sm: '0.875rem', md: '1rem' },
  lineHeights: { xs: '1.5714285714', sm: '1.5714285714', md: '1.5' },
  defaultRadius: 'sm',
  radius: { xs: '0.25rem', sm: '0.375rem', md: '0.5rem' },
  respectReducedMotion: true,
  primaryColor: 'accent',
  colors: {
    accent: virtualColor({ name: 'accent', light: 'blue', dark: 'blue' }),
    danger: virtualColor({ name: 'danger', light: 'red', dark: 'red' }),
    success: virtualColor({ name: 'success', light: 'green', dark: 'green' }),
    warning: virtualColor({ name: 'warning', light: 'yellow', dark: 'yellow' }),
  },
  focusClassName: classes.focus,
  components: {
    Button: Button.extend({
      defaultProps: { size: 'sm', variant: 'default' },
      classNames: {
        root: classes.button,
        inner: classes.buttonInner,
        label: classes.buttonLabel,
      },
      vars: (_theme, { size = 'sm' }) => ({
        root: {
          '--button-height': controlHeights[size],
          '--button-fz': size === 'md' ? '1rem' : '0.875rem',
          '--button-padding-x': size === 'xs' ? '0.4375rem' : '0.9375rem',
        },
      }),
    }),
    ActionIcon: ActionIcon.extend({
      defaultProps: { variant: 'subtle', size: 'sm' },
      classNames: { root: classes.actionIcon },
      vars: (_theme, { color, variant, size = 'sm' }) => ({
        root: {
          ...(variant === 'outline' ? { '--ai-size': controlHeights[size] } : {}),
          ...(!color && variant === 'subtle'
            ? { '--ai-color': 'var(--app-muted)', '--ai-hover': 'var(--app-hover)' }
            : {}),
        },
      }),
    }),
    Input: Input.extend({
      defaultProps: { size: 'sm' },
      classNames: { input: classes.input, wrapper: classes.inputWrapper },
      vars: (_theme, { size = 'sm' }) => ({
        wrapper: {
          '--input-height': controlHeights[size],
          '--input-fz': size === 'md' ? '1rem' : '0.875rem',
        },
      }),
    }),
    InputWrapper: InputWrapper.extend({
      defaultProps: { inputWrapperOrder: ['label', 'input', 'error', 'description'] },
      classNames: {
        label: classes.fieldLabel,
        error: classes.fieldError,
        description: classes.fieldDescription,
      },
    }),
    PasswordInput: PasswordInput.extend({
      defaultProps: { visibilityToggleFocusable: true },
      classNames: { innerInput: classes.passwordInput },
    }),
    DateInput: DateInput.extend({ classNames: calendarClassNames }),
    DateTimePicker: DateTimePicker.extend({ classNames: calendarClassNames }),
    InlineDateTimePicker: InlineDateTimePicker.extend({ classNames: calendarClassNames }),
    Tabs: Tabs.extend({
      classNames: { root: classes.tabs, tab: classes.tab, tabLabel: classes.tabLabel },
    }),
    TimePicker: TimePicker.extend({
      classNames: {
        fieldsRoot: classes.timeFieldsRoot,
        fieldsGroup: classes.timeFieldsGroup,
        field: classes.timeField,
      },
    }),
    Checkbox: Checkbox.extend({
      defaultProps: { size: 'xs' },
      classNames: (_theme, { label }) => ({
        root: classes.checkbox,
        label: classes.choiceLabel,
        inner: label ? classes.checkboxInner : undefined,
      }),
    }),
    Radio: Radio.extend({
      classNames: (_theme, { label }) => ({
        root: classes.radio,
        label: classes.choiceLabel,
        inner: label ? classes.radioInner : undefined,
      }),
    }),
    Switch: Switch.extend({
      defaultProps: { size: 'xs' },
      classNames: { track: classes.switchTrack, label: classes.choiceLabel },
    }),
    Alert: Alert.extend({
      defaultProps: { role: 'alert' },
      classNames: {
        root: classes.alert,
        title: classes.alertTitle,
        message: classes.alertMessage,
      },
      vars: (_theme, { color }) => ({
        root: {
          '--alert-bd': `1px solid var(--app-${
            color === 'danger' || color === 'warning' || color === 'success'
              ? `${color}-border`
              : 'border'
          })`,
        },
      }),
    }),
    Modal: Modal.extend({
      defaultProps: {
        size: 520,
        padding: 20,
        yOffset: 100,
        overlayProps: { backgroundOpacity: 0.45 },
      },
      classNames: {
        content: `${classes.overlayContent} ${classes.modalContent}`,
        header: classes.modalHeader,
        title: classes.overlayTitle,
        body: classes.modalBody,
      },
    }),
    Drawer: Drawer.extend({
      classNames: { content: classes.overlayContent, title: classes.overlayTitle },
    }),
    Menu: Menu.extend({ classNames: { dropdown: classes.dropdown, item: classes.menuItem } }),
    Combobox: Combobox.extend({
      classNames: { dropdown: classes.dropdown, option: classes.menuItem },
    }),
    Popover: Popover.extend({ classNames: { dropdown: classes.dropdown } }),
    Tooltip: Tooltip.extend({
      defaultProps: {
        multiline: true,
        maw: 320,
        withArrow: true,
        events: { hover: true, focus: true, touch: false },
      },
      classNames: { tooltip: classes.tooltip },
    }),
    Notification: Notification.extend({
      classNames: {
        root: classes.notification,
        icon: classes.notificationIcon,
        description: classes.alertMessage,
      },
    }),
    Notifications: Notifications.extend({
      classNames: { root: classes.notifications },
    }),
  },
});

export const mantineCssVariables: CSSVariablesResolver = () => {
  const colors = {
    '--mantine-color-body': 'var(--app-page)',
    '--mantine-color-text': 'var(--app-content)',
    '--mantine-color-dimmed': 'var(--app-description)',
    '--mantine-color-placeholder': 'var(--app-placeholder)',
    '--mantine-color-anchor': 'var(--app-accent)',
    '--mantine-color-error': 'var(--app-danger)',
    '--mantine-color-default': 'var(--app-surface)',
    '--mantine-color-default-hover': 'var(--app-hover)',
    '--mantine-color-default-color': 'var(--app-content)',
    '--mantine-color-default-border': 'var(--app-border)',
    '--mantine-color-disabled': 'var(--app-hover)',
    '--mantine-color-disabled-color': 'var(--app-disabled)',
    '--mantine-color-disabled-border': 'var(--app-border)',
    '--mantine-color-accent-filled': 'var(--app-primary)',
    '--mantine-color-accent-filled-hover': 'var(--app-primary-hover)',
    '--mantine-color-accent-light': 'var(--app-primary-soft)',
    '--mantine-color-accent-light-hover': 'var(--app-selected)',
    '--mantine-color-accent-light-color': 'var(--app-accent)',
    '--mantine-color-accent-outline': 'var(--app-accent)',
    '--mantine-color-accent-outline-hover': 'var(--app-hover)',
    '--mantine-color-danger-filled': 'var(--app-error-fill)',
    '--mantine-color-danger-filled-hover': 'var(--app-danger)',
    '--mantine-color-danger-light': 'var(--app-danger-soft)',
    '--mantine-color-danger-light-color': 'var(--app-danger)',
    '--mantine-color-danger-outline': 'var(--app-danger)',
    '--mantine-color-danger-outline-hover': 'var(--app-hover)',
    '--mantine-color-success-filled': 'var(--app-success-fill)',
    '--mantine-color-success-light': 'var(--app-success-soft)',
    '--mantine-color-success-light-color': 'var(--app-success)',
    '--mantine-color-warning-light': 'var(--app-warning-soft)',
    '--mantine-color-warning-light-color': 'var(--app-warning)',
  };

  return { variables: {}, light: colors, dark: colors };
};

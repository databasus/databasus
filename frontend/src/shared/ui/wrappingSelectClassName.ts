// A single AntD Select keeps a fixed height and one line, which clips translated values at 320
// CSS pixels with enlarged text.
export const WRAPPING_SELECT_CLASS_NAME =
  // eslint-disable-next-line i18next/no-literal-string -- CSS class names
  '!h-auto [&_.ant-select-selection-item]:!leading-normal [&_.ant-select-selection-item]:!whitespace-normal [&_.ant-select-selector]:!h-auto';

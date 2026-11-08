import { Menu } from '@mantine/core';
import { Monitor, Moon, Sun } from 'lucide-react';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { THEME_MODE_LABEL_KEYS, useTheme } from '@/shared/theme';

interface Props {
  isSplitEnd?: boolean;
}

export function ThemeToggleComponent({ isSplitEnd = false }: Props) {
  const [isOpen, setIsOpen] = useState(false);
  const controlRef = useRef<HTMLButtonElement>(null);
  const { t } = useTranslation();

  const { theme, setTheme, resolvedTheme } = useTheme();
  const CurrentIcon = theme === 'system' ? Monitor : resolvedTheme === 'dark' ? Moon : Sun;

  return (
    <Menu
      opened={isOpen}
      onChange={(isNextOpen) => {
        setIsOpen(isNextOpen);
        if (!isNextOpen) controlRef.current?.focus();
      }}
      returnFocus={false}
      closeOnEscape={false}
      position="bottom-end"
      withInitialFocusPlaceholder={false}
    >
      <Menu.Target>
        <button
          ref={controlRef}
          type="button"
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown') {
              event.preventDefault();
              setIsOpen(true);
            } else if (event.key === 'Escape' && isOpen) {
              event.preventDefault();
              event.stopPropagation();
              setIsOpen(false);
              controlRef.current?.focus();
            }
          }}
          className={`flex cursor-pointer items-center gap-1.5 border border-border bg-surface px-2.5 py-1 text-sm text-content transition-colors hover:bg-hover ${
            isSplitEnd ? 'rounded-r-md border-l-0' : 'rounded-md'
          }`}
          aria-label={t('app.theme.title', { theme: t(THEME_MODE_LABEL_KEYS[theme]) })}
        >
          <CurrentIcon aria-hidden size={14} />
          <span className="hidden sm:inline">{t(THEME_MODE_LABEL_KEYS[theme])}</span>
        </button>
      </Menu.Target>
      <Menu.Dropdown
        onKeyDownCapture={(event) => {
          if (event.key !== 'Escape') return;

          event.preventDefault();
          event.stopPropagation();
          setIsOpen(false);
          controlRef.current?.focus();
        }}
      >
        <Menu.RadioGroup
          value={theme}
          onChange={(value) => {
            if (value === 'system' || value === 'light' || value === 'dark') setTheme(value);
          }}
        >
          {(
            [
              ['light', Sun],
              ['dark', Moon],
              ['system', Monitor],
            ] as const
          ).map(([mode, Icon]) => (
            <Menu.RadioItem
              key={mode}
              value={mode}
              closeMenuOnClick
              data-autofocus={mode === theme || undefined}
            >
              <span className="flex items-center gap-2">
                <Icon aria-hidden size={14} />
                {t(THEME_MODE_LABEL_KEYS[mode])}
              </span>
            </Menu.RadioItem>
          ))}
        </Menu.RadioGroup>
      </Menu.Dropdown>
    </Menu>
  );
}

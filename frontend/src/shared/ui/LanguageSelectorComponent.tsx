import { Menu } from '@mantine/core';
import { Globe } from 'lucide-react';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { DICTIONARIES, SUPPORTED_LOCALES, useLocale } from '@/shared/i18n';

interface Props {
  isSplitStart?: boolean;
}

export function LanguageSelectorComponent({ isSplitStart = false }: Props) {
  const [isOpen, setIsOpen] = useState(false);
  const controlRef = useRef<HTMLButtonElement>(null);
  const { t } = useTranslation();

  const { locale, setLocale } = useLocale();

  return (
    <Menu
      opened={isOpen}
      onChange={(isNextOpen) => {
        setIsOpen(isNextOpen);
        if (!isNextOpen) controlRef.current?.focus();
      }}
      returnFocus={false}
      closeOnEscape={false}
      trigger="click-hover"
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
            isSplitStart ? 'rounded-l-md' : 'rounded-md'
          }`}
          aria-label={t('app.languageControl.title', {
            language: DICTIONARIES[locale].language.nativeName,
          })}
        >
          <Globe aria-hidden size="1em" className="inline-block shrink-0 align-middle" />
          <span>{locale.toUpperCase()}</span>
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
          value={locale}
          onChange={(value) => {
            const selectedLocale = SUPPORTED_LOCALES.find((supported) => supported === value);
            if (selectedLocale) setLocale(selectedLocale);
          }}
        >
          {SUPPORTED_LOCALES.map((supportedLocale) => (
            <Menu.RadioItem
              key={supportedLocale}
              value={supportedLocale}
              closeMenuOnClick
              data-autofocus={supportedLocale === locale || undefined}
            >
              <span lang={supportedLocale}>
                {DICTIONARIES[supportedLocale].language.nativeName}
              </span>
            </Menu.RadioItem>
          ))}
        </Menu.RadioGroup>
      </Menu.Dropdown>
    </Menu>
  );
}

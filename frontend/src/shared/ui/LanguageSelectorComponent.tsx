import { GlobalOutlined } from '@ant-design/icons';
import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { DICTIONARIES, SUPPORTED_LOCALES, useLocale } from '@/shared/i18n';
import { useDropdownFocus } from '@/shared/ui/useDropdownFocus';

interface Props {
  isSplitStart?: boolean;
}

// Opens on hover like the website's selector; click is kept for touch screens, which have no
// hover. Each language is listed by the name its own dictionary gives it.
export function LanguageSelectorComponent({ isSplitStart = false }: Props) {
  const [isOpen, setIsOpen] = useState(false);

  const changeOpen = (isNextOpen: boolean) => {
    setIsOpen(isNextOpen);
    if (!isNextOpen) controlRef.current?.focus();
  };
  const { t } = useTranslation();
  const controlRef = useRef<HTMLButtonElement>(null);
  const focusMenu = useDropdownFocus(isOpen);

  const { locale, setLocale } = useLocale();

  const items: MenuProps['items'] = SUPPORTED_LOCALES.map((supportedLocale) => ({
    key: supportedLocale,
    label: <span lang={supportedLocale}>{DICTIONARIES[supportedLocale].language.nativeName}</span>,
    onClick: () => {
      setLocale(supportedLocale);
      changeOpen(false);
    },
  }));

  return (
    <Dropdown
      open={isOpen}
      onOpenChange={changeOpen}
      popupRender={(menu) => <div ref={focusMenu}>{menu}</div>}
      menu={{
        selectable: true,
        items,
        selectedKeys: [locale],
        onKeyDown: (event) => {
          if (event.key === 'Enter') event.preventDefault();
          if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            changeOpen(false);
          }
        },
      }}
      trigger={['hover', 'click']}
      placement="bottomRight"
    >
      <button
        ref={controlRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        onKeyDown={(event) => {
          if (['ArrowDown', 'Enter', ' '].includes(event.key)) {
            event.preventDefault();
            changeOpen(true);
          } else if (event.key === 'Escape' && isOpen) {
            event.preventDefault();
            event.stopPropagation();
            changeOpen(false);
          }
        }}
        className={`flex cursor-pointer items-center gap-1.5 border border-border bg-surface px-2.5 py-1 text-sm text-content transition-colors hover:bg-hover ${
          isSplitStart ? 'rounded-l-md' : 'rounded-md'
        }`}
        aria-label={t('app.languageControl.title', {
          language: DICTIONARIES[locale].language.nativeName,
        })}
      >
        <GlobalOutlined aria-hidden />
        <span>{locale.toUpperCase()}</span>
      </button>
    </Dropdown>
  );
}

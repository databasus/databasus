import { SunOutlined, MoonOutlined, DesktopOutlined } from '@ant-design/icons';
import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { THEME_MODE_LABEL_KEYS, useTheme } from '@/shared/theme';
import { useDropdownFocus } from '@/shared/ui/useDropdownFocus';

interface Props {
  isSplitEnd?: boolean;
}

export function ThemeToggleComponent({ isSplitEnd = false }: Props) {
  const [isOpen, setIsOpen] = useState(false);

  const changeOpen = (isNextOpen: boolean) => {
    setIsOpen(isNextOpen);
    if (!isNextOpen) controlRef.current?.focus();
  };

  const getCurrentIcon = () => {
    if (theme === 'system') {
      return <DesktopOutlined aria-hidden />;
    }
    return resolvedTheme === 'dark' ? <MoonOutlined aria-hidden /> : <SunOutlined aria-hidden />;
  };
  const { t } = useTranslation();
  const controlRef = useRef<HTMLButtonElement>(null);
  const focusMenu = useDropdownFocus(isOpen);

  const { theme, setTheme, resolvedTheme } = useTheme();

  const items: MenuProps['items'] = [
    {
      key: 'light',
      label: (
        <div className="flex items-center gap-2">
          <SunOutlined aria-hidden />
          <span>{t(THEME_MODE_LABEL_KEYS.light)}</span>
        </div>
      ),
      onClick: () => {
        setTheme('light');
        changeOpen(false);
      },
    },
    {
      key: 'dark',
      label: (
        <div className="flex items-center gap-2">
          <MoonOutlined aria-hidden />
          <span>{t(THEME_MODE_LABEL_KEYS.dark)}</span>
        </div>
      ),
      onClick: () => {
        setTheme('dark');
        changeOpen(false);
      },
    },
    {
      key: 'system',
      label: (
        <div className="flex items-center gap-2">
          <DesktopOutlined aria-hidden />
          <span>{t(THEME_MODE_LABEL_KEYS.system)}</span>
        </div>
      ),
      onClick: () => {
        setTheme('system');
        changeOpen(false);
      },
    },
  ];

  return (
    <Dropdown
      open={isOpen}
      onOpenChange={changeOpen}
      popupRender={(menu) => <div ref={focusMenu}>{menu}</div>}
      menu={{
        selectable: true,
        items,
        selectedKeys: [theme],
        onKeyDown: (event) => {
          if (event.key === 'Enter') event.preventDefault();
          if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            changeOpen(false);
          }
        },
      }}
      trigger={['click']}
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
          isSplitEnd ? 'rounded-r-md border-l-0' : 'rounded-md'
        }`}
        aria-label={t('app.theme.title', { theme: t(THEME_MODE_LABEL_KEYS[theme]) })}
      >
        {getCurrentIcon()}
        <span className="hidden sm:inline">{t(THEME_MODE_LABEL_KEYS[theme])}</span>
      </button>
    </Dropdown>
  );
}

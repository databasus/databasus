// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

import { type MainTab, SidebarComponent, type SidebarTab } from './SidebarComponent';

const server = setUpMockApi();

const buildTab = ({
  name,
  text,
  onSelect,
}: {
  name: MainTab;
  text: string;
  onSelect: (name: MainTab) => void;
}): SidebarTab => ({
  text,
  name,
  icon: `/icons/menu/${name}-gray.svg`,
  selectedIcon: `/icons/menu/${name}-white.svg`,
  onClick: () => onSelect(name),
  isAdminOnly: false,
  marginTop: '0px',
  isVisible: true,
});

const SelectableSidebar = () => {
  const [selectedTab, setSelectedTab] = useState<MainTab>('databases');

  return (
    <SidebarComponent
      isOpen
      onClose={() => {}}
      selectedTab={selectedTab}
      tabs={[
        buildTab({ name: 'databases', text: 'Databases', onSelect: setSelectedTab }),
        buildTab({ name: 'storages', text: 'Storages', onSelect: setSelectedTab }),
      ]}
    />
  );
};

const getTabIcon = (tab: HTMLElement) => tab.querySelector('img')!;

describe('SidebarComponent', () => {
  describe.each([
    { layout: 'desktop', viewportWidth: 1200 },
    { layout: 'mobile', viewportWidth: 320 },
  ])('in the $layout layout', ({ viewportWidth }) => {
    beforeEach(() => {
      vi.stubGlobal('innerWidth', viewportWidth);
      server.use(
        http.get('https://api.github.com/repos/databasus/databasus', () =>
          HttpResponse.json({ stargazers_count: 1 }),
        ),
      );
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it('marks the selected tab with aria-current', () => {
      renderScreen(<SelectableSidebar />);

      const selectedTab = screen.getByRole('button', { name: 'Databases' });

      expect(selectedTab.getAttribute('aria-current')).toBe('page');
      expect(
        screen.getByRole('button', { name: 'Storages' }).getAttribute('aria-current'),
      ).toBeNull();
      expect(document.querySelectorAll('[aria-current="page"]')).toHaveLength(1);
      expect(getTabIcon(selectedTab).getAttribute('src')).toBe('/icons/menu/databases-white.svg');
      expect(getTabIcon(selectedTab).getAttribute('alt')).toBe('');
    });

    it('moves aria-current to the activated tab', async () => {
      const { user } = renderScreen(<SelectableSidebar />);

      await user.click(screen.getByRole('button', { name: 'Storages' }));

      const activatedTab = screen.getByRole('button', { name: 'Storages' });
      const previousTab = screen.getByRole('button', { name: 'Databases' });

      expect(activatedTab.getAttribute('aria-current')).toBe('page');
      expect(previousTab.getAttribute('aria-current')).toBeNull();
      expect(document.querySelectorAll('[aria-current="page"]')).toHaveLength(1);
      expect(getTabIcon(activatedTab).getAttribute('src')).toBe('/icons/menu/storages-white.svg');
      expect(getTabIcon(previousTab).getAttribute('src')).toBe('/icons/menu/databases-gray.svg');
    });
  });
});

// @vitest-environment happy-dom
import { act, screen } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { AppShell } from '@/widgets/app-shell/ui/AppShell';

const server = setUpMockApi();

describe('AppShell', () => {
  it('names the logo link', () => {
    server.use(
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
    );

    renderScreen(
      <AppShell
        workspaceSelector={<div>Workspace selector</div>}
        sidebar={<nav>Navigation</nav>}
        isSidebarOpen={false}
        onOpenSidebar={() => {}}
      >
        <div />
      </AppShell>,
      { locale: 'fr' },
    );

    expect(screen.getByRole('link', { name: DICTIONARIES.fr.common.websiteLink })).toBeTruthy();
  });

  it('resizes its boundary and removes viewport listeners when the shell unmounts after replay', async () => {
    server.use(
      http.get('https://api.github.com/repos/databasus/databasus', () =>
        HttpResponse.json({ stargazers_count: 1 }),
      ),
    );

    const viewport = new EventTarget() as EventTarget & { height: number };
    viewport.height = 600;
    const addListener = vi.spyOn(viewport, 'addEventListener');
    const removeListener = vi.spyOn(viewport, 'removeEventListener');
    const previous = Object.getOwnPropertyDescriptor(window, 'visualViewport');
    Object.defineProperty(window, 'visualViewport', { configurable: true, value: viewport });

    const view = renderScreen(
      <AppShell
        workspaceSelector={<div>Workspace selector</div>}
        sidebar={<nav>Navigation</nav>}
        isSidebarOpen={false}
        onOpenSidebar={() => {}}
      >
        <input aria-label="Draft" defaultValue="Retained input" />
      </AppShell>,
      { isStrictMode: true },
    );

    try {
      const boundary = screen.getByRole('banner').parentElement!;
      expect(boundary.style.height).toBe('600px');

      act(() => {
        viewport.height = 420;
        viewport.dispatchEvent(new Event('resize'));
      });

      expect(boundary.style.height).toBe('420px');
      expect(screen.getByDisplayValue('Retained input')).toBeTruthy();

      view.unmount();
      expect(addListener.mock.calls.filter((call) => call[0] === 'resize')).toHaveLength(2);
      expect(removeListener.mock.calls.filter((call) => call[0] === 'resize')).toHaveLength(2);
    } finally {
      view.unmount();
      if (previous) Object.defineProperty(window, 'visualViewport', previous);
      else Reflect.deleteProperty(window, 'visualViewport');
    }
  });
});

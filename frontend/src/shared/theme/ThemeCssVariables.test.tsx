// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { renderScreen } from '@/shared/lib/test/renderScreen';
import { useTheme } from '@/shared/theme/useTheme';

const ThemeScreen = () => {
  const { setTheme, resolvedTheme } = useTheme();
  return (
    <button onClick={() => setTheme(resolvedTheme === 'light' ? 'dark' : 'light')}>Switch</button>
  );
};

describe('ThemeCssVariables', () => {
  it('updates document and portal colors without replacing descendants when the theme changes', () => {
    document.documentElement.style.setProperty('--app-surface', 'previous');
    const view = renderScreen(<ThemeScreen />, { theme: 'light', isStrictMode: true });
    const control = screen.getByRole('button', { name: 'Switch' });

    expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe('#ffffff');
    fireEvent.click(control);
    expect(screen.getByRole('button', { name: 'Switch' })).toBe(control);
    expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe('#141414');

    for (const property of ['track', 'thumb', 'thumb-hover']) {
      expect(
        document.documentElement.style.getPropertyValue(`--color-scrollbar-${property}`),
      ).not.toBe('');
    }

    view.unmount();
    expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe('previous');
    document.documentElement.style.removeProperty('--app-surface');
  });

  it('follows system changes and removes its media listener after StrictMode cleanup', () => {
    const media = Object.assign(new EventTarget(), { matches: false });
    const add = vi.spyOn(media, 'addEventListener');
    const remove = vi.spyOn(media, 'removeEventListener');
    vi.spyOn(window, 'matchMedia').mockReturnValue(media as MediaQueryList);
    const view = renderScreen(<ThemeScreen />, { theme: 'system', isStrictMode: true });

    expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe('#ffffff');
    act(() => media.dispatchEvent(Object.assign(new Event('change'), { matches: true })));
    expect(document.documentElement.style.getPropertyValue('--app-surface')).toBe('#141414');

    view.unmount();
    expect(remove.mock.calls).toHaveLength(add.mock.calls.length);
  });
});

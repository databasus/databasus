// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DICTIONARIES } from '../i18n';
import { renderScreen } from '../lib/test/renderScreen';
import { LanguageThemeControlComponent } from './LanguageThemeControlComponent';

describe('LanguageThemeControlComponent', () => {
  it('opens the language menu when activated from the keyboard', async () => {
    const { user } = renderScreen(<LanguageThemeControlComponent />, { locale: 'en' });
    const language = screen.getByRole('button', { name: /Language/ });
    language.focus();
    await user.keyboard('{ArrowDown}');
    expect(language.getAttribute('aria-expanded')).toBe('true');
    const menu = await screen.findByRole('menu');
    await waitFor(() => expect(menu.contains(document.activeElement)).toBe(true));
    await user.click(screen.getByRole('menuitem', { name: DICTIONARIES.es.language.nativeName }));
    expect(document.documentElement.lang).toBe('es');
    expect(screen.getAllByRole('button')).toHaveLength(2);
    expect(document.activeElement).toBe(language);
    await user.keyboard('{ArrowDown}');
    const selectedLanguage = await screen.findByRole('menuitem', {
      name: DICTIONARIES.es.language.nativeName,
    });
    expect(selectedLanguage.classList.contains('ant-dropdown-menu-item-selected')).toBe(true);
    await waitFor(() => expect(menu.contains(document.activeElement)).toBe(true));
    await user.keyboard('{Escape}');
    await waitFor(() => {
      expect(language.getAttribute('aria-expanded')).toBe('false');
      expect(document.activeElement).toBe(language);
    });
  });

  it('opens the language menu on hover and restores focus when Escape dismisses it', async () => {
    const { user } = renderScreen(<LanguageThemeControlComponent />, { locale: 'en' });
    const language = screen.getByRole('button', { name: /Language/ });
    await user.hover(language);
    const menu = await screen.findByRole('menu');
    fireEvent.keyDown(menu, { key: 'Escape' });
    await waitFor(() => {
      expect(language.getAttribute('aria-expanded')).toBe('false');
      expect(document.activeElement).toBe(language);
    });
  });
});

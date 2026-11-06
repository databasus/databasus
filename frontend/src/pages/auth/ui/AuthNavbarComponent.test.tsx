// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { AuthNavbarComponent } from '@/pages/auth/ui/AuthNavbarComponent';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { renderScreen } from '@/shared/lib/test/renderScreen';

describe('AuthNavbarComponent', () => {
  it('names the logo link in the selected language', () => {
    renderScreen(<AuthNavbarComponent />, { locale: 'fr' });

    expect(screen.getByRole('link', { name: DICTIONARIES.fr.common.websiteLink })).toBeTruthy();
  });
});

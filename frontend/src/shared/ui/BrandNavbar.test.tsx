// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { BrandNavbar } from '@/shared/ui/BrandNavbar';

describe('BrandNavbar', () => {
  it('names the logo link in the selected language', () => {
    renderScreen(<BrandNavbar />, { locale: 'fr' });

    expect(screen.getByRole('link', { name: DICTIONARIES.fr.common.websiteLink })).toBeTruthy();
  });
});

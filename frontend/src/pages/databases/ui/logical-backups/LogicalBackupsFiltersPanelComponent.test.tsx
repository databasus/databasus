// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DICTIONARIES } from '../../../../shared/i18n/dictionaries';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { LogicalBackupsFiltersPanelComponent } from './LogicalBackupsFiltersPanelComponent';

describe('LogicalBackupsFiltersPanelComponent', () => {
  it('names the status and date filters by their translated labels', () => {
    renderScreen(<LogicalBackupsFiltersPanelComponent filter={{}} onFilterChange={() => {}} />, {
      locale: 'fr',
    });

    expect(
      screen.getByRole('combobox', { name: DICTIONARIES.fr.common.fields.status }),
    ).toBeTruthy();
    expect(screen.getByLabelText(DICTIONARIES.fr.backups.filters.before)).toBeTruthy();
  });
});

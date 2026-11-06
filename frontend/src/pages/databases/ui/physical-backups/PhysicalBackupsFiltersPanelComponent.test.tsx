// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { PhysicalBackupsFiltersPanelComponent } from '@/pages/databases/ui/physical-backups/PhysicalBackupsFiltersPanelComponent';
import { renderScreen } from '@/shared/lib/test/renderScreen';

describe('PhysicalBackupsFiltersPanelComponent', () => {
  it('names the type, status and date filters by their labels', () => {
    renderScreen(<PhysicalBackupsFiltersPanelComponent filters={{}} onFiltersChange={() => {}} />);

    expect(screen.getByRole('combobox', { name: 'Type' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Status' })).toBeTruthy();
    expect(screen.getByLabelText('Before')).toBeTruthy();
  });
});

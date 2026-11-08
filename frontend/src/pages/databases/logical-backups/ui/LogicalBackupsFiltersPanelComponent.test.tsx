// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { LogicalBackupsFiltersPanelComponent } from '@/pages/databases/logical-backups/ui/LogicalBackupsFiltersPanelComponent';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { renderScreen } from '@/shared/lib/test/renderScreen';

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
  it('closes the calendar when Escape is pressed after keyboard focus enters it', async () => {
    const onDateChange = vi.fn();
    const view = renderScreen(
      <LogicalBackupsFiltersPanelComponent filter={{}} onFilterChange={onDateChange} />,
    );
    const date = screen.getByRole('textbox', { name: 'Before' });

    await view.user.click(date);

    const day = screen.getByRole('button', { name: /, 15 / });

    await act(async () => day.focus());

    expect(document.activeElement).toBe(day);

    await view.user.keyboard('{Escape}');

    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Before' })).toBeNull());
    expect(document.activeElement).toBe(date);

    await view.user.click(date);
    await act(async () => screen.getByRole('button', { name: /, 15 / }).focus());
    await view.user.keyboard('{Enter}');

    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Before' })).toBeNull());
    expect(document.activeElement).toBe(date);
    expect(onDateChange).toHaveBeenCalledOnce();
    expect(onDateChange).toHaveBeenCalledWith({ createdBefore: expect.any(String) });
  });
});

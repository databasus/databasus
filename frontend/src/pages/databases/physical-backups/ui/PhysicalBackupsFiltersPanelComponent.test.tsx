// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { PhysicalBackupsFiltersPanelComponent } from '@/pages/databases/physical-backups/ui/PhysicalBackupsFiltersPanelComponent';
import { renderScreen } from '@/shared/lib/test/renderScreen';

describe('PhysicalBackupsFiltersPanelComponent', () => {
  it('names the type, status and date filters by their labels', () => {
    renderScreen(<PhysicalBackupsFiltersPanelComponent filters={{}} onFiltersChange={() => {}} />);

    expect(screen.getByRole('combobox', { name: 'Type' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Status' })).toBeTruthy();
    expect(screen.getByLabelText('Before')).toBeTruthy();
  });
  it('closes the calendar when Escape is pressed after keyboard focus enters it', async () => {
    const onDateChange = vi.fn();
    const view = renderScreen(
      <PhysicalBackupsFiltersPanelComponent filters={{}} onFiltersChange={onDateChange} />,
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

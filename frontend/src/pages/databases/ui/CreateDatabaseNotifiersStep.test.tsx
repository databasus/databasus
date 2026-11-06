// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { CreateDatabaseNotifiersStep } from '@/pages/databases/ui/CreateDatabaseNotifiersStep';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

describe('CreateDatabaseNotifiersStep', () => {
  it('opens creation from the empty dropdown with the keyboard and restores focus after cancel', async () => {
    mockServer.use(mockApi.get('/notifiers', 200, { items: [] }));
    const view = renderScreen(
      <CreateDatabaseNotifiersStep
        workspaceId="workspace-id"
        initialNotifierIds={[]}
        onContinue={() => {}}
      />,
    );
    const selector = await screen.findByRole('combobox', { name: 'Notifiers' });

    expect(screen.queryByRole('button', { name: 'Create new notifier' })).toBeNull();

    await view.user.click(selector);
    await view.user.keyboard('{Tab}');
    const createButton = await screen.findByRole('button', { name: 'Create new notifier' });

    expect(document.activeElement).toBe(createButton);
    expect(document.querySelector('.ant-empty-image')).toBeNull();

    await view.user.keyboard('{Enter}');
    const dialog = within(await screen.findByRole('dialog', { name: 'Add notifier' }));
    await view.user.click(dialog.getByRole('button', { name: 'Close' }));

    await waitFor(() => expect(document.activeElement).toBe(selector));
    expect(selector.getAttribute('aria-expanded')).toBe('false');
  });

  it('offers Back and Complete while the notifier list loads', () => {
    const notifiers = mockApi.pending('get', '/notifiers');
    mockServer.use(notifiers.handler);
    const view = renderScreen(
      <CreateDatabaseNotifiersStep
        workspaceId="workspace-id"
        initialNotifierIds={[]}
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(document.querySelector('.ant-spin')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Back' })).toHaveProperty('disabled', false);
    expect(screen.getByRole('button', { name: 'Complete' })).toHaveProperty('disabled', false);

    view.unmount();
    notifiers.cancel();
  });
});

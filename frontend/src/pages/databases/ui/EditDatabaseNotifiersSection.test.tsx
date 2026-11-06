// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
  databaseApi,
} from '@/entities/databases';
import { EditDatabaseNotifiersSection } from '@/pages/databases/ui/EditDatabaseNotifiersSection';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: [],
};
const alertsNotifier = {
  id: 'alerts-id',
  workspaceId: database.workspaceId,
  name: 'Alerts',
  lastSendError: null,
  settings: { type: 'DISCORD' as const },
};

const renderSection = async (onSaved: (database: Database) => void = () => {}) => {
  mockServer.use(mockApi.get('/notifiers', 200, { items: [alertsNotifier] }));
  renderScreen(
    <EditDatabaseNotifiersSection database={database} onCancel={() => {}} onSaved={onSaved} />,
  );

  return screen.findByRole('combobox', { name: 'Notifiers' });
};

const toggleAlertsNotifier = async (combobox: HTMLElement) => {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const alertsOption = document.querySelector('.ant-select-item-option[title="Alerts"]');
    expect(alertsOption).toBeTruthy();
    return alertsOption!;
  });

  fireEvent.click(option);
};

describe('EditDatabaseNotifiersSection', () => {
  it('offers Save only while the selection differs from the saved one', async () => {
    const combobox = await renderSection();

    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();

    await toggleAlertsNotifier(combobox);

    expect(await screen.findByRole('button', { name: 'Save' })).toBeTruthy();

    await toggleAlertsNotifier(combobox);

    await waitFor(() => expect(screen.queryByRole('button', { name: 'Save' })).toBeNull());
  });

  it('disables Cancel during a pending save', async () => {
    const saveCall = vi.spyOn(databaseApi, 'updateDatabase');
    const save = mockApi.delayed({
      method: 'put',
      path: '/databases/{databaseId}',
      status: 200,
      body: { ...database, notifierIds: [alertsNotifier.id] },
    });
    mockServer.use(save.handler);
    const onSaved = vi.fn();
    const combobox = await renderSection(onSaved);

    await toggleAlertsNotifier(combobox);
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
    await waitFor(() => expect(saveCall).toHaveBeenCalled());

    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', true);

    await releaseAndSettle(saveCall, save.release);

    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it('offers Cancel while the notifier list loads', () => {
    const notifiers = mockApi.pending('get', '/notifiers');
    mockServer.use(notifiers.handler);
    const view = renderScreen(
      <EditDatabaseNotifiersSection database={database} onCancel={() => {}} onSaved={() => {}} />,
    );

    expect(document.querySelector('.ant-spin')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', false);

    view.unmount();
    notifiers.cancel();
  });

  it('shows a refused save in the form with the selection and reports no save', async () => {
    const onSaved = vi.fn();
    mockServer.use(
      mockApi.put('/databases/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );
    const combobox = await renderSection(onSaved);

    await toggleAlertsNotifier(combobox);
    fireEvent.click(await screen.findByRole('button', { name: 'Save' }));

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(combobox.closest('.ant-select')!.textContent).toContain('Alerts');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveProperty('disabled', false);
    expect(onSaved).not.toHaveBeenCalled();
  });

  it('shows a failed notifier load with a retry that lists the notifiers and keeps the selection', async () => {
    mockServer.use(mockApi.get('/notifiers', 500, buildProblem('internal', 500)));
    const view = renderScreen(
      <EditDatabaseNotifiersSection
        database={{ ...database, notifierIds: [alertsNotifier.id] }}
        onCancel={() => {}}
        onSaved={() => {}}
      />,
    );

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByRole('alert')).toBeTruthy();

    mockServer.use(mockApi.get('/notifiers', 200, { items: [alertsNotifier] }));
    await view.user.click(retryButton);

    const combobox = await screen.findByRole('combobox', { name: 'Notifiers' });

    await waitFor(() => expect(combobox.closest('.ant-select')!.textContent).toContain('Alerts'));
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });
});

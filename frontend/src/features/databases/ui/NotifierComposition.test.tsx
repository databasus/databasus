// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../entities/databases';
import type { Notifier } from '../../../entities/notifiers';
import { API_BASE_PATH } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { DatabaseTransferDialogComponent } from './DatabaseTransferDialogComponent';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';

const server = setUpMockApi();
const database: Database = {
  id: 'database-id',
  workspaceId: 'source-id',
  name: 'Orders',
  settings: createDefaultDatabaseSettings(DatabaseType.MYSQL),
  notifierIds: ['source-notifier'],
};
const notifier: Notifier = {
  id: 'created-id',
  workspaceId: 'target-id',
  name: 'Created SMTP',
  lastSendError: null,
  settings: {
    type: 'EMAIL',
    targetEmail: 'ops@example.com',
    smtpHost: 'smtp.example.com',
    smtpPort: 587,
    security: 'STARTTLS',
    isInsecureSkipVerify: false,
  },
};

const chooseOption = async (combobox: HTMLElement, label: string) => {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const matchingOption = Array.from(document.querySelectorAll('.ant-select-item-option')).find(
      (candidate) => candidate.textContent === label,
    );
    expect(matchingOption).toBeTruthy();
    return matchingOption as HTMLElement;
  });
  fireEvent.click(option);
};

describe('Database notifier creation composition', () => {
  it.each(['selection', 'transfer'] as const)(
    'preserves newer edits and the created notifier when the %s parent refreshes and editing is cancelled',
    async (composition) => {
      const creation = mockApi.delayed({
        method: 'post',
        path: '/notifiers',
        status: 201,
        body: notifier,
      });
      let listedNotifiers: Notifier[] = [];
      let listRequests = 0;
      server.use(
        creation.handler,
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        http.get(`*${API_BASE_PATH}/notifiers`, () => {
          listRequests += 1;
          return HttpResponse.json({ items: listedNotifiers });
        }),
        mockApi.get('/workspaces', 200, {
          items: [
            {
              id: 'target-id',
              name: 'Target workspace',
              currentUserRole: 'WORKSPACE_OWNER',
              createdAt: '2026-10-01T00:00:00Z',
            },
          ],
        }),
        mockApi.get('/storages', 200, { items: [] }),
      );
      const view = renderScreen(
        composition === 'selection' ? (
          <EditDatabaseNotifiersComponent
            database={database}
            workspaceId="target-id"
            isShowBackButton={false}
            onBack={() => {}}
            onCancel={() => {}}
            isShowSaveOnlyForUnsaved={false}
            isSaveToApi={false}
            onSaved={() => {}}
          />
        ) : (
          <DatabaseTransferDialogComponent
            database={database}
            onClose={() => {}}
            onTransferred={() => {}}
          />
        ),
        { locale: 'en' },
      );
      if (composition === 'transfer') {
        await chooseOption(await screen.findByRole('combobox'), 'Target workspace');
        await waitFor(() => expect(screen.getAllByRole('combobox')).toHaveLength(3));
      }
      const selectors = await screen.findAllByRole('combobox');
      await chooseOption(selectors[selectors.length - 1], 'Create new notifier');
      const editor = await screen.findByRole('region', { name: 'Notifier settings' });
      const dialog = within(editor.closest('[role=dialog]') as HTMLElement);
      await view.user.click(dialog.getByRole('combobox', { name: 'Type' }));
      await view.user.click(screen.getByRole('option', { name: 'Email' }));
      const nameInput = dialog.getByRole('textbox', { name: 'Name' });
      fireEvent.change(nameInput, { target: { value: notifier.name } });
      fireEvent.change(dialog.getByLabelText('Target email'), {
        target: { value: 'ops@example.com' },
      });
      fireEvent.change(dialog.getByLabelText('SMTP host'), {
        target: { value: 'smtp.example.com' },
      });
      fireEvent.change(dialog.getByLabelText('SMTP port'), { target: { value: '587' } });
      fireEvent.click(dialog.getByRole('button', { name: /Send test notification/ }));
      fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));
      await waitFor(() =>
        expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
      );
      fireEvent.change(nameInput, { target: { value: 'Newer local name' } });
      const previousListRequests = listRequests;
      listedNotifiers = [notifier];
      creation.release();
      await waitFor(() => expect(listRequests).toBeGreaterThan(previousListRequests));
      await waitFor(() => expect(dialog.queryByRole('combobox', { name: 'Type' })).toBeNull());
      expect(dialog.getByDisplayValue('Newer local name')).toBe(nameInput);
      fireEvent.click(dialog.getByRole('button', { name: 'Close' }));
      await waitFor(() =>
        expect(screen.queryByRole('region', { name: 'Notifier settings' })).toBeNull(),
      );
      const remainingSelectors = screen.getAllByRole('combobox');
      await chooseOption(remainingSelectors[remainingSelectors.length - 1], notifier.name);
      expect(
        server
          .getRequests()
          .filter(
            (request) =>
              request.method === 'POST' && new URL(request.url).pathname.endsWith('/notifiers'),
          ),
      ).toHaveLength(1);
    },
  );
});

// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import type { Notifier } from '@/entities/notifiers';
import { CreateDatabaseNotifiersStep } from '@/pages/databases/create-wizard/ui/CreateDatabaseNotifiersStep';
import { DatabaseTransferDialogComponent } from '@/pages/databases/settings/ui/DatabaseTransferDialogComponent';
import { EditDatabaseNotifiersSection } from '@/pages/databases/settings/ui/EditDatabaseNotifiersSection';
import { API_BASE_PATH } from '@/shared/api';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});
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
  if (combobox.getAttribute('aria-expanded') !== 'true') fireEvent.click(combobox);

  const option = await waitFor(() => {
    const matchingOption = Array.from(screen.queryAllByRole('option')).find(
      (candidate) => candidate.textContent === label,
    );

    expect(matchingOption).toBeTruthy();

    return matchingOption as HTMLElement;
  });

  fireEvent.click(option);
};

const fillNotifierCreation = async (
  interaction: ReturnType<typeof renderScreen>['user'],
  composition: 'selection' | 'transfer',
) => {
  if (composition === 'selection')
    await interaction.click(await screen.findByRole('combobox', { name: 'Notifiers' }));

  await interaction.click(await screen.findByRole('button', { name: 'Create new notifier' }));
  const editor = await screen.findByRole('region', { name: 'Notifier settings' });
  const dialog = within(editor.closest('[role=dialog]') as HTMLElement);

  await interaction.click(dialog.getByRole('combobox', { name: 'Type' }));
  await interaction.click(screen.getByRole('option', { name: 'Email' }));
  fireEvent.change(dialog.getByRole('textbox', { name: 'Name' }), {
    target: { value: notifier.name },
  });
  fireEvent.change(dialog.getByLabelText('Target email'), {
    target: { value: 'ops@example.com' },
  });
  fireEvent.change(dialog.getByLabelText('SMTP host'), {
    target: { value: 'smtp.example.com' },
  });
  fireEvent.change(dialog.getByLabelText('SMTP port'), { target: { value: '587' } });

  return dialog;
};

describe('Database notifier creation composition', () => {
  it('filters notifier choices without losing selected values and saves the combined selection', async () => {
    const existingNotifier = {
      ...notifier,
      id: 'source-notifier',
      name: 'Existing alerts',
      workspaceId: database.workspaceId,
    };
    const additionalNotifier = { ...existingNotifier, id: 'audit-id', name: 'Audit destination' };
    const onSaved = vi.fn();
    server.use(
      mockApi.get('/notifiers', 200, { items: [existingNotifier, additionalNotifier] }),
      mockApi.put('/databases/{databaseId}', 200, {
        ...database,
        notifierIds: [existingNotifier.id, additionalNotifier.id],
      }),
    );

    const { user } = renderScreen(
      <EditDatabaseNotifiersSection database={database} onCancel={() => {}} onSaved={onSaved} />,
    );
    const selection = await screen.findByRole('combobox', { name: 'Notifiers' });

    await user.click(selection);
    const search = screen.getByRole('textbox', { name: 'Notifiers' });
    await user.type(search, 'aUdIt');

    expect(screen.queryByRole('option', { name: existingNotifier.name })).toBeNull();
    expect(screen.getByRole('button', { name: `Remove ${existingNotifier.name}` })).toBeTruthy();

    await user.click(screen.getByRole('option', { name: additionalNotifier.name }));
    await user.clear(search);

    expect(
      screen.getByRole('option', { name: existingNotifier.name }).getAttribute('aria-selected'),
    ).toBe('true');
    expect(
      screen.getByRole('option', { name: additionalNotifier.name }).getAttribute('aria-selected'),
    ).toBe('true');

    await user.keyboard('{Escape}');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1));

    const update = server.getRequests().find((request) => request.method === 'PUT')!;
    expect(await update.json()).toMatchObject({
      notifierIds: [existingNotifier.id, additionalNotifier.id],
    });
  });

  it.each(
    (['selection', 'transfer'] as const).flatMap((composition) =>
      (['close', 'workspace', 'unmount'] as const).flatMap((contextChange) =>
        [false, true].map((hasFailed) => ({ composition, contextChange, hasFailed })),
      ),
    ),
  )(
    'ignores pending creation when $composition changes $contextChange with failure=$hasFailed',
    async ({ composition, contextChange, hasFailed }) => {
      const creation = mockApi.pending('post', '/notifiers');
      const fetchObservation = vi.spyOn(globalThis, 'fetch');
      const currentNotifier = { ...notifier, id: 'current-notifier', name: 'Current alerts' };
      const onSaved = vi.fn();
      const onTransferred = vi.fn();
      const editedDatabase = { ...database, notifierIds: [currentNotifier.id] };

      server.use(
        creation.handler,
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) =>
          HttpResponse.json({
            items: [
              {
                ...currentNotifier,
                workspaceId: new URL(request.url).searchParams.get('workspaceId'),
              },
            ],
          }),
        ),
        mockApi.get('/databases', 200, { items: [editedDatabase] }),
        mockApi.get('/storages', 200, { items: [] }),
        mockApi.get('/workspaces', 200, {
          items: ['target-id', 'other-id'].map((id) => ({
            id,
            name: id === 'target-id' ? 'Target workspace' : 'Other workspace',
            currentUserRole: 'WORKSPACE_OWNER' as const,
            createdAt: '2026-10-01T00:00:00Z',
          })),
        }),
      );
      const renderParent = (workspaceId = 'target-id') =>
        composition === 'selection' ? (
          <CreateDatabaseNotifiersStep
            workspaceId={workspaceId}
            initialNotifierIds={editedDatabase.notifierIds}
            onContinue={onSaved}
          />
        ) : (
          <DatabaseTransferDialogComponent
            database={editedDatabase}
            onClose={() => {}}
            onTransferred={onTransferred}
          />
        );

      const view = renderScreen(renderParent(), { isStrictMode: true });

      if (composition === 'transfer') {
        await chooseOption(
          await screen.findByRole('combobox', { name: 'Target workspace' }),
          'Target workspace',
        );
        await chooseOption(
          await screen.findByRole('combobox', { name: 'Notifiers (optional)' }),
          currentNotifier.name,
        );
      }
      const dialog = await fillNotifierCreation(view.user, composition);

      fireEvent.click(dialog.getByRole('button', { name: 'Send test notification' }));
      fireEvent.click(await dialog.findByRole('button', { name: 'Save' }));

      await waitFor(() =>
        expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
      );

      const creatingFetch =
        fetchObservation.mock.results[fetchObservation.mock.results.length - 1].value;

      if (contextChange === 'unmount') view.unmount();
      else if (contextChange === 'close')
        fireEvent.click(dialog.getByRole('button', { name: 'Close' }));
      else if (composition === 'selection') view.rerender(renderParent('other-id'));
      else
        await chooseOption(
          screen.getByRole('combobox', { name: 'Target workspace' }),
          'Other workspace',
        );

      await waitFor(() =>
        expect(screen.queryByRole('region', { name: 'Notifier settings' })).toBeNull(),
      );

      if (contextChange !== 'unmount') {
        await screen.findByRole(composition === 'selection' ? 'combobox' : 'button', {
          name: composition === 'selection' ? 'Notifiers' : 'Create new notifier',
        });
        if (contextChange === 'workspace' && composition === 'transfer')
          await chooseOption(
            screen.getByRole('combobox', { name: 'Notifiers (optional)' }),
            currentNotifier.name,
          );
      }

      const previousListRequests = server
        .getRequests()
        .filter((request) => request.method === 'GET').length;

      await act(async () => {
        creation.respond(
          hasFailed
            ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
            : HttpResponse.json(notifier, { status: 201 }),
        );
        await creatingFetch;
      });

      expect(onSaved).not.toHaveBeenCalled();
      expect(onTransferred).not.toHaveBeenCalled();
      expect(server.getRequests().filter((request) => request.method === 'GET')).toHaveLength(
        previousListRequests,
      );
      expect(
        screen.queryByText('An internal server error occurred. Please try again later.'),
      ).toBeNull();

      if (contextChange !== 'unmount') {
        const selector = screen.getByRole('combobox', {
          name: composition === 'selection' ? 'Notifiers' : 'Notifiers (optional)',
        });
        expect(selector.closest('div')!.textContent).toContain(currentNotifier.name);
        if (selector.getAttribute('aria-expanded') !== 'true') fireEvent.click(selector);
        await waitFor(() => expect(screen.queryAllByRole('option')[0]).toBeTruthy());
        expect(screen.queryByRole('option', { name: notifier.name })).toBeNull();
      }

      fetchObservation.mockRestore();
    },
  );

  it.each(
    (['selection', 'transfer'] as const).flatMap((composition) =>
      [false, true].map((hasFailedRefresh) => ({ composition, hasFailedRefresh })),
    ),
  )(
    'retains persisted choices when $composition cancels newer edits with refresh failure=$hasFailedRefresh',
    async ({ composition, hasFailedRefresh }) => {
      const creation = mockApi.delayed({
        method: 'post',
        path: '/notifiers',
        status: 201,
        body: notifier,
      });
      let listedNotifiers: Notifier[] = [];
      let listRequests = 0;
      let hasPersisted = false;

      server.use(
        creation.handler,
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        http.get(`*${API_BASE_PATH}/notifiers`, () => {
          listRequests += 1;

          if (hasPersisted && hasFailedRefresh)
            return HttpResponse.json(buildProblem('internal', 500), { status: 500 });

          return HttpResponse.json({ items: listedNotifiers });
        }),
        mockApi.get('/workspaces', 200, {
          items: [
            {
              id: 'target-id',
              name: 'Target workspace',
              currentUserRole: 'WORKSPACE_OWNER' as const,
              createdAt: '2026-10-01T00:00:00Z',
            },
          ],
        }),
        mockApi.get('/storages', 200, { items: [] }),
      );

      const view = renderScreen(
        composition === 'selection' ? (
          <CreateDatabaseNotifiersStep
            workspaceId="target-id"
            initialNotifierIds={database.notifierIds}
            onContinue={() => {}}
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
      const dialog = await fillNotifierCreation(view.user, composition);
      const nameInput = dialog.getByRole('textbox', { name: 'Name' });

      fireEvent.click(dialog.getByRole('button', { name: /Send test notification/ }));
      fireEvent.click(await dialog.findByRole('button', { name: /Save/ }));

      await waitFor(() =>
        expect(server.getRequests().filter((request) => request.method === 'POST')).toHaveLength(2),
      );

      fireEvent.change(nameInput, { target: { value: 'Newer local name' } });
      const previousListRequests = listRequests;
      listedNotifiers = [notifier];
      hasPersisted = true;

      creation.release();

      await waitFor(() => expect(listRequests).toBeGreaterThan(previousListRequests));
      await waitFor(() => expect(dialog.queryByRole('combobox', { name: 'Type' })).toBeNull());
      expect(dialog.getByDisplayValue('Newer local name')).toBe(nameInput);

      if (hasFailedRefresh)
        await screen.findByText('An internal server error occurred. Please try again later.');

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

  it.each(['selection', 'transfer'] as const)(
    'automatically selects the created notifier when the %s editor completes',
    async (composition) => {
      const targetWorkspaceId = composition === 'selection' ? database.workspaceId : 'target-id';
      const createdNotifier = { ...notifier, workspaceId: targetWorkspaceId };
      const existingSource = {
        ...notifier,
        id: 'source-notifier',
        workspaceId: database.workspaceId,
        name: 'Source alerts',
      };
      const existingTarget = { ...notifier, id: 'existing-target', name: 'Existing target alerts' };
      const targetStorage = {
        id: 'target-storage',
        workspaceId: targetWorkspaceId,
        name: 'Target disk',
        settings: { type: 'LOCAL' as const },
      };

      server.use(
        mockApi.get('/workspaces', 200, {
          items: [
            {
              id: 'target-id',
              name: 'Target workspace',
              currentUserRole: 'WORKSPACE_OWNER' as const,
              createdAt: '2026-10-01T00:00:00Z',
            },
          ],
        }),
        mockApi.get('/storages', 200, { items: [targetStorage] }),
        http.get(`*${API_BASE_PATH}/notifiers`, ({ request }) =>
          HttpResponse.json({
            items:
              new URL(request.url).searchParams.get('workspaceId') === database.workspaceId
                ? [existingSource]
                : [existingTarget],
          }),
        ),
        mockApi.get('/databases', 200, { items: [database] }),
        mockApi.post('/notifier-tests', 200, { isDelivered: true }),
        mockApi.post('/notifiers', 201, createdNotifier),
        mockApi.put('/databases/{databaseId}', 200, {
          ...database,
          notifierIds: [existingSource.id, createdNotifier.id],
        }),
        mockApi.post('/databases/{databaseId}/transfer', 204, undefined),
      );

      const onSaved = vi.fn();
      const onTransferred = vi.fn();
      const view = renderScreen(
        composition === 'selection' ? (
          <EditDatabaseNotifiersSection database={database} onCancel={() => {}} onSaved={onSaved} />
        ) : (
          <DatabaseTransferDialogComponent
            database={database}
            onClose={() => {}}
            onTransferred={onTransferred}
          />
        ),
      );

      if (composition === 'transfer') {
        await chooseOption(await screen.findByRole('combobox'), 'Target workspace');
        await chooseOption(
          await screen.findByRole('combobox', { name: 'Storage' }),
          targetStorage.name,
        );
        await chooseOption(
          screen.getByRole('combobox', { name: 'Notifiers (optional)' }),
          existingTarget.name,
        );
      }
      const dialog = await fillNotifierCreation(view.user, composition);

      fireEvent.click(dialog.getByRole('button', { name: 'Send test notification' }));
      fireEvent.click(await dialog.findByRole('button', { name: 'Save' }));

      await waitFor(() =>
        expect(screen.queryByRole('region', { name: 'Notifier settings' })).toBeNull(),
      );

      const choice = screen
        .getByRole('combobox', {
          name: composition === 'selection' ? 'Notifiers' : 'Notifiers (optional)',
        })
        .closest('div')!;
      expect(choice.textContent).toContain(createdNotifier.name);
      expect(choice.textContent).toContain(
        composition === 'selection' ? existingSource.name : existingTarget.name,
      );

      fireEvent.click(
        screen.getByRole('button', { name: composition === 'selection' ? 'Save' : 'Transfer' }),
      );

      await waitFor(() =>
        expect(composition === 'selection' ? onSaved : onTransferred).toHaveBeenCalled(),
      );

      const request = server
        .getRequests()
        .find((request) =>
          composition === 'selection'
            ? request.method === 'PUT' &&
              new URL(request.url).pathname.endsWith(`/databases/${database.id}`)
            : request.method === 'POST' && new URL(request.url).pathname.endsWith('/transfer'),
        )!;
      const submission = await request.json();
      if (composition === 'selection')
        expect(submission.notifierIds).toEqual([existingSource.id, createdNotifier.id]);
      else
        expect(submission).toMatchObject({
          targetWorkspaceId,
          targetNotifierIds: [existingTarget.id, createdNotifier.id],
          storage: { type: 'SWITCH_TO_STORAGE', storageId: targetStorage.id },
        });
    },
  );
});

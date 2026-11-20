// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import type { Storage } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import {
  buildSettings,
  buildStatus,
} from '@/pages/instance-settings/ui/instance-backups/instanceBackups.fixture';
import { InstanceBackupsSection } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupsSection';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceBackups;

const createdWorkspace: Workspace = {
  id: 'cccccccc-cccc-4ccc-8ccc-cccccccccccc',
  name: 'Backups',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-11-14T10:00:00Z',
};
const existingWorkspace: Workspace = {
  id: 'dddddddd-dddd-4ddd-8ddd-dddddddddddd',
  name: 'Production',
  currentUserRole: 'WORKSPACE_OWNER',
  createdAt: '2026-10-01T00:00:00Z',
};
const createdStorage: Storage = {
  id: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee',
  workspaceId: createdWorkspace.id,
  name: 'Databasus archive',
  settings: { type: 'LOCAL' },
};

const useSectionResponses = () =>
  server.use(
    mockApi.get('/instance-backup-settings', 200, buildSettings()),
    mockApi.get('/instance-backup-statuses', 200, buildStatus()),
    mockApi.get('/instance-backup-destinations', 200, { items: [] }),
    mockApi.get('/instance-backups', 200, { items: [] }),
    mockApi.post('/storage-tests', 200, { isConnected: true }),
  );

const startSetup = async (user: ReturnType<typeof renderScreen>['user']) => {
  await user.click(await screen.findByRole('button', { name: text.setup.start }));
};

const renderSetup = async (workspaces: Workspace[], onWorkspaceCreated = vi.fn()) => {
  activateSession({ token: 'root-session', userId: 'root' });
  const view = renderScreen(
    <InstanceBackupsSection
      workspaces={workspaces}
      onWorkspaceCreated={onWorkspaceCreated}
      onConfigureSmtp={vi.fn()}
    />,
  );
  await startSetup(view.user);

  return view;
};

const SectionWithWorkspaces = ({ initialWorkspaces }: { initialWorkspaces: Workspace[] }) => {
  const [workspaces, setWorkspaces] = useState(initialWorkspaces);

  return (
    <InstanceBackupsSection
      workspaces={workspaces}
      onWorkspaceCreated={(workspace) => setWorkspaces((current) => [...current, workspace])}
      onConfigureSmtp={vi.fn()}
    />
  );
};

const editRetention = async (user: ReturnType<typeof renderScreen>['user']) => {
  const retention = await screen.findByRole('textbox', { name: text.settings.retentionDays });
  await user.clear(retention);
  await user.type(retention, '45');
};

const getRetentionValue = () =>
  (screen.getByRole('textbox', { name: text.settings.retentionDays }) as HTMLInputElement).value;

const openCreation = async (user: ReturnType<typeof renderScreen>['user']) => {
  await user.click(await screen.findByRole('combobox', { name: text.destinations.picker }));
  await user.click(await screen.findByRole('button', { name: text.destinations.create }));

  return within(await screen.findByRole('dialog', { name: text.createDestination.title }));
};

const saveStorage = async (
  user: ReturnType<typeof renderScreen>['user'],
  dialog: ReturnType<typeof within>,
) => {
  await user.type(dialog.getByRole('textbox', { name: 'Name' }), createdStorage.name);
  await user.click(dialog.getByRole('button', { name: 'Test connection' }));
  await user.click(await dialog.findByRole('button', { name: 'Save' }));
};

const countRequests = (method: string, pathname: string) =>
  server
    .getRequests()
    .filter(
      (request) => request.method === method && new URL(request.url).pathname.endsWith(pathname),
    ).length;

describe('CreateDestinationDialog', () => {
  it('creates a workspace and a storage and selects it without leaving the page when no workspace exists', async () => {
    useSectionResponses();
    server.use(
      mockApi.post('/workspaces', 201, createdWorkspace),
      mockApi.post('/storages', 201, createdStorage),
    );
    const onWorkspaceCreated = vi.fn();
    const { user } = await renderSetup([], onWorkspaceCreated);

    const dialog = await openCreation(user);
    await user.click(dialog.getByRole('button', { name: text.createDestination.createWorkspace }));
    await saveStorage(user, dialog);

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
    );
    expect(onWorkspaceCreated).toHaveBeenCalledWith(createdWorkspace);
    expect(screen.getByText(createdStorage.name)).toBeTruthy();
    expect(screen.getByRole('button', { name: text.settings.save })).toBeTruthy();
  });

  it.each([
    { scenario: 'a new workspace', initialWorkspaces: [] as Workspace[] },
    { scenario: 'an existing workspace', initialWorkspaces: [existingWorkspace] },
  ])(
    'keeps the edited draft beside the new destination after creating it in $scenario',
    async ({ initialWorkspaces }) => {
      useSectionResponses();
      server.use(
        mockApi.post('/workspaces', 201, createdWorkspace),
        mockApi.post('/storages', 201, createdStorage),
      );
      activateSession({ token: 'root-session', userId: 'root' });
      const { user } = renderScreen(
        <SectionWithWorkspaces initialWorkspaces={initialWorkspaces} />,
      );

      await startSetup(user);
      await editRetention(user);
      const dialog = await openCreation(user);
      if (initialWorkspaces.length > 0)
        await user.click(dialog.getByRole('radio', { name: text.createDestination.newWorkspace }));
      const workspaceName = dialog.getByRole('textbox', {
        name: DICTIONARIES.en.workspaces.fields.name,
      });
      await user.clear(workspaceName);
      await user.type(workspaceName, createdWorkspace.name);
      await user.click(
        dialog.getByRole('button', { name: text.createDestination.createWorkspace }),
      );
      await saveStorage(user, dialog);

      await waitFor(() =>
        expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
      );
      expect(screen.getByText(createdStorage.name)).toBeTruthy();
      expect(getRetentionValue()).toBe('45');
    },
  );

  it('retries the storage in the workspace it already created when the storage step fails', async () => {
    useSectionResponses();
    let storageAttempts = 0;
    server.use(
      mockApi.post('/workspaces', 201, createdWorkspace),
      http.post(`*${API_BASE_PATH}/storages`, () => {
        storageAttempts += 1;
        return storageAttempts === 1
          ? HttpResponse.json(buildProblem('internal', 500), { status: 500 })
          : HttpResponse.json(createdStorage, { status: 201 });
      }),
    );
    const { user } = await renderSetup([]);

    const dialog = await openCreation(user);
    await user.click(dialog.getByRole('button', { name: text.createDestination.createWorkspace }));
    await saveStorage(user, dialog);
    await dialog.findByText(DICTIONARIES.en.errors.internal);
    expect((dialog.getByRole('textbox', { name: 'Name' }) as HTMLInputElement).value).toBe(
      createdStorage.name,
    );

    await user.click(dialog.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
    );
    expect(countRequests('POST', '/workspaces')).toBe(1);
    expect(screen.getByText(createdStorage.name)).toBeTruthy();
  });

  it('creates the storage in a chosen existing workspace', async () => {
    useSectionResponses();
    let storageWorkspaceId: unknown;
    server.use(
      http.post(`*${API_BASE_PATH}/storages`, async ({ request }) => {
        storageWorkspaceId = ((await request.json()) as { workspaceId: string }).workspaceId;
        return HttpResponse.json(
          { ...createdStorage, workspaceId: existingWorkspace.id },
          { status: 201 },
        );
      }),
    );
    const { user } = await renderSetup([existingWorkspace]);

    const dialog = await openCreation(user);
    await user.click(dialog.getByRole('button', { name: text.createDestination.continue }));
    await saveStorage(user, dialog);

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
    );
    expect(storageWorkspaceId).toBe(existingWorkspace.id);
    expect(countRequests('POST', '/workspaces')).toBe(0);
  });

  it('returns focus to the picker and changes nothing when creation is cancelled', async () => {
    useSectionResponses();
    const { user } = await renderSetup([existingWorkspace]);

    const dialog = await openCreation(user);
    await user.click(dialog.getByRole('button', { name: DICTIONARIES.en.common.actions.cancel }));

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole('combobox', { name: text.destinations.picker }),
      ),
    );
    expect(screen.getByText(text.destinations.none)).toBeTruthy();
    expect(screen.getByRole('button', { name: text.settings.save }).hasAttribute('disabled')).toBe(
      true,
    );
  });

  it('leaves a created storage inactive when the unsaved setup is cancelled', async () => {
    useSectionResponses();
    server.use(
      mockApi.post('/storages', 201, { ...createdStorage, workspaceId: existingWorkspace.id }),
    );
    const { user } = await renderSetup([existingWorkspace]);

    const dialog = await openCreation(user);
    await user.click(dialog.getByRole('button', { name: text.createDestination.continue }));
    await saveStorage(user, dialog);
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: text.createDestination.title })).toBeNull(),
    );
    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.cancel }));

    expect(await screen.findByRole('button', { name: text.setup.start })).toBeTruthy();
    expect(countRequests('PUT', '/instance-backup-settings')).toBe(0);
  });
});

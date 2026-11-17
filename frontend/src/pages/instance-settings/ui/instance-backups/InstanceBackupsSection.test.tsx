// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  buildBackup,
  buildSettings,
  buildStatus,
  localChoice,
  s3Choice,
} from '@/pages/instance-settings/ui/instance-backups/instanceBackups.fixture';
import { InstanceBackupsSection } from '@/pages/instance-settings/ui/instance-backups/InstanceBackupsSection';
import { InstanceSettingsPage } from '@/pages/instance-settings/ui/InstanceSettingsPage';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { LanguageThemeControlComponent } from '@/shared/ui';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceBackups;

const useSectionResponses = ({
  settings = buildSettings(),
  status = buildStatus(),
  backups = [] as ReturnType<typeof buildBackup>[],
} = {}) =>
  server.use(
    mockApi.get('/instance-backup-settings', 200, settings),
    mockApi.get('/instance-backup-statuses', 200, status),
    mockApi.get('/instance-backup-destinations', 200, { items: [s3Choice, localChoice] }),
    mockApi.get('/instance-backups', 200, { items: backups }),
  );

const openSection = () => {
  activateSession({ token: 'root-session', userId: 'root' });

  return renderScreen(
    <InstanceBackupsSection
      workspaces={[]}
      onWorkspaceCreated={vi.fn()}
      onConfigureSmtp={vi.fn()}
    />,
  );
};

const selectDestination = async (
  user: ReturnType<typeof renderScreen>['user'],
  storageName: string,
) => {
  await user.click(await screen.findByRole('combobox', { name: text.destinations.picker }));
  await user.click(
    await screen.findByRole('option', {
      name: (accessibleName) => accessibleName.includes(storageName),
    }),
  );
  await user.keyboard('{Escape}');
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe('InstanceBackupsSection', () => {
  it('shows the section only to root admin', async () => {
    activateSession({ token: 'admin-session', userId: 'admin' });
    server.use(
      mockApi.get('/instance-settings', 200, {
        isAllowExternalRegistrations: false,
        isAllowMemberInvitations: false,
        isMemberAllowedToCreateWorkspaces: false,
        isTwoFactorAuthRequired: false,
        isEmailConfigured: true,
      }),
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.get('/audit-events', 200, { items: [] }),
    );
    useSectionResponses();

    const view = renderScreen(<InstanceSettingsPage isRootAdmin={false} />);
    await screen.findByRole('switch', {
      name: DICTIONARIES.en.settings.externalRegistrations.title,
    });

    expect(screen.queryByRole('heading', { name: text.title })).toBeNull();
    expect(server.getRequests().some((request) => request.url.includes('/instance-backup-'))).toBe(
      false,
    );

    view.unmount();
    renderScreen(<InstanceSettingsPage isRootAdmin />);

    expect(await screen.findByRole('heading', { name: text.title })).toBeTruthy();
  });

  it('shows the master-key status and copy instructions, and reads the key only when asked', async () => {
    useSectionResponses({ status: buildStatus({ isMasterKeyAcknowledged: true }) });
    server.use(
      mockApi.post('/master-key-exports', 201, {
        token: 'key-token',
        fileName: 'databasus-master-key.txt',
        expiresAt: '2999-01-01T00:00:00Z',
      }),
      http.get(`*${API_BASE_PATH}/master-key-downloads`, () =>
        HttpResponse.text('panel-master-key-0123456789'),
      ),
    );
    const { user } = openSection();
    const masterKey = DICTIONARIES.en.masterKey;

    expect(await screen.findByText(masterKey.statusSaved)).toBeTruthy();
    expect(screen.getByText(masterKey.host.description)).toBeTruthy();
    expect(screen.getByText('/databasus-data/secret.key')).toBeTruthy();
    expect(server.getRequests().some((request) => request.url.includes('/master-key-'))).toBe(
      false,
    );

    await user.click(screen.getByRole('button', { name: masterKey.saveAgain }));

    expect(await screen.findByText('pane••••••••••••6789')).toBeTruthy();
    expect(screen.getByRole('button', { name: masterKey.copy })).toBeTruthy();
    expect(screen.getByRole('button', { name: masterKey.download })).toBeTruthy();
  });

  it('asks for an emailed code in the master-key panel when two-factor sign-in is on', async () => {
    useSectionResponses({
      status: buildStatus({ isMasterKeyAcknowledged: false, isMasterKeyExportCodeRequired: true }),
    });
    const { user } = openSection();
    const masterKey = DICTIONARIES.en.masterKey;

    expect(await screen.findByText(masterKey.statusNotSaved)).toBeTruthy();
    await user.click(screen.getByRole('button', { name: masterKey.saveAgain }));

    expect(screen.getByRole('button', { name: masterKey.proof.sendCode })).toBeTruthy();
    expect(server.getRequests().some((request) => request.url.includes('/master-key-'))).toBe(
      false,
    );
  });

  it('retries the status after it failed to load', async () => {
    server.use(mockApi.get('/instance-backup-statuses', 500, buildProblem('internal', 500)));
    const { user } = openSection();

    await user.click(await screen.findByRole('button', { name: 'Retry' }));
    useSectionResponses();

    expect(await screen.findByText(text.health.notConfigured)).toBeTruthy();
  });

  it('requests the first backup and refreshes the status when the first destination is saved', async () => {
    useSectionResponses();
    let savedSettings: unknown;
    server.use(
      http.put(`*${API_BASE_PATH}/instance-backup-settings`, async ({ request }) => {
        savedSettings = await request.json();
        server.use(
          mockApi.get(
            '/instance-backup-statuses',
            200,
            buildStatus({
              health: 'INITIALIZING',
              destinations: [
                {
                  storageId: s3Choice.storageId,
                  storageName: s3Choice.storageName,
                  storageType: 'S3',
                  workspaceId: s3Choice.workspaceId,
                  workspaceName: s3Choice.workspaceName,
                  health: 'INITIALIZING',
                },
              ],
            }),
          ),
        );
        return HttpResponse.json(buildSettings({ storageIds: [s3Choice.storageId] }));
      }),
    );
    const { user } = openSection();

    await selectDestination(user, s3Choice.storageName);
    await user.click(screen.getByRole('button', { name: text.settings.save }));

    expect(await screen.findByText(text.settings.firstBackupRequested)).toBeTruthy();
    expect(savedSettings).toEqual(buildSettings({ storageIds: [s3Choice.storageId] }));
    expect((await screen.findAllByText(text.health.initializing)).length).toBeGreaterThan(0);
    expect(screen.queryByRole('button', { name: text.settings.save })).toBeNull();
  });

  it('keeps a newer retention edit when the save that preceded it is acknowledged', async () => {
    useSectionResponses();
    const save = mockApi.delayed({
      method: 'put',
      path: '/instance-backup-settings',
      status: 200,
      body: buildSettings({ retentionDays: 30, storageIds: [s3Choice.storageId] }),
    });
    server.use(save.handler);
    const { user } = openSection();

    await selectDestination(user, s3Choice.storageName);
    await user.click(screen.getByRole('button', { name: text.settings.save }));
    await waitFor(() =>
      expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1),
    );
    const retention = screen.getByRole('textbox', { name: text.settings.retentionDays });
    await user.clear(retention);
    await user.type(retention, '45');
    save.release();

    await screen.findByText(text.settings.firstBackupRequested);
    expect((retention as HTMLInputElement).value).toBe('45');
    expect(screen.getByRole('button', { name: text.settings.save })).toBeTruthy();
  });

  it('reports stopped backups when the last destination is removed', async () => {
    useSectionResponses({
      settings: buildSettings({ storageIds: [s3Choice.storageId] }),
      status: buildStatus({ health: 'HEALTHY' }),
    });
    server.use(
      http.put(`*${API_BASE_PATH}/instance-backup-settings`, () => {
        server.use(
          mockApi.get(
            '/instance-backup-statuses',
            200,
            buildStatus({
              health: 'STOPPED',
              incident: {
                id: '55555555-5555-4555-8555-555555555555',
                kind: 'STOPPED',
                startedAt: '2026-11-14T10:00:00Z',
                mailStatus: 'PENDING',
              },
            }),
          ),
        );
        return HttpResponse.json(buildSettings());
      }),
    );
    const { user } = openSection();

    await user.click(
      await screen.findByRole('button', {
        name: text.destinations.remove.replace('{{storage}}', s3Choice.storageName),
      }),
    );
    await user.click(screen.getByRole('button', { name: text.settings.save }));

    expect(await screen.findByText(text.settings.stopped)).toBeTruthy();
    expect(await screen.findByText(text.incidents.stopped)).toBeTruthy();
    expect(screen.getByText(text.destinations.none)).toBeTruthy();
  });

  it('keeps the draft and shows the refusal when saving fails', async () => {
    useSectionResponses();
    server.use(mockApi.put('/instance-backup-settings', 400, buildProblem('invalid_request', 400)));
    const { user } = openSection();

    await selectDestination(user, s3Choice.storageName);
    await user.click(screen.getByRole('button', { name: text.settings.save }));

    expect(await screen.findByText(DICTIONARIES.en.errors.invalidRequest)).toBeTruthy();
    expect(screen.getByText(s3Choice.storageName)).toBeTruthy();
  });

  it('warns that a local destination does not protect against losing the server', async () => {
    useSectionResponses();
    const { user } = openSection();

    await selectDestination(user, localChoice.storageName);

    expect(screen.getByText(text.destinations.localWarning)).toBeTruthy();
  });

  it('shows the refusal when a backup is already running', async () => {
    useSectionResponses({
      settings: buildSettings({ storageIds: [s3Choice.storageId] }),
      status: buildStatus({
        health: 'HEALTHY',
        destinations: [
          {
            storageId: s3Choice.storageId,
            storageName: s3Choice.storageName,
            workspaceId: s3Choice.workspaceId,
            workspaceName: s3Choice.workspaceName,
            health: 'HEALTHY',
          },
        ],
      }),
    });
    server.use(mockApi.post('/instance-backups', 409, buildProblem('already_running', 409)));
    const { user } = openSection();

    await user.click(await screen.findByRole('button', { name: text.history.start }));

    expect(await screen.findByText(DICTIONARIES.en.errors.alreadyRunning)).toBeTruthy();
  });

  it('follows the file link with a single-use token when a stored copy is downloaded', async () => {
    const followedLinks: { href: string; download: string }[] = [];
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      followedLinks.push({ href: this.href, download: this.download });
    });
    const backup = buildBackup();
    useSectionResponses({ backups: [backup] });
    server.use(
      mockApi.post('/instance-backups/{instanceBackupId}/download-tokens', 201, {
        token: 'single-use',
        fileName: 'databasus-backup.dbsbackup',
        expiresAt: '2026-11-14T10:05:00Z',
      }),
    );
    const { user } = openSection();

    await user.click(
      await screen.findByRole('button', {
        name: text.history.downloadFrom.replace('{{storage}}', s3Choice.storageName),
      }),
    );

    await waitFor(() => expect(followedLinks).toHaveLength(1));
    expect(followedLinks[0].href).toContain('/instance-backup-downloads?token=single-use');
    expect(followedLinks[0].download).toBe('databasus-backup.dbsbackup');
  });

  it('discards a status that arrives after the session changed', async () => {
    useSectionResponses();
    const status = mockApi.delayed({
      method: 'get',
      path: '/instance-backup-statuses',
      status: 200,
      body: buildStatus({
        health: 'FAILED',
        incident: {
          id: '66666666-6666-4666-8666-666666666666',
          kind: 'FAILED',
          startedAt: '2026-11-14T10:00:00Z',
          mailStatus: 'SENT',
        },
      }),
    });
    server.use(status.handler);
    openSection();
    await waitFor(() =>
      expect(
        server.getRequests().filter((request) => request.url.includes('/instance-backup-statuses')),
      ).toHaveLength(1),
    );

    server.use(mockApi.get('/instance-backup-statuses', 200, buildStatus({ health: 'HEALTHY' })));
    activateSession({ token: 'other-session', userId: 'other' });
    status.release();

    expect(await screen.findByText(text.health.healthy)).toBeTruthy();
    expect(screen.queryByText(text.incidents.failed)).toBeNull();
  });

  it('translates the status and keeps the draft when the language changes', async () => {
    useSectionResponses();
    const { user } = renderScreen(
      <>
        <LanguageThemeControlComponent />
        <InstanceBackupsSection
          workspaces={[]}
          onWorkspaceCreated={vi.fn()}
          onConfigureSmtp={vi.fn()}
        />
      </>,
    );
    activateSession({ token: 'root-session', userId: 'root' });

    await selectDestination(user, s3Choice.storageName);
    await user.click(screen.getByRole('button', { name: /Language:/ }));
    await user.click(
      await screen.findByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName }),
    );

    const section = screen.getByRole('region', { name: DICTIONARIES.ru.instanceBackups.title });
    expect(
      within(section).getByText(DICTIONARIES.ru.instanceBackups.health.notConfigured),
    ).toBeTruthy();
    expect(
      within(section).getByRole('button', { name: DICTIONARIES.ru.instanceBackups.settings.save }),
    ).toBeTruthy();
  });

  it('asks to configure the mail server and calls back when the action is chosen', async () => {
    useSectionResponses({ status: buildStatus({ isEmailConfigured: false }) });
    activateSession({ token: 'root-session', userId: 'root' });
    const onConfigureSmtp = vi.fn();
    const { user } = renderScreen(
      <InstanceBackupsSection
        workspaces={[]}
        onWorkspaceCreated={vi.fn()}
        onConfigureSmtp={onConfigureSmtp}
      />,
    );

    await user.click(await screen.findByRole('button', { name: text.smtp.configure }));

    expect(onConfigureSmtp).toHaveBeenCalledTimes(1);
  });

  it('directs the owner to the profile without Configure SMTP until the status reports a usable address', async () => {
    useSectionResponses({
      status: buildStatus({ isEmailConfigured: true, isRootEmailUsable: false }),
    });
    server.use(
      http.put(`*${API_BASE_PATH}/instance-backup-settings`, async ({ request }) => {
        server.use(mockApi.get('/instance-backup-statuses', 200, buildStatus()));

        return HttpResponse.json(await request.json());
      }),
    );
    const { user } = openSection();

    expect(await screen.findByText(text.smtp.addressMissing)).toBeTruthy();
    expect(screen.queryByRole('button', { name: text.smtp.configure })).toBeNull();

    const retention = await screen.findByRole('textbox', { name: text.settings.retentionDays });
    await user.clear(retention);
    await user.type(retention, '45');
    await user.click(screen.getByRole('button', { name: text.settings.save }));

    await waitFor(() => expect(screen.queryByText(text.smtp.addressMissing)).toBeNull());
    expect(screen.queryByRole('button', { name: text.smtp.configure })).toBeNull();
  });

  it('moves focus to the mail server section and keeps the draft when Configure SMTP is chosen', async () => {
    activateSession({ token: 'root-session', userId: 'root' });
    server.use(
      mockApi.get('/instance-settings', 200, {
        isAllowExternalRegistrations: false,
        isAllowMemberInvitations: false,
        isMemberAllowedToCreateWorkspaces: false,
        isTwoFactorAuthRequired: false,
        isEmailConfigured: false,
      }),
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.get('/audit-events', 200, { items: [] }),
    );
    useSectionResponses({ status: buildStatus({ isEmailConfigured: false }) });
    const scrollIntoView = vi.fn();
    HTMLElement.prototype.scrollIntoView = scrollIntoView;
    const { user } = renderScreen(<InstanceSettingsPage isRootAdmin />);

    await selectDestination(user, s3Choice.storageName);
    await user.click(await screen.findByRole('button', { name: text.smtp.configure }));

    const mailHeading = screen.getByRole('heading', {
      name: DICTIONARIES.en.settings.mailServer.title,
    });
    expect(document.activeElement).toBe(mailHeading);
    expect(scrollIntoView).toHaveBeenCalled();
    expect(screen.getByRole('button', { name: text.settings.save })).toBeTruthy();
  });
});

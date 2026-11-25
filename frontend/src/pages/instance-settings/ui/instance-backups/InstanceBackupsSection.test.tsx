// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InstanceBackupDestinationHealth } from '@/entities/instance-backups';
import {
  buildBackup,
  buildRecovery,
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
import { formatDateTime } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { mockSignedInAccount } from '@/shared/lib/test/mockSignedInAccount';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { getUserClockFormat } from '@/shared/time';
import { LanguageThemeControlComponent } from '@/shared/ui';

const server = setUpMockApi();
const dictionary = DICTIONARIES.en;
const text = dictionary.instanceBackups;

const s3Destination: InstanceBackupDestinationHealth = {
  storageId: s3Choice.storageId,
  storageName: s3Choice.storageName,
  storageType: 'S3',
  workspaceId: s3Choice.workspaceId,
  workspaceName: s3Choice.workspaceName,
  health: 'HEALTHY',
  lastSuccessfulSnapshotAt: '2026-11-14T10:00:05Z',
};

const localDestination: InstanceBackupDestinationHealth = {
  storageId: localChoice.storageId,
  storageName: localChoice.storageName,
  storageType: 'LOCAL',
  workspaceId: localChoice.workspaceId,
  workspaceName: localChoice.workspaceName,
  health: 'FAILED',
  lastError: 'disk is full',
};

dayjs.extend(relativeTime);

const shownTime = (instant: string) =>
  formatDateTime(instant, 'en-GB', getUserClockFormat().use12Hours);
const shownTimeWithRelativeTime = (instant: string) =>
  dictionary.common.dateTimeWithRelativeTime
    .replace('{{time}}', shownTime(instant))
    .replace('{{relativeTime}}', dayjs(instant).fromNow());
const healthyStatus = buildStatus({ health: 'HEALTHY', destinations: [s3Destination] });
const partialStatus = buildStatus({ health: 'PARTIAL', destinations: [s3Destination] });
const healthySettings = buildSettings({ storageIds: [s3Choice.storageId] });

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
    mockSignedInAccount(),
  );

const renderSection = (onConfigureSmtp = vi.fn()) =>
  renderScreen(
    <InstanceBackupsSection
      workspaces={[]}
      onWorkspaceCreated={vi.fn()}
      onConfigureSmtp={onConfigureSmtp}
    />,
  );

const openSection = () => {
  activateSession({ token: 'root-session', userId: 'root' });

  return renderSection();
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

const settingsWrites = () => server.getRequests().filter((request) => request.method === 'PUT');

const masterKeyRequests = () =>
  server.getRequests().filter((request) => request.url.includes('/master-key-'));

const keyExportRequests = () =>
  server
    .getRequests()
    .filter((request) => request.method === 'POST' && request.url.includes('/master-key-exports'));

const buildBackups = (count: number) =>
  Array.from({ length: count }, (_, index) => {
    const snapshotAt = new Date(Date.UTC(2026, 10, 14, 10 - index)).toISOString();

    return buildBackup({
      id: crypto.randomUUID(),
      snapshotAt,
      startedAt: snapshotAt,
      copies: [],
    });
  });

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
        twoFactorRequirement: 'OFF',
        isEmailConfigured: true,
      }),
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.get('/audit-events', 200, { items: [] }),
    );
    useSectionResponses();

    const view = renderScreen(<InstanceSettingsPage isRootAdmin={false} />);
    await screen.findByRole('switch', {
      name: dictionary.settings.externalRegistrations.title,
    });

    expect(screen.queryByRole('heading', { name: text.title })).toBeNull();
    expect(server.getRequests().some((request) => request.url.includes('/instance-backup-'))).toBe(
      false,
    );

    view.unmount();
    renderScreen(<InstanceSettingsPage isRootAdmin />);

    expect(await screen.findByRole('heading', { name: text.title })).toBeTruthy();
  });

  describe('on a new installation', () => {
    it('shows only the not-backed-up status and the three actions', async () => {
      useSectionResponses();
      openSection();

      expect(await screen.findByText(text.status.titles.notConfigured)).toBeTruthy();
      expect(screen.getByText(text.setup.description)).toBeTruthy();
      expect(screen.getByRole('button', { name: text.setup.start })).toBeTruthy();
      expect(screen.getByRole('button', { name: dictionary.instanceRecovery.open })).toBeTruthy();
      expect(screen.getByRole('button', { name: dictionary.masterKey.open })).toBeTruthy();
      expect(screen.queryByRole('heading', { name: text.settings.storages })).toBeNull();
      expect(screen.queryByRole('button', { name: dictionary.common.actions.edit })).toBeNull();
      expect(screen.queryByRole('heading', { name: text.history.title })).toBeNull();
      expect(document.querySelector('input[type="file"]')).toBeNull();
    });

    it('returns to the empty state without a request when the first setup is cancelled', async () => {
      useSectionResponses();
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: text.setup.start }));
      await selectDestination(user, s3Choice.storageName);
      await user.click(screen.getByRole('button', { name: dictionary.common.actions.cancel }));

      expect(await screen.findByRole('button', { name: text.setup.start })).toBeTruthy();
      expect(screen.queryByRole('combobox', { name: text.destinations.picker })).toBeNull();
      expect(settingsWrites()).toHaveLength(0);
    });

    it('requests the first backup and shows the saved settings when the first destination is saved', async () => {
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
                  { ...s3Destination, health: 'INITIALIZING', lastSuccessfulSnapshotAt: null },
                ],
              }),
            ),
          );
          return HttpResponse.json(healthySettings);
        }),
      );
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: text.setup.start }));
      await selectDestination(user, s3Choice.storageName);
      await user.click(screen.getByRole('button', { name: text.settings.save }));

      expect(await screen.findByText(text.settings.firstBackupRequested)).toBeTruthy();
      expect(savedSettings).toEqual(healthySettings);
      expect(await screen.findByText(text.status.titles.initializing)).toBeTruthy();
      expect(screen.getByText(text.status.noSuccessfulBackup)).toBeTruthy();
      expect(screen.queryByText(/^Last backup/)).toBeNull();
      expect(document.querySelector('section time')).toBeNull();
      expect(screen.getByRole('button', { name: dictionary.common.actions.edit })).toBeTruthy();
      expect(screen.queryByRole('button', { name: text.settings.save })).toBeNull();
    });
  });

  describe('status', () => {
    it('shows no status card for a healthy instance and hides every editing and recovery control', async () => {
      useSectionResponses({
        settings: buildSettings({
          storageIds: [s3Choice.storageId],
          schedule: { type: 'HOURLY' },
          retentionDays: 30,
        }),
        status: healthyStatus,
      });
      openSection();

      expect(await screen.findByRole('heading', { name: text.settings.storages })).toBeTruthy();
      expect(screen.queryByText(text.settings.schedule)).toBeNull();
      expect(screen.queryByText(text.status.titles.healthy)).toBeNull();
      expect(screen.queryByText(/^Last backup/)).toBeNull();
      expect(screen.queryByRole('combobox', { name: text.destinations.picker })).toBeNull();
      expect(document.querySelector('input[type="file"]')).toBeNull();
      expect(screen.queryByLabelText(dictionary.instanceRecovery.form.masterKey)).toBeNull();
      expect(screen.queryByText('databasus-data/secret.key')).toBeNull();
      expect(screen.queryByText(text.health.healthy)).toBeNull();
      expect(screen.queryByText(text.smtp.missingTitle)).toBeNull();
      expect(screen.queryByText(text.smtp.addressMissing)).toBeNull();
      expect(screen.queryByText(/^Since /)).toBeNull();
      expect(masterKeyRequests()).toHaveLength(0);
    });

    it('says how many pending deletions the latest recovery dropped and that files may need cleanup', async () => {
      useSectionResponses({
        status: buildStatus({
          health: 'HEALTHY',
          destinations: [s3Destination],
          recovery: buildRecovery({ status: 'COMPLETED', discardedCleanupCount: 3 }),
        }),
      });
      openSection();

      expect(
        await screen.findByText(
          dictionary.instanceRecovery.discardedCleanup.replace('{{count}}', '3'),
        ),
      ).toBeTruthy();
    });

    it('shows no cleanup note after a recovery that dropped no pending deletion', async () => {
      useSectionResponses({
        status: buildStatus({
          health: 'HEALTHY',
          destinations: [s3Destination],
          recovery: buildRecovery({ status: 'COMPLETED', discardedCleanupCount: 0 }),
        }),
      });
      openSection();

      expect(await screen.findByRole('heading', { name: text.settings.storages })).toBeTruthy();
      expect(screen.queryByText(/manual cleanup/)).toBeNull();
    });

    it('shows a failing storage and the undelivered alert inside the status without a second notice', async () => {
      useSectionResponses({
        settings: buildSettings({ storageIds: [s3Choice.storageId, localChoice.storageId] }),
        status: buildStatus({
          health: 'FAILED',
          destinations: [s3Destination, localDestination],
          incident: {
            id: '66666666-6666-4666-8666-666666666666',
            kind: 'FAILED',
            startedAt: '2026-11-14T10:00:00Z',
            mailStatus: 'RETRYING',
            lastMailError: 'mail server refused the sender',
          },
        }),
      });
      openSection();

      const status = within(
        (await screen.findByText(text.status.titles.failed)).closest('[role="status"]')!,
      );

      expect(
        status.getByText(
          text.incidents.since.replace(
            '{{time}}',
            shownTimeWithRelativeTime('2026-11-14T10:00:00Z'),
          ),
        ),
      ).toBeTruthy();
      expect(status.getByText(`${localChoice.storageName}: disk is full`)).toBeTruthy();
      expect(
        status.getByText(
          text.incidents.mailFailed.replace('{{error}}', 'mail server refused the sender'),
        ),
      ).toBeTruthy();
      expect(screen.getAllByText(text.status.titles.failed)).toHaveLength(1);
    });

    it('keeps the last known health and says it may be out of date when a refresh fails', async () => {
      useSectionResponses({ settings: healthySettings, status: partialStatus });
      server.use(mockApi.post('/instance-backups', 202, buildBackup({ status: 'IN_PROGRESS' })));
      const { user } = openSection();

      expect(await screen.findByText(text.status.titles.partial)).toBeTruthy();

      server.use(mockApi.get('/instance-backup-statuses', 500, buildProblem('internal', 500)));
      await user.click(screen.getByRole('button', { name: text.history.start }));

      expect(
        await screen.findByText(
          text.status.refreshFailed.replace('{{reason}}', dictionary.errors.internal),
        ),
      ).toBeTruthy();
      expect(screen.getByText(text.status.titles.partial)).toBeTruthy();
    });

    it('retries the status after it failed to load', async () => {
      server.use(
        mockApi.get('/instance-backup-settings', 200, buildSettings()),
        mockApi.get('/instance-backup-statuses', 500, buildProblem('internal', 500)),
      );
      const { user } = openSection();
      const retry = await screen.findByRole('button', { name: 'Retry' });
      useSectionResponses();

      await user.click(retry);

      expect(await screen.findByText(text.status.titles.notConfigured)).toBeTruthy();
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
          server
            .getRequests()
            .filter((request) => request.url.includes('/instance-backup-statuses')),
        ).toHaveLength(1),
      );

      server.use(mockApi.get('/instance-backup-statuses', 200, partialStatus));
      activateSession({ token: 'other-session', userId: 'other' });
      status.release();

      expect(await screen.findByText(text.status.titles.partial)).toBeTruthy();
      expect(screen.queryByText(text.status.titles.failed)).toBeNull();
    });
  });

  describe('read-only settings', () => {
    it('shows the health and error only for a failing storage and warns about a local one', async () => {
      useSectionResponses({
        settings: buildSettings({ storageIds: [s3Choice.storageId, localChoice.storageId] }),
        status: buildStatus({ health: 'PARTIAL', destinations: [s3Destination, localDestination] }),
      });
      openSection();

      const failingStorage = within(
        (await screen.findAllByText(localChoice.storageName))
          .map((element) => element.closest('li'))
          .find(Boolean)!,
      );

      const shownWorkspace = text.destinations.storageInWorkspace
        .replace(/^.*<details>|<\/details>$/g, '')
        .replace('<workspace/>', localChoice.workspaceName);

      expect(
        failingStorage.getByText((_, element) => element?.textContent === shownWorkspace),
      ).toBeTruthy();
      expect(failingStorage.getByText(text.health.failed)).toBeTruthy();
      expect(failingStorage.getByText('disk is full')).toBeTruthy();
      expect(failingStorage.getByText(text.destinations.localWarning)).toBeTruthy();
      expect(screen.queryByText(text.health.healthy)).toBeNull();
      expect(screen.queryByRole('button', { name: /^Remove / })).toBeNull();
    });

    it('says no storage is selected and keeps the history after the last storage was removed', async () => {
      const backup = buildBackup();
      useSectionResponses({
        status: buildStatus({
          health: 'STOPPED',
          incident: {
            id: '55555555-5555-4555-8555-555555555555',
            kind: 'STOPPED',
            startedAt: '2026-11-14T10:00:00Z',
            mailStatus: 'PENDING',
          },
        }),
        backups: [backup],
      });
      openSection();

      expect(await screen.findByText(text.status.titles.stopped)).toBeTruthy();
      expect(await screen.findByText(text.destinations.none)).toBeTruthy();
      expect(await screen.findByText(shownTimeWithRelativeTime(backup.snapshotAt!))).toBeTruthy();
      expect(screen.getByText(text.history.startNeedsDestination)).toBeTruthy();
    });
  });

  describe('editing settings', () => {
    it('saves a new retention and returns to the read-only view', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      server.use(
        http.put(`*${API_BASE_PATH}/instance-backup-settings`, async ({ request }) =>
          HttpResponse.json(await request.json()),
        ),
      );
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      const save = await screen.findByRole('button', { name: text.settings.save });
      expect(save.hasAttribute('disabled')).toBe(true);

      const retention = screen.getByRole('textbox', { name: text.settings.retentionDays });
      await user.clear(retention);
      await user.type(retention, '45');
      await user.click(save);

      expect(
        await screen.findByRole('button', { name: dictionary.common.actions.edit }),
      ).toBeTruthy();
      expect(screen.queryByRole('button', { name: text.settings.save })).toBeNull();
      expect(settingsWrites()).toHaveLength(1);
      expect(await settingsWrites()[0].clone().json()).toMatchObject({ retentionDays: 45 });
    });

    it('saves a new schedule and returns to the read-only view', async () => {
      useSectionResponses({
        settings: buildSettings({
          storageIds: [s3Choice.storageId],
          schedule: { type: 'WEEKLY', weekday: 2, timeOfDay: '04:00' },
        }),
        status: healthyStatus,
      });
      let savedSettings: unknown;
      server.use(
        http.put(`*${API_BASE_PATH}/instance-backup-settings`, async ({ request }) => {
          savedSettings = await request.json();
          return HttpResponse.json(savedSettings as object);
        }),
      );
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      await user.click(await screen.findByRole('combobox', { name: text.settings.schedule }));
      await user.click(
        await screen.findByRole('option', { name: dictionary.schedules.types.daily }),
      );
      await user.click(screen.getByRole('button', { name: text.settings.save }));

      expect(
        await screen.findByRole('button', { name: dictionary.common.actions.edit }),
      ).toBeTruthy();
      expect(screen.queryByRole('button', { name: text.settings.save })).toBeNull();
      expect(savedSettings).toMatchObject({ schedule: { type: 'DAILY', timeOfDay: '04:00' } });
    });

    it('cancels an edit without sending a change', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      await user.click(
        await screen.findByRole('button', {
          name: text.destinations.remove.replace('{{storage}}', s3Choice.storageName),
        }),
      );
      await user.click(screen.getByRole('button', { name: dictionary.common.actions.cancel }));

      expect(await screen.findByText(s3Choice.storageName)).toBeTruthy();
      expect(screen.queryByText(text.destinations.none)).toBeNull();
      expect(settingsWrites()).toHaveLength(0);
    });

    it('keeps the dialog and the draft when root admin clicks outside it', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      await user.click(
        await screen.findByRole('button', {
          name: text.destinations.remove.replace('{{storage}}', s3Choice.storageName),
        }),
      );
      await user.click(document.querySelector('.mantine-Modal-overlay')!);

      const dialog = within(screen.getByRole('dialog', { name: text.settings.title }));
      expect(dialog.getByText(text.destinations.none)).toBeTruthy();
      expect(settingsWrites()).toHaveLength(0);
    });

    it('discards the draft and sends nothing when the dialog is closed with its close button', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      await user.click(
        await screen.findByRole('button', {
          name: text.destinations.remove.replace('{{storage}}', s3Choice.storageName),
        }),
      );
      await user.click(
        within(screen.getByRole('dialog', { name: text.settings.title })).getByRole('button', {
          name: dictionary.common.actions.close,
        }),
      );

      await waitFor(() =>
        expect(screen.queryByRole('dialog', { name: text.settings.title })).toBeNull(),
      );
      expect(screen.getByText(s3Choice.storageName)).toBeTruthy();
      expect(screen.queryByText(text.destinations.none)).toBeNull();
      expect(settingsWrites()).toHaveLength(0);
    });

    it('keeps the form and the draft when saving fails', async () => {
      useSectionResponses();
      server.use(
        mockApi.put('/instance-backup-settings', 400, buildProblem('invalid_request', 400)),
      );
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: text.setup.start }));
      await selectDestination(user, s3Choice.storageName);
      await user.click(screen.getByRole('button', { name: text.settings.save }));

      expect(await screen.findByText(dictionary.errors.invalidRequest)).toBeTruthy();
      expect(screen.getByText(s3Choice.storageName)).toBeTruthy();
      expect(screen.getByRole('button', { name: text.settings.save })).toBeTruthy();
    });

    it('keeps the form open with a newer retention edit when the save that preceded it is acknowledged', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      const save = mockApi.delayed({
        method: 'put',
        path: '/instance-backup-settings',
        status: 200,
        body: buildSettings({ retentionDays: 31, storageIds: [s3Choice.storageId] }),
      });
      server.use(save.handler);
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      const retention = await screen.findByRole('textbox', { name: text.settings.retentionDays });
      await user.clear(retention);
      await user.type(retention, '31');
      await user.click(screen.getByRole('button', { name: text.settings.save }));
      await waitFor(() => expect(settingsWrites()).toHaveLength(1));
      await user.clear(retention);
      await user.type(retention, '45');
      save.release();

      await screen.findByText(text.settings.saved);
      expect((retention as HTMLInputElement).value).toBe('45');
      expect(screen.getByRole('button', { name: text.settings.save })).toBeTruthy();
    });

    it('reports stopped backups when the last destination is removed', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
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

      await user.click(await screen.findByRole('button', { name: dictionary.common.actions.edit }));
      await user.click(
        await screen.findByRole('button', {
          name: text.destinations.remove.replace('{{storage}}', s3Choice.storageName),
        }),
      );
      await user.click(screen.getByRole('button', { name: text.settings.save }));

      expect(await screen.findByText(text.settings.stopped)).toBeTruthy();
      expect(await screen.findByText(text.status.titles.stopped)).toBeTruthy();
      expect(screen.getByText(text.destinations.none)).toBeTruthy();
    });

    it('translates the form and keeps the draft when the language changes', async () => {
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

      await user.click(await screen.findByRole('button', { name: text.setup.start }));
      await selectDestination(user, s3Choice.storageName);
      await user.click(screen.getByRole('button', { name: /Language:/ }));
      await user.click(
        await screen.findByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName }),
      );

      const section = screen.getByRole('region', { name: DICTIONARIES.ru.instanceBackups.title });
      expect(
        within(section).getByText(DICTIONARIES.ru.instanceBackups.status.titles.notConfigured),
      ).toBeTruthy();
      expect(
        within(section).getByRole('button', {
          name: DICTIONARIES.ru.instanceBackups.settings.save,
        }),
      ).toBeTruthy();
      expect(within(section).getByText(s3Choice.storageName)).toBeTruthy();
    });
  });

  describe('history', () => {
    it('shows the refusal when a backup is already running', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      server.use(mockApi.post('/instance-backups', 409, buildProblem('already_running', 409)));
      const { user } = openSection();

      await user.click(await screen.findByRole('button', { name: text.history.start }));

      expect(await screen.findByText(dictionary.errors.alreadyRunning)).toBeTruthy();
    });

    it('follows the file link with a single-use token when the only stored copy is downloaded', async () => {
      const followedLinks: { href: string; download: string }[] = [];
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
        this: HTMLAnchorElement,
      ) {
        followedLinks.push({ href: this.href, download: this.download });
      });
      useSectionResponses({
        settings: healthySettings,
        status: healthyStatus,
        backups: [buildBackup()],
      });
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
          name: text.history.downloadBackup.replace(
            '{{time}}',
            shownTime(buildBackup().snapshotAt!),
          ),
        }),
      );

      await waitFor(() => expect(followedLinks).toHaveLength(1));
      expect(followedLinks[0].href).toContain('/instance-backup-downloads?token=single-use');
      expect(followedLinks[0].download).toBe('databasus-backup.dbsbackup');
    });

    it('downloads from the next copy when the first storage cannot serve it', async () => {
      const backup = buildBackup();
      const localCopy = {
        ...backup.copies[0],
        id: '77777777-7777-4777-8777-777777777777',
        storageId: localChoice.storageId,
        storageName: localChoice.storageName,
        workspaceName: localChoice.workspaceName,
      };
      useSectionResponses({
        settings: healthySettings,
        status: healthyStatus,
        backups: [{ ...backup, copies: [...backup.copies, localCopy] }],
      });
      const requestedCopies: unknown[] = [];
      server.use(
        http.post(
          `*${API_BASE_PATH}/instance-backups/:instanceBackupId/download-tokens`,
          async ({ request }) => {
            const requestedCopy = (await request.json()) as { copyId: string };
            requestedCopies.push(requestedCopy.copyId);

            if (requestedCopy.copyId === backup.copies[0].id)
              return HttpResponse.json(buildProblem('storage_unavailable', 409), { status: 409 });

            return HttpResponse.json(
              {
                token: 'single-use',
                fileName: 'databasus.dbsbackup',
                expiresAt: '2999-01-01T00:00:00Z',
              },
              { status: 201 },
            );
          },
        ),
      );
      const followedLinks: string[] = [];
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
        this: HTMLAnchorElement,
      ) {
        followedLinks.push(this.href);
      });
      const { user } = openSection();

      await user.click(
        await screen.findByRole('button', {
          name: text.history.downloadBackup.replace('{{time}}', shownTime(backup.snapshotAt!)),
        }),
      );

      await waitFor(() => expect(followedLinks).toHaveLength(1));
      expect(requestedCopies).toEqual([backup.copies[0].id, localCopy.id]);
      expect(screen.queryByRole('menuitem')).toBeNull();
    });

    it('shows the failed copies of a partial backup in its row', async () => {
      const backup = buildBackup({
        status: 'PARTIAL',
        copies: [
          { ...buildBackup().copies[0] },
          {
            ...buildBackup().copies[0],
            id: '88888888-8888-4888-8888-888888888888',
            storageId: localChoice.storageId,
            storageName: localChoice.storageName,
            workspaceName: localChoice.workspaceName,
            status: 'FAILED',
            isDownloadable: false,
            lastError: 'disk is full',
          },
        ],
      });
      useSectionResponses({ settings: healthySettings, status: healthyStatus, backups: [backup] });
      openSection();

      expect(await screen.findByText(text.backupStatuses.partial)).toBeTruthy();
      expect(screen.getByText('disk is full')).toBeTruthy();
      expect(
        screen.getByText(
          text.history.copy
            .replace('{{storage}}', localChoice.storageName)
            .replace('{{workspace}}', localChoice.workspaceName)
            .replace('{{status}}', text.copyStatuses.failed),
        ),
      ).toBeTruthy();
      expect(
        screen.queryByText(
          text.history.copy
            .replace('{{storage}}', s3Choice.storageName)
            .replace('{{workspace}}', s3Choice.workspaceName)
            .replace('{{status}}', text.copyStatuses.stored),
        ),
      ).toBeNull();
    });

    it('lists no copy of a backup in progress whose copy retries after an error', async () => {
      const backup = buildBackup({
        status: 'IN_PROGRESS',
        copies: [
          {
            ...buildBackup().copies[0],
            status: 'RETRYING',
            isDownloadable: false,
            lastError: 'connection reset',
          },
        ],
      });
      useSectionResponses({ settings: healthySettings, status: healthyStatus, backups: [backup] });
      openSection();

      expect(await screen.findByText(text.backupStatuses.inProgress)).toBeTruthy();
      expect(screen.queryByText('connection reset')).toBeNull();
    });

    it('shows the five newest backups and expands in place to page through all of them', async () => {
      const backups = buildBackups(12);
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      server.use(
        http.get(`*${API_BASE_PATH}/instance-backups`, ({ request }) => {
          const query = new URL(request.url).searchParams;
          const start = Number(query.get('after') ?? 0);
          const end = start + Number(query.get('limit'));

          return HttpResponse.json({
            items: backups.slice(start, end),
            ...(end < backups.length ? { nextCursor: String(end) } : {}),
          });
        }),
      );
      const { user } = openSection();
      const shownTimes = backups.map((backup) => shownTimeWithRelativeTime(backup.snapshotAt!));

      expect(await screen.findByText(shownTimes[4])).toBeTruthy();
      expect(screen.queryByText(shownTimes[5])).toBeNull();

      await user.click(screen.getByRole('button', { name: text.history.showAll }));
      expect(screen.getByText(shownTimes[9])).toBeTruthy();
      expect(screen.queryByText(shownTimes[10])).toBeNull();
      expect(screen.queryByRole('button', { name: text.history.showAll })).toBeNull();

      await user.click(screen.getByRole('button', { name: text.history.showMore }));

      expect(await screen.findByText(shownTimes[11])).toBeTruthy();
      expect(screen.queryByRole('button', { name: text.history.showMore })).toBeNull();
    });
  });

  describe('dialogs', () => {
    it('reads the master key each time Download master key opens its dialog and forgets it on close', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
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
      const masterKey = dictionary.masterKey;

      await screen.findByRole('button', { name: masterKey.open });
      expect(masterKeyRequests()).toHaveLength(0);

      await user.click(screen.getByRole('button', { name: masterKey.open }));
      const dialog = within(await screen.findByRole('dialog', { name: masterKey.title }));

      expect(await dialog.findByText('pane••••••••••••6789')).toBeTruthy();
      expect(dialog.getByRole('button', { name: masterKey.copy })).toBeTruthy();
      expect(dialog.getByRole('button', { name: masterKey.download })).toBeTruthy();

      const hostToggle = dialog.getByRole('button', { name: masterKey.host.toggle });
      expect(hostToggle.getAttribute('aria-expanded')).toBe('false');
      await user.click(hostToggle);
      expect(dialog.getByText('databasus-data/secret.key')).toBeTruthy();
      expect(dialog.queryByText(/docker cp/)).toBeNull();

      await user.click(dialog.getByRole('button', { name: dictionary.common.actions.close }));
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());

      expect(screen.queryByText('pane••••••••••••6789')).toBeNull();
      expect(keyExportRequests()).toHaveLength(1);

      await user.click(screen.getByRole('button', { name: masterKey.open }));
      const reopened = within(await screen.findByRole('dialog', { name: masterKey.title }));

      expect(await reopened.findByText('pane••••••••••••6789')).toBeTruthy();
      expect(keyExportRequests()).toHaveLength(2);
    });

    it('asks for an emailed code in the master-key dialog when two-factor sign-in is on', async () => {
      useSectionResponses({
        status: buildStatus({
          isMasterKeyAcknowledged: false,
          isConfirmationCodeRequired: true,
        }),
      });
      const { user } = openSection();
      const masterKey = dictionary.masterKey;

      await user.click(await screen.findByRole('button', { name: masterKey.open }));
      const dialog = within(await screen.findByRole('dialog', { name: masterKey.title }));

      expect(await dialog.findByRole('button', { name: masterKey.proof.sendCode })).toBeTruthy();
      expect(masterKeyRequests()).toHaveLength(0);
    });

    it('asks for an app code in the master-key dialog when root admin has an app', async () => {
      useSectionResponses({
        status: buildStatus({
          isMasterKeyAcknowledged: false,
          isConfirmationCodeRequired: true,
        }),
      });
      const exportBodies: unknown[] = [];
      server.use(
        mockSignedInAccount({ isAuthenticatorAppConnected: true }),
        http.post(`*${API_BASE_PATH}/master-key-exports`, async ({ request }) => {
          exportBodies.push(await request.json());
          return HttpResponse.json(
            {
              token: 'key-token',
              fileName: 'databasus-master-key.txt',
              expiresAt: '2999-01-01T00:00:00Z',
            },
            { status: 201 },
          );
        }),
        http.get(`*${API_BASE_PATH}/master-key-downloads`, () =>
          HttpResponse.text('panel-master-key-0123456789'),
        ),
      );
      const { user } = openSection();
      const masterKey = dictionary.masterKey;

      await user.click(await screen.findByRole('button', { name: masterKey.open }));
      const dialog = within(await screen.findByRole('dialog', { name: masterKey.title }));
      await user.type(
        await dialog.findByLabelText(dictionary.authenticatorApp.code.appCode),
        '123456',
      );

      expect(dialog.queryByRole('button', { name: masterKey.proof.sendCode })).toBeNull();
      await user.click(dialog.getByRole('button', { name: masterKey.proof.confirm }));

      expect(await dialog.findByText('pane••••••••••••6789')).toBeTruthy();
      expect(exportBodies).toEqual([{ code: '123456' }]);
      expect(
        server.getRequests().some((request) => request.url.includes('/master-key-confirmations')),
      ).toBe(false);
    });

    it('opens recovery from a backup file in a dialog', async () => {
      useSectionResponses({ settings: healthySettings, status: healthyStatus });
      const { user } = openSection();

      await user.click(
        await screen.findByRole('button', { name: dictionary.instanceRecovery.open }),
      );
      const dialog = within(
        await screen.findByRole('dialog', { name: dictionary.instanceRecovery.form.title }),
      );

      expect(dialog.getByText(dictionary.instanceRecovery.form.archiveHint)).toBeTruthy();
      expect(
        dialog.getByRole('button', { name: dictionary.instanceRecovery.form.upload }),
      ).toBeTruthy();
    });
  });

  describe('mail server', () => {
    it('asks to configure the mail server and calls back when the action is chosen', async () => {
      useSectionResponses({ status: buildStatus({ isEmailConfigured: false }) });
      activateSession({ token: 'root-session', userId: 'root' });
      const onConfigureSmtp = vi.fn();
      const { user } = renderSection(onConfigureSmtp);

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

      await user.click(screen.getByRole('button', { name: text.setup.start }));
      const retention = await screen.findByRole('textbox', { name: text.settings.retentionDays });
      await user.clear(retention);
      await user.type(retention, '45');
      await user.click(screen.getByRole('button', { name: text.settings.save }));

      await waitFor(() => expect(screen.queryByText(text.smtp.addressMissing)).toBeNull());
      expect(screen.queryByRole('button', { name: text.smtp.configure })).toBeNull();
    });

    it('moves focus to the mail server section when Configure SMTP is chosen', async () => {
      activateSession({ token: 'root-session', userId: 'root' });
      server.use(
        mockApi.get('/instance-settings', 200, {
          isAllowExternalRegistrations: false,
          isAllowMemberInvitations: false,
          isMemberAllowedToCreateWorkspaces: false,
          twoFactorRequirement: 'OFF',
          isEmailConfigured: false,
        }),
        mockApi.get('/verification-agents', 200, { items: [] }),
        mockApi.get('/audit-events', 200, { items: [] }),
      );
      useSectionResponses({ status: buildStatus({ isEmailConfigured: false }) });
      const scrollIntoView = vi
        .spyOn(HTMLElement.prototype, 'scrollIntoView')
        .mockImplementation(() => {});
      const { user } = renderScreen(<InstanceSettingsPage isRootAdmin />);

      await user.click(await screen.findByRole('button', { name: text.smtp.configure }));

      const mailHeading = screen.getByRole('heading', {
        name: dictionary.settings.mailServer.title,
      });
      expect(document.activeElement).toBe(mailHeading);
      expect(scrollIntoView).toHaveBeenCalled();
    });
  });
});

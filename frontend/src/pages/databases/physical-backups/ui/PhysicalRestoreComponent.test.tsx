// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { physicalBackupsApi } from '@/pages/databases/physical-backups/api/physicalBackupsApi';
import { type PhysicalBackupListItem } from '@/pages/databases/physical-backups/model/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import { PhysicalBackupType } from '@/pages/databases/physical-backups/model/PhysicalBackupType';
import { PhysicalRestoreComponent } from '@/pages/databases/physical-backups/ui/PhysicalRestoreComponent';
import { API_BASE_PATH } from '@/shared/api';
import { DICTIONARIES, useLocale } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboardWrite } from '@/shared/lib/test/stubClipboard';

// main.tsx registers the plugin for the application; the screen sends and shows times through it.
dayjs.extend(utc);

const server = setUpMockApi();

const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  notifierIds: [],
};

const backup: PhysicalBackupListItem = {
  id: 'backup-id',
  type: PhysicalBackupType.FULL,
  status: PhysicalBackupStatus.COMPLETED,
  timelineId: 1,
  startLsn: '0/10000000',
  stopLsn: '1/0',
  hasGapBefore: false,
  sizeMb: 100,
  createdAt: '2026-10-10T03:00:00Z',
};

const FIRST_TIME = '2026-10-18 10:00:00';
const SECOND_TIME = '2026-10-18 11:00:00';
const LATEST_TARGET = 'latest';

const restoreCopy = DICTIONARIES.en.backups.physical.restore;

const toRequestedTarget = (localTime: string) => dayjs(localTime).utc().toISOString();
const toShownTarget = (localTime: string) => dayjs(localTime).utc().format('YYYY-MM-DD HH:mm:ssZ');

interface TokenRequestBody {
  type: string;
  backupId?: string;
  targetTime?: string;
}

const declareHeldTokenRequests = () => {
  const heldRequests = new Map<string, ReturnType<typeof createPendingApiResponse>>();
  const requestBodies: TokenRequestBody[] = [];

  server.use(
    http.post(`*${API_BASE_PATH}/physical-restore-tokens`, async ({ request }) => {
      const requestBody = (await request.json()) as TokenRequestBody;
      requestBodies.push(requestBody);

      const heldRequest = createPendingApiResponse();
      heldRequests.set(requestBody.targetTime ?? LATEST_TARGET, heldRequest);

      return heldRequest.response;
    }),
  );

  return { heldRequests, requestBodies };
};

const issueToken = (token: string) => HttpResponse.json({ token }, { status: 201 });

const refuse = (code: Parameters<typeof buildProblem>[0], status: number) =>
  HttpResponse.json(buildProblem(code, status), { status });

const queryCommandWithToken = (token: string) =>
  screen.queryByText((content, element) => element?.tagName === 'PRE' && content.includes(token));

const queryAnyCommand = () => document.querySelector('pre');

const chooseTargetTime = async (localTime: string) => {
  const picker = screen.getByPlaceholderText(restoreCopy.latestAvailable);

  fireEvent.click(picker);
  fireEvent.change(picker, { target: { value: localTime } });
  fireEvent.keyDown(picker, { key: 'Enter' });
};

const clearTargetTime = () => {
  fireEvent.click(document.querySelector('button[aria-label="Clear"]') as Element);
};

const spyOnTokenRequests = () => vi.spyOn(physicalBackupsApi, 'createPhysicalRestoreToken');

const waitForHeldRequest = async (
  heldRequests: ReturnType<typeof declareHeldTokenRequests>['heldRequests'],
  target: string,
) => {
  await waitFor(() => expect(heldRequests.has(target)).toBe(true));

  return heldRequests.get(target)!;
};

const renderPointInTimeRestore = () =>
  renderScreen(<PhysicalRestoreComponent database={database} onClose={() => {}} />);

const SwitchableRestoreScreen = () => {
  const [isOpen, setIsOpen] = useState(true);

  return (
    <>
      <button onClick={() => setIsOpen((isShown) => !isShown)}>Toggle restore</button>
      {isOpen && <PhysicalRestoreComponent database={database} onClose={() => setIsOpen(false)} />}
    </>
  );
};

const RestoreScreenWithLanguageSwitch = () => {
  const { setLocale } = useLocale();

  return (
    <>
      <button onClick={() => setLocale('fr')}>Change language</button>
      <PhysicalRestoreComponent database={database} onClose={() => {}} />
    </>
  );
};

describe('PhysicalRestoreComponent', () => {
  it('keeps the restore commands when the language changes', async () => {
    const { heldRequests, requestBodies } = declareHeldTokenRequests();
    renderScreen(<RestoreScreenWithLanguageSwitch />);

    const latestRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    latestRequest.respond(issueToken('latest-token'));
    await waitFor(() => expect(queryCommandWithToken('latest-token')).toBeTruthy());

    fireEvent.click(screen.getByRole('button', { name: 'Change language' }));

    expect(
      await screen.findByText(DICTIONARIES.fr.backups.physical.restore.scriptTitles.host),
    ).toBeTruthy();
    expect(screen.queryByText(restoreCopy.scriptTitles.host)).toBeNull();
    expect(queryCommandWithToken('latest-token')).toBeTruthy();
    expect(requestBodies).toHaveLength(1);
  });

  it('names the target time, restore method and environment fields', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    renderPointInTimeRestore();

    expect(screen.getByLabelText(restoreCopy.targetTime)).toBeTruthy();

    const latestRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    latestRequest.respond(issueToken('latest-token'));
    await waitFor(() => expect(queryCommandWithToken('latest-token')).toBeTruthy());

    expect(screen.getByRole('radiogroup', { name: restoreCopy.methods.label })).toBeTruthy();

    fireEvent.click(screen.getByRole('radio', { name: restoreCopy.methods.manual }));

    expect(
      await screen.findByRole('textbox', { name: restoreCopy.fields.restoreDirectory }),
    ).toBeTruthy();
    expect(screen.getByRole('textbox', { name: restoreCopy.fields.pgBinPath })).toBeTruthy();

    fireEvent.click(screen.getByRole('tab', { name: 'Docker' }));

    expect(await screen.findByRole('textbox', { name: restoreCopy.fields.pgImage })).toBeTruthy();
  });

  it('requests one token for a backup and shows commands without a target time', async () => {
    const { heldRequests, requestBodies } = declareHeldTokenRequests();
    renderScreen(
      <PhysicalRestoreComponent database={database} backup={backup} onClose={() => {}} />,
    );

    const heldRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    heldRequest.respond(issueToken('backup-token'));

    await waitFor(() => expect(queryCommandWithToken('backup-token')).toBeTruthy());
    expect(requestBodies).toEqual([{ type: 'BACKUP', backupId: backup.id }]);
    expect(screen.queryByPlaceholderText(restoreCopy.latestAvailable)).toBeNull();
    expect(queryAnyCommand()?.textContent).not.toContain('--target-time');
  });

  it('requests a token for the latest moment on open and for the chosen time afterwards', async () => {
    const { heldRequests, requestBodies } = declareHeldTokenRequests();
    renderPointInTimeRestore();

    const latestRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    latestRequest.respond(issueToken('latest-token'));
    await waitFor(() => expect(queryCommandWithToken('latest-token')).toBeTruthy());

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    firstRequest.respond(issueToken('first-token'));

    await waitFor(() => expect(queryCommandWithToken('first-token')).toBeTruthy());
    expect(queryCommandWithToken('first-token')?.textContent).toContain(toShownTarget(FIRST_TIME));
    expect(requestBodies[0]).toEqual({ type: 'POINT_IN_TIME', databaseId: database.id });
    expect(requestBodies[1]).toEqual({
      type: 'POINT_IN_TIME',
      databaseId: database.id,
      targetTime: toRequestedTarget(FIRST_TIME),
    });
  });

  it('keeps the newest token and time when an earlier token answers last', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    const tokenRequests = spyOnTokenRequests();
    renderPointInTimeRestore();
    await waitForHeldRequest(heldRequests, LATEST_TARGET);

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    const firstCallIndex = tokenRequests.mock.results.length - 1;
    await chooseTargetTime(SECOND_TIME);
    const secondRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(SECOND_TIME));

    secondRequest.respond(issueToken('second-token'));
    await waitFor(() => expect(queryCommandWithToken('second-token')).toBeTruthy());
    await releaseAndSettle(
      tokenRequests,
      () => firstRequest.respond(issueToken('first-token')),
      firstCallIndex,
    );

    expect(queryCommandWithToken('first-token')).toBeNull();
    expect(queryCommandWithToken('second-token')?.textContent).toContain(
      toShownTarget(SECOND_TIME),
    );
  });

  it('keeps the commands and shows no error when an earlier request fails last', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    const tokenRequests = spyOnTokenRequests();
    renderPointInTimeRestore();
    await waitForHeldRequest(heldRequests, LATEST_TARGET);

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    const firstCallIndex = tokenRequests.mock.results.length - 1;
    await chooseTargetTime(SECOND_TIME);
    const secondRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(SECOND_TIME));

    secondRequest.respond(issueToken('second-token'));
    await waitFor(() => expect(queryCommandWithToken('second-token')).toBeTruthy());
    await releaseAndSettle(
      tokenRequests,
      () => firstRequest.respond(refuse('internal', 500)),
      firstCallIndex,
    );

    expect(queryCommandWithToken('second-token')).toBeTruthy();
    expect(screen.queryByText(DICTIONARIES.en.errors.internal)).toBeNull();
  });

  it('shows that it is preparing and no commands when an earlier token answers while the newest is pending', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    const tokenRequests = spyOnTokenRequests();
    renderPointInTimeRestore();
    await waitForHeldRequest(heldRequests, LATEST_TARGET);

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    const firstCallIndex = tokenRequests.mock.results.length - 1;
    await chooseTargetTime(SECOND_TIME);
    await waitForHeldRequest(heldRequests, toRequestedTarget(SECOND_TIME));

    await releaseAndSettle(
      tokenRequests,
      () => firstRequest.respond(issueToken('first-token')),
      firstCallIndex,
    );

    expect(screen.getByText(restoreCopy.preparing)).toBeTruthy();
    expect(queryAnyCommand()).toBeNull();
  });

  it('shows the reason with its hint and no commands when the newest request is refused', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    renderPointInTimeRestore();

    const latestRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    latestRequest.respond(issueToken('latest-token'));
    await waitFor(() => expect(queryCommandWithToken('latest-token')).toBeTruthy());

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    firstRequest.respond(refuse('target_time_unreachable', 422));

    expect(await screen.findByText(DICTIONARIES.en.errors.targetTimeUnreachable)).toBeTruthy();
    expect(screen.getByText(restoreCopy.errorHints.targetTimeUnreachable)).toBeTruthy();
    expect(queryAnyCommand()).toBeNull();
    expect(screen.queryByText(restoreCopy.preparing)).toBeNull();
  });

  it('uses the token issued without a target time when a chosen time is cleared and its answer arrives last', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    const tokenRequests = spyOnTokenRequests();
    renderPointInTimeRestore();
    const openingRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    openingRequest.respond(issueToken('opening-token'));
    await waitFor(() => expect(queryCommandWithToken('opening-token')).toBeTruthy());
    heldRequests.delete(LATEST_TARGET);

    await chooseTargetTime(FIRST_TIME);
    const firstRequest = await waitForHeldRequest(heldRequests, toRequestedTarget(FIRST_TIME));
    const firstCallIndex = tokenRequests.mock.results.length - 1;
    clearTargetTime();
    const latestRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);

    latestRequest.respond(issueToken('latest-token'));
    await waitFor(() => expect(queryCommandWithToken('latest-token')).toBeTruthy());
    await releaseAndSettle(
      tokenRequests,
      () => firstRequest.respond(issueToken('first-token')),
      firstCallIndex,
    );

    expect(queryCommandWithToken('first-token')).toBeNull();
    expect(queryCommandWithToken('latest-token')?.textContent).not.toContain('--target-time');
  });

  it('ignores an answer after the screen closes and starts its own request when it opens again', async () => {
    const { heldRequests, requestBodies } = declareHeldTokenRequests();
    const tokenRequests = spyOnTokenRequests();
    renderScreen(<SwitchableRestoreScreen />);
    const closedRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    heldRequests.delete(LATEST_TARGET);

    fireEvent.click(screen.getByRole('button', { name: 'Toggle restore' }));
    await releaseAndSettle(tokenRequests, () => closedRequest.respond(refuse('internal', 500)));

    expect(screen.queryByText(DICTIONARIES.en.errors.internal)).toBeNull();

    fireEvent.click(screen.getByRole('button', { name: 'Toggle restore' }));
    const reopenedRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);

    expect(screen.getByText(restoreCopy.preparing)).toBeTruthy();
    expect(requestBodies).toHaveLength(2);

    reopenedRequest.respond(issueToken('reopened-token'));

    await waitFor(() => expect(queryCommandWithToken('reopened-token')).toBeTruthy());
  });

  it('shows that a command was copied and clears its timer when the screen closes', async () => {
    const { heldRequests } = declareHeldTokenRequests();
    renderScreen(<SwitchableRestoreScreen />);
    stubClipboardWrite(() => Promise.resolve());
    const openingRequest = await waitForHeldRequest(heldRequests, LATEST_TARGET);
    openingRequest.respond(issueToken('copied-token'));
    await waitFor(() => expect(queryCommandWithToken('copied-token')).toBeTruthy());
    const clearTimeoutSpy = vi.spyOn(window, 'clearTimeout');
    const setTimeoutSpy = vi.spyOn(window, 'setTimeout');

    fireEvent.click(screen.getAllByRole('button', { name: /Copy/ })[0]);

    expect(await screen.findByRole('button', { name: /Copied/ })).toBeTruthy();
    const copiedTimers = setTimeoutSpy.mock.results
      .filter((_, index) => setTimeoutSpy.mock.calls[index][1] === 2000)
      .map((timerResult) => timerResult.value);
    expect(copiedTimers).toHaveLength(1);

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Toggle restore' }));
    });

    expect(clearTimeoutSpy).toHaveBeenCalledWith(copiedTimers[0]);
  });
});

describe('PhysicalRestoreComponent commands', () => {
  const RESTORE_TOKEN = 'restore-token';
  const DEFAULT_DIRECTORY = './databasus-restore';

  const getScriptUrl = () => physicalBackupsApi.getPhysicalRecoveryScriptUrl();
  const getBundleUrl = () => physicalBackupsApi.getPhysicalRestoreSetUrl(RESTORE_TOKEN);

  const buildDatabaseOfVersion = (version: '17' | '18'): Database => ({
    ...database,
    settings: { ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL), version },
  });

  // The hidden tab keeps its commands in the document, so only the shown tab counts.
  const getShownCommands = () =>
    Array.from(screen.getByRole('tabpanel').querySelectorAll('pre')).map(
      (command) => command.textContent,
    );

  const renderIssuedRestore = async (shownDatabase = buildDatabaseOfVersion('17')) => {
    server.use(
      mockApi.post('/physical-restore-tokens', 201, {
        token: RESTORE_TOKEN,
        expiresAt: '2026-10-18T12:00:00Z',
      }),
    );
    renderScreen(<PhysicalRestoreComponent database={shownDatabase} onClose={() => {}} />);
    await waitFor(() => expect(queryCommandWithToken(RESTORE_TOKEN)).toBeTruthy());
  };

  const switchToDocker = () => fireEvent.click(screen.getByRole('tab', { name: 'Docker' }));

  const switchToManual = () =>
    fireEvent.click(screen.getByRole('radio', { name: restoreCopy.methods.manual }));

  const editField = async (fieldName: string, value: string) => {
    fireEvent.change(await screen.findByRole('textbox', { name: fieldName }), {
      target: { value },
    });
  };

  it('uses edited paths and tools in Host preparation and startup instructions', async () => {
    await renderIssuedRestore(buildDatabaseOfVersion('18'));

    await editField(restoreCopy.fields.restoreDirectory, './host restore');
    await editField(restoreCopy.fields.pgBinPath, '/opt/pg tools');

    expect(getShownCommands()[0]).toContain('--pg-bin "/opt/pg tools"');
    expect(getShownCommands()[0]).toContain('"./host restore"');
    expect(screen.getByRole('tabpanel').textContent).toContain(
      'chown -R postgres:postgres "./host restore/18/docker"',
    );
    expect(screen.getByRole('tabpanel').textContent).toContain(
      '"/opt/pg tools/pg_ctl" -D "./host restore/18/docker" start',
    );
  });

  it.each(['17', '18'] as const)(
    'uses the edited absolute path and image for Docker preparation and startup on PostgreSQL %s',
    async (version) => {
      await renderIssuedRestore(buildDatabaseOfVersion(version));
      switchToDocker();
      const image = `postgres:${version}@sha256:abcdef`;

      await editField(restoreCopy.fields.restoreDirectory, '/restore-work/docker restore');
      await editField(restoreCopy.fields.pgImage, image);

      expect(getShownCommands()[0]).toContain(`--combine-image "${image}"`);
      expect(getShownCommands()[0]).toContain('"/restore-work/docker restore"');
      const mount =
        version === '18'
          ? '/restore-work/docker restore:/var/lib/postgresql'
          : '/restore-work/docker restore/data:/var/lib/postgresql/data';
      expect(screen.getByRole('tabpanel').textContent).toContain(
        `docker run -e POSTGRES_PASSWORD=... -v "${mount}" "${image}"`,
      );
    },
  );

  it('replaces the host script with the image script after switching to Docker', async () => {
    await renderIssuedRestore();

    expect(getShownCommands()).toEqual([
      `curl -fsSL "${getScriptUrl()}" | sh -s -- "${getBundleUrl()}" "${DEFAULT_DIRECTORY}"`,
    ]);

    switchToDocker();

    await waitFor(() =>
      expect(getShownCommands()).toEqual([
        `curl -fsSL "${getScriptUrl()}" | sh -s -- --combine-image "postgres:17" "${getBundleUrl()}" "${DEFAULT_DIRECTORY}"`,
      ]),
    );
  });

  it('replaces the script with the manual steps after switching to Manual', async () => {
    await renderIssuedRestore();

    switchToManual();

    await waitFor(() =>
      expect(getShownCommands()[0]).toBe(`curl -fsSL "${getBundleUrl()}" -o restore.tar`),
    );
    expect(getShownCommands().join('\n')).not.toContain('sh -s --');
    expect(getShownCommands().join('\n')).toContain('pg_combinebackup');
    expect(getShownCommands().join('\n')).not.toContain('docker run');

    switchToDocker();

    await waitFor(() =>
      expect(getShownCommands().join('\n')).toContain(
        '-v "$PWD:/work" -w /work postgres:17 \\\n  pg_combinebackup',
      ),
    );
  });

  it('shows the new restore directory in the script and the manual steps after an edit', async () => {
    await renderIssuedRestore();

    await editField(restoreCopy.fields.restoreDirectory, './edited-restore');

    expect(getShownCommands()).toEqual([
      `curl -fsSL "${getScriptUrl()}" | sh -s -- "${getBundleUrl()}" "./edited-restore"`,
    ]);

    switchToManual();

    await waitFor(() =>
      expect(getShownCommands().join('\n')).toContain('-o "./edited-restore/data"'),
    );
  });

  it('shows the new pg_bin path in the script and the manual steps after an edit', async () => {
    await renderIssuedRestore();

    await editField(restoreCopy.fields.pgBinPath, '/opt/postgres/bin');

    expect(getShownCommands()).toEqual([
      `curl -fsSL "${getScriptUrl()}" | sh -s -- --pg-bin "/opt/postgres/bin" "${getBundleUrl()}" "${DEFAULT_DIRECTORY}"`,
    ]);

    switchToManual();

    await waitFor(() =>
      expect(getShownCommands().join('\n')).toContain('"/opt/postgres/bin/pg_combinebackup"'),
    );
  });

  it('shows the new image in the Docker script and the Docker manual steps after an edit', async () => {
    await renderIssuedRestore();
    switchToDocker();

    await editField(restoreCopy.fields.pgImage, 'registry.example.com/postgres:17');

    expect(getShownCommands()).toEqual([
      `curl -fsSL "${getScriptUrl()}" | sh -s -- --combine-image "registry.example.com/postgres:17" "${getBundleUrl()}" "${DEFAULT_DIRECTORY}"`,
    ]);

    switchToManual();

    await waitFor(() =>
      expect(getShownCommands().join('\n')).toContain(
        '-w /work registry.example.com/postgres:17 \\',
      ),
    );
  });

  it.each(['17', '18'] as const)(
    'offers and uses postgres:%s by default with a database of that version',
    async (version) => {
      await renderIssuedRestore(buildDatabaseOfVersion(version));

      switchToDocker();

      const imageField = await screen.findByRole<HTMLInputElement>('textbox', {
        name: restoreCopy.fields.pgImage,
      });

      expect(imageField.value).toBe(`postgres:${version}`);
      expect(getShownCommands()[0]).toContain(`--combine-image "postgres:${version}"`);
    },
  );
});

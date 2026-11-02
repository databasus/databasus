// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '../../../../entities/databases';
import { API_BASE_PATH } from '../../../../shared/api';
import { DICTIONARIES } from '../../../../shared/i18n';
import { buildProblem, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { stubClipboardWrite } from '../../../../shared/lib/test/stubClipboard';
import { physicalBackupsApi } from '../../api/physicalBackupsApi';
import { type PhysicalBackupListItem } from '../../model/physical-backups/PhysicalBackupListItem';
import { PhysicalBackupStatus } from '../../model/physical-backups/PhysicalBackupStatus';
import { PhysicalBackupType } from '../../model/physical-backups/PhysicalBackupType';
import { PhysicalRestoreComponent } from './PhysicalRestoreComponent';

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

  fireEvent.mouseDown(picker);
  fireEvent.change(picker, { target: { value: localTime } });
  fireEvent.click(await screen.findByRole('button', { name: 'OK' }));
};

const clearTargetTime = () => {
  fireEvent.click(document.querySelector('.ant-picker-clear') as Element);
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

describe('PhysicalRestoreComponent', () => {
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

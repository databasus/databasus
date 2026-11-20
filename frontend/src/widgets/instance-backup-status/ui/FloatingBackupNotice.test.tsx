// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { InstanceBackupStatusWidget } from '@/widgets/instance-backup-status/ui/InstanceBackupStatusWidget';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceBackupNotices;
const incidents = DICTIONARIES.en.instanceBackups.incidents;

const buildStatus = (status: Partial<InstanceBackupStatus> = {}): InstanceBackupStatus => ({
  health: 'NOT_CONFIGURED',
  destinations: [],
  incident: null,
  isEmailConfigured: true,
  isRootEmailUsable: true,
  isMasterKeyAcknowledged: true,
  isConfirmationCodeRequired: false,
  reminder: { isEligible: false },
  recovery: null,
  ...status,
});

const stoppedIncident = {
  id: '77777777-7777-4777-8777-777777777777',
  kind: 'STOPPED' as const,
  startedAt: '2026-11-14T10:00:00Z',
  mailStatus: 'SENT' as const,
};

const renderWidget = (
  status: InstanceBackupStatus,
  { isBackupSectionOpen = false, onOpenBackupSection = vi.fn() } = {},
) => {
  activateSession({ token: 'root-session', userId: 'root' });
  server.use(mockApi.get('/instance-backup-statuses', 200, status));

  return renderScreen(
    <InstanceBackupStatusWidget
      isRootAdmin
      isBackupSectionOpen={isBackupSectionOpen}
      onOpenBackupSection={onOpenBackupSection}
    />,
  );
};

const waitForStatusRead = () =>
  waitFor(() =>
    expect(
      server.getRequests().some((request) => request.url.includes('/instance-backup-statuses')),
    ).toBe(true),
  );

describe('FloatingBackupNotice', () => {
  it('reports an incident with an action to the backup section even after the email was sent', async () => {
    const onOpenBackupSection = vi.fn();
    const { user } = renderWidget(buildStatus({ health: 'STOPPED', incident: stoppedIncident }), {
      onOpenBackupSection,
    });

    expect(await screen.findByText(incidents.stopped)).toBeTruthy();
    expect(screen.getByRole('status', { name: text.label })).toBeTruthy();
    expect(document.activeElement).toBe(document.body);

    await user.click(screen.getByRole('button', { name: text.open }));

    expect(onOpenBackupSection).toHaveBeenCalledTimes(1);
  });

  it('hides the floating duplicate while the backup section is open', async () => {
    renderWidget(
      buildStatus({ health: 'FAILED', incident: { ...stoppedIncident, kind: 'FAILED' } }),
      {
        isBackupSectionOpen: true,
      },
    );
    await waitForStatusRead();

    expect(screen.queryByText(incidents.failed)).toBeNull();
  });

  it('shows the stopped incident instead of the setup reminder', async () => {
    renderWidget(
      buildStatus({
        health: 'STOPPED',
        incident: stoppedIncident,
        reminder: { isEligible: true },
      }),
    );

    expect(await screen.findByText(incidents.stopped)).toBeTruthy();
    expect(screen.queryByText(text.setup)).toBeNull();
  });

  it('reports resumption until a new copy succeeds', async () => {
    renderWidget(buildStatus({ health: 'RESUMING' }));

    expect(await screen.findByText(text.resuming)).toBeTruthy();
  });

  it('suggests setup only when the instance holds a database or storage', async () => {
    const view = renderWidget(buildStatus({ reminder: { isEligible: false } }));
    await waitForStatusRead();

    expect(screen.queryByText(text.setup)).toBeNull();

    view.unmount();
    renderWidget(buildStatus({ reminder: { isEligible: true } }));

    expect(await screen.findByText(text.setup)).toBeTruthy();
  });

  it('offers no way to put the setup reminder off', async () => {
    renderWidget(buildStatus({ reminder: { isEligible: true } }));

    const notice = await screen.findByRole('status', { name: text.label });

    expect(within(notice).getByText(text.setup)).toBeTruthy();
    expect(
      within(notice)
        .getAllByRole('button')
        .map((button) => button.textContent),
    ).toEqual([text.configure]);
  });

  it('keeps the last known incident and offers a retry when a status read fails', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      renderWidget(buildStatus({ health: 'STOPPED', incident: stoppedIncident }));
      await screen.findByText(incidents.stopped);
      server.use(mockApi.get('/instance-backup-statuses', 500, buildProblem('internal', 500)));

      await advanceFakeClock(30_000);

      expect(await screen.findByText(text.stale)).toBeTruthy();
      expect(screen.getByText(incidents.stopped)).toBeTruthy();

      server.use(mockApi.get('/instance-backup-statuses', 200, buildStatus({ health: 'HEALTHY' })));
      fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.retry }));

      await waitFor(() => expect(screen.queryByText(incidents.stopped)).toBeNull());
    } finally {
      vi.useRealTimers();
    }
  });

  it('lets setup wait while the master key is unsaved', async () => {
    renderWidget(
      buildStatus({
        isMasterKeyAcknowledged: false,
        reminder: { isEligible: true },
      }),
    );
    await waitForStatusRead();

    expect(screen.queryByText(text.setup)).toBeNull();
  });
});

// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { InstanceBackupStatus, InstanceRecovery } from '@/entities/instance-backups';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { InstanceBackupStatusWidget } from '@/widgets/instance-backup-status/ui/InstanceBackupStatusWidget';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceRecovery.pause;

const pausedRecovery: InstanceRecovery = {
  id: '88888888-8888-4888-8888-888888888888',
  status: 'PAUSED',
  origin: 'UPLOAD',
  archiveSha256: 'a'.repeat(64),
  runningRelease: '3.1.0',
  sourceRelease: '3.1.0',
  snapshotAt: '2026-11-14T09:00:00Z',
  createdAt: '2026-11-14T10:00:00Z',
  expiresAt: null,
  completedAt: '2026-11-14T10:05:00Z',
  failure: null,
  discardedCleanupCount: 0,
};

const buildStatus = (status: Partial<InstanceBackupStatus> = {}): InstanceBackupStatus => ({
  health: 'HEALTHY',
  destinations: [],
  incident: null,
  isEmailConfigured: true,
  isRootEmailUsable: true,
  isMasterKeyAcknowledged: true,
  isMasterKeyExportCodeRequired: false,
  identityProofMethods: ['PASSWORD'],
  masterKeyPath: '/databasus-data/secret.key',
  reminder: { isEligible: false },
  isRecoveryPaused: true,
  recovery: pausedRecovery,
  ...status,
});

const renderWidget = (isRootAdmin = true) => {
  activateSession({ token: 'root-session', userId: 'root' });
  server.use(mockApi.get('/instance-backup-statuses', 200, buildStatus()));

  return renderScreen(
    <InstanceBackupStatusWidget
      isRootAdmin={isRootAdmin}
      isBackupSectionOpen={false}
      onOpenBackupSection={vi.fn()}
    />,
  );
};

const activationRequests = () =>
  server.getRequests().filter((request) => request.url.endsWith('/activate'));

const openConfirmation = async (user: ReturnType<typeof renderWidget>['user']) => {
  await user.click(await screen.findByRole('button', { name: text.activate }));

  return within(await screen.findByRole('dialog', { name: text.confirmTitle }));
};

describe('RecoveryPauseNotice', () => {
  it('explains the pause and the need to stop the old instance to root admin', async () => {
    renderWidget();

    expect(await screen.findByText(text.title)).toBeTruthy();
    expect(screen.getByText(text.description)).toBeTruthy();
  });

  it('shows no pause controls to an administrator who is not root admin', async () => {
    renderWidget(false);

    expect(screen.queryByText(text.title)).toBeNull();
    expect(server.getRequests()).toHaveLength(0);
  });

  it('requires the explicit confirmation before it activates', async () => {
    const { user } = renderWidget();
    const dialog = await openConfirmation(user);

    expect(dialog.getByRole('button', { name: text.activate }).hasAttribute('disabled')).toBe(true);

    await user.click(dialog.getByRole('button', { name: DICTIONARIES.en.common.actions.cancel }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(activationRequests()).toHaveLength(0);
    expect(screen.getByText(text.title)).toBeTruthy();
  });

  it('keeps the pause and shows the refusal when activation fails', async () => {
    server.use(
      mockApi.post(
        '/instance-recoveries/{instanceRecoveryId}/activate',
        409,
        buildProblem('recovery_not_ready', 409),
      ),
    );
    const { user } = renderWidget();
    const dialog = await openConfirmation(user);

    await user.click(dialog.getByRole('checkbox', { name: text.oldInstanceStopped }));
    await user.click(dialog.getByRole('button', { name: text.activate }));

    expect(await dialog.findByText(DICTIONARIES.en.errors.recoveryNotReady)).toBeTruthy();
    expect(screen.getByText(text.title)).toBeTruthy();
  });

  it('clears the pause only after the server acknowledges activation', async () => {
    const activation = mockApi.delayed({
      method: 'post',
      path: '/instance-recoveries/{instanceRecoveryId}/activate',
      status: 200,
      body: { ...pausedRecovery, status: 'ACTIVE' },
    });
    server.use(activation.handler);
    const { user } = renderWidget();
    const dialog = await openConfirmation(user);

    await user.click(dialog.getByRole('checkbox', { name: text.oldInstanceStopped }));
    await user.click(dialog.getByRole('button', { name: text.activate }));
    await waitFor(() => expect(activationRequests()).toHaveLength(1));

    expect(screen.getByText(text.title)).toBeTruthy();

    server.use(
      mockApi.get(
        '/instance-backup-statuses',
        200,
        buildStatus({ isRecoveryPaused: false, recovery: { ...pausedRecovery, status: 'ACTIVE' } }),
      ),
    );
    activation.release();

    await waitFor(() => expect(screen.queryByText(text.title)).toBeNull());
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('keeps the pause when activation is acknowledged after the session changed', async () => {
    let isReleased = false;
    server.use(
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/activate`, async () => {
        await waitFor(() => expect(isReleased).toBe(true));
        return HttpResponse.json({ ...pausedRecovery, status: 'ACTIVE' });
      }),
    );
    const { user } = renderWidget();
    const dialog = await openConfirmation(user);

    await user.click(dialog.getByRole('checkbox', { name: text.oldInstanceStopped }));
    await user.click(dialog.getByRole('button', { name: text.activate }));
    await waitFor(() => expect(activationRequests()).toHaveLength(1));
    activateSession({ token: 'other-session', userId: 'root' });
    isReleased = true;

    expect(await screen.findByText(text.title)).toBeTruthy();
  });
});

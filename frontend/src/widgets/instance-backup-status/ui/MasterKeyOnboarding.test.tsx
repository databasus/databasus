// @vitest-environment happy-dom
import { screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { LanguageThemeControlComponent } from '@/shared/ui';
import { InstanceBackupStatusWidget } from '@/widgets/instance-backup-status/ui/InstanceBackupStatusWidget';

const server = setUpMockApi();
const text = DICTIONARIES.en.masterKey;

const buildStatus = (status: Partial<InstanceBackupStatus> = {}): InstanceBackupStatus => ({
  health: 'NOT_CONFIGURED',
  destinations: [],
  incident: null,
  isEmailConfigured: true,
  isRootEmailUsable: true,
  isMasterKeyAcknowledged: false,
  identityProofMethods: ['PASSWORD'],
  masterKeyPath: '/databasus-data/secret.key',
  reminder: { isEligible: false, deferredUntil: null },
  isRecoveryPaused: false,
  recovery: null,
  ...status,
});

const renderWidget = (status: InstanceBackupStatus, isRootAdmin = true) => {
  activateSession({ token: 'root-session', userId: 'root' });
  server.use(mockApi.get('/instance-backup-statuses', 200, status));

  return renderScreen(
    <InstanceBackupStatusWidget
      isRootAdmin={isRootAdmin}
      isBackupSectionOpen={false}
      onOpenBackupSection={vi.fn()}
    />,
  );
};

const masterKeyRequests = () =>
  server.getRequests().filter((request) => new URL(request.url).pathname.includes('/master-key'));

const followedLinks: string[] = [];

afterEach(() => {
  followedLinks.length = 0;
  sessionStorage.clear();
  vi.restoreAllMocks();
});

const recordFollowedLinks = () =>
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    followedLinks.push(this.href);
  });

describe('Master key onboarding', () => {
  it('opens on the first visit without requesting key material', async () => {
    renderWidget(buildStatus());

    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    expect(dialog.getByText(text.insecure.title)).toBeTruthy();
    expect(dialog.getByText('/databasus-data/secret.key')).toBeTruthy();
    expect(masterKeyRequests()).toHaveLength(0);
  });

  it('stays closed when the key was acknowledged in another browser', async () => {
    renderWidget(buildStatus({ isMasterKeyAcknowledged: true }));

    await waitFor(() =>
      expect(
        server.getRequests().some((request) => request.url.includes('/instance-backup-statuses')),
      ).toBe(true),
    );

    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('shows nothing to an administrator who is not root admin', async () => {
    renderWidget(buildStatus(), false);

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(server.getRequests()).toHaveLength(0);
  });

  it('leaves a smaller reminder after deferral and reopens the dialog from it', async () => {
    const { user } = renderWidget(buildStatus());

    await user.click(await screen.findByRole('button', { name: text.onboarding.later }));

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getByText(DICTIONARIES.en.instanceBackupNotices.masterKey)).toBeTruthy();
    expect(masterKeyRequests()).toHaveLength(0);

    await user.click(
      screen.getByRole('button', { name: DICTIONARIES.en.instanceBackupNotices.saveMasterKey }),
    );

    expect(await screen.findByRole('dialog', { name: text.onboarding.title })).toBeTruthy();
  });

  it('keeps the deferral after a reload within the same sign-in', async () => {
    const view = renderWidget(buildStatus());

    await view.user.click(await screen.findByRole('button', { name: text.onboarding.later }));
    view.unmount();
    renderWidget(buildStatus());

    expect(await screen.findByText(DICTIONARIES.en.instanceBackupNotices.masterKey)).toBeTruthy();
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('shows the dialog again after another sign-in', async () => {
    const { user } = renderWidget(buildStatus());

    await user.click(await screen.findByRole('button', { name: text.onboarding.later }));
    activateSession({ token: 'next-session', userId: 'root' });

    expect(await screen.findByRole('dialog', { name: text.onboarding.title })).toBeTruthy();
  });

  it('downloads the key over HTTP only after the warning is accepted and identity is confirmed', async () => {
    recordFollowedLinks();
    let exportRequest: unknown;
    server.use(
      http.post(`*${API_BASE_PATH}/master-key-exports`, async ({ request }) => {
        exportRequest = await request.json();
        return HttpResponse.json(
          { token: 'key-token', fileName: 'secret.key', expiresAt: '2026-11-14T10:01:00Z' },
          { status: 201 },
        );
      }),
    );
    const { user } = renderWidget(buildStatus());
    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    expect(dialog.queryByLabelText(text.proof.password)).toBeNull();

    await user.click(dialog.getByRole('button', { name: text.insecure.continue }));
    await user.type(dialog.getByLabelText(text.proof.password), 'correct horse');
    await user.click(dialog.getByRole('button', { name: text.download }));

    await waitFor(() => expect(followedLinks).toHaveLength(1));
    expect(followedLinks[0]).toContain('/master-key-downloads?token=key-token');
    expect(exportRequest).toEqual({
      proof: { password: 'correct horse' },
      isInsecureTransferAcknowledged: true,
    });
    expect(dialog.getByText(text.downloaded)).toBeTruthy();
    expect(dialog.getByRole('button', { name: text.acknowledge })).toBeTruthy();
  });

  it('shows the refusal and downloads nothing when the password is wrong', async () => {
    recordFollowedLinks();
    server.use(
      mockApi.post('/master-key-exports', 400, buildProblem('identity_proof_invalid', 400)),
    );
    const { user } = renderWidget(buildStatus());
    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    await user.click(dialog.getByRole('button', { name: text.insecure.continue }));
    await user.type(dialog.getByLabelText(text.proof.password), 'wrong');
    await user.click(dialog.getByRole('button', { name: text.download }));

    expect(await dialog.findByText(DICTIONARIES.en.errors.identityProofInvalid)).toBeTruthy();
    expect(followedLinks).toHaveLength(0);
  });

  it('confirms an account without a password through an emailed code', async () => {
    recordFollowedLinks();
    let exportRequest: unknown;
    server.use(
      mockApi.post('/master-key-confirmations', 201, { expiresAt: '2026-11-14T10:05:00Z' }),
      http.post(`*${API_BASE_PATH}/master-key-exports`, async ({ request }) => {
        exportRequest = await request.json();
        return HttpResponse.json(
          { token: 'key-token', fileName: 'secret.key', expiresAt: '2026-11-14T10:01:00Z' },
          { status: 201 },
        );
      }),
    );
    const { user } = renderWidget(buildStatus({ identityProofMethods: ['EMAIL_CODE'] }));
    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    await user.click(dialog.getByRole('button', { name: text.insecure.continue }));
    expect(dialog.queryByLabelText(text.proof.password)).toBeNull();
    await user.click(dialog.getByRole('button', { name: text.proof.sendCode }));
    await user.type(await dialog.findByLabelText(text.proof.code), '123456');
    await user.click(dialog.getByRole('button', { name: text.download }));

    await waitFor(() => expect(followedLinks).toHaveLength(1));
    expect(exportRequest).toEqual({
      proof: { code: '123456' },
      isInsecureTransferAcknowledged: true,
    });
  });

  it('offers only the host file when no identity proof is possible', async () => {
    renderWidget(buildStatus({ identityProofMethods: [] }));
    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    await dialog.findByText(text.host.onlyOption);
    expect(dialog.queryByRole('button', { name: text.download })).toBeNull();
    expect(dialog.queryByRole('button', { name: text.proof.sendCode })).toBeNull();
  });

  it('closes after the saved key is acknowledged', async () => {
    let acknowledgment: unknown;
    server.use(
      http.put(`*${API_BASE_PATH}/master-key-acknowledgments`, async ({ request }) => {
        acknowledgment = await request.json();
        server.use(
          mockApi.get(
            '/instance-backup-statuses',
            200,
            buildStatus({ isMasterKeyAcknowledged: true }),
          ),
        );
        return HttpResponse.json({ isSaved: true });
      }),
    );
    const { user } = renderWidget(buildStatus());
    const dialog = within(await screen.findByRole('dialog', { name: text.onboarding.title }));

    await user.click(dialog.getByRole('button', { name: text.acknowledge }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(acknowledgment).toEqual({ isSaved: true });
  });

  it('translates the dialog when the language changes', async () => {
    activateSession({ token: 'root-session', userId: 'root' });
    server.use(mockApi.get('/instance-backup-statuses', 200, buildStatus()));
    const { user } = renderScreen(
      <>
        <LanguageThemeControlComponent />
        <InstanceBackupStatusWidget
          isRootAdmin
          isBackupSectionOpen={false}
          onOpenBackupSection={vi.fn()}
        />
      </>,
    );
    await screen.findByRole('dialog', { name: text.onboarding.title });

    await user.click(screen.getByRole('button', { name: /Language:/ }));
    await user.click(
      await screen.findByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName }),
    );

    expect(
      await screen.findByRole('dialog', { name: DICTIONARIES.ru.masterKey.onboarding.title }),
    ).toBeTruthy();
  });
});

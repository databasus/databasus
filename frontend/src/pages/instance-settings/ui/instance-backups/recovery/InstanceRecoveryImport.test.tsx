// @vitest-environment happy-dom
import { createEvent, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  type InstanceRecovery,
  instanceRecoveryApi,
  isRestoredSignInExpected,
} from '@/entities/instance-backups';
import { InstanceRecoveryImport } from '@/pages/instance-settings/ui/instance-backups/recovery/InstanceRecoveryImport';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceRecovery;
const archiveFile = new File(['encrypted archive'], 'databasus.dbsbackup', {
  type: 'application/octet-stream',
});

const buildRecovery = (recovery: Partial<InstanceRecovery> = {}): InstanceRecovery => ({
  id: '99999999-9999-4999-8999-999999999999',
  status: 'VALIDATING',
  origin: 'UPLOAD',
  archiveSha256: 'b'.repeat(64),
  runningRelease: '3.1.0',
  sourceRelease: null,
  snapshotAt: null,
  createdAt: '2026-11-14T10:00:00Z',
  expiresAt: '2026-11-14T11:00:00Z',
  completedAt: null,
  failure: null,
  discardedCleanupCount: 0,
  ...recovery,
});

const readyRecovery = buildRecovery({
  status: 'READY',
  sourceRelease: '3.1.0',
  snapshotAt: '2026-11-14T09:00:00Z',
});

const renderImport = () => {
  activateSession({ token: 'root-session', userId: 'root' });

  return renderScreen(<InstanceRecoveryImport identityProofMethods={['PASSWORD']} />);
};

const uploads = () =>
  server
    .getRequests()
    .filter(
      (request) =>
        request.method === 'POST' && new URL(request.url).pathname.endsWith('/instance-recoveries'),
    );

const chooseArchive = async (user: ReturnType<typeof renderImport>['user']) => {
  const input = document.querySelector<HTMLInputElement>('input[type="file"]')!;
  await user.upload(input, archiveFile);
};

const pasteKey = (key: string) => {
  const field = screen.getByRole('textbox', { name: new RegExp(text.form.masterKey) });
  const pasteEvent = createEvent.paste(field, {
    clipboardData: { getData: () => key },
  });
  fireEvent(field, pasteEvent);
};

const acceptHttp = async (user: ReturnType<typeof renderImport>['user']) =>
  user.click(screen.getByRole('checkbox', { name: text.form.insecureAccept }));

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  sessionStorage.clear();
});

describe('InstanceRecoveryImport', () => {
  it('names the missing archive and the empty key without uploading', async () => {
    const { user } = renderImport();

    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(screen.getByText(text.form.archiveMissing)).toBeTruthy();
    expect(screen.getByText(text.form.masterKeyMissing)).toBeTruthy();
    expect(uploads()).toHaveLength(0);
  });

  it('uploads nothing over HTTP until the risk is accepted', async () => {
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');

    expect(screen.getByRole('button', { name: text.form.upload }).hasAttribute('disabled')).toBe(
      true,
    );
    expect(uploads()).toHaveLength(0);
  });

  it('sends the pasted key with its exact whitespace and line endings', async () => {
    let uploadedKey: unknown;
    let uploadedFileName: unknown;
    server.use(
      http.post(`*${API_BASE_PATH}/instance-recoveries`, async ({ request }) => {
        const form = await request.formData();
        uploadedKey = await (form.get('masterKey') as Blob).text();
        uploadedFileName = (form.get('archive') as File).name;
        return HttpResponse.json(readyRecovery, { status: 202 });
      }),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('  source-key\r\nsecond line\n');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(text.form.ready)).toBeTruthy();
    expect(uploadedKey).toBe('  source-key\r\nsecond line\n');
    expect(uploadedFileName).toBe(archiveFile.name);
  });

  it('shows the snapshot after preparation finishes', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    server.use(
      mockApi.post('/instance-recoveries', 202, buildRecovery()),
      mockApi.get('/instance-recoveries/{instanceRecoveryId}', 200, readyRecovery),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    expect(await screen.findByText(text.form.preparing)).toBeTruthy();

    await advanceFakeClock(2_000);

    expect(await screen.findByText(text.form.ready)).toBeTruthy();
    expect(screen.getByText(/Databasus 3\.1\.0/)).toBeTruthy();
  });

  it('keeps the inputs for correction when the key does not open the archive', async () => {
    server.use(
      mockApi.post(
        '/instance-recoveries',
        202,
        buildRecovery({ status: 'FAILED', failure: 'MASTER_KEY_INVALID' }),
      ),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('wrong-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(text.failures.masterKeyInvalid)).toBeTruthy();
    expect(screen.getByText(text.form.correctAndRetry)).toBeTruthy();
    expect(screen.getByRole('button', { name: text.form.upload })).toBeTruthy();
  });

  it('names the release the archive needs when versions differ', async () => {
    server.use(
      mockApi.post(
        '/instance-recoveries',
        202,
        buildRecovery({ status: 'FAILED', failure: 'RELEASE_MISMATCH', sourceRelease: '3.0.4' }),
      ),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(/Recover it with Databasus 3\.0\.4/)).toBeTruthy();
  });

  it('shows the upload limit refusal', async () => {
    server.use(mockApi.post('/instance-recoveries', 413, buildProblem('archive_too_large', 413)));
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(DICTIONARIES.en.errors.archiveTooLarge)).toBeTruthy();
  });

  it('cancels a prepared recovery and clears the form', async () => {
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      mockApi.post('/instance-recoveries/{instanceRecoveryId}/cancel', 200, {
        ...readyRecovery,
        status: 'CANCELLED',
      }),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await user.click(
      await screen.findByRole('button', { name: DICTIONARIES.en.common.actions.cancel }),
    );

    expect(await screen.findByRole('button', { name: text.form.upload })).toBeTruthy();
    expect(
      (
        screen.getByRole('textbox', {
          name: new RegExp(text.form.masterKey),
        }) as HTMLTextAreaElement
      ).value,
    ).toBe('');
  });

  it('sends no apply when the replacement confirmation is cancelled', async () => {
    server.use(mockApi.post('/instance-recoveries', 202, readyRecovery));
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await user.click(await screen.findByRole('button', { name: text.form.replace }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(dialog.getByRole('button', { name: DICTIONARIES.en.common.actions.cancel }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(server.getRequests().filter((request) => request.url.endsWith('/apply'))).toHaveLength(
      0,
    );
    expect(screen.getByText(text.form.ready)).toBeTruthy();
  });

  it('applies once with confirmation and identity proof and guides to restored sign-in', async () => {
    const applyRequests: unknown[] = [];
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/apply`, async ({ request }) => {
        applyRequests.push(await request.json());
        return HttpResponse.json({ ...readyRecovery, status: 'APPLYING' }, { status: 202 });
      }),
      mockApi.network('get', '/instance-recoveries/{instanceRecoveryId}'),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await user.click(await screen.findByRole('button', { name: text.form.replace }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    const submit = dialog.getByRole('button', { name: text.apply.submit });

    expect(submit.hasAttribute('disabled')).toBe(true);

    await user.click(dialog.getByRole('checkbox', { name: text.apply.confirmReplacement }));
    await user.type(dialog.getByLabelText(DICTIONARIES.en.masterKey.proof.password), 'root-pass');
    await user.dblClick(submit);

    expect(await screen.findByText(text.form.applying)).toBeTruthy();
    expect(screen.getByText(text.form.signInWithRestored)).toBeTruthy();
    expect(applyRequests).toEqual([
      { isReplacementConfirmed: true, proof: { password: 'root-pass' } },
    ]);
    expect(isRestoredSignInExpected()).toBe(true);
  });

  it('shows nothing of a preparation that finishes after the session changed', async () => {
    const upload = mockApi.delayed({
      method: 'post',
      path: '/instance-recoveries',
      status: 202,
      body: readyRecovery,
    });
    server.use(upload.handler);
    const createRecovery = vi.spyOn(instanceRecoveryApi, 'createRecovery');
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await acceptHttp(user);
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await waitFor(() => expect(uploads()).toHaveLength(1));
    activateSession({ token: 'other-session', userId: 'other' });
    await releaseAndSettle(createRecovery, upload.release);

    expect(screen.queryByText(text.form.ready)).toBeNull();
    expect(screen.queryByRole('button', { name: text.form.replace })).toBeNull();
  });
});

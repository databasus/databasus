// @vitest-environment happy-dom
import { createEvent, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { instanceRecoveryApi, isRestoredSignInExpected } from '@/entities/instance-backups';
import { buildRecovery } from '@/pages/instance-settings/ui/instance-backups/instanceBackups.fixture';
import { InstanceRecoveryDialog } from '@/pages/instance-settings/ui/instance-backups/recovery/InstanceRecoveryDialog';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { mockSignedInAccount } from '@/shared/lib/test/mockSignedInAccount';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const text = DICTIONARIES.en.instanceRecovery;
const archiveFile = new File(['encrypted archive'], 'databasus.dbsbackup', {
  type: 'application/octet-stream',
});

const readyRecovery = buildRecovery({
  status: 'READY',
  sourceRelease: '3.1.0',
  snapshotAt: '2026-11-14T09:00:00Z',
});

const REOPEN_LABEL = 'Open recovery';

const RecoveryDialogHost = ({ isCodeRequired }: { isCodeRequired: boolean }) => {
  const [isOpen, setIsOpen] = useState(true);

  return isOpen ? (
    <InstanceRecoveryDialog isCodeRequired={isCodeRequired} onClose={() => setIsOpen(false)} />
  ) : (
    <button type="button" onClick={() => setIsOpen(true)}>
      {REOPEN_LABEL}
    </button>
  );
};

const renderImport = ({ isCodeRequired = false, isAuthenticatorAppConnected = false } = {}) => {
  activateSession({ token: 'root-session', userId: 'root' });
  server.use(mockSignedInAccount({ isAuthenticatorAppConnected }));

  return renderScreen(<RecoveryDialogHost isCodeRequired={isCodeRequired} />);
};

const cancellations = () =>
  server.getRequests().filter((request) => new URL(request.url).pathname.endsWith('/cancel'));

const readKeyField = () => (screen.getByLabelText(text.form.masterKey) as HTMLInputElement).value;

const reopen = async (user: ReturnType<typeof renderImport>['user']) => {
  await user.click(await screen.findByRole('button', { name: REOPEN_LABEL }));
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
  const field = screen.getByLabelText(text.form.masterKey);
  const pasteEvent = createEvent.paste(field, {
    clipboardData: { getData: () => key },
  });
  fireEvent(field, pasteEvent);
};

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  sessionStorage.clear();
});

describe('InstanceRecoveryDialog', () => {
  it('names the missing archive and the empty key without uploading', async () => {
    const { user } = renderImport();

    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(screen.getByText(text.form.archiveMissing)).toBeTruthy();
    expect(screen.getByText(text.form.masterKeyMissing)).toBeTruthy();
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
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(/Recover it with Databasus 3\.0\.4/)).toBeTruthy();
  });

  it('shows the upload limit refusal', async () => {
    server.use(mockApi.post('/instance-recoveries', 413, buildProblem('archive_too_large', 413)));
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));

    expect(await screen.findByText(DICTIONARIES.en.errors.archiveTooLarge)).toBeTruthy();
  });

  it('cancels a prepared recovery and starts empty when reopened', async () => {
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
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(dialog.getByRole('button', { name: DICTIONARIES.en.common.actions.cancel }));
    await reopen(user);

    expect(cancellations()).toHaveLength(1);
    expect(server.getRequests().filter((request) => request.url.endsWith('/apply'))).toHaveLength(
      0,
    );
    expect(readKeyField()).toBe('');
    expect(screen.queryByText(archiveFile.name)).toBeNull();
  });

  it('cancels the recovery on the server when the dialog closes during preparation', async () => {
    server.use(
      mockApi.post('/instance-recoveries', 202, buildRecovery()),
      mockApi.post('/instance-recoveries/{instanceRecoveryId}/cancel', 200, {
        ...buildRecovery(),
        status: 'CANCELLED',
      }),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await screen.findByText(text.form.preparing);
    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.close }));

    expect(await screen.findByRole('button', { name: REOPEN_LABEL })).toBeTruthy();
    expect(cancellations()).toHaveLength(1);
  });

  it('stays open with the error when the recovery cannot be cancelled', async () => {
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      mockApi.post(
        '/instance-recoveries/{instanceRecoveryId}/cancel',
        500,
        buildProblem('internal', 500),
      ),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = await screen.findByRole('dialog', { name: text.apply.title });
    await user.click(
      within(dialog).getByRole('button', { name: DICTIONARIES.en.common.actions.close }),
    );

    expect(await within(dialog).findByText(DICTIONARIES.en.errors.internal)).toBeTruthy();
    expect(screen.getByRole('dialog', { name: text.apply.title })).toBeTruthy();
    expect(screen.queryByRole('button', { name: REOPEN_LABEL })).toBeNull();
  });

  it('aborts the upload without a server-side cancellation when the dialog closes', async () => {
    let uploadSignal: AbortSignal | undefined;
    server.use(
      http.post(`*${API_BASE_PATH}/instance-recoveries`, async ({ request }) => {
        uploadSignal = request.signal;
        await new Promise((resolve) => request.signal.addEventListener('abort', resolve));
        return HttpResponse.json(readyRecovery, { status: 202 });
      }),
    );
    const { user } = renderImport();

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await waitFor(() => expect(uploadSignal).toBeDefined());
    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.close }));

    await waitFor(() => expect(uploadSignal?.aborted).toBe(true));
    expect(await screen.findByRole('button', { name: REOPEN_LABEL })).toBeTruthy();
    expect(cancellations()).toHaveLength(0);
  });

  it('applies once with the confirmation alone without two-factor sign-in and cannot be closed while restoring', async () => {
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
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    expect(dialog.getByText(/Databasus 3\.1\.0/)).toBeTruthy();
    const submit = dialog.getByRole('button', { name: text.apply.submit });

    expect(submit.hasAttribute('disabled')).toBe(true);

    await user.click(dialog.getByRole('checkbox', { name: text.apply.confirmReplacement }));
    await user.dblClick(submit);

    expect(await screen.findByText(text.form.applying)).toBeTruthy();
    expect(screen.getByText(text.form.signInWithRestored)).toBeTruthy();
    expect(applyRequests).toEqual([{ isReplacementConfirmed: true }]);
    expect(isRestoredSignInExpected()).toBe(true);

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('button', { name: DICTIONARIES.en.common.actions.close })).toBeNull();
    expect(screen.getByText(text.form.applying)).toBeTruthy();
  });

  it('asks for the emailed code with two-factor sign-in and says beside it that the code is wrong', async () => {
    const applyRequests: unknown[] = [];
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      mockApi.post('/instance-recoveries/{instanceRecoveryId}/confirmations', 201, {
        method: 'EMAIL',
        expiresAt: '2026-11-18T10:05:00Z',
      }),
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/apply`, async ({ request }) => {
        applyRequests.push(await request.json());
        return HttpResponse.json(buildProblem('identity_proof_invalid', 400), { status: 400 });
      }),
    );
    const { user } = renderImport({ isCodeRequired: true });

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(dialog.getByRole('checkbox', { name: text.apply.confirmReplacement }));
    const submit = dialog.getByRole('button', { name: text.apply.submit });

    expect(submit.hasAttribute('disabled')).toBe(true);

    await user.click(
      await dialog.findByRole('button', { name: DICTIONARIES.en.masterKey.proof.sendCode }),
    );
    const codeField = await dialog.findByLabelText(DICTIONARIES.en.masterKey.proof.code);
    await user.type(codeField, '000000');
    await user.click(submit);

    expect(await dialog.findByText(DICTIONARIES.en.masterKey.proof.wrongCode)).toBeTruthy();
    expect(dialog.queryByText(DICTIONARIES.en.errors.identityProofInvalid)).toBeNull();
    expect(applyRequests).toEqual([{ isReplacementConfirmed: true, code: '000000' }]);

    await user.type(codeField, '1');

    expect(dialog.queryByText(DICTIONARIES.en.masterKey.proof.wrongCode)).toBeNull();
  });

  it('shows the sending error in the replacement dialog when the code cannot be sent', async () => {
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/confirmations`, () =>
        HttpResponse.json(buildProblem('identity_proof_unavailable', 409), { status: 409 }),
      ),
    );
    const { user } = renderImport({ isCodeRequired: true });

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(
      await dialog.findByRole('button', { name: DICTIONARIES.en.masterKey.proof.sendCode }),
    );

    expect(await dialog.findByText(DICTIONARIES.en.errors.identityProofUnavailable)).toBeTruthy();
    expect(dialog.queryByLabelText(DICTIONARIES.en.masterKey.proof.code)).toBeNull();
  });

  it('starts the replacement after a correct app code when root admin has an app', async () => {
    const applyRequests: unknown[] = [];
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/apply`, async ({ request }) => {
        applyRequests.push(await request.json());
        return HttpResponse.json({ ...readyRecovery, status: 'APPLYING' }, { status: 202 });
      }),
      mockApi.network('get', '/instance-recoveries/{instanceRecoveryId}'),
    );
    const { user } = renderImport({ isCodeRequired: true, isAuthenticatorAppConnected: true });

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(dialog.getByRole('checkbox', { name: text.apply.confirmReplacement }));
    await user.type(
      await dialog.findByLabelText(DICTIONARIES.en.authenticatorApp.code.appCode),
      '123456',
    );
    await user.click(dialog.getByRole('button', { name: text.apply.submit }));

    expect(await screen.findByText(text.form.applying)).toBeTruthy();
    expect(screen.getByText(text.form.signInWithRestored)).toBeTruthy();
    expect(applyRequests).toEqual([{ isReplacementConfirmed: true, code: '123456' }]);
    expect(isRestoredSignInExpected()).toBe(true);
  });

  it('asks for an app or recovery code without a send action when root admin has an app', async () => {
    const applyRequests: unknown[] = [];
    server.use(
      mockApi.post('/instance-recoveries', 202, readyRecovery),
      http.post(`*${API_BASE_PATH}/instance-recoveries/:recoveryId/apply`, async ({ request }) => {
        applyRequests.push(await request.json());
        return HttpResponse.json(buildProblem('identity_proof_invalid', 400), { status: 400 });
      }),
    );
    const { user } = renderImport({ isCodeRequired: true, isAuthenticatorAppConnected: true });
    const codeText = DICTIONARIES.en.authenticatorApp.code;

    await chooseArchive(user);
    pasteKey('source-key');
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    const dialog = within(await screen.findByRole('dialog', { name: text.apply.title }));
    await user.click(dialog.getByRole('checkbox', { name: text.apply.confirmReplacement }));
    await user.click(await dialog.findByRole('button', { name: codeText.useRecoveryCode }));
    await user.type(dialog.getByLabelText(codeText.recoveryCode), 'abcde-fghij');
    await user.click(dialog.getByRole('button', { name: text.apply.submit }));

    expect(await dialog.findByText(DICTIONARIES.en.errors.secondFactorInvalid)).toBeTruthy();
    expect(
      dialog.queryByRole('button', { name: DICTIONARIES.en.masterKey.proof.sendCode }),
    ).toBeNull();
    expect(applyRequests).toEqual([{ isReplacementConfirmed: true, code: 'abcde-fghij' }]);
    expect(
      server
        .getRequests()
        .some((request) => new URL(request.url).pathname.endsWith('/confirmations')),
    ).toBe(false);
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
    await user.click(screen.getByRole('button', { name: text.form.upload }));
    await waitFor(() => expect(uploads()).toHaveLength(1));
    activateSession({ token: 'other-session', userId: 'other' });
    await releaseAndSettle(createRecovery, upload.release);

    expect(screen.queryByText(text.form.ready)).toBeNull();
    expect(screen.queryByRole('button', { name: text.apply.submit })).toBeNull();
  });
});

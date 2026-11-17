// @vitest-environment happy-dom
import { screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { InstanceBackupStatus } from '@/entities/instance-backups';
import { MasterKeyOnboardingPage } from '@/pages/master-key-onboarding/ui/MasterKeyOnboardingPage';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const server = setUpMockApi();
const text = DICTIONARIES.en.masterKey;
const installedKey = 'k3Jd-installed-master-key-9fQa\n';
const keyPreview = 'k3Jd••••••••••••9fQa';

const buildStatus = (status: Partial<InstanceBackupStatus> = {}): InstanceBackupStatus => ({
  health: 'NOT_CONFIGURED',
  destinations: [],
  incident: null,
  isEmailConfigured: true,
  isRootEmailUsable: true,
  isMasterKeyAcknowledged: false,
  isMasterKeyExportCodeRequired: false,
  identityProofMethods: ['PASSWORD'],
  masterKeyPath: '/databasus-data/secret.key',
  reminder: { isEligible: false },
  isRecoveryPaused: false,
  recovery: null,
  ...status,
});

const renderPage = (
  status: InstanceBackupStatus = buildStatus(),
  onAcknowledged: () => Promise<unknown> = async () => {},
) => {
  activateSession({ token: 'root-session', userId: 'root' });
  server.use(
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
  );

  return renderScreen(<MasterKeyOnboardingPage status={status} onAcknowledged={onAcknowledged} />);
};

afterEach(() => {
  vi.restoreAllMocks();
});

const useKeyExport = () => {
  const exportRequests: unknown[] = [];
  server.use(
    http.post(`*${API_BASE_PATH}/master-key-exports`, async ({ request }) => {
      exportRequests.push(await request.json());
      return HttpResponse.json(
        {
          token: 'key-token',
          fileName: 'databasus-master-key.txt',
          expiresAt: '2999-01-01T00:00:00Z',
        },
        { status: 201 },
      );
    }),
    http.get(`*${API_BASE_PATH}/master-key-downloads`, ({ request }) =>
      new URL(request.url).searchParams.get('token') === 'key-token'
        ? HttpResponse.text(installedKey)
        : HttpResponse.json(buildProblem('unauthenticated', 401), { status: 401 }),
    ),
  );

  return exportRequests;
};

const useAcknowledgment = () => {
  const acknowledgments: unknown[] = [];
  server.use(
    http.put(`*${API_BASE_PATH}/master-key-acknowledgments`, async ({ request }) => {
      acknowledgments.push(await request.json());
      return HttpResponse.json({ isSaved: true });
    }),
  );

  return acknowledgments;
};

const recordSavedFiles = () => {
  const savedFiles: { fileName: string; blob: Blob }[] = [];
  const blobs = new Map<string, Blob>();
  vi.spyOn(URL, 'createObjectURL').mockImplementation((blob) => {
    const url = `blob:saved-${blobs.size}`;
    blobs.set(url, blob as Blob);
    return url;
  });
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    const blob = blobs.get(this.href);
    if (blob) savedFiles.push({ fileName: this.download, blob });
  });

  return savedFiles;
};

const getCloseButton = () => screen.getByRole('button', { name: text.onboarding.close });

const findTitle = () => screen.findByRole('heading', { name: text.onboarding.title, level: 1 });

describe('Master key onboarding page', () => {
  it('reads the key without a code and keeps the confirmation disabled until it is saved', async () => {
    const exportRequests = useKeyExport();
    renderPage();

    await findTitle();
    expect(await screen.findByText(keyPreview)).toBeTruthy();
    expect(screen.getByText(text.description)).toBeTruthy();
    expect(screen.queryByText(installedKey.trim())).toBeNull();
    expect(getCloseButton().hasAttribute('disabled')).toBe(true);
    expect(exportRequests).toEqual([{}]);
    expect(screen.queryByRole('button', { name: /later/i })).toBeNull();
  });

  it('copies the whole key, then records the acknowledgment and reports it', async () => {
    const copiedTexts: string[] = [];
    vi.spyOn(BrowserClipboard, 'copyToClipboard').mockImplementation(async (copiedText) => {
      copiedTexts.push(copiedText);
    });
    useKeyExport();
    const acknowledgments = useAcknowledgment();
    const onAcknowledged = vi.fn(async () => {});
    const { user } = renderPage(buildStatus(), onAcknowledged);
    await screen.findByText(keyPreview);

    await user.click(screen.getByRole('button', { name: text.copy }));

    expect(await screen.findByRole('button', { name: text.copied })).toBeTruthy();
    expect(copiedTexts).toEqual([installedKey]);
    expect(getCloseButton().hasAttribute('disabled')).toBe(false);

    await user.click(getCloseButton());

    await waitFor(() => expect(onAcknowledged).toHaveBeenCalledTimes(1));
    expect(acknowledgments).toEqual([{ isSaved: true }]);
  });

  it('downloads the key as a text file from the key already read and enables the confirmation', async () => {
    const savedFiles = recordSavedFiles();
    const exportRequests = useKeyExport();
    const { user } = renderPage();
    await screen.findByText(keyPreview);

    await user.click(screen.getByRole('button', { name: text.download }));

    expect(await screen.findByRole('button', { name: text.downloaded })).toBeTruthy();
    expect(savedFiles.map((savedFile) => savedFile.fileName)).toEqual(['databasus-master-key.txt']);
    expect(await savedFiles[0].blob.text()).toBe(installedKey);
    expect(exportRequests).toHaveLength(1);
    expect(getCloseButton().hasAttribute('disabled')).toBe(false);
  });

  it('reports a refused copy and keeps the confirmation disabled', async () => {
    vi.spyOn(BrowserClipboard, 'copyToClipboard').mockRejectedValue(new Error('refused'));
    useKeyExport();
    const { user } = renderPage();
    await screen.findByText(keyPreview);

    await user.click(screen.getByRole('button', { name: text.copy }));

    expect(await screen.findByText(text.copyFailed)).toBeTruthy();
    expect(getCloseButton().hasAttribute('disabled')).toBe(true);
  });

  it('shows the refusal and stays in place when the acknowledgment fails', async () => {
    vi.spyOn(BrowserClipboard, 'copyToClipboard').mockResolvedValue();
    useKeyExport();
    server.use(
      mockApi.put('/master-key-acknowledgments', 403, buildProblem('permission_denied', 403)),
    );
    const onAcknowledged = vi.fn(async () => {});
    const { user } = renderPage(buildStatus(), onAcknowledged);
    await screen.findByText(keyPreview);

    await user.click(screen.getByRole('button', { name: text.copy }));
    await user.click(getCloseButton());

    expect(await screen.findByText(DICTIONARIES.en.errors.permissionDenied)).toBeTruthy();
    expect(onAcknowledged).not.toHaveBeenCalled();
  });

  it('reads the key with an emailed code when two-factor sign-in is on', async () => {
    server.use(
      mockApi.post('/master-key-confirmations', 201, { expiresAt: '2999-01-01T00:00:00Z' }),
    );
    const exportRequests = useKeyExport();
    const { user } = renderPage(buildStatus({ isMasterKeyExportCodeRequired: true }));
    await findTitle();

    expect(screen.getByText(text.codeRequired)).toBeTruthy();
    expect(exportRequests).toHaveLength(0);

    await user.click(screen.getByRole('button', { name: text.proof.sendCode }));
    await user.type(await screen.findByLabelText(text.proof.code), '123456');
    await user.click(screen.getByRole('button', { name: text.proof.confirm }));

    expect(await screen.findByText(keyPreview)).toBeTruthy();
    expect(exportRequests).toEqual([{ code: '123456' }]);
    expect(getCloseButton().hasAttribute('disabled')).toBe(true);
  });

  it('explains copying the key from the server when the code cannot be sent', async () => {
    server.use(
      mockApi.post(
        '/master-key-confirmations',
        409,
        buildProblem('identity_proof_unavailable', 409),
      ),
    );
    const { user } = renderPage(buildStatus({ isMasterKeyExportCodeRequired: true }));
    await findTitle();

    await user.click(screen.getByRole('button', { name: text.proof.sendCode }));

    expect(await screen.findByText(DICTIONARIES.en.errors.identityProofUnavailable)).toBeTruthy();
    expect(screen.getByText(text.codeFailed)).toBeTruthy();
    expect(screen.getByText('/databasus-data/secret.key')).toBeTruthy();
    expect(screen.getByText(/docker cp databasus:\/databasus-data\/secret\.key/)).toBeTruthy();
    expect(getCloseButton().hasAttribute('disabled')).toBe(true);
  });

  it('shows the refusal and no key when the code is wrong', async () => {
    server.use(
      mockApi.post('/master-key-confirmations', 201, { expiresAt: '2999-01-01T00:00:00Z' }),
      mockApi.post('/master-key-exports', 400, buildProblem('identity_proof_invalid', 400)),
    );
    const { user } = renderPage(buildStatus({ isMasterKeyExportCodeRequired: true }));
    await findTitle();

    await user.click(screen.getByRole('button', { name: text.proof.sendCode }));
    await user.type(await screen.findByLabelText(text.proof.code), '000000');
    await user.click(screen.getByRole('button', { name: text.proof.confirm }));

    expect(await screen.findByText(DICTIONARIES.en.errors.identityProofInvalid)).toBeTruthy();
    expect(screen.queryByRole('button', { name: text.copy })).toBeNull();
  });

  it('translates the page through its language control', async () => {
    useKeyExport();
    const { user } = renderPage();
    await findTitle();

    await user.click(screen.getByRole('button', { name: /Language:/ }));
    await user.click(
      await screen.findByRole('menuitemradio', { name: DICTIONARIES.ru.language.nativeName }),
    );

    expect(
      await screen.findByRole('heading', { name: DICTIONARIES.ru.masterKey.onboarding.title }),
    ).toBeTruthy();
  });
});

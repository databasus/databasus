// @vitest-environment happy-dom
import { act, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, onTestFinished, vi } from 'vitest';

import { verificationRunsApi } from '@/pages/databases/verification/api/verificationRunsApi';
import type { VerificationDetails } from '@/pages/databases/verification/model/verificationContract';
import { VerificationDetailDrawer } from '@/pages/databases/verification/ui/VerificationDetailDrawer';
import { API_BASE_PATH } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboardWrite } from '@/shared/lib/test/stubClipboard';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const buildDetails = (change: Partial<VerificationDetails> = {}): VerificationDetails => ({
  id: 'verification-id',
  databaseId: 'database-id',
  backupId: 'backup-id',
  trigger: 'MANUAL',
  status: 'COMPLETED',
  attemptCount: 1,
  createdAt: '2026-10-27T10:00:00Z',
  tableStats: [],
  ...change,
});

const renderDrawer = (verificationId: string) => (
  <VerificationDetailDrawer
    verificationId={verificationId}
    isOpen={true}
    onClose={() => {}}
    onClosed={() => {}}
  />
);

describe('VerificationDetailDrawer', () => {
  it('shows the newest verification when an earlier one answers after it', async () => {
    const heldFirst = createPendingApiResponse();
    const detailsCall = vi.spyOn(verificationRunsApi, 'getById');
    mockServer.use(
      http.get(`*${API_BASE_PATH}/verifications/:verificationId`, ({ params }) =>
        params.verificationId === 'first'
          ? heldFirst.response
          : HttpResponse.json(buildDetails({ schemaCount: 22 })),
      ),
    );
    const { rerender } = renderScreen(renderDrawer('first'));
    await waitFor(() => expect(detailsCall).toHaveBeenCalledOnce());

    rerender(renderDrawer('second'));
    expect(await screen.findByText('22')).toBeTruthy();

    await releaseAndSettle(
      detailsCall,
      () => heldFirst.respond(HttpResponse.json(buildDetails({ schemaCount: 11 }))),
      0,
    );

    expect(screen.getByText('22')).toBeTruthy();
    expect(screen.queryByText('11')).toBeNull();
  });

  it('gives each table its own row key when two schemas hold a table of the same name', async () => {
    const reportError = vi.spyOn(console, 'error').mockImplementation(() => {});
    mockServer.use(
      mockApi.get(
        '/verifications/{verificationId}',
        200,
        buildDetails({
          tableStats: [
            { schemaName: 'public', name: 'orders', rowCount: 20 },
            { schemaName: 'archive', name: 'orders', rowCount: 10 },
          ],
        }),
      ),
    );

    renderScreen(renderDrawer('first'));

    expect(await screen.findAllByText('orders')).toHaveLength(2);
    expect(screen.getByText('public')).toBeTruthy();
    expect(screen.getByText('archive')).toBeTruthy();
    expect(reportError.mock.calls.filter(([report]) => /\bkey\b/.test(String(report)))).toEqual([]);
  });
});

const diagnosticCopy = DICTIONARIES.en.verification.runs.details.diagnostic;
const failureDetails = (change: Partial<VerificationDetails> = {}) =>
  buildDetails({
    status: 'FAILED',
    failMessage: 'Restore failed with code 1',
    failureDiagnostic: {
      output: 'pg_restore: error: type geometry does not exist\n<literal>{{output}}</literal>',
      isTruncated: false,
    },
    ...change,
  });

const ChangeLanguage = () => {
  const { setLocale } = useLocale();
  return <button onClick={() => setLocale('ru')}>Change language</button>;
};

describe('Verification failure diagnostics', () => {
  it('copies the literal diagnostic when initially collapsed details are expanded', async () => {
    const verification = failureDetails();
    mockServer.use(mockApi.get('/verifications/{verificationId}', 200, verification));
    const { user } = renderScreen(renderDrawer('first'));
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboardWrite(writeText);

    const disclosure = await screen.findByRole('button', {
      name: new RegExp(diagnosticCopy.title),
    });
    expect(disclosure.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText(verification.failureDiagnostic!.output)).toBeNull();

    await user.click(disclosure);

    expect(
      screen.getByText(verification.failureDiagnostic!.output, { normalizer: (text) => text }),
    ).toBeTruthy();
    expect(document.querySelector('literal')).toBeNull();

    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.copy }));

    expect(await screen.findByText(diagnosticCopy.copied)).toBeTruthy();
    expect(writeText).toHaveBeenCalledWith(verification.failureDiagnostic!.output);
  });

  it('omits error details when no diagnostic was recorded', async () => {
    mockServer.use(
      mockApi.get(
        '/verifications/{verificationId}',
        200,
        failureDetails({ failureDiagnostic: undefined }),
      ),
    );
    renderScreen(renderDrawer('first'));

    await screen.findByText('Restore failed with code 1');

    expect(screen.queryByRole('button', { name: new RegExp(diagnosticCopy.title) })).toBeNull();
    expect(screen.queryByText(diagnosticCopy.unavailable)).toBeNull();
    expect(screen.queryByRole('button', { name: DICTIONARIES.en.common.actions.copy })).toBeNull();
  });

  it('keeps truncated output open and translates active copy feedback after a language change', async () => {
    const verification = failureDetails({
      failureDiagnostic: {
        output: 'first error\n[... diagnostic output truncated ...]\nlast error',
        isTruncated: true,
      },
    });
    mockServer.use(mockApi.get('/verifications/{verificationId}', 200, verification));
    const { user } = renderScreen(
      <>
        {renderDrawer('first')}
        <ChangeLanguage />
      </>,
    );
    stubClipboardWrite(vi.fn().mockRejectedValue(new Error('permission denied')));

    await user.click(await screen.findByRole('button', { name: new RegExp(diagnosticCopy.title) }));

    expect(screen.getByText(diagnosticCopy.truncated)).toBeTruthy();

    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.copy }));

    expect(await screen.findByText(diagnosticCopy.copyFailed)).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Change language' }));

    const russianCopy = DICTIONARIES.ru.verification.runs.details.diagnostic;
    expect(await screen.findByText(russianCopy.copyFailed)).toBeTruthy();
    expect(
      screen
        .getByRole('button', { name: new RegExp(russianCopy.title) })
        .getAttribute('aria-expanded'),
    ).toBe('true');
    expect(screen.getByText(russianCopy.truncated)).toBeTruthy();
    expect(
      screen.getByText(verification.failureDiagnostic!.output, { normalizer: (text) => text }),
    ).toBeTruthy();
  });

  for (const status of ['COMPLETED', 'CANCELED', 'RUNNING', 'PENDING'] as const) {
    it(`omits failure details when the verification is ${status}`, async () => {
      mockServer.use(
        mockApi.get(
          '/verifications/{verificationId}',
          200,
          buildDetails({ status, schemaCount: 31 }),
        ),
      );
      renderScreen(renderDrawer('first'));

      await screen.findByText('31');

      expect(screen.queryByRole('button', { name: new RegExp(diagnosticCopy.title) })).toBeNull();
    });
  }

  it('resets disclosure and discards late copy feedback when the selection changes', async () => {
    mockServer.use(
      http.get(`*${API_BASE_PATH}/verifications/:verificationId`, ({ params }) =>
        HttpResponse.json(
          failureDetails({
            id: String(params.verificationId),
            failureDiagnostic: {
              output: String(params.verificationId) + ' error',
              isTruncated: false,
            },
          }),
        ),
      ),
    );
    const { user, rerender, unmount } = renderScreen(renderDrawer('first'));
    let finishCopy = () => {};
    const pendingCopy = new Promise<void>((resolve) => {
      finishCopy = resolve;
    });
    onTestFinished(() => {
      unmount();
      finishCopy();
    });
    const writeText = vi.fn(() => pendingCopy);
    stubClipboardWrite(writeText);

    await user.click(await screen.findByRole('button', { name: new RegExp(diagnosticCopy.title) }));
    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.copy }));
    await waitFor(() => expect(writeText).toHaveBeenCalledOnce());

    rerender(renderDrawer('second'));
    const disclosure = await screen.findByRole('button', {
      name: new RegExp(diagnosticCopy.title),
    });
    expect(disclosure.getAttribute('aria-expanded')).toBe('false');

    await act(async () => {
      finishCopy();
      await pendingCopy;
    });

    expect(screen.queryByText(diagnosticCopy.copied)).toBeNull();

    await user.click(disclosure);

    expect(screen.getByText('second error')).toBeTruthy();
    expect(screen.queryByText('first error')).toBeNull();
  });

  it('discards pending copy feedback when the drawer closes', async () => {
    mockServer.use(mockApi.get('/verifications/{verificationId}', 200, failureDetails()));
    const { user, rerender, unmount } = renderScreen(renderDrawer('first'));
    let finishCopy = () => {};
    const pendingCopy = new Promise<void>((resolve) => {
      finishCopy = resolve;
    });
    onTestFinished(() => {
      unmount();
      finishCopy();
    });
    stubClipboardWrite(() => pendingCopy);

    await user.click(await screen.findByRole('button', { name: new RegExp(diagnosticCopy.title) }));
    await user.click(screen.getByRole('button', { name: DICTIONARIES.en.common.actions.copy }));

    rerender(
      <VerificationDetailDrawer
        verificationId="first"
        isOpen={false}
        onClose={() => {}}
        onClosed={() => {}}
      />,
    );

    await act(async () => {
      finishCopy();
      await pendingCopy;
    });

    expect(screen.queryByText(diagnosticCopy.copied)).toBeNull();
  });
});

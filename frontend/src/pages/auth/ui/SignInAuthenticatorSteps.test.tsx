// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { AuthPage } from '@/pages/auth';
import { API_BASE_PATH } from '@/shared/api';
import { getSessionSnapshot } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const text = DICTIONARIES.en;
const codeText = text.authenticatorApp.code;
const setupSecret = 'JBSWY3DPEHPK3PXP';

const session = { token: 'completed-session', userId: 'owner-id', email: 'owner@example.com' };
const setupOutcome = {
  outcome: 'AUTHENTICATOR_SETUP_REQUIRED',
  pendingSignInId: 'pending-id',
  secret: setupSecret,
  otpauthUri: `otpauth://totp/Databasus:owner@example.com?secret=${setupSecret}&issuer=Databasus`,
  expiresAt: '2999-01-01T00:00:00Z',
} as const;
const recoveryCodes = Array.from({ length: 10 }, (_, index) => `code${index}-abcde`);

const signInWithPassword = async (outcome: object) => {
  mockServer.use(
    mockApi.get('/sign-in-options', 200, { hasAnyUser: true }),
    http.get('https://api.github.com/repos/databasus/databasus', () =>
      HttpResponse.json({ stargazers_count: 1 }),
    ),
    http.post(`*${API_BASE_PATH}/sessions`, () => HttpResponse.json(outcome)),
  );
  renderScreen(<AuthPage />);

  await screen.findByRole('button', { name: text.users.signIn.submit });
  fireEvent.change(screen.getByPlaceholderText('your@email.com'), {
    target: { value: 'owner@example.com' },
  });
  fireEvent.change(screen.getByPlaceholderText('********'), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: text.users.signIn.submit }));
};

const captureVerification = (status: number, body: object) => {
  const submittedCodes: string[] = [];
  mockServer.use(
    http.post(`*${API_BASE_PATH}/pending-sign-ins/:pendingSignInId/verify`, async ({ request }) => {
      submittedCodes.push(((await request.json()) as { code: string }).code);
      return HttpResponse.json(body, { status });
    }),
  );

  return submittedCodes;
};

const enterAppCode = async (code: string) => {
  fireEvent.change(await screen.findByLabelText(codeText.appCode), { target: { value: code } });
  fireEvent.click(screen.getByRole('button', { name: text.users.signInAuthenticatorCode.submit }));
};

describe('Sign-in app code step', () => {
  const appCodeOutcome = { outcome: 'AUTHENTICATOR_CODE_REQUIRED', pendingSignInId: 'pending-id' };

  it('opens the application with the issued session after a correct app code', async () => {
    const submittedCodes = captureVerification(200, session);
    await signInWithPassword(appCodeOutcome);

    expect(await screen.findByText(text.users.signInAuthenticatorCode.title)).toBeTruthy();
    expect(getSessionSnapshot().token).toBeUndefined();
    await enterAppCode('123456');

    await waitFor(() => expect(getSessionSnapshot().token).toBe('completed-session'));
    expect(submittedCodes).toEqual(['123456']);
  });

  it('signs in with a recovery code entered instead of the app code', async () => {
    const submittedCodes = captureVerification(200, session);
    await signInWithPassword(appCodeOutcome);

    fireEvent.click(await screen.findByRole('button', { name: codeText.useRecoveryCode }));
    fireEvent.change(screen.getByLabelText(codeText.recoveryCode), {
      target: { value: 'abcde-fghij' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: text.users.signInAuthenticatorCode.submit }),
    );

    await waitFor(() => expect(getSessionSnapshot().token).toBe('completed-session'));
    expect(submittedCodes).toEqual(['abcde-fghij']);
  });

  it('offers no action that sends a code', async () => {
    await signInWithPassword(appCodeOutcome);

    await screen.findByText(text.users.signInAuthenticatorCode.title);

    expect(screen.queryByRole('button', { name: text.users.signInCode.resend })).toBeNull();
  });

  it('keeps the step open with a message and a cleared entry after a wrong code', async () => {
    captureVerification(400, buildProblem('sign_in_code_incorrect', 400));
    await signInWithPassword(appCodeOutcome);

    await enterAppCode('000000');

    expect(await screen.findByText(text.errors.signInCodeIncorrect)).toBeTruthy();
    expect((screen.getByLabelText(codeText.appCode) as HTMLInputElement).value).toBe('');
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it('returns to the password step when the pending sign-in can no longer be completed', async () => {
    captureVerification(410, buildProblem('pending_sign_in_not_usable', 410));
    await signInWithPassword(appCodeOutcome);

    await enterAppCode('000000');

    expect(await screen.findByText(text.errors.pendingSignInNotUsable)).toBeTruthy();
    expect(await screen.findByPlaceholderText('********')).toBeTruthy();
    expect(screen.queryByLabelText(codeText.appCode)).toBeNull();
  });

  it('says that too many wrong codes were entered and activates no session', async () => {
    captureVerification(429, buildProblem('too_many_wrong_codes', 429));
    await signInWithPassword(appCodeOutcome);

    await enterAppCode('000000');

    expect(await screen.findByText(text.errors.tooManyWrongCodes)).toBeTruthy();
    expect(getSessionSnapshot().token).toBeUndefined();
  });
});

describe('Sign-in app setup step', () => {
  const captureSetupConfirmation = (status: number, body: object) => {
    const submittedCodes: string[] = [];
    mockServer.use(
      http.post(
        `*${API_BASE_PATH}/pending-sign-ins/:pendingSignInId/confirm-authenticator-app-setup`,
        async ({ request }) => {
          submittedCodes.push(((await request.json()) as { code: string }).code);
          return HttpResponse.json(body, { status });
        },
      ),
    );

    return submittedCodes;
  };

  const confirmSetup = async (code: string) => {
    fireEvent.change(await screen.findByLabelText(codeText.appCode), { target: { value: code } });
    fireEvent.click(
      screen.getByRole('button', { name: text.users.signInAuthenticatorSetup.submit }),
    );
  };

  it('shows the recovery codes and opens the application only after they are saved', async () => {
    const submittedCodes = captureSetupConfirmation(200, { ...session, recoveryCodes });
    await signInWithPassword(setupOutcome);

    expect(
      await screen.findByRole('img', { name: text.authenticatorApp.setup.qrCodeAlt }),
    ).toBeTruthy();
    expect(screen.getByText(setupSecret)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /skip/i })).toBeNull();
    await confirmSetup('123456');

    expect(await screen.findByText(recoveryCodes[0])).toBeTruthy();
    expect(screen.getByText(recoveryCodes[9])).toBeTruthy();
    expect(getSessionSnapshot().token).toBeUndefined();

    fireEvent.click(
      screen.getByRole('button', { name: text.authenticatorApp.recoveryCodes.saved }),
    );

    await waitFor(() => expect(getSessionSnapshot().token).toBe('completed-session'));
    expect(submittedCodes).toEqual(['123456']);
  });

  it('keeps the same QR code, says the code is wrong and clears the entry', async () => {
    captureSetupConfirmation(400, buildProblem('sign_in_code_incorrect', 400));
    await signInWithPassword(setupOutcome);

    await confirmSetup('000000');

    expect(await screen.findByText(text.errors.signInCodeIncorrect)).toBeTruthy();
    expect(screen.getByText(setupSecret)).toBeTruthy();
    expect((screen.getByLabelText(codeText.appCode) as HTMLInputElement).value).toBe('');
    expect(getSessionSnapshot().token).toBeUndefined();
  });

  it('returns to the password step when the setup can no longer be completed', async () => {
    captureSetupConfirmation(410, buildProblem('pending_sign_in_not_usable', 410));
    await signInWithPassword(setupOutcome);

    await confirmSetup('000000');

    expect(await screen.findByText(text.errors.pendingSignInNotUsable)).toBeTruthy();
    expect(await screen.findByPlaceholderText('********')).toBeTruthy();
    expect(screen.queryByText(setupSecret)).toBeNull();
  });
});

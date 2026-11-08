import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { ProfilePage } from '@/pages/profile/ui/ProfilePage';
// @vitest-environment happy-dom
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, getSessionSnapshot, subscribeSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();

const openProfile = async (onUserChanged = vi.fn()) => {
  activateSession({ token: 'old-session', userId: 'owner-id' });
  mockServer.use(
    mockApi.get('/users/{userId}', 200, {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Owner',
      role: 'ADMIN',
      status: 'ACTIVE',
      createdAt: '2026-10-01T00:00:00Z',
    }),
  );
  const view = renderScreen(<ProfilePage onUserChanged={onUserChanged} />);
  await screen.findByDisplayValue('owner@example.com');
  return view;
};

const submitPasswordChange = () => {
  fireEvent.change(screen.getByPlaceholderText('Enter new password'), {
    target: { value: 'replacement-password' },
  });
  fireEvent.change(screen.getByPlaceholderText('Confirm new password'), {
    target: { value: 'replacement-password' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Change password' }));
};

describe('ProfilePage', () => {
  it.each([
    { field: 'Name', value: '', message: DICTIONARIES.en.users.validation.nameRequired },
    { field: 'Name', value: '   ', message: DICTIONARIES.en.users.validation.nameRequired },
    { field: 'Email', value: '', message: DICTIONARIES.en.users.validation.emailRequired },
  ])(
    'retains the draft and session without a mutation when $field is blank ($value)',
    async ({ field, value, message }) => {
      const onUserChanged = vi.fn();
      const { user } = await openProfile(onUserChanged);
      const session = getSessionSnapshot();
      const name = screen.getByRole('textbox', { name: 'Name' });
      const email = screen.getByRole('textbox', { name: 'Email' });

      await user.clear(name);
      await user.type(name, 'Correction owner');
      await user.clear(email);
      await user.type(email, 'correction@example.com');

      const invalidField = screen.getByRole('textbox', { name: field });
      await user.clear(invalidField);
      if (value) await user.type(invalidField, value);

      await user.click(screen.getByRole('button', { name: 'Save changes' }));
      await screen.findByText(message);

      expect(invalidField.getAttribute('aria-invalid')).toBe('true');
      expect(
        document.getElementById(invalidField.getAttribute('aria-describedby')!)?.textContent,
      ).toBe(message);
      expect((name as HTMLInputElement).value).toBe(field === 'Name' ? value : 'Correction owner');
      expect((email as HTMLInputElement).value).toBe(
        field === 'Email' ? value : 'correction@example.com',
      );
      expect(getSessionSnapshot()).toBe(session);
      expect(onUserChanged).not.toHaveBeenCalled();
      expect(mockServer.getRequests().filter((request) => request.method !== 'GET')).toHaveLength(
        0,
      );
    },
  );

  it.each([
    {
      newPassword: 'short',
      confirmation: 'short',
      invalidField: 'Password',
      message: DICTIONARIES.en.users.validation.passwordTooShort.replace('{{minLength}}', '8'),
    },
    {
      newPassword: '',
      confirmation: 'correction-password',
      invalidField: 'Password',
      message: DICTIONARIES.en.users.validation.passwordTooShort.replace('{{minLength}}', '8'),
    },
    {
      newPassword: 'correction-password',
      confirmation: 'other-password',
      invalidField: 'Confirm New Password',
      message: DICTIONARIES.en.users.profile.changePassword.passwordsDoNotMatch,
    },
    {
      newPassword: 'correction-password',
      confirmation: '',
      invalidField: 'Confirm New Password',
      message: DICTIONARIES.en.users.profile.changePassword.passwordsDoNotMatch,
    },
  ])(
    'retains correction values and the session without a mutation when $invalidField is invalid ($newPassword / $confirmation)',
    async ({ newPassword, confirmation, invalidField, message }) => {
      const { user } = await openProfile();
      const session = getSessionSnapshot();
      const password = screen.getByLabelText('Password', { exact: true });
      const confirm = screen.getByLabelText('Confirm New Password', { exact: true });

      if (newPassword) await user.type(password, newPassword);
      if (confirmation) await user.type(confirm, confirmation);

      await user.click(screen.getByRole('button', { name: 'Change password' }));
      await screen.findByText(message);
      const invalidControl = screen.getByLabelText(invalidField, { exact: true });
      expect(invalidControl.getAttribute('aria-invalid')).toBe('true');
      expect(
        document.getElementById(invalidControl.getAttribute('aria-describedby')!)?.textContent,
      ).toBe(message);

      expect((password as HTMLInputElement).value).toBe(newPassword);
      expect((confirm as HTMLInputElement).value).toBe(confirmation);
      expect(getSessionSnapshot()).toBe(session);
      expect(mockServer.getRequests().filter((request) => request.method !== 'GET')).toHaveLength(
        0,
      );
      expect(
        screen.queryByText(DICTIONARIES.en.users.profile.changePassword.signInRequired),
      ).toBeNull();
    },
  );

  it.each([
    {
      field: 'Name',
      enteredValue: 'Renamed owner',
      savedProfile: { name: 'Renamed owner', email: 'owner@example.com' },
    },
    {
      field: 'Email',
      enteredValue: 'renamed@example.com',
      savedProfile: { name: 'Owner', email: 'renamed@example.com' },
    },
  ])(
    'sends the entered $field beside the stored value when the profile is saved',
    async ({ field, enteredValue, savedProfile }) => {
      const savedBodies: unknown[] = [];
      const savedUserIds: unknown[] = [];
      const acknowledgedUser = {
        id: 'owner-id',
        role: 'ADMIN' as const,
        status: 'ACTIVE' as const,
        createdAt: '2026-10-01T00:00:00Z',
        ...savedProfile,
      };
      mockServer.use(
        http.put(`*${API_BASE_PATH}/users/:userId`, async ({ request, params }) => {
          const { body } = await inspectApiRequest(request, {
            method: 'put',
            path: '/users/{userId}',
          });
          savedBodies.push(body);
          savedUserIds.push(params.userId);

          return HttpResponse.json(acknowledgedUser);
        }),
      );
      const onUserChanged = vi.fn();
      const { user } = await openProfile(onUserChanged);

      const changedField = screen.getByRole('textbox', { name: field });
      await user.clear(changedField);
      await user.type(changedField, enteredValue);
      await user.click(screen.getByRole('button', { name: 'Save changes' }));

      await screen.findByText('Profile updated successfully');

      expect(savedBodies).toEqual([savedProfile]);
      expect(savedUserIds).toEqual(['owner-id']);
      expect(onUserChanged).toHaveBeenCalledWith(acknowledgedUser);
    },
  );

  it('sends the entered password when the password is changed', async () => {
    const changedPasswordBodies: unknown[] = [];
    const changedPasswordUserIds: unknown[] = [];
    mockServer.use(
      http.put(`*${API_BASE_PATH}/users/:userId/password`, async ({ request, params }) => {
        const { body } = await inspectApiRequest(request, {
          method: 'put',
          path: '/users/{userId}/password',
        });
        changedPasswordBodies.push(body);
        changedPasswordUserIds.push(params.userId);

        return new HttpResponse(null, { status: 204 });
      }),
    );
    await openProfile();

    submitPasswordChange();

    await screen.findByText('Password changed. Sign in again with your new password.');

    expect(changedPasswordBodies).toEqual([{ newPassword: 'replacement-password' }]);
    expect(changedPasswordUserIds).toEqual(['owner-id']);
  });

  it('clears authentication and notifies listeners after a successful password change', async () => {
    mockServer.use(
      mockApi.put('/users/{userId}/password', 204, undefined),
      mockApi.post('/sessions', 200, {
        outcome: 'CODE_REQUIRED',
        pendingSignInId: 'pending-id',
        email: 'owner@example.com',
      }),
    );
    await openProfile();
    const onAuthChanged = vi.fn();
    const dispose = subscribeSession(onAuthChanged);
    try {
      submitPasswordChange();
      await waitFor(() => expect(onAuthChanged).toHaveBeenCalledOnce());
      expect(getSessionSnapshot().token).toBeUndefined();
      expect(getSessionSnapshot().userId).toBeUndefined();
      expect(
        await screen.findByText('Password changed. Sign in again with your new password.'),
      ).toBeTruthy();
    } finally {
      dispose();
    }
  });

  it('keeps the profile and session when the password change is refused', async () => {
    mockServer.use(
      mockApi.put('/users/{userId}/password', 403, buildProblem('permission_denied', 403)),
    );
    await openProfile();
    submitPasswordChange();
    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(getSessionSnapshot().token).toBe('old-session');
    expect(getSessionSnapshot().userId).toBe('owner-id');
    expect(screen.getByDisplayValue('owner@example.com')).toBeTruthy();
    expect(
      screen.queryByText('Password changed. Sign in again with your new password.'),
    ).toBeNull();
  });
  it('preserves newer input after the submitted profile is acknowledged', async () => {
    const savedUser = {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Submitted',
      role: 'ADMIN' as const,
      status: 'ACTIVE' as const,
      createdAt: '2026-10-01T00:00:00Z',
    };
    const save = mockApi.delayed({
      method: 'put',
      path: '/users/{userId}',
      status: 200,
      body: savedUser,
    });
    mockServer.use(save.handler);
    await openProfile();

    fireEvent.change(screen.getByPlaceholderText('Enter your name'), {
      target: { value: 'Submitted' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
    await waitFor(() =>
      expect(mockServer.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );

    fireEvent.change(screen.getByPlaceholderText('Enter your name'), {
      target: { value: 'Newer' },
    });
    save.release();
    await screen.findByText('Profile updated successfully');

    expect(screen.getByDisplayValue('Newer')).toBeTruthy();
    expect(await screen.findByRole('button', { name: 'Save changes' })).toBeTruthy();
  });

  it('leaves the form clean when newer input equals the normalized acknowledgement', async () => {
    const acknowledgement = {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Normalized',
      role: 'ADMIN' as const,
      status: 'ACTIVE' as const,
      createdAt: '2026-10-01T00:00:00Z',
    };
    const save = mockApi.delayed({
      method: 'put',
      path: '/users/{userId}',
      status: 200,
      body: acknowledgement,
    });
    mockServer.use(save.handler);

    await openProfile();
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Submitted' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
    await waitFor(() =>
      expect(mockServer.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Normalized' },
    });
    save.release();

    await screen.findByText('Profile updated successfully');
    expect(screen.getByDisplayValue('Normalized')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Save changes' })).toBeNull();
  });

  it('removes save eligibility when the draft returns to the acknowledged values', async () => {
    await openProfile();
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Changed' },
    });
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeTruthy();
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value: 'Owner' } });
    expect(screen.queryByRole('button', { name: 'Save changes' })).toBeNull();
  });

  it('preserves correction values when the profile save is refused', async () => {
    mockServer.use(mockApi.put('/users/{userId}', 403, buildProblem('permission_denied', 403)));
    await openProfile();

    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Correction' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
    await screen.findByText('You do not have permission to do this.');
    expect(screen.getByDisplayValue('Correction')).toBeTruthy();
  });

  it.each([
    { operation: 'save', isRefused: false },
    { operation: 'password', isRefused: false },
    { operation: 'save', isRefused: true },
    { operation: 'password', isRefused: true },
  ] as const)(
    'suppresses callbacks and notices after a pending $operation outlives its page with refusal=$isRefused',
    async ({ operation, isRefused }) => {
      const callback = vi.fn();
      const pending = isRefused
        ? mockApi.delayed({
            method: 'put',
            path: operation === 'save' ? '/users/{userId}' : '/users/{userId}/password',
            status: 403,
            body: buildProblem('permission_denied', 403),
          })
        : operation === 'save'
          ? mockApi.delayed({
              method: 'put',
              path: '/users/{userId}',
              status: 200,
              body: {
                id: 'owner-id',
                email: 'owner@example.com',
                name: 'Changed',
                role: 'ADMIN',
                status: 'ACTIVE',
                createdAt: '2026-10-01T00:00:00Z',
              },
            })
          : mockApi.delayed({
              method: 'put',
              path: '/users/{userId}/password',
              status: 204,
              body: undefined,
            });
      mockServer.use(pending.handler);
      const view = await openProfile(callback);

      if (operation === 'save') {
        fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
          target: { value: 'Changed' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
      } else submitPasswordChange();

      await waitFor(() =>
        expect(mockServer.getRequests().some((request) => request.method === 'PUT')).toBe(true),
      );

      const provider = document.querySelector('style[data-mantine-styles="classes"]');
      expect(provider).not.toBeNull();
      view.rerender(<div>Profile closed</div>);
      const responded = new Promise<void>((resolve) =>
        mockServer.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        pending.release();
        await responded;
      });

      expect(document.querySelector('style[data-mantine-styles="classes"]')).toBe(provider);
      expect(callback).not.toHaveBeenCalled();
      expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
      expect(screen.queryByText('Profile updated successfully')).toBeNull();
      expect(
        screen.queryByText('Password changed. Sign in again with your new password.'),
      ).toBeNull();
      expect(getSessionSnapshot().token).toBe('old-session');
    },
  );

  it('offers no retry when saving fails with the data loaded', async () => {
    mockServer.use(mockApi.put('/users/{userId}', 403, buildProblem('permission_denied', 403)));
    await openProfile();

    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: 'Correction' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
    await screen.findByText('You do not have permission to do this.');

    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it('recovers the current profile when the initial request fails and is retried', async () => {
    activateSession({ token: 'current', userId: 'owner-id' });
    mockServer.use(mockApi.get('/users/{userId}', 500, buildProblem('internal', 500)));
    renderScreen(<ProfilePage />);
    const retry = await screen.findByRole('button', { name: 'Retry' });

    mockServer.use(
      mockApi.get('/users/{userId}', 200, {
        id: 'owner-id',
        email: 'owner@example.com',
        name: 'Owner',
        role: 'ADMIN',
        status: 'ACTIVE',
        createdAt: '2026-10-01T00:00:00Z',
      }),
    );
    fireEvent.click(retry);

    expect(await screen.findByDisplayValue('Owner')).toBeTruthy();
  });
});

it.each([false, true])(
  'applies server normalization to an unchanged submitted field when another field has newer edits=%s',
  async (hasNewerEmail) => {
    const acknowledgedUser = {
      id: 'owner-id',
      email: 'owner@example.com',
      name: 'Normalized name',
      role: 'ADMIN' as const,
      status: 'ACTIVE' as const,
      createdAt: '2026-10-01T00:00:00Z',
    };
    const save = mockApi.delayed({
      method: 'put',
      path: '/users/{userId}',
      status: 200,
      body: acknowledgedUser,
    });
    mockServer.use(save.handler);
    const onUserChanged = vi.fn();
    const { user } = await openProfile(onUserChanged);

    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), {
      target: { value: '  Normalized name  ' },
    });
    await user.click(screen.getByRole('button', { name: 'Save changes' }));
    await waitFor(() =>
      expect(mockServer.getRequests().some((request) => request.method === 'PUT')).toBe(true),
    );

    if (hasNewerEmail)
      fireEvent.change(screen.getByRole('textbox', { name: 'Email' }), {
        target: { value: 'newer@example.com' },
      });
    save.release();
    await screen.findByText('Profile updated successfully');

    expect(screen.getByRole('textbox', { name: 'Name' }).getAttribute('value')).toBe(
      'Normalized name',
    );
    expect(screen.getByRole('textbox', { name: 'Email' }).getAttribute('value')).toBe(
      hasNewerEmail ? 'newer@example.com' : 'owner@example.com',
    );
    expect(!!screen.queryByRole('button', { name: 'Save changes' })).toBe(hasNewerEmail);
    expect(onUserChanged).toHaveBeenCalledWith(acknowledgedUser);
  },
);

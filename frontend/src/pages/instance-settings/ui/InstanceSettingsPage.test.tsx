// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type InstanceSettings } from '@/entities/users';
import { InstanceSettingsPage } from '@/pages/instance-settings/ui/InstanceSettingsPage';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { LanguageThemeControlComponent } from '@/shared/ui';

const server = setUpMockApi();
const settings: InstanceSettings = {
  isAllowExternalRegistrations: false,
  isAllowMemberInvitations: false,
  isMemberAllowedToCreateWorkspaces: false,
  isTwoFactorAuthRequired: false,
  isEmailConfigured: true,
};
const openSettings = async (onSettingsChanged = vi.fn(), initialSettings = settings) => {
  activateSession({ token: 'current-session', userId: 'admin' });
  server.use(
    mockApi.get('/instance-settings', 200, initialSettings),
    mockApi.get('/verification-agents', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
  );
  const view = renderScreen(<InstanceSettingsPage onSettingsChanged={onSettingsChanged} />);
  await screen.findByRole('switch', { name: DICTIONARIES.en.settings.externalRegistrations.title });
  return view;
};
const externalToggle = () =>
  screen.getByRole('switch', { name: DICTIONARIES.en.settings.externalRegistrations.title });

describe('InstanceSettingsPage', () => {
  it.each([true, false])(
    'derives save eligibility from the acknowledged response when a newer draft differs=%s',
    async (shouldDiffer) => {
      const callback = vi.fn();
      const save = mockApi.delayed({
        method: 'put',
        path: '/instance-settings',
        status: 200,
        body: { ...settings, isAllowExternalRegistrations: true },
      });
      server.use(save.handler);
      await openSettings(callback);

      fireEvent.click(externalToggle());
      fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));
      await waitFor(() =>
        expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
      );

      fireEvent.click(externalToggle());
      if (!shouldDiffer) fireEvent.click(externalToggle());
      save.release();
      await screen.findByText(DICTIONARIES.en.settings.updated);

      expect(callback).toHaveBeenCalledWith({ ...settings, isAllowExternalRegistrations: true });
      expect(externalToggle().getAttribute('aria-checked')).toBe(String(!shouldDiffer));
      expect(!!screen.queryByRole('button', { name: DICTIONARIES.en.settings.saveChanges })).toBe(
        shouldDiffer,
      );
    },
  );

  it.each([
    {
      settingName: 'member invitations',
      switchName: DICTIONARIES.en.settings.memberInvitations.title,
      turnedOn: { isAllowMemberInvitations: true },
    },
    {
      settingName: 'member workspace creation',
      switchName: DICTIONARIES.en.settings.memberWorkspaceCreation.title,
      turnedOn: { isMemberAllowedToCreateWorkspaces: true },
    },
    {
      settingName: 'two-factor authentication',
      switchName: DICTIONARIES.en.settings.twoFactorAuth.title,
      turnedOn: { isTwoFactorAuthRequired: true },
    },
  ])(
    'sends $settingName turned on and the other settings unchanged when it is saved with mail configured',
    async ({ switchName, turnedOn }) => {
      const savedBodies: unknown[] = [];
      server.use(
        http.put(`*${API_BASE_PATH}/instance-settings`, async ({ request }) => {
          const { body } = await inspectApiRequest(request, {
            method: 'put',
            path: '/instance-settings',
          });
          savedBodies.push(body);

          return HttpResponse.json({ ...settings, ...turnedOn });
        }),
      );
      await openSettings();

      fireEvent.click(screen.getByRole('switch', { name: switchName }));
      fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));

      await screen.findByText(DICTIONARIES.en.settings.updated);

      expect(savedBodies).toEqual([
        {
          isAllowExternalRegistrations: false,
          isAllowMemberInvitations: false,
          isMemberAllowedToCreateWorkspaces: false,
          isTwoFactorAuthRequired: false,
          ...turnedOn,
        },
      ]);
      expect(screen.getByRole('switch', { name: switchName }).getAttribute('aria-checked')).toBe(
        'true',
      );
      expect(
        screen.queryByRole('button', { name: DICTIONARIES.en.settings.saveChanges }),
      ).toBeNull();
    },
  );

  it('restores the acknowledged baseline when reset is selected', async () => {
    await openSettings();
    fireEvent.click(externalToggle());
    fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.reset }));
    expect(externalToggle().getAttribute('aria-checked')).toBe('false');
    expect(screen.queryByRole('button', { name: DICTIONARIES.en.settings.saveChanges })).toBeNull();
  });

  it('retains the draft and safe failure when saving is refused', async () => {
    server.use(mockApi.put('/instance-settings', 403, buildProblem('permission_denied', 403)));
    await openSettings();

    fireEvent.click(externalToggle());
    fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));
    await screen.findByText('You do not have permission to do this.');

    expect(externalToggle().getAttribute('aria-checked')).toBe('true');
  });

  it('offers no retry when saving fails with the data loaded', async () => {
    server.use(mockApi.put('/instance-settings', 403, buildProblem('permission_denied', 403)));
    await openSettings();

    fireEvent.click(externalToggle());
    fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));
    await screen.findByText('You do not have permission to do this.');

    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
  });

  it.each([
    { reason: 'unmount', isRefused: false },
    { reason: 'session replacement', isRefused: false },
    { reason: 'unmount', isRefused: true },
    { reason: 'session replacement', isRefused: true },
  ] as const)(
    'suppresses settings publication after $reason with refusal=$isRefused while providers remain',
    async ({ reason, isRefused }) => {
      const callback = vi.fn();
      const save = isRefused
        ? mockApi.delayed({
            method: 'put',
            path: '/instance-settings',
            status: 403,
            body: buildProblem('permission_denied', 403),
          })
        : mockApi.delayed({
            method: 'put',
            path: '/instance-settings',
            status: 200,
            body: { ...settings, isAllowExternalRegistrations: true },
          });

      server.use(save.handler);
      const view = await openSettings(callback);

      fireEvent.click(externalToggle());
      fireEvent.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));
      await waitFor(() =>
        expect(server.getRequests().some((request) => request.method === 'PUT')).toBe(true),
      );

      if (reason === 'unmount') view.rerender(<div>Settings closed</div>);
      else activateSession({ token: 'new-session', userId: 'new-owner' });
      const responded = new Promise<void>((resolve) =>
        server.events.on('response:mocked', () => resolve()),
      );
      await act(async () => {
        save.release();
        await responded;
      });

      expect(callback).not.toHaveBeenCalled();
      expect(screen.queryByText('You do not have permission to do this.')).toBeNull();
      expect(screen.queryByText(DICTIONARIES.en.settings.updated)).toBeNull();
    },
  );
});

it.each([false, true])(
  'permits only disabling two-factor authentication when mail is unavailable and requirement=%s',
  async (isTwoFactorAuthRequired) => {
    await openSettings(vi.fn(), { ...settings, isEmailConfigured: false, isTwoFactorAuthRequired });
    const toggle = screen.getByRole('switch', {
      name: DICTIONARIES.en.settings.twoFactorAuth.title,
    });
    expect(toggle.hasAttribute('disabled')).toBe(!isTwoFactorAuthRequired);
    expect(toggle.getAttribute('aria-checked')).toBe(String(isTwoFactorAuthRequired));
    if (isTwoFactorAuthRequired) {
      fireEvent.click(toggle);
      expect(toggle.getAttribute('aria-checked')).toBe('false');
      expect(
        screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }),
      ).toBeTruthy();
    }
  },
);

it('recovers the settings form when the initial request is retried after a refusal', async () => {
  activateSession({ token: 'current-session', userId: 'admin' });
  server.use(
    mockApi.get('/instance-settings', 500, buildProblem('internal', 500)),
    mockApi.get('/verification-agents', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
  );

  const { user } = renderScreen(<InstanceSettingsPage />);
  const retry = await screen.findByRole('button', { name: 'Retry' });
  server.use(mockApi.get('/instance-settings', 200, settings));
  await user.click(retry);

  expect(
    await screen.findByRole('switch', {
      name: DICTIONARIES.en.settings.externalRegistrations.title,
    }),
  ).toBeTruthy();
});

it('retains settings edits and translates the refusal when language changes during saving', async () => {
  activateSession({ token: 'settings-session', userId: 'admin' });
  const save = mockApi.delayed({
    method: 'put',
    path: '/instance-settings',
    status: 403,
    body: buildProblem('permission_denied', 403),
  });
  server.use(
    mockApi.get('/instance-settings', 200, settings),
    mockApi.get('/verification-agents', 200, { items: [] }),
    mockApi.get('/audit-events', 200, { items: [] }),
    save.handler,
  );

  const { user } = renderScreen(
    <>
      <LanguageThemeControlComponent />
      <InstanceSettingsPage />
    </>,
  );

  await user.click(
    await screen.findByRole('switch', {
      name: DICTIONARIES.en.settings.externalRegistrations.title,
    }),
  );
  await user.click(screen.getByRole('button', { name: DICTIONARIES.en.settings.saveChanges }));
  await waitFor(() =>
    expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1),
  );

  await user.click(screen.getByRole('button', { name: /Language:/ }));
  await user.click(
    await screen.findByRole('menuitem', { name: DICTIONARIES.ru.language.nativeName }),
  );

  const toggle = screen.getByRole('switch', {
    name: DICTIONARIES.ru.settings.externalRegistrations.title,
  });
  expect(toggle.getAttribute('aria-checked')).toBe('true');
  expect(
    screen
      .getByRole('button', { name: DICTIONARIES.ru.settings.saveChanges })
      .hasAttribute('disabled'),
  ).toBe(true);
  expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1);

  save.release();
  await screen.findByText(DICTIONARIES.ru.errors.permissionDenied);

  expect(toggle.getAttribute('aria-checked')).toBe('true');
  expect(
    screen
      .getByRole('button', { name: DICTIONARIES.ru.settings.saveChanges })
      .hasAttribute('disabled'),
  ).toBe(false);
  expect(server.getRequests().filter((request) => request.method === 'PUT')).toHaveLength(1);
});

// @vitest-environment happy-dom
import { act, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  type DatabaseEditDraft,
  DatabaseType,
  createDefaultDatabaseSettings,
} from '@/entities/databases';
import { CreateLimitedUserComponent } from '@/pages/databases/ui/CreateLimitedUserComponent';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const settings = createDefaultDatabaseSettings(DatabaseType.MYSQL);
const draft = {
  scenario: 'edit' as const,
  workspaceId: 'workspace-id',
  databaseId: 'database-id',
  name: 'Shop',
  notifierIds: [],
  savedSettings: settings,
  settings,
  hasUserChosenSslMode: false,
};
const renderSuggestion = (privileges: string[]) => {
  mockServer.use(mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges }));

  return renderScreen(
    <CreateLimitedUserComponent
      draft={draft}
      onLimitedUserCreated={() => {}}
      onGoBack={() => {}}
      onSkipped={() => {}}
      onLimitedUserNotSuggested={() => {}}
    />,
  );
};

const CREATE_READ_ONLY_USER = /Yes, create read-only user$/;
const FORCED_WAL_ROTATION_WARNING = /would not grant EXECUTE on pg_switch_wal\(\)/;

const generatedUser: ApiComponents['schemas']['LimitedUser'] = {
  username: 'databasus_reader',
  password: 'generated-secret',
  isForcedWalRotationAvailable: false,
};

const renderChoice = (renderedDraft: DatabaseEditDraft = draft) => {
  const callbacks = {
    onLimitedUserCreated: vi.fn(),
    onGoBack: vi.fn(),
    onSkipped: vi.fn(),
    onLimitedUserNotSuggested: vi.fn(),
  };

  mockServer.use(mockApi.post('/limited-user-checks', 200, { isAdvisable: true, privileges: [] }));

  return {
    ...callbacks,
    ...renderScreen(<CreateLimitedUserComponent draft={renderedDraft} {...callbacks} />),
  };
};

const physicalSettings = createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL);
const physicalDraft = { ...draft, savedSettings: physicalSettings, settings: physicalSettings };

describe('CreateLimitedUserComponent', () => {
  it('expands the privileges from the keyboard and exposes that state', async () => {
    const privileges = Array.from({ length: 20 }, (_, index) => `PRIVILEGE_${index}`);
    const { user } = renderSuggestion(privileges);

    const toggle = await screen.findByRole('button', { name: /PRIVILEGE_0/, expanded: false });

    expect(toggle.textContent).not.toContain('PRIVILEGE_19');

    toggle.focus();
    await user.keyboard('{Enter}');

    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    expect(toggle.textContent).toContain('PRIVILEGE_19');
  });

  it('shows short privileges as plain text with no toggle', async () => {
    renderSuggestion(['SELECT']);

    expect(await screen.findByText(/SELECT/)).toBeTruthy();
    expect(screen.queryByRole('button', { name: /SELECT/ })).toBeNull();
  });

  it('sends the connection of the draft when the suggestion is accepted', async () => {
    const sentCreations: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
    mockServer.use(
      http.post(`*${API_BASE_PATH}/limited-users`, async ({ request }) => {
        sentCreations.push(
          (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
        );

        return HttpResponse.json(generatedUser, { status: 201 });
      }),
    );
    const typedDraft = {
      ...draft,
      settings: { ...settings, host: 'db.example.com', username: 'admin', password: 'typed' },
    };
    const { user, onLimitedUserCreated } = renderChoice(typedDraft);

    await user.click(await screen.findByRole('button', { name: CREATE_READ_ONLY_USER }));

    await waitFor(() => expect(onLimitedUserCreated).toHaveBeenCalledWith(generatedUser));
    expect(sentCreations).toHaveLength(1);
    expect(sentCreations[0]).toMatchObject({
      workspaceId: 'workspace-id',
      databaseId: 'database-id',
      settings: { type: 'MYSQL', host: 'db.example.com', username: 'admin', password: 'typed' },
    });
  });

  it('shows the reason and keeps every choice when the create request is refused', async () => {
    mockServer.use(mockApi.post('/limited-users', 403, buildProblem('permission_denied', 403)));
    const { user, onLimitedUserCreated, onSkipped } = renderChoice();

    await user.click(await screen.findByRole('button', { name: CREATE_READ_ONLY_USER }));

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(onLimitedUserCreated).not.toHaveBeenCalled();
    expect(onSkipped).not.toHaveBeenCalled();
    expect(screen.getByText('Create a read-only user for Databasus?')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Back' })).toHaveProperty('disabled', false);
    expect(screen.getByRole('button', { name: 'Skip' })).toHaveProperty('disabled', false);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: CREATE_READ_ONLY_USER })).toHaveProperty(
        'disabled',
        false,
      ),
    );
  });

  it('reports the user when a second attempt follows a refused create request', async () => {
    mockServer.use(mockApi.post('/limited-users', 403, buildProblem('permission_denied', 403)));
    const { user, onLimitedUserCreated } = renderChoice();

    await user.click(await screen.findByRole('button', { name: CREATE_READ_ONLY_USER }));
    await screen.findByText('You do not have permission to do this.');
    mockServer.use(mockApi.post('/limited-users', 201, generatedUser));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: CREATE_READ_ONLY_USER })).toHaveProperty(
        'disabled',
        false,
      ),
    );
    await user.click(screen.getByRole('button', { name: CREATE_READ_ONLY_USER }));

    await waitFor(() => expect(onLimitedUserCreated).toHaveBeenCalledWith(generatedUser));
  });

  it('reports the skip only after the risks are accepted', async () => {
    const { user, onSkipped } = renderChoice();

    await user.click(await screen.findByRole('button', { name: 'Skip' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', {
        name: "Let's continue with the secure way",
      }),
    );

    expect(onSkipped).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Skip' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('button', {
        name: 'Yes, I accept risks',
      }),
    );

    expect(onSkipped).toHaveBeenCalledTimes(1);
  });

  const expectNoNoticeAfterCreation = async () => {
    await waitFor(() => expect(document.querySelector('.ant-btn-loading')).toBeNull());
    await act(async () => {
      await new Promise((resolve) => requestAnimationFrame(resolve));
    });

    expect(screen.queryByText(FORCED_WAL_ROTATION_WARNING)).toBeNull();
  };

  it('warns about WAL streaming when a physical user cannot force WAL rotation', async () => {
    mockServer.use(mockApi.post('/limited-users', 201, generatedUser));
    const { user, onLimitedUserCreated } = renderChoice(physicalDraft);

    await user.click(
      await screen.findByRole('button', { name: 'Yes, create replication-only user' }),
    );

    expect(await screen.findByText(FORCED_WAL_ROTATION_WARNING)).toBeTruthy();
    expect(onLimitedUserCreated).toHaveBeenCalledWith(generatedUser);
  });

  it('shows no WAL warning when a physical user can force WAL rotation', async () => {
    const rotatingUser = { ...generatedUser, isForcedWalRotationAvailable: true };
    mockServer.use(mockApi.post('/limited-users', 201, rotatingUser));
    const { user, onLimitedUserCreated } = renderChoice(physicalDraft);

    await user.click(
      await screen.findByRole('button', { name: 'Yes, create replication-only user' }),
    );

    await waitFor(() => expect(onLimitedUserCreated).toHaveBeenCalledWith(rotatingUser));
    await expectNoNoticeAfterCreation();
  });

  it('shows no WAL warning when a read-only user is created', async () => {
    mockServer.use(mockApi.post('/limited-users', 201, generatedUser));
    const { user, onLimitedUserCreated } = renderChoice();

    await user.click(await screen.findByRole('button', { name: CREATE_READ_ONLY_USER }));

    await waitFor(() => expect(onLimitedUserCreated).toHaveBeenCalledWith(generatedUser));
    await expectNoNoticeAfterCreation();
  });
});

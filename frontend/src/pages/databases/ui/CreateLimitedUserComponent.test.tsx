// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DatabaseType, createDefaultDatabaseSettings } from '../../../entities/databases';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { CreateLimitedUserComponent } from './CreateLimitedUserComponent';

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
});

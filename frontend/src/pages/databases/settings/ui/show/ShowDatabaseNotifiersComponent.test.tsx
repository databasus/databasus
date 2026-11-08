// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { type Database, DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import { NotifierType, notifierApi } from '@/entities/notifiers';
import { ShowDatabaseNotifiersComponent } from '@/pages/databases/settings/ui/show/ShowDatabaseNotifiersComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, invalidateSession } from '@/shared/auth';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';

const mockServer = setUpMockApi();
const REFUSAL_MESSAGE = 'You do not have permission to do this.';
const alertsNotifier = {
  id: 'alerts-id',
  workspaceId: 'workspace-id',
  name: 'Alerts',
  lastSendError: null,
  settings: { type: NotifierType.TELEGRAM, targetChatId: '1', isProxyEnabled: false },
};
const database: Database = {
  id: 'database-id',
  workspaceId: 'workspace-id',
  name: 'Shop',
  settings: createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  notifierIds: [alertsNotifier.id],
};

const secondWorkspaceDatabase = { ...database, workspaceId: 'second-workspace-id' };
const secondNotifier = { ...alertsNotifier, id: 'second-id', name: 'Second' };
const notifiersPath = `*${API_BASE_PATH}/notifiers`;
const ANSWER_OUTCOMES = [
  { outcome: 'successful', buildAnswer: () => HttpResponse.json({ items: [alertsNotifier] }) },
  {
    outcome: 'refused',
    buildAnswer: () => HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 }),
  },
];

describe('ShowDatabaseNotifiersComponent', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each(ANSWER_OUTCOMES)(
    'keeps the spinner when a $outcome load answers after the session ended',
    async ({ buildAnswer }) => {
      const loadCall = vi.spyOn(notifierApi, 'listAllNotifiers');
      const heldLoad = createPendingApiResponse();
      activateSession({ token: 'first-token', userId: 'first-user' });
      mockServer.use(http.get(notifiersPath, () => heldLoad.response));

      renderScreen(<ShowDatabaseNotifiersComponent database={database} />);
      act(() => {
        invalidateSession();
      });
      await waitFor(() => expect(loadCall).toHaveBeenCalledTimes(1));

      await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()));

      expect(document.querySelector('[role="status"]')).toBeTruthy();
      expect(screen.queryByRole('alert')).toBeNull();
      expect(screen.queryByText(/Alerts/)).toBeNull();
    },
  );

  it.each(ANSWER_OUTCOMES)(
    'keeps the view of the second workspace when a $outcome load of the first one answers late',
    async ({ buildAnswer }) => {
      const loadCall = vi.spyOn(notifierApi, 'listAllNotifiers');
      const heldLoad = createPendingApiResponse();
      mockServer.use(
        http.get(notifiersPath, ({ request }) =>
          new URL(request.url).searchParams.get('workspaceId') === database.workspaceId
            ? heldLoad.response
            : HttpResponse.json({ items: [secondNotifier] }),
        ),
      );

      const { rerender } = renderScreen(<ShowDatabaseNotifiersComponent database={database} />);
      rerender(
        <ShowDatabaseNotifiersComponent
          database={{ ...secondWorkspaceDatabase, notifierIds: [secondNotifier.id] }}
        />,
      );
      await screen.findByText(/Second/);

      await releaseAndSettle(loadCall, () => heldLoad.respond(buildAnswer()), 0);

      expect(screen.getByText(/Second/)).toBeTruthy();
      expect(screen.queryByText(/Alerts/)).toBeNull();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );

  it('hides the notifier logo beside its name from assistive technology', async () => {
    mockServer.use(mockApi.get('/notifiers', 200, { items: [alertsNotifier] }));

    renderScreen(<ShowDatabaseNotifiersComponent database={database} />);
    await screen.findByText(/Alerts/);

    expect(screen.queryByRole('img')).toBeNull();
  });

  it('shows the reason with a retry instead of a spinner when the load fails, and the notifier after a successful retry', async () => {
    const showAlert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    mockServer.use(mockApi.get('/notifiers', 403, buildProblem('permission_denied', 403)));

    renderScreen(<ShowDatabaseNotifiersComponent database={database} />);

    const retryButton = await screen.findByRole('button', { name: 'Retry' });

    expect(screen.getByText(REFUSAL_MESSAGE)).toBeTruthy();
    expect(document.querySelector('[role="status"]')).toBeNull();
    expect(showAlert).not.toHaveBeenCalled();

    mockServer.use(mockApi.get('/notifiers', 200, { items: [alertsNotifier] }));
    fireEvent.click(retryButton);

    expect(await screen.findByText(/Alerts/)).toBeTruthy();
    expect(screen.queryByText(REFUSAL_MESSAGE)).toBeNull();
  });
});

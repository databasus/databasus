// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { Notifier } from '../../../entity/notifiers';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { NotifierComponent } from './NotifierComponent';

const mockServer = setUpMockApi();

afterEach(() => {
  vi.restoreAllMocks();
});

const usedNotifier: Notifier = {
  id: 'notifier-id',
  workspaceId: 'workspace-id',
  name: 'Team alerts',
  lastSendError: null,
  settings: { type: 'DISCORD' },
};

describe('NotifierComponent', () => {
  it('shows the in_use text when a notifier that databases use is deleted', async () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onNotifierDeleted = vi.fn();
    mockServer.use(
      mockApi.get('/notifiers/{notifierId}', 200, usedNotifier),
      mockApi.delete('/notifiers/{notifierId}', 409, buildProblem('in_use', 409)),
    );
    renderScreen(
      <NotifierComponent
        notifierId={usedNotifier.id}
        onNotifierChanged={() => {}}
        onNotifierDeleted={onNotifierDeleted}
        onNotifierTransferred={() => {}}
        isCanManageNotifiers
      />,
    );

    await screen.findByText('Team alerts');
    fireEvent.click(screen.getByRole('button', { name: 'delete' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Remove' }));

    await waitFor(() =>
      expect(alertSpy).toHaveBeenCalledWith(
        'It is still in use by a database or by a running backup or restore. Try again once nothing uses it.',
      ),
    );
    expect(onNotifierDeleted).not.toHaveBeenCalled();
  });
});

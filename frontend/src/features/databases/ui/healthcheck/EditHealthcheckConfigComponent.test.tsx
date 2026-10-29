// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { ApiComponents } from '../../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { EditHealthcheckConfigComponent } from './EditHealthcheckConfigComponent';

const mockServer = setUpMockApi();
const databaseId = 'database-id';
const config: ApiComponents['schemas']['HealthcheckConfig'] = {
  databaseId,
  isHealthcheckEnabled: true,
  isNotificationEnabled: true,
  intervalMinutes: 1,
  failedAttemptsBeforeUnavailable: 3,
  attemptRetentionDays: 7,
};

describe('EditHealthcheckConfigComponent', () => {
  it('shows translated permission_denied after a refused config save', async () => {
    const alert = vi.spyOn(window, 'alert').mockImplementation(() => {});
    const onClose = vi.fn();
    mockServer.use(
      mockApi.get('/healthcheck-configs/{databaseId}', 200, config),
      mockApi.put('/healthcheck-configs/{databaseId}', 403, buildProblem('permission_denied', 403)),
    );
    try {
      renderScreen(<EditHealthcheckConfigComponent databaseId={databaseId} onClose={onClose} />);
      fireEvent.click((await screen.findAllByRole('switch'))[0]);
      fireEvent.click(await screen.findByRole('button', { name: 'Save' }));
      await waitFor(() =>
        expect(alert).toHaveBeenCalledWith('You do not have permission to do this.'),
      );
      expect(onClose).not.toHaveBeenCalled();
    } finally {
      alert.mockRestore();
    }
  });
});

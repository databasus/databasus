// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { VerificationConfig } from '../../../../entities/verification/config';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { ShowVerificationConfigComponent } from './ShowVerificationConfigComponent';

const mockServer = setUpMockApi();
const config: VerificationConfig = {
  databaseId: 'database-id',
  isAutomaticVerificationEnabled: true,
  scheduleType: 'INTERVAL',
  schedule: { type: 'WEEKLY', timeOfDay: '12:00', weekday: 0 },
  notificationTriggers: [],
};

describe('ShowVerificationConfigComponent', () => {
  beforeEach(() => {
    vi.stubEnv('TZ', 'UTC');
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('shows a saved weekday 0 as Sunday', async () => {
    mockServer.use(mockApi.get('/verification-configs/{databaseId}', 200, config));

    renderScreen(<ShowVerificationConfigComponent databaseId={config.databaseId} />);

    expect(await screen.findByText('Sun')).toBeTruthy();
  });
});

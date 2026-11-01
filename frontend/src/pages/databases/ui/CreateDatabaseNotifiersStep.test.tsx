// @vitest-environment happy-dom
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { CreateDatabaseNotifiersStep } from './CreateDatabaseNotifiersStep';

const mockServer = setUpMockApi();

describe('CreateDatabaseNotifiersStep', () => {
  it('offers Back and Complete while the notifier list loads', () => {
    const notifiers = mockApi.pending('get', '/notifiers');
    mockServer.use(notifiers.handler);
    const view = renderScreen(
      <CreateDatabaseNotifiersStep
        workspaceId="workspace-id"
        initialNotifierIds={[]}
        onBack={() => {}}
        onContinue={() => {}}
      />,
    );

    expect(document.querySelector('.ant-spin')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Back' })).toHaveProperty('disabled', false);
    expect(screen.getByRole('button', { name: 'Complete' })).toHaveProperty('disabled', false);

    view.unmount();
    notifiers.cancel();
  });
});

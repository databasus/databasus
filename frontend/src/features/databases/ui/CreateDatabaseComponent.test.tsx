// @vitest-environment happy-dom
import { fireEvent, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import type { DatabaseResourceComposition } from '../model/DatabaseResourceComposition';
import { CreateDatabaseComponent } from './CreateDatabaseComponent';

const mockServer = setUpMockApi();

// The scenario ends before the steps that render storages, notifiers or transfers.
const unusedResourceComposition: DatabaseResourceComposition = {
  renderStorageSelection: () => null,
  renderNotifiers: () => null,
  renderTransfer: () => null,
};

describe('CreateDatabaseComponent', () => {
  it('advances to the read-only user step with the submitted connection settings', async () => {
    const limitedUserChecks: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
    mockServer.use(
      mockApi.post('/database-tests', 200, { isConnected: true }),
      http.post(`*${API_BASE_PATH}/limited-user-checks`, async ({ request }) => {
        limitedUserChecks.push(
          (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
        );

        return HttpResponse.json({ isAdvisable: true, privileges: [] });
      }),
    );
    renderScreen(
      <CreateDatabaseComponent
        resourceComposition={unusedResourceComposition}
        workspaceId="workspace-id"
        onCreated={() => {}}
        onClose={() => {}}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('My favourite DB'), { target: { value: 'Shop' } });
    fireEvent.click(screen.getByText('MySQL'));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByPlaceholderText(/host$/), {
      target: { value: 'db.example.com' },
    });
    fireEvent.change(screen.getByPlaceholderText(/username$/), { target: { value: 'backup' } });
    fireEvent.change(screen.getByPlaceholderText(/password$/), { target: { value: 'secret' } });
    fireEvent.change(screen.getByPlaceholderText(/database name$/), { target: { value: 'shop' } });
    fireEvent.click(screen.getByRole('button', { name: 'Test connection' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Continue' }));

    expect(await screen.findByText('Create a read-only user for Databasus?')).toBeTruthy();
    expect(limitedUserChecks).toHaveLength(1);
    expect(limitedUserChecks[0].settings).toMatchObject({
      type: 'MYSQL',
      host: 'db.example.com',
      username: 'backup',
      password: 'secret',
      database: 'shop',
    });
  });
});

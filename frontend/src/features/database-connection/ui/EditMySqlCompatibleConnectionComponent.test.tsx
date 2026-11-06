// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DatabaseType, createDefaultDatabaseSettings } from '@/entities/databases';
import {
  buildCreateDraft,
  buildEditDraft,
  buildRestoreDraft,
  renderEditableForm,
  renderRestoreForm,
} from '@/features/database-connection/ui/connectionForm.fixture';
import { mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { stubClipboard } from '@/shared/lib/test/stubClipboard';

const mockServer = setUpMockApi();

const mysqlSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.MYSQL),
  host: 'db.example.com',
  username: 'backup',
  database: 'shop',
};

const mariadbSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.MARIADB),
  host: 'db.example.com',
  username: 'backup',
  database: 'shop',
};

const openAdvanced = () => fireEvent.click(screen.getByText('Advanced settings'));

describe('EditMySqlCompatibleConnectionComponent', () => {
  it('opens with the table exclusion label that has no data-only counterpart', () => {
    renderEditableForm(buildEditDraft({ ...mysqlSettings, excludeTables: ['audit_log'] }));

    expect(screen.getByText('Exclude tables')).toBeTruthy();
    expect(screen.getByText('audit_log')).toBeTruthy();
    expect(screen.queryByText('Exclude whole tables')).toBeNull();
  });

  it('offers the event and Galera options only for MariaDB', () => {
    const mysqlForm = renderEditableForm(buildCreateDraft(mysqlSettings));
    openAdvanced();

    expect(screen.queryByText('Exclude events')).toBeNull();
    expect(screen.queryByText('Galera replication')).toBeNull();

    mysqlForm.unmount();
    renderEditableForm(buildCreateDraft(mariadbSettings));
    openAdvanced();

    expect(screen.getByText('Exclude events')).toBeTruthy();
    expect(screen.getByText('Galera replication')).toBeTruthy();
  });

  it('fills a MariaDB form from a MariaDB string and asks for a new test', async () => {
    mockServer.use(mockApi.post('/database-tests', 200, { isConnected: true }));
    renderEditableForm(buildEditDraft(mariadbSettings));
    stubClipboard(async () => 'mariadb://user:secret@imported.example.com:3307/imported');
    fireEvent.click(screen.getByRole('button', { name: /Test connection$/ }));
    await screen.findByRole('button', { name: 'Submit draft' });

    fireEvent.click(screen.getByRole('button', { name: 'Parse from clipboard' }));

    await waitFor(() =>
      expect((screen.getByLabelText('Host') as HTMLInputElement).value).toBe(
        'imported.example.com',
      ),
    );
    expect((screen.getByLabelText('Port') as HTMLInputElement).value).toBe('3307');
    expect((screen.getByLabelText('Username') as HTMLInputElement).value).toBe('user');
    expect((screen.getByLabelText('Password') as HTMLInputElement).value).toBe('secret');
    expect((screen.getByLabelText('DB name') as HTMLInputElement).value).toBe('imported');
    expect(screen.queryByRole('button', { name: 'Submit draft' })).toBeNull();
  });

  it('refuses a MariaDB string in a MySQL form and changes nothing', async () => {
    renderEditableForm(buildCreateDraft(mysqlSettings));
    stubClipboard(async () => 'mariadb://user:secret@imported.example.com:3306/shop');

    fireEvent.click(screen.getByRole('button', { name: 'Parse from clipboard' }));

    expect(await screen.findByText(/format/i)).toBeTruthy();
    expect((screen.getByLabelText('Host') as HTMLInputElement).value).toBe('db.example.com');
  });

  it('hides the settings a restore ignores and keeps the Galera option', () => {
    renderRestoreForm(buildRestoreDraft(mariadbSettings));
    openAdvanced();

    expect(screen.queryByText('Exclude tables')).toBeNull();
    expect(screen.queryByText('Exclude events')).toBeNull();
    expect(screen.getByText('Galera replication')).toBeTruthy();
  });

  it('offers the start and no test once a restore target is filled', () => {
    renderRestoreForm(buildRestoreDraft({ ...mysqlSettings, password: 'secret' }));

    expect(screen.getByRole('button', { name: 'Submit draft' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Test connection$/ })).toBeNull();
  });
});

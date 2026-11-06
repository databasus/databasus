// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  DatabaseType,
  type PostgresqlLogicalSettings,
  PostgresSslMode,
  createDefaultDatabaseSettings,
} from '@/entities/databases';
import {
  NO_RESTORE_OPTIONS,
  buildCreateDraft,
  buildEditDraft,
  buildRestoreDraft,
  renderEditableForm,
  renderRestoreForm,
} from '@/features/database-connection/ui/connectionForm.fixture';
import { en } from '@/shared/i18n/locales/en';
import { setUpMockApi } from '@/shared/lib/test/mockApi';
import { stubClipboard } from '@/shared/lib/test/stubClipboard';

setUpMockApi();

const buildSettings = (
  change: Partial<PostgresqlLogicalSettings> = {},
): PostgresqlLogicalSettings => ({
  ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_LOGICAL),
  version: '16',
  host: 'db.example.com',
  username: 'backup',
  database: 'orders',
  sslMode: PostgresSslMode.Require,
  ...change,
});

const getHostInput = () => screen.getByLabelText('Host') as HTMLInputElement;

const getSelectedSslMode = () =>
  screen.getByText('SSL mode').parentElement!.querySelector('.ant-select-selection-item')!
    .textContent;

describe('EditPostgresqlLogicalConnectionComponent', () => {
  it('opens the advanced settings with both exclusion fields when only table data is excluded', () => {
    renderEditableForm(buildEditDraft(buildSettings({ excludeTableData: ['logs'] })));

    expect(screen.getByText('Exclude whole tables')).toBeTruthy();
    expect(screen.getByText('Exclude table data')).toBeTruthy();
    expect(screen.getByText('logs')).toBeTruthy();
  });

  it('offers no test until the client key is entered when a database with a client certificate moves', () => {
    renderEditableForm(
      buildEditDraft(buildSettings({ password: undefined, sslClientCert: 'saved-certificate' })),
    );
    const getTestButton = () =>
      screen.getByRole('button', { name: /Test connection$/ }) as HTMLButtonElement;

    fireEvent.change(getHostInput(), { target: { value: 'moved.example.com' } });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'typed-password' } });

    expect(screen.getAllByText(en.common.messages.addressChangedEnterSecretAgain)).toHaveLength(1);
    expect(getTestButton().disabled).toBe(true);

    fireEvent.change(screen.getByLabelText('Client key'), { target: { value: 'typed-key' } });

    expect(getTestButton().disabled).toBe(false);
  });

  it('asks for no client key when the certificates are replaced with nothing at a moved host', () => {
    renderEditableForm(
      buildEditDraft(buildSettings({ password: undefined, sslClientCert: 'saved-certificate' })),
    );

    fireEvent.change(getHostInput(), { target: { value: 'moved.example.com' } });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'typed-password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Replace' }));

    expect(
      (screen.getByRole('button', { name: /Test connection$/ }) as HTMLButtonElement).disabled,
    ).toBe(false);
  });

  it('keeps the advanced settings collapsed when no advanced value is set', () => {
    renderEditableForm(buildEditDraft(buildSettings()));

    expect(screen.getByText('Advanced settings')).toBeTruthy();
    expect(screen.queryByText('Exclude whole tables')).toBeNull();
    expect(screen.queryByText('Exclude table data')).toBeNull();
  });

  it('opens the advanced settings when only a client key is held', () => {
    renderEditableForm(buildCreateDraft(buildSettings({ sslClientKey: 'typed-client-key' })));

    expect((screen.getByLabelText('Client key') as HTMLTextAreaElement).value).toBe(
      'typed-client-key',
    );
  });

  it('turns SSL off for a new database on localhost without a chosen mode', () => {
    renderEditableForm(buildCreateDraft(buildSettings()));

    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Disable/);
  });

  it('clears the typed certificates when a derived mode turns SSL off', () => {
    renderEditableForm(buildCreateDraft(buildSettings({ sslClientCert: 'typed-client-cert' })));

    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });
    fireEvent.change(getHostInput(), { target: { value: 'https://db.example.com' } });

    expect(getSelectedSslMode()).toMatch(/^Require/);
    expect(screen.queryByDisplayValue('typed-client-cert')).toBeNull();
  });

  it('keeps an imported mode of a new database when the host then changes', async () => {
    renderEditableForm(buildCreateDraft(buildSettings()));
    stubClipboard(
      async () => 'postgresql://importer:secret@db.example.com:5432/shop?sslmode=require',
    );

    fireEvent.click(screen.getByText('Parse from clipboard'));
    await waitFor(() => expect(screen.getByDisplayValue('importer')).toBeTruthy());
    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Require/);
  });

  it('keeps the stored mode of a saved database when the host changes', () => {
    renderEditableForm(buildEditDraft(buildSettings()));

    fireEvent.change(getHostInput(), { target: { value: 'localhost' } });

    expect(getSelectedSslMode()).toMatch(/^Require/);
  });

  it('turns SSL on for a saved database whose new host starts with https', () => {
    renderEditableForm(buildEditDraft(buildSettings({ sslMode: PostgresSslMode.Disable })));

    fireEvent.change(getHostInput(), { target: { value: 'https://db.example.com' } });

    expect(getSelectedSslMode()).toMatch(/^Require/);
    expect(getHostInput().value).toBe('db.example.com');
  });

  it('shows a saved certificate as stored and offers Replace', () => {
    renderEditableForm(buildEditDraft(buildSettings({ sslClientCert: 'stored-client-cert' })));

    expect(screen.getByRole('button', { name: 'Replace' })).toBeTruthy();
    expect(screen.queryByDisplayValue('stored-client-cert')).toBeNull();
  });

  it('shows empty editable fields when Replace is left empty', async () => {
    renderEditableForm(buildEditDraft(buildSettings({ sslClientCert: 'stored-client-cert' })));

    fireEvent.click(screen.getByRole('button', { name: 'Replace' }));

    await waitFor(() => expect(screen.queryByRole('button', { name: 'Replace' })).toBeNull());
    expect((screen.getByLabelText('Client certificate') as HTMLTextAreaElement).value).toBe('');
    expect((screen.getByLabelText('Client key') as HTMLTextAreaElement).value).toBe('');
  });

  it('opens the advanced settings for a restore option that is on', () => {
    renderRestoreForm(buildRestoreDraft(buildSettings()), {
      ...NO_RESTORE_OPTIONS,
      isRestoreOwnership: true,
    });

    const ownershipCheckbox = screen.getByLabelText('Restore ownership') as HTMLInputElement;

    expect(ownershipCheckbox.checked).toBe(true);
    expect(screen.queryByText('Exclude whole tables')).toBeNull();
  });

  it('keeps the advanced settings collapsed for a restore with default options', () => {
    renderRestoreForm(buildRestoreDraft(buildSettings()));

    expect(screen.queryByLabelText('Restore ownership')).toBeNull();
  });
});

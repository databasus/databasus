// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import {
  DatabaseType,
  type DatabaseSettings,
  createDefaultDatabaseSettings,
} from '@/entities/databases';
import { API_BASE_PATH, type ApiComponents } from '@/shared/api';
import { useLocale } from '@/shared/i18n';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboard, stubClipboardUnavailable } from '@/shared/lib/test/stubClipboard';

import { buildCreateDraft, buildEditDraft, renderEditableForm } from './connectionForm.fixture';
import { DatabaseConnectionForm } from './DatabaseConnectionForm';

const mockServer = setUpMockApi();

const savedMysqlSettings: DatabaseSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.MYSQL),
  password: undefined,
  host: 'db.example.com',
  username: 'backup',
  database: 'shop',
};

const savedPhysicalSettings: DatabaseSettings = {
  ...createDefaultDatabaseSettings(DatabaseType.POSTGRES_PHYSICAL),
  password: undefined,
  host: 'db.example.com',
  username: 'replicator',
};

const clickTest = () => fireEvent.click(screen.getByRole('button', { name: /Test connection$/ }));

const getHostInput = () => screen.getByLabelText('Host') as HTMLInputElement;

describe('DatabaseConnectionForm', () => {
  it('sends no password in the test when a saved database leaves the password empty', async () => {
    const sentTests: ApiComponents['schemas']['DatabaseSettingsTarget'][] = [];
    mockServer.use(
      http.post(`*${API_BASE_PATH}/database-tests`, async ({ request }) => {
        sentTests.push(
          (await request.json()) as ApiComponents['schemas']['DatabaseSettingsTarget'],
        );

        return HttpResponse.json({ isConnected: true });
      }),
    );
    renderEditableForm(buildEditDraft(savedMysqlSettings));
    const passwordInput = screen.getByLabelText('Password');

    fireEvent.change(passwordInput, { target: { value: 'typed by mistake' } });
    fireEvent.change(passwordInput, { target: { value: '' } });
    clickTest();

    await screen.findByRole('button', { name: 'Submit draft' });
    expect(sentTests[0].databaseId).toBe('database-id');
    expect(sentTests[0].settings).not.toHaveProperty('password');
  });

  it('shows the reason and the server error when a physical test fails', async () => {
    mockServer.use(
      mockApi.post('/database-tests', 200, {
        isConnected: false,
        reason: 'PG_HBA_NO_ENTRY',
        error: 'no pg_hba.conf entry for replication connection from host "10.0.0.5"',
      }),
    );
    renderEditableForm(buildEditDraft(savedPhysicalSettings));

    clickTest();

    expect(await screen.findByText('Replication is not allowed from this host')).toBeTruthy();
    expect(
      screen.getByText('no pg_hba.conf entry for replication connection from host "10.0.0.5"'),
    ).toBeTruthy();
    expect(screen.queryByText(/IP whitelist/)).toBeNull();
    expect(screen.queryByRole('button', { name: 'Submit draft' })).toBeNull();
  });

  it('reports a classified failure and its removal to the wizard', async () => {
    mockServer.use(
      mockApi.post('/database-tests', 200, {
        isConnected: false,
        reason: 'PG_HBA_NO_ENTRY',
        error: 'no pg_hba.conf entry',
      }),
    );
    const onConnectionErrorChange = vi.fn();
    const Form = () => (
      <DatabaseConnectionForm
        draft={buildEditDraft(savedPhysicalSettings)}
        onDraftChange={() => {}}
        onSubmit={() => {}}
        submitLabel="Submit draft"
        onConnectionErrorChange={onConnectionErrorChange}
      />
    );
    const view = renderScreen(<Form />);

    clickTest();

    expect(await screen.findByText('Replication is not allowed from this host')).toBeTruthy();
    await waitFor(() => expect(onConnectionErrorChange).toHaveBeenLastCalledWith(true));

    view.unmount();

    expect(onConnectionErrorChange).toHaveBeenLastCalledWith(false);
  });

  it('removes a classified failure when the connection changes', async () => {
    mockServer.use(
      mockApi.post('/database-tests', 200, {
        isConnected: false,
        reason: 'PG_HBA_NO_ENTRY',
        error: 'no pg_hba.conf entry',
      }),
    );
    const onConnectionErrorChange = vi.fn();
    renderScreen(
      <DatabaseConnectionForm
        draft={buildEditDraft(savedPhysicalSettings)}
        onDraftChange={() => {}}
        onSubmit={() => {}}
        submitLabel="Submit draft"
        onConnectionErrorChange={onConnectionErrorChange}
      />,
    );
    clickTest();
    await screen.findByText('Replication is not allowed from this host');
    expect(screen.getByText(/^Add the line below to pg_hba\.conf/)).toBeTruthy();
    expect(screen.getByText(/^host\s+replication\s+all/)).toBeTruthy();

    fireEvent.change(getHostInput(), { target: { value: 'other.example.com' } });

    await waitFor(() =>
      expect(screen.queryByText('Replication is not allowed from this host')).toBeNull(),
    );
    expect(screen.queryByText(/^Add the line below to pg_hba\.conf/)).toBeNull();
    expect(screen.queryByText(/^host\s+replication\s+all/)).toBeNull();
    expect(onConnectionErrorChange).toHaveBeenLastCalledWith(false);
  });

  it('shows a refused test inline with its translated reason and no dialog', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    mockServer.use(mockApi.post('/database-tests', 403, buildProblem('permission_denied', 403)));
    renderEditableForm(buildEditDraft(savedMysqlSettings));

    clickTest();

    expect(await screen.findByText('You do not have permission to do this.')).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
  });

  it('identifies the password field by its visible label', () => {
    renderEditableForm(buildEditDraft(savedMysqlSettings));

    expect(screen.getByLabelText('Password').getAttribute('type')).toBe('password');
  });

  it('starts the import by keyboard focus and activation', async () => {
    const { user } = renderEditableForm(buildCreateDraft(savedMysqlSettings));
    stubClipboard(async () => 'mysql://user:secret@imported.example.com:3306/shop');
    const importControl = screen.getByRole('button', { name: 'Parse from clipboard' });

    importControl.focus();
    await user.keyboard('{Enter}');

    await waitFor(() => expect(getHostInput().value).toBe('imported.example.com'));
    expect(document.activeElement).toBe(importControl);
  });

  it('offers a manual paste when the clipboard API is unavailable and imports the pasted string', async () => {
    renderEditableForm(buildCreateDraft(savedMysqlSettings));
    stubClipboardUnavailable();

    fireEvent.click(screen.getByRole('button', { name: 'Parse from clipboard' }));
    fireEvent.change(await screen.findByPlaceholderText('Paste your connection string here...'), {
      target: { value: 'mysql://user:secret@pasted.example.com:3306/shop' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

    await waitFor(() => expect(getHostInput().value).toBe('pasted.example.com'));
  });

  it('shows an error, opens no manual paste and changes nothing when reading the clipboard fails', async () => {
    renderEditableForm(buildCreateDraft(savedMysqlSettings));
    stubClipboard(async () => {
      throw new Error('denied');
    });

    fireEvent.click(screen.getByRole('button', { name: 'Parse from clipboard' }));

    expect(await screen.findByText(/Failed to read clipboard/)).toBeTruthy();
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(getHostInput().value).toBe('db.example.com');
  });

  it('keeps the literal reason of a failure and its values across a language change', async () => {
    mockServer.use(
      mockApi.post('/database-tests', 200, { isConnected: false, error: 'dial tcp: refused' }),
    );
    const LanguageSwitch = () => {
      const { setLocale } = useLocale();

      return <button onClick={() => setLocale('ru')}>Switch language</button>;
    };
    renderScreen(
      <>
        <LanguageSwitch />
        <DatabaseConnectionForm
          draft={buildEditDraft({ ...savedMysqlSettings, password: 'typed' })}
          onDraftChange={() => {}}
          onSubmit={() => {}}
          submitLabel="Submit draft"
        />
      </>,
    );
    clickTest();
    await screen.findByText('The database could not be reached: dial tcp: refused');

    fireEvent.click(screen.getByRole('button', { name: 'Switch language' }));

    await waitFor(() =>
      expect(screen.queryByText('The database could not be reached: dial tcp: refused')).toBeNull(),
    );
    expect(screen.getByText(/dial tcp: refused/)).toBeTruthy();
    expect(screen.getByDisplayValue('db.example.com')).toBeTruthy();
  });

  it('shows a refused submission inline, keeps the input and accepts input again', async () => {
    const showAlert = vi.fn();
    vi.stubGlobal('alert', showAlert);
    mockServer.use(mockApi.post('/database-tests', 200, { isConnected: true }));
    renderEditableForm(buildEditDraft(savedMysqlSettings), () => {
      throw new Error('refused');
    });
    fireEvent.change(getHostInput(), { target: { value: 'replica.example.com' } });
    clickTest();

    fireEvent.click(await screen.findByRole('button', { name: 'Submit draft' }));

    expect(await screen.findByText('Something went wrong. Please try again.')).toBeTruthy();
    expect(showAlert).not.toHaveBeenCalled();
    expect(getHostInput().value).toBe('replica.example.com');
    expect(getHostInput().disabled).toBe(false);
  });

  it('accepts no input and no second submission while a submission is pending', async () => {
    mockServer.use(mockApi.post('/database-tests', 200, { isConnected: true }));
    let finishSubmission = () => {};
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finishSubmission = resolve;
        }),
    );
    renderEditableForm(buildEditDraft(savedMysqlSettings), onSubmit);
    clickTest();
    const submitButton = await screen.findByRole('button', { name: 'Submit draft' });

    fireEvent.click(submitButton);
    fireEvent.click(submitButton);

    await waitFor(() => expect(getHostInput().disabled).toBe(true));
    expect(onSubmit).toHaveBeenCalledTimes(1);

    await act(async () => finishSubmission());

    expect(getHostInput().disabled).toBe(false);
  });

  it('submits a snapshot that later changes do not reach', async () => {
    mockServer.use(mockApi.post('/database-tests', 200, { isConnected: true }));
    const onSubmit = vi.fn();
    const draft = buildEditDraft({ ...savedMysqlSettings, password: 'typed' });
    renderEditableForm(draft, onSubmit);
    clickTest();

    fireEvent.click(await screen.findByRole('button', { name: 'Submit draft' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).toEqual(draft);
    expect(onSubmit.mock.calls[0][0]).not.toBe(draft);
  });
});

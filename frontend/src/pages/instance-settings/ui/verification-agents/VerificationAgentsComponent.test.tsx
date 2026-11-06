// @vitest-environment happy-dom
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { type VerificationAgent, verificationAgentApi } from '@/entities/verification-agents';
import { TOKEN_PLACEHOLDER } from '@/pages/instance-settings/lib/verificationAgentCommands';
import { VerificationAgentsComponent } from '@/pages/instance-settings/ui/verification-agents/VerificationAgentsComponent';
import { API_BASE_PATH } from '@/shared/api';
import { activateSession, getSessionSnapshot } from '@/shared/auth';
import { DICTIONARIES } from '@/shared/i18n';
import { advanceFakeClock } from '@/shared/lib/test/advanceFakeClock';
import { inspectApiRequest } from '@/shared/lib/test/inspectApiRequest';
import { buildProblem, mockApi, setUpMockApi } from '@/shared/lib/test/mockApi';
import { createPendingApiResponse } from '@/shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '@/shared/lib/test/releaseAndSettle';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { stubClipboardWrite } from '@/shared/lib/test/stubClipboard';

const mockServer = setUpMockApi();

const LIST_REFRESH_MS = 15_000;
const AGENTS_PATH = `*${API_BASE_PATH}/verification-agents`;
const INTERNAL_ERROR = DICTIONARIES.en.errors.internal;
const REFUSAL = DICTIONARIES.en.errors.permissionDenied;
const agentsCopy = DICTIONARIES.en.verification.agents;
const commonActions = DICTIONARIES.en.common.actions;
const ISSUED_TOKEN = 'issued-once-token';

const buildAgent = (change: Partial<VerificationAgent> = {}): VerificationAgent => ({
  id: 'agent-id',
  name: 'Original agent',
  maxCpu: 0,
  maxRamMb: 0,
  maxDiskGb: 0,
  maxConcurrentJobs: 0,
  createdAt: '2026-10-27T10:00:00Z',
  ...change,
});

const firstAgent = buildAgent({ id: 'first-agent', name: 'First agent' });
const secondAgent = buildAgent({ id: 'second-agent', name: 'Second agent' });

const listResponse = (...agents: VerificationAgent[]) => HttpResponse.json({ items: agents });
const failedResponse = () => HttpResponse.json(buildProblem('internal', 500), { status: 500 });
const refusal = () => buildProblem('permission_denied', 403);

const declareListAnswersRepeatingLast = (...answers: (() => Response | Promise<Response>)[]) => {
  let answerIndex = 0;
  mockServer.use(
    http.get(AGENTS_PATH, () => {
      const answer = answers[Math.min(answerIndex, answers.length - 1)];
      answerIndex += 1;

      return answer();
    }),
  );
};

// The clock also runs by itself, so the overlays that wait on timers still open, and a test moves
// it by a whole refresh interval to tick.
const useRunningFakeClock = () => vi.useFakeTimers({ shouldAdvanceTime: true });
const tickListRefresh = () => advanceFakeClock(LIST_REFRESH_MS);

const getCreateButton = () =>
  screen.getAllByRole<HTMLButtonElement>('button', { name: /Create/ })[0];

const pressRowDelete = async (agentName = 'Original agent') => {
  const row = screen.getByText(agentName).closest('tr') as HTMLElement;
  fireEvent.click(within(row).getByRole('button', { name: commonActions.delete }));

  const confirmation = await waitFor(() => {
    const popconfirm = document.querySelector<HTMLElement>('.ant-popconfirm');
    if (!popconfirm) throw new Error('The delete confirmation is not open');

    return popconfirm;
  });
  fireEvent.click(within(confirmation).getByRole('button', { name: commonActions.delete }));
};

const queryMessages = () => document.querySelectorAll('.ant-message-notice');

const getAgentRow = (agentName: string) => screen.getByText(agentName).closest('tr') as HTMLElement;

const openInstallCommands = async (agentName: string) => {
  fireEvent.click(
    within(getAgentRow(agentName)).getByRole('button', {
      name: agentsCopy.actions.viewInstallCommands,
    }),
  );

  return screen.findByRole('dialog');
};

const getDownloadBlock = (dialog: HTMLElement) => within(dialog).getByText(/^curl /);
const getLaunchBlock = (dialog: HTMLElement) =>
  within(dialog).getByText(/^\.\/verification-agent start/);

const buildDownloadCommand = (arch: string) =>
  `curl -L -o verification-agent "${window.location.origin}/api/v1/system/verification-agent?arch=${arch}" && chmod +x verification-agent`;

const buildPlaceholderLaunchCommand = (agentId: string) =>
  [
    './verification-agent start \\',
    `  --databasus-host=${window.location.origin} \\`,
    `  --agent-id=${agentId} \\`,
    `  --token=${TOKEN_PLACEHOLDER} \\`,
    '  --max-cpu=2 \\',
    '  --max-ram-mb=2048 \\',
    '  --max-disk-gb=20 \\',
    '  --max-concurrent-jobs=1',
  ].join('\n');

describe('VerificationAgentsComponent', () => {
  describe('its first load', () => {
    it('offers a retry with no empty list and Create disabled, and only the retry reloads', async () => {
      useRunningFakeClock();
      let isListFailing = true;
      mockServer.use(
        http.get(AGENTS_PATH, () =>
          isListFailing ? failedResponse() : listResponse(buildAgent()),
        ),
      );
      const listCall = vi.spyOn(verificationAgentApi, 'listAgents');
      renderScreen(<VerificationAgentsComponent />);

      const retryButton = await screen.findByRole('button', { name: commonActions.retry });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(screen.queryByText(agentsCopy.empty)).toBeNull();
      expect(getCreateButton().disabled).toBe(true);

      await tickListRefresh();
      await tickListRefresh();

      expect(listCall).toHaveBeenCalledOnce();

      fireEvent.click(retryButton);
      await waitFor(() => expect(listCall).toHaveBeenCalledTimes(2));

      const retryAfterFailure = await screen.findByRole('button', { name: commonActions.retry });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(getCreateButton().disabled).toBe(true);

      isListFailing = false;
      fireEvent.click(retryAfterFailure);

      expect(await screen.findByText('Original agent')).toBeTruthy();
      expect(screen.queryByText(INTERNAL_ERROR)).toBeNull();
      expect(getCreateButton().disabled).toBe(false);
      expect(queryMessages()).toHaveLength(0);
    });

    it('keeps Create disabled while the first load and a retry run', async () => {
      const heldLoad = createPendingApiResponse();
      const heldRetry = createPendingApiResponse();
      declareListAnswersRepeatingLast(
        () => heldLoad.response,
        () => heldRetry.response,
      );
      renderScreen(<VerificationAgentsComponent />);

      expect(getCreateButton().disabled).toBe(true);

      heldLoad.respond(failedResponse());
      fireEvent.click(await screen.findByRole('button', { name: commonActions.retry }));

      await waitFor(() => expect(screen.queryByText(INTERNAL_ERROR)).toBeNull());
      expect(getCreateButton().disabled).toBe(true);

      heldRetry.respond(listResponse(buildAgent()));

      expect(await screen.findByText('Original agent')).toBeTruthy();
      expect(getCreateButton().disabled).toBe(false);
    });
  });

  describe('its background refresh', () => {
    it('shows no message and keeps the agents while refreshes fail, and a later success shows the current ones', async () => {
      useRunningFakeClock();
      let isListFailing = false;
      let listedAgent = buildAgent({ lastSeenAt: new Date().toISOString() });
      mockServer.use(
        http.get(AGENTS_PATH, () => (isListFailing ? failedResponse() : listResponse(listedAgent))),
      );
      const listCall = vi.spyOn(verificationAgentApi, 'listAgents');
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      isListFailing = true;
      for (let refresh = 0; refresh < 7; refresh++) await tickListRefresh();

      expect(listCall.mock.calls.length).toBeGreaterThan(2);
      expect(queryMessages()).toHaveLength(0);
      expect(screen.getByText('Original agent')).toBeTruthy();
      expect(screen.getByText(DICTIONARIES.en.status.agent.stale)).toBeTruthy();

      isListFailing = false;
      listedAgent = buildAgent({ name: 'Renamed agent' });
      await tickListRefresh();

      expect(await screen.findByText('Renamed agent')).toBeTruthy();
      expect(screen.queryByText('Original agent')).toBeNull();
    });

    it('refreshes the list behind an open dialog and leaves its input as it is', async () => {
      useRunningFakeClock();
      let listedAgent = buildAgent();
      mockServer.use(http.get(AGENTS_PATH, () => listResponse(listedAgent)));
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      fireEvent.click(getCreateButton());
      const dialog = await screen.findByRole('dialog');
      const nameInput = within(dialog).getByPlaceholderText<HTMLInputElement>(
        agentsCopy.create.namePlaceholder,
      );
      fireEvent.change(nameInput, { target: { value: 'Typed name' } });

      listedAgent = buildAgent({ name: 'Renamed agent' });
      await tickListRefresh();

      expect(await screen.findByText('Renamed agent')).toBeTruthy();
      expect(screen.getByRole('dialog')).toBe(dialog);
      expect(nameInput.value).toBe('Typed name');
    });

    it('sends no request after an unmount and ignores an answer that arrives after it', async () => {
      useRunningFakeClock();
      const heldRefresh = createPendingApiResponse();
      declareListAnswersRepeatingLast(
        () => listResponse(buildAgent()),
        () => heldRefresh.response,
      );
      const listCall = vi.spyOn(verificationAgentApi, 'listAgents');
      const view = renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      await tickListRefresh();
      await waitFor(() => expect(listCall).toHaveBeenCalledTimes(2));

      view.unmount();
      await tickListRefresh();
      await releaseAndSettle(listCall, () =>
        heldRefresh.respond(listResponse(buildAgent({ name: 'Obsolete agent' }))),
      );

      expect(listCall).toHaveBeenCalledTimes(2);
      expect(screen.queryByText('Obsolete agent')).toBeNull();
    });
  });

  describe('its deletion', () => {
    it('removes the row at once and keeps Create disabled from the request until the reload succeeds', async () => {
      const heldDeletion = mockApi.pending('delete', '/verification-agents/{agentId}');
      const heldReload = createPendingApiResponse();
      declareListAnswersRepeatingLast(
        () => listResponse(buildAgent()),
        () => heldReload.response,
      );
      mockServer.use(heldDeletion.handler);
      const deleteCall = vi.spyOn(verificationAgentApi, 'deleteAgent');
      const listCall = vi.spyOn(verificationAgentApi, 'listAgents');
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      await pressRowDelete();
      await waitFor(() => expect(deleteCall).toHaveBeenCalledOnce());

      expect(getCreateButton().disabled).toBe(true);

      heldDeletion.respond(new HttpResponse(null, { status: 204 }));
      await waitFor(() => expect(listCall).toHaveBeenCalledTimes(2));

      await waitFor(() => expect(screen.queryByText('Original agent')).toBeNull());
      expect(getCreateButton().disabled).toBe(true);

      heldReload.respond(listResponse());

      expect(await screen.findByText(agentsCopy.empty)).toBeTruthy();
      expect(getCreateButton().disabled).toBe(false);
    });

    it('keeps the deleted agent gone and the older names out when a held refresh answers after the reload', async () => {
      useRunningFakeClock();
      const deletedAgent = buildAgent({ id: 'deleted-agent', name: 'Deleted agent' });
      const remainingAgent = buildAgent({ id: 'remaining-agent', name: 'Remaining agent' });
      const heldRefresh = createPendingApiResponse();
      declareListAnswersRepeatingLast(
        () => listResponse(deletedAgent, remainingAgent),
        () => heldRefresh.response,
        () => listResponse(remainingAgent),
      );
      mockServer.use(mockApi.delete('/verification-agents/{agentId}', 204, undefined));
      const listCall = vi.spyOn(verificationAgentApi, 'listAgents');
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Deleted agent');

      await tickListRefresh();
      await waitFor(() => expect(listCall).toHaveBeenCalledTimes(2));

      await pressRowDelete('Deleted agent');

      expect(await screen.findByText('Remaining agent')).toBeTruthy();
      await waitFor(() => expect(screen.queryByText('Deleted agent')).toBeNull());

      await releaseAndSettle(
        listCall,
        () =>
          heldRefresh.respond(
            listResponse(deletedAgent, { ...remainingAgent, name: 'Obsolete agent' }),
          ),
        1,
      );

      expect(screen.queryByText('Deleted agent')).toBeNull();
      expect(screen.queryByText('Obsolete agent')).toBeNull();
      expect(screen.getByText('Remaining agent')).toBeTruthy();
    });

    it('shows the reason with a retry when the reload fails', async () => {
      declareListAnswersRepeatingLast(
        () => listResponse(buildAgent()),
        () => failedResponse(),
        () => listResponse(),
      );
      mockServer.use(mockApi.delete('/verification-agents/{agentId}', 204, undefined));
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      await pressRowDelete();

      const retryButton = await screen.findByRole('button', { name: commonActions.retry });

      expect(screen.getByText(INTERNAL_ERROR)).toBeTruthy();
      expect(screen.queryByText('Original agent')).toBeNull();
      expect(getCreateButton().disabled).toBe(true);

      fireEvent.click(retryButton);

      expect(await screen.findByText(agentsCopy.empty)).toBeTruthy();
      expect(getCreateButton().disabled).toBe(false);
    });

    it('keeps the agent and reports once when the deletion is refused', async () => {
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [buildAgent()] }),
        mockApi.delete('/verification-agents/{agentId}', 403, refusal()),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      await pressRowDelete();

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getByText('Original agent')).toBeTruthy();
      await waitFor(() => expect(getCreateButton().disabled).toBe(false));
    });
  });

  describe('its refused mutations', () => {
    it('keeps the entered name and reports once when the creation is refused', async () => {
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [] }),
        mockApi.post('/verification-agents', 403, refusal()),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText(agentsCopy.empty);

      fireEvent.click(getCreateButton());
      const dialog = await screen.findByRole('dialog');
      const nameInput = within(dialog).getByPlaceholderText<HTMLInputElement>(
        agentsCopy.create.namePlaceholder,
      );
      fireEvent.change(nameInput, { target: { value: 'Typed name' } });
      fireEvent.click(within(dialog).getByRole('button', { name: commonActions.create }));

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getByRole('dialog')).toBe(dialog);
      expect(nameInput.value).toBe('Typed name');
    });

    it('keeps the dialog, reports once and reveals no token when the rotation is refused', async () => {
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [buildAgent()] }),
        mockApi.post('/verification-agents/{agentId}/rotate-token', 403, refusal()),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      fireEvent.click(screen.getByRole('button', { name: agentsCopy.actions.rotateToken }));
      const dialog = await screen.findByRole('dialog');
      fireEvent.click(within(dialog).getByRole('button', { name: agentsCopy.actions.rotateToken }));

      expect(await screen.findAllByText(REFUSAL)).toHaveLength(1);
      expect(screen.getByRole('dialog')).toBe(dialog);
      expect(screen.queryByRole('button', { name: agentsCopy.token.confirmSaved })).toBeNull();
    });
  });

  describe('its revealed token', () => {
    const REVEALS = [
      {
        origin: 'a creation',
        declareReveal: () =>
          mockServer.use(
            mockApi.post('/verification-agents', 201, {
              agent: buildAgent({ id: 'created-agent', name: 'Typed name' }),
              token: ISSUED_TOKEN,
            }),
          ),
        reveal: async () => {
          fireEvent.click(getCreateButton());
          const dialog = await screen.findByRole('dialog');
          fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), {
            target: { value: 'Typed name' },
          });
          fireEvent.click(within(dialog).getByRole('button', { name: commonActions.create }));
        },
      },
      {
        origin: 'a rotation',
        declareReveal: () =>
          mockServer.use(
            mockApi.post('/verification-agents/{agentId}/rotate-token', 200, {
              token: ISSUED_TOKEN,
            }),
          ),
        reveal: async () => {
          fireEvent.click(
            screen.getAllByRole('button', { name: agentsCopy.actions.rotateToken })[0],
          );
          const dialog = await screen.findByRole('dialog');
          fireEvent.click(
            within(dialog).getByRole('button', { name: agentsCopy.actions.rotateToken }),
          );
        },
      },
    ];

    const readBrowserStorage = () =>
      JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage });

    it.each(REVEALS)(
      'removes the token from $origin from the page and browser storage on close, and later install commands show the placeholder',
      async ({ declareReveal, reveal }) => {
        mockServer.use(mockApi.get('/verification-agents', 200, { items: [buildAgent()] }));
        declareReveal();
        renderScreen(<VerificationAgentsComponent />);
        await screen.findByText('Original agent');

        await reveal();

        expect(await screen.findByText(ISSUED_TOKEN)).toBeTruthy();
        expect(document.body.textContent).toContain(`--token=${ISSUED_TOKEN}`);
        expect(readBrowserStorage()).not.toContain(ISSUED_TOKEN);

        fireEvent.click(screen.getByRole('button', { name: agentsCopy.token.confirmSaved }));
        await waitFor(() => expect(document.body.innerHTML).not.toContain(ISSUED_TOKEN));

        expect(readBrowserStorage()).not.toContain(ISSUED_TOKEN);

        fireEvent.click(
          screen.getAllByRole('button', { name: agentsCopy.actions.viewInstallCommands })[0],
        );

        await waitFor(() =>
          expect(document.body.textContent).toContain(`--token=${TOKEN_PLACEHOLDER}`),
        );
        expect(document.body.innerHTML).not.toContain(ISSUED_TOKEN);
      },
    );
  });

  describe('its icon actions', () => {
    it('names every action of an agent row', async () => {
      mockServer.use(mockApi.get('/verification-agents', 200, { items: [buildAgent()] }));
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');

      for (const name of [
        agentsCopy.actions.viewInstallCommands,
        agentsCopy.actions.rotateToken,
        commonActions.delete,
      ]) {
        expect(screen.getByRole('button', { name })).toBeTruthy();
      }
    });

    it('names the copy actions of the install commands and copies the chosen block', async () => {
      mockServer.use(mockApi.get('/verification-agents', 200, { items: [buildAgent()] }));
      const view = renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');
      const writeText = vi.fn(() => Promise.resolve());
      stubClipboardWrite(writeText);

      await view.user.click(
        screen.getByRole('button', { name: agentsCopy.actions.viewInstallCommands }),
      );
      const dialog = await screen.findByRole('dialog');

      expect(within(dialog).getAllByRole('button', { name: commonActions.copy })).toHaveLength(3);

      await view.user.click(within(dialog).getAllByRole('button', { name: commonActions.copy })[0]);

      await waitFor(() => expect(writeText).toHaveBeenCalledWith('agent-id'));
    });

    it('names the copy actions of the revealed token and copies the token', async () => {
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [buildAgent()] }),
        mockApi.post('/verification-agents/{agentId}/rotate-token', 200, { token: ISSUED_TOKEN }),
      );
      const view = renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Original agent');
      const writeText = vi.fn(() => Promise.resolve());
      stubClipboardWrite(writeText);

      await view.user.click(screen.getByRole('button', { name: agentsCopy.actions.rotateToken }));
      await view.user.click(
        within(await screen.findByRole('dialog')).getByRole('button', {
          name: agentsCopy.actions.rotateToken,
        }),
      );
      const tokenBlock = await screen.findByText(ISSUED_TOKEN);
      const dialog = tokenBlock.closest('[role="dialog"]') as HTMLElement;

      expect(within(dialog).getAllByRole('button', { name: commonActions.copy })).toHaveLength(4);

      await view.user.click(within(dialog).getAllByRole('button', { name: commonActions.copy })[0]);

      await waitFor(() => expect(writeText).toHaveBeenCalledWith(ISSUED_TOKEN));
    });
  });

  describe('its install commands', () => {
    const renderTwoAgents = async () => {
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [firstAgent, secondAgent] }),
      );
      const view = renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Second agent');

      return view;
    };

    it('shows the host, the chosen agent, the token placeholder and the limits in the launch command after its dialog opens', async () => {
      await renderTwoAgents();

      const dialog = await openInstallCommands('Second agent');

      expect(getLaunchBlock(dialog).textContent).toBe(
        buildPlaceholderLaunchCommand('second-agent'),
      );
    });

    it('shows the download URL of the chosen architecture and updates it after the switch', async () => {
      await renderTwoAgents();

      const dialog = await openInstallCommands('First agent');
      const getArchitectureButton = (arch: string) =>
        within(dialog).getByRole('button', { name: arch });

      expect(getDownloadBlock(dialog).textContent).toBe(buildDownloadCommand('amd64'));
      expect(getArchitectureButton('amd64').getAttribute('aria-pressed')).toBe('true');

      fireEvent.click(getArchitectureButton('arm64'));

      expect(getDownloadBlock(dialog).textContent).toBe(buildDownloadCommand('arm64'));
      expect(getArchitectureButton('arm64').getAttribute('aria-pressed')).toBe('true');
      expect(getArchitectureButton('amd64').getAttribute('aria-pressed')).toBe('false');

      fireEvent.click(getArchitectureButton('amd64'));

      expect(getDownloadBlock(dialog).textContent).toBe(buildDownloadCommand('amd64'));
    });

    it('copies the shown commands after their copy buttons are pressed', async () => {
      const view = await renderTwoAgents();
      const writeText = vi.fn(() => Promise.resolve());
      stubClipboardWrite(writeText);

      const dialog = await openInstallCommands('Second agent');
      fireEvent.click(within(dialog).getByRole('button', { name: 'arm64' }));

      const pressBlockCopy = (block: HTMLElement) =>
        view.user.click(
          within(block.parentElement as HTMLElement).getByRole('button', {
            name: commonActions.copy,
          }),
        );

      await pressBlockCopy(getDownloadBlock(dialog));
      await waitFor(() => expect(writeText).toHaveBeenCalledWith(buildDownloadCommand('arm64')));

      await pressBlockCopy(getLaunchBlock(dialog));
      await waitFor(() =>
        expect(writeText).toHaveBeenCalledWith(buildPlaceholderLaunchCommand('second-agent')),
      );
    });
  });

  describe('its requests', () => {
    it('sends the typed name without its surrounding spaces when an agent is created', async () => {
      const createdAgentBodies: unknown[] = [];
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [] }),
        http.post(AGENTS_PATH, async ({ request }) => {
          const { body } = await inspectApiRequest(request, {
            method: 'post',
            path: '/verification-agents',
          });
          createdAgentBodies.push(body);

          return HttpResponse.json(
            { agent: buildAgent({ name: body.name }), token: ISSUED_TOKEN },
            { status: 201 },
          );
        }),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText(agentsCopy.empty);

      fireEvent.click(getCreateButton());
      const dialog = await screen.findByRole('dialog');
      fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), {
        target: { value: '  Build server  ' },
      });
      fireEvent.click(within(dialog).getByRole('button', { name: commonActions.create }));

      expect(await screen.findByText(ISSUED_TOKEN)).toBeTruthy();
      expect(createdAgentBodies).toEqual([{ name: 'Build server' }]);
    });

    it('rotates the token of the chosen agent when another agent is listed first', async () => {
      const rotatedAgentIds: unknown[] = [];
      mockServer.use(
        mockApi.get('/verification-agents', 200, { items: [firstAgent, secondAgent] }),
        http.post(`${AGENTS_PATH}/:agentId/rotate-token`, ({ params }) => {
          rotatedAgentIds.push(params.agentId);

          return HttpResponse.json({ token: ISSUED_TOKEN });
        }),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Second agent');

      fireEvent.click(
        within(getAgentRow('Second agent')).getByRole('button', {
          name: agentsCopy.actions.rotateToken,
        }),
      );
      const dialog = await screen.findByRole('dialog');
      fireEvent.click(within(dialog).getByRole('button', { name: agentsCopy.actions.rotateToken }));

      expect(await screen.findByText(ISSUED_TOKEN)).toBeTruthy();
      expect(rotatedAgentIds).toEqual(['second-agent']);
    });

    it('deletes the chosen agent and keeps the other when another agent is listed first', async () => {
      const deletedAgentIds: unknown[] = [];
      declareListAnswersRepeatingLast(
        () => listResponse(firstAgent, secondAgent),
        () => listResponse(firstAgent),
      );
      mockServer.use(
        http.delete(`${AGENTS_PATH}/:agentId`, ({ params }) => {
          deletedAgentIds.push(params.agentId);

          return new HttpResponse(null, { status: 204 });
        }),
      );
      renderScreen(<VerificationAgentsComponent />);
      await screen.findByText('Second agent');

      await pressRowDelete('Second agent');

      await waitFor(() => expect(screen.queryByText('Second agent')).toBeNull());
      expect(deletedAgentIds).toEqual(['second-agent']);
      expect(screen.getByText('First agent')).toBeTruthy();
    });
  });

  it('reveals the token once after an agent is created and clears it after confirmation', async () => {
    mockServer.use(
      mockApi.get('/verification-agents', 200, { items: [] }),
      mockApi.post('/verification-agents', 201, {
        agent: {
          id: 'agent-id',
          name: 'My agent',
          maxCpu: 0,
          maxRamMb: 0,
          maxDiskGb: 0,
          maxConcurrentJobs: 0,
          createdAt: '2026-10-27T10:00:00Z',
        },
        token: 'issued-once-token',
      }),
    );

    renderScreen(<VerificationAgentsComponent />);
    fireEvent.click(await screen.findByRole('button', { name: /Create/ }));
    const dialog = await screen.findByRole('dialog');

    fireEvent.change(within(dialog).getByPlaceholderText('Agent name'), {
      target: { value: 'My agent' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create' }));

    expect(await screen.findByText('issued-once-token')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: "I've saved the token" }));
    await waitFor(() => expect(screen.queryByText('issued-once-token')).toBeNull());
    expect(screen.getByText('My agent')).toBeTruthy();
  });
  it.each([
    { memory: 512, label: '512 MB RAM' },
    { memory: 4096, label: '4 GB RAM' },
  ])('shows $label when the agent reports $memory megabytes', async ({ memory, label }) => {
    mockServer.use(
      mockApi.get('/verification-agents', 200, {
        items: [
          {
            id: 'agent-id',
            name: 'My agent',
            maxCpu: 0,
            maxRamMb: memory,
            maxDiskGb: 0,
            maxConcurrentJobs: 0,
            createdAt: '2026-10-27T10:00:00Z',
          },
        ],
      }),
    );

    renderScreen(<VerificationAgentsComponent />);
    expect(await screen.findByText(label)).toBeTruthy();
  });
});

it.each([{ isRefused: false }, { isRefused: true }])(
  'suppresses old agent completion after the screen unmounts for a new session with refusal=$isRefused',
  async ({ isRefused }) => {
    activateSession({ token: 'old-session', userId: 'old-owner' });

    mockServer.use(mockApi.get('/verification-agents', 200, { items: [buildAgent()] }));
    const view = renderScreen(<VerificationAgentsComponent />);
    await screen.findByText('Original agent');
    const pending = mockApi.pending('delete', '/verification-agents/{agentId}');
    mockServer.use(pending.handler);

    await pressRowDelete();
    await waitFor(() => expect(mockServer.getRequests()).toHaveLength(2));

    act(() => activateSession({ token: 'new-session', userId: 'new-owner' }));
    const provider = document.querySelector('.ant-app');
    view.rerender(<div>New account</div>);

    const processed = new Promise<void>((resolve) => {
      const onResponse = () => {
        mockServer.events.removeListener('response:mocked', onResponse);
        resolve();
      };
      mockServer.events.on('response:mocked', onResponse);
    });

    await act(async () => {
      pending.respond(
        isRefused
          ? HttpResponse.json(buildProblem('permission_denied', 403), { status: 403 })
          : new HttpResponse(null, { status: 204 }),
      );
      await processed;
    });

    expect(document.querySelector('.ant-app')).toBe(provider);
    expect(document.querySelector('.ant-message-notice')).toBeNull();
    expect(getSessionSnapshot().token).toBe('new-session');
  },
);

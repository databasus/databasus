// @vitest-environment happy-dom
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import type { User } from '../../../entities/users';
import type { Workspace, WorkspaceMember } from '../../../entities/workspaces';
import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { buildProblem, mockApi, setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { WorkspaceMembersComponent } from './WorkspaceMembersComponent';

const mockServer = setUpMockApi();

const owner: User = {
  id: 'owner-id',
  email: 'owner@example.com',
  name: 'Owner',
  role: 'MEMBER',
  status: 'ACTIVE',
  createdAt: '2026-10-01T10:00:00Z',
};

const selectedWorkspace: Workspace = {
  id: 'workspace-id',
  name: 'Team',
  createdAt: '2026-10-01T10:00:00Z',
  currentUserRole: 'WORKSPACE_OWNER',
};

const buildMember = (email: string, role: WorkspaceMember['role']): WorkspaceMember => ({
  id: `${email}-member-id`,
  workspaceId: selectedWorkspace.id,
  userId: `${email}-user-id`,
  email,
  name: email.split('@')[0],
  role,
  isInvited: false,
  createdAt: '2026-10-01T10:00:00Z',
});

const mockMemberList = (members: WorkspaceMember[]) =>
  mockServer.use(mockApi.get('/workspace-members', 200, { items: members }));

// Records every addition the screen sends and answers it with the given response.
const mockAddition = (
  status: number,
  body: ApiComponents['schemas']['AddedWorkspaceMember'] | ApiComponents['schemas']['Problem'],
) => {
  const sentAdditions: ApiComponents['schemas']['WorkspaceMemberAddition'][] = [];
  mockServer.use(
    http.post(`*${API_BASE_PATH}/workspace-members`, async ({ request }) => {
      sentAdditions.push(
        (await request.json()) as ApiComponents['schemas']['WorkspaceMemberAddition'],
      );

      return HttpResponse.json(body, { status });
    }),
  );

  return sentAdditions;
};

const openAddMemberDialog = async () => {
  await screen.findByText('owner@example.com');
  fireEvent.click(screen.getByRole('button', { name: /Add member/ }));
};

const submitAddition = (email: string) => {
  fireEvent.change(screen.getByPlaceholderText('Enter email address'), {
    target: { value: email },
  });

  const addButtons = screen.getAllByRole('button', { name: /Add member/ });

  fireEvent.click(addButtons[addButtons.length - 1]);
};

describe('WorkspaceMembersComponent', () => {
  it.each(['ADMIN', 'MEMBER'] as const)(
    'associates the visible email label with the control when the user role is %s',
    async (role) => {
      mockMemberList([buildMember('owner@example.com', 'WORKSPACE_OWNER')]);
      renderScreen(
        <WorkspaceMembersComponent
          selectedWorkspace={selectedWorkspace}
          user={{ ...owner, role }}
        />,
      );
      await openAddMemberDialog();
      const dialog = within(screen.getByRole('dialog', { name: 'Add member' }));

      const emailControl = dialog.getByRole(role === 'ADMIN' ? 'combobox' : 'textbox', {
        name: 'Email address',
      });
      const emailLabel = dialog.getByText('Email address', { selector: 'label' });

      expect(emailControl.id).toBeTruthy();
      expect(emailLabel.getAttribute('for')).toBe(emailControl.id);
      expect(dialog.getByLabelText('Email address', { exact: true })).toBe(emailControl);
      expect(document.querySelectorAll(`[id="${emailControl.id}"]`)).toHaveLength(1);
    },
  );

  it('adds an existing user with the selected role when the address has an account', async () => {
    mockMemberList([buildMember('owner@example.com', 'WORKSPACE_OWNER')]);
    const sentAdditions = mockAddition(201, {
      ...buildMember('colleague@example.com', 'WORKSPACE_MEMBER'),
      outcome: 'ADDED',
    });
    renderScreen(<WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />);
    await openAddMemberDialog();

    submitAddition('colleague@example.com');

    expect(await screen.findByText('Member added successfully')).toBeTruthy();
    expect(sentAdditions).toEqual([
      { workspaceId: 'workspace-id', email: 'colleague@example.com', role: 'WORKSPACE_MEMBER' },
    ]);
  });

  it('shows the invitation dialog when the address has no account', async () => {
    mockMemberList([buildMember('owner@example.com', 'WORKSPACE_OWNER')]);
    mockAddition(201, {
      ...buildMember('newcomer@example.com', 'WORKSPACE_MEMBER'),
      isInvited: true,
      outcome: 'INVITED',
    });
    renderScreen(<WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />);
    await openAddMemberDialog();

    submitAddition('newcomer@example.com');

    expect(await screen.findByText('User invited')).toBeTruthy();
    expect(screen.getByText('Invitation sent to newcomer@example.com')).toBeTruthy();
  });

  it('shows the translated refusal when the user is already a member', async () => {
    mockMemberList([buildMember('owner@example.com', 'WORKSPACE_OWNER')]);
    mockAddition(409, buildProblem('already_member', 409));
    renderScreen(<WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />);
    await openAddMemberDialog();

    submitAddition('owner@example.com');

    expect(await screen.findByText('This user is already a member of the workspace.')).toBeTruthy();
  });

  it('marks a member as invited when sign-up is incomplete', async () => {
    mockMemberList([
      buildMember('owner@example.com', 'WORKSPACE_OWNER'),
      { ...buildMember('newcomer@example.com', 'WORKSPACE_VIEWER'), isInvited: true },
    ]);
    renderScreen(<WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />);

    expect(await screen.findByText('Invited')).toBeTruthy();
  });

  it('offers no owner role when a role is selected for a new member', async () => {
    mockMemberList([buildMember('owner@example.com', 'WORKSPACE_OWNER')]);
    renderScreen(<WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />);
    await openAddMemberDialog();

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Role' }));

    await waitFor(() =>
      expect(getOfferedRoleLabels()).toEqual(expect.arrayContaining(['Viewer', 'Member', 'Admin'])),
    );
    expect(getOfferedRoleLabels()).not.toContain('Owner');
  });

  const getOfferedRoleLabels = () =>
    Array.from(document.querySelectorAll('.ant-select-item-option')).map(
      (option) => option.textContent,
    );

  it.each([false, true])(
    'shows the acknowledged member role when a role change is refused=%s',
    async (isRefused) => {
      const colleague = buildMember('colleague@example.com', 'WORKSPACE_MEMBER');
      mockMemberList([buildMember(owner.email, 'WORKSPACE_OWNER'), colleague]);
      mockServer.use(
        isRefused
          ? mockApi.put(
              '/workspace-members/{memberId}',
              403,
              buildProblem('permission_denied', 403),
            )
          : mockApi.put('/workspace-members/{memberId}', 200, {
              ...colleague,
              role: 'WORKSPACE_VIEWER',
            }),
      );
      const { user: interaction } = renderScreen(
        <WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />,
      );
      const row = (await screen.findByText(colleague.email)).closest('tr')!;

      await interaction.click(within(row).getByRole('combobox', { name: 'Role' }));
      await interaction.click(
        await screen.findByText('Viewer', { selector: '.ant-select-item-option-content' }),
      );

      await screen.findByText(
        isRefused ? 'You do not have permission to do this.' : 'Member role updated successfully',
      );
      const request = mockServer.getRequests().find((request) => request.method === 'PUT')!;

      expect(decodeURIComponent(new URL(request.url).pathname)).toBe(
        `/api/v1/workspace-members/${colleague.id}`,
      );
      expect(await request.json()).toEqual({ role: 'WORKSPACE_VIEWER' });
      expect(
        within(row).getByText(isRefused ? 'Member' : 'Viewer', {
          selector: '.ant-select-selection-item',
        }),
      ).toBeTruthy();
      expect(screen.getByText(owner.email)).toBeTruthy();
    },
  );

  it.each([false, true])(
    'removes only the acknowledged member when removal is refused=%s',
    async (isRefused) => {
      const colleague = buildMember('colleague@example.com', 'WORKSPACE_MEMBER');
      mockMemberList([buildMember(owner.email, 'WORKSPACE_OWNER'), colleague]);
      mockServer.use(
        isRefused
          ? mockApi.delete(
              '/workspace-members/{memberId}',
              403,
              buildProblem('permission_denied', 403),
            )
          : mockApi.delete('/workspace-members/{memberId}', 204, undefined),
      );
      const { user: interaction } = renderScreen(
        <WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />,
      );
      const row = (await screen.findByText(colleague.email)).closest('tr')!;

      await interaction.click(within(row).getByRole('button', { name: 'Remove member' }));
      await interaction.click(await screen.findByRole('button', { name: 'Remove' }));

      await screen.findByText(
        isRefused
          ? 'You do not have permission to do this.'
          : 'Member "colleague@example.com" removed successfully',
      );
      const request = mockServer.getRequests().find((request) => request.method === 'DELETE')!;

      expect(decodeURIComponent(new URL(request.url).pathname)).toBe(
        `/api/v1/workspace-members/${colleague.id}`,
      );
      expect(screen.queryByText(colleague.email) !== null).toBe(isRefused);
      expect(screen.getByText(owner.email)).toBeTruthy();
    },
  );

  it.each(['WORKSPACE_OWNER', 'WORKSPACE_ADMIN'] as const)(
    'limits member controls when the current workspace role is %s',
    async (role) => {
      const workspaceOwner = {
        ...buildMember('workspace-owner@example.com', 'WORKSPACE_OWNER'),
        userId: 'workspace-owner',
      };
      const currentMember = { ...buildMember(owner.email, role), userId: owner.id };
      const administrator = buildMember('administrator@example.com', 'WORKSPACE_ADMIN');
      const colleague = buildMember('colleague@example.com', 'WORKSPACE_MEMBER');
      mockMemberList([workspaceOwner, currentMember, administrator, colleague]);

      const { user: interaction } = renderScreen(
        <WorkspaceMembersComponent
          selectedWorkspace={{ ...selectedWorkspace, currentUserRole: role }}
          user={owner}
        />,
      );
      await screen.findByText(colleague.email);
      const controlsFor = (email: string) => within(screen.getByText(email).closest('tr')!);

      for (const email of [workspaceOwner.email, currentMember.email]) {
        expect(controlsFor(email).queryByRole('combobox')).toBeNull();
        expect(controlsFor(email).queryByRole('button', { name: 'Remove member' })).toBeNull();
      }

      if (role === 'WORKSPACE_ADMIN') {
        expect(controlsFor(administrator.email).queryByRole('combobox')).toBeNull();
        expect(
          controlsFor(administrator.email).queryByRole('button', { name: 'Remove member' }),
        ).toBeNull();
      } else expect(controlsFor(administrator.email).getByRole('combobox')).toBeTruthy();

      await interaction.click(controlsFor(colleague.email).getByRole('combobox', { name: 'Role' }));
      await screen.findByText('Viewer', { selector: '.ant-select-item-option-content' });
      const offered = getOfferedRoleLabels();

      expect(offered).not.toContain('Owner');
      expect(offered.includes('Admin')).toBe(role === 'WORKSPACE_OWNER');
    },
  );

  it.each([320, 1280])(
    'distinguishes failed membership loading from an accepted empty list when width is %s',
    async (width) => {
      vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(width);
      mockServer.use(mockApi.get('/workspace-members', 500, buildProblem('internal', 500)));
      const view = renderScreen(
        <WorkspaceMembersComponent selectedWorkspace={selectedWorkspace} user={owner} />,
      );
      const retry = await screen.findByRole('button', { name: 'Retry' });

      expect(screen.queryAllByText('No members found')).toHaveLength(0);
      expect(screen.queryByRole('table')).toBeNull();

      mockMemberList([]);

      await view.user.click(retry);

      await screen.findAllByText('No members found');

      expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull();
    },
  );
});

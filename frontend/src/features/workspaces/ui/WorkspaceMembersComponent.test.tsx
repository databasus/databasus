// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

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

  it('marks an invited member who has not finished sign-up', async () => {
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
});

const getOfferedRoleLabels = () =>
  Array.from(document.querySelectorAll('.ant-select-item-option')).map(
    (option) => option.textContent,
  );

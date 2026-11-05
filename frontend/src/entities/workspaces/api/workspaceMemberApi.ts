import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { collectCursorPages } from '../../../shared/lib/collectCursorPages';
import type { AddedWorkspaceMember } from '../model/AddedWorkspaceMember';
import type { AssignableWorkspaceRole } from '../model/AssignableWorkspaceRole';
import type { WorkspaceMember } from '../model/WorkspaceMember';
import type { WorkspaceMemberAddition } from '../model/WorkspaceMemberAddition';

// The largest page the contract serves, so the members arrive in as few requests as possible.
const MEMBER_PAGE_LIMIT = 100;

export const workspaceMemberApi = {
  // The member screen shows every member and offers each of them for an ownership transfer, so
  // this follows the cursor to the last page.
  async listAllMembers(workspaceId: string): Promise<WorkspaceMember[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/workspace-members', {
          params: { query: { workspaceId, limit: MEMBER_PAGE_LIMIT, after } },
        }),
      ),
    );
  },

  async addMember(addition: WorkspaceMemberAddition): Promise<AddedWorkspaceMember> {
    return requireData(await getApiClient().POST('/workspace-members', { body: addition }));
  },

  async changeMemberRole(
    memberId: string,
    role: AssignableWorkspaceRole,
  ): Promise<WorkspaceMember> {
    return requireData(
      await getApiClient().PUT('/workspace-members/{memberId}', {
        params: { path: { memberId } },
        body: { role },
      }),
    );
  },

  async removeMember(memberId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/workspace-members/{memberId}', {
        params: { path: { memberId } },
      }),
    );
  },
};

import type { AddedWorkspaceMember } from '@/entities/workspaces/model/AddedWorkspaceMember';
import type { AssignableWorkspaceRole } from '@/entities/workspaces/model/AssignableWorkspaceRole';
import type { WorkspaceMember } from '@/entities/workspaces/model/WorkspaceMember';
import type { WorkspaceMemberAddition } from '@/entities/workspaces/model/WorkspaceMemberAddition';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

// The largest page the contract serves, so the members arrive in as few requests as possible.
const MEMBER_PAGE_LIMIT = 100;

export const workspaceMemberApi = {
  // The member screen shows every member and offers each of them for an ownership transfer, so
  // this follows the cursor to the last page.
  async listAllMembers(workspaceId: string, options: ReadOptions = {}): Promise<WorkspaceMember[]> {
    return collectCursorPages(
      async (after: string | undefined) =>
        requireData(
          await getApiClient(options).GET('/workspace-members', {
            params: { query: { workspaceId, limit: MEMBER_PAGE_LIMIT, after } },
          }),
        ),
      options,
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

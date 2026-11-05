import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { collectCursorPages } from '../../../shared/lib/collectCursorPages';
import type { Workspace } from '../model/Workspace';

// The largest page the contract serves, so a team's workspaces arrive in as few requests as possible.
const WORKSPACE_PAGE_LIMIT = 100;

export const workspaceApi = {
  // The workspace selector and the transfer dialogs need every workspace, so this follows the
  // cursor to the last page.
  async listAllWorkspaces(): Promise<Workspace[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/workspaces', {
          params: { query: { limit: WORKSPACE_PAGE_LIMIT, after } },
        }),
      ),
    );
  },

  async createWorkspace(name: string): Promise<Workspace> {
    return requireData(await getApiClient().POST('/workspaces', { body: { name } }));
  },

  async getWorkspace(workspaceId: string): Promise<Workspace> {
    return requireData(
      await getApiClient().GET('/workspaces/{workspaceId}', { params: { path: { workspaceId } } }),
    );
  },

  async updateWorkspace(workspaceId: string, name: string): Promise<Workspace> {
    return requireData(
      await getApiClient().PUT('/workspaces/{workspaceId}', {
        params: { path: { workspaceId } },
        body: { name },
      }),
    );
  },

  async deleteWorkspace(workspaceId: string): Promise<void> {
    requireSuccess(
      await getApiClient().DELETE('/workspaces/{workspaceId}', {
        params: { path: { workspaceId } },
      }),
    );
  },

  async transferOwnership(workspaceId: string, newOwnerMemberId: string): Promise<void> {
    requireSuccess(
      await getApiClient().POST('/workspaces/{workspaceId}/transfer-ownership', {
        params: { path: { workspaceId } },
        body: { newOwnerMemberId },
      }),
    );
  },
};

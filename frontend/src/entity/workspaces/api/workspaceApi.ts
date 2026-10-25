import { getApplicationServer } from '../../../constants';
import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { apiHelper } from '../../../shared/api/apiHelper';
import RequestOptions from '../../../shared/api/RequestOptions';
import type { GetAuditLogsResponse } from '../../audit-logs/model/GetAuditLogsResponse';
import type { Workspace } from '../model/Workspace';

// The largest page the contract serves, so a team's workspaces arrive in as few requests as possible.
const WORKSPACE_PAGE_LIMIT = 100;

export const workspaceApi = {
  // The workspace selector and the transfer dialogs need every workspace, so this follows the
  // cursor to the last page.
  async listAllWorkspaces(): Promise<Workspace[]> {
    const workspaces: Workspace[] = [];
    let after: string | undefined;

    do {
      const page = requireData(
        await getApiClient().GET('/workspaces', {
          params: { query: { limit: WORKSPACE_PAGE_LIMIT, after } },
        }),
      );

      workspaces.push(...page.items);
      after = page.nextCursor;
    } while (after !== undefined);

    return workspaces;
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

  async getWorkspaceAuditLogs(
    workspaceId: string,
    params?: {
      limit?: number;
      offset?: number;
      beforeDate?: string;
    },
  ): Promise<GetAuditLogsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/workspaces/${workspaceId}/audit-logs`;
    const urlParams = new URLSearchParams();

    if (params?.limit) {
      urlParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      urlParams.append('offset', params.offset.toString());
    }
    if (params?.beforeDate) {
      urlParams.append('beforeDate', params.beforeDate);
    }

    if (urlParams.toString()) {
      url += `?${urlParams.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },
};

import { getApiClient, requireData, requireSuccess } from '../../../shared/api';
import { collectCursorPages } from '../../../shared/lib/collectCursorPages';
import type { VerificationAgent } from '../model/VerificationAgent';

export const verificationAgentApi = {
  async listAgents(): Promise<VerificationAgent[]> {
    return collectCursorPages(async (after: string | undefined) =>
      requireData(
        await getApiClient().GET('/verification-agents', {
          params: { query: { limit: 100, after } },
        }),
      ),
    );
  },
  async getAgentCount() {
    return requireData(await getApiClient().GET('/verification-agent-counts'));
  },
  async createAgent(name: string) {
    return requireData(await getApiClient().POST('/verification-agents', { body: { name } }));
  },
  async rotateToken(agentId: string) {
    return requireData(
      await getApiClient().POST('/verification-agents/{agentId}/rotate-token', {
        params: { path: { agentId } },
      }),
    );
  },
  async deleteAgent(agentId: string) {
    requireSuccess(
      await getApiClient().DELETE('/verification-agents/{agentId}', {
        params: { path: { agentId } },
      }),
    );
  },
};

import type { VerificationAgent } from '@/entities/verification-agents/model/VerificationAgent';
import type { ReadOptions } from '@/shared/api';
import { getApiClient, requireData, requireSuccess } from '@/shared/api';
import { collectCursorPages } from '@/shared/lib/collectCursorPages';

export const verificationAgentApi = {
  async listAgents(options: ReadOptions = {}): Promise<VerificationAgent[]> {
    return collectCursorPages(
      async (after: string | undefined) =>
        requireData(
          await getApiClient(options).GET('/verification-agents', {
            params: { query: { limit: 100, after } },
          }),
        ),
      options,
    );
  },
  async getAgentCount(options: ReadOptions = {}) {
    return requireData(await getApiClient(options).GET('/verification-agent-counts'));
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

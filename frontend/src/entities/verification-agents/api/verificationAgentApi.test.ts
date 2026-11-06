import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { verificationAgentApi } from '@/entities/verification-agents/api/verificationAgentApi';
import { setUpMockApi } from '@/shared/lib/test/mockApi';

const mockServer = setUpMockApi();

describe('verificationAgentApi.listAgents', () => {
  it('returns the agents of every page and sends the limit and cursor', async () => {
    const requestedQueries: URLSearchParams[] = [];
    mockServer.use(
      http.get('*/api/v1/verification-agents', ({ request }) => {
        const query = new URL(request.url).searchParams;
        requestedQueries.push(query);

        return HttpResponse.json(
          query.get('after') === null
            ? { items: [{ id: 'first-agent' }], nextCursor: 'second-page' }
            : { items: [{ id: 'second-agent' }] },
        );
      }),
    );

    const listedAgents = await verificationAgentApi.listAgents();

    expect(listedAgents.map((agent) => agent.id)).toEqual(['first-agent', 'second-agent']);
    expect(requestedQueries.map((query) => query.get('after'))).toEqual([null, 'second-page']);
    expect(requestedQueries.map((query) => query.get('limit'))).toEqual(['100', '100']);
  });
});

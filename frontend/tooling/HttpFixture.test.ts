import { describe, expect, it } from 'vitest';

describe('Browser HTTP fixture declarations', () => {
  it('rejects declarations when methods or paths are unrestricted or duplicated', async () => {
    const { validateHttpFixtures } = await import('../browser/HttpFixture.ts');
    const response = { method: 'GET', path: '/notifiers/{notifierId}', status: 200, body: {} };
    expect(() => validateHttpFixtures([response])).not.toThrow();
    expect(() => validateHttpFixtures([{ ...response, path: '/notifiers/*' }])).toThrow(
      'finite pathname',
    );
    expect(() => validateHttpFixtures([{ ...response, method: '*' }])).toThrow('explicit method');
    expect(() => validateHttpFixtures([response, response])).toThrow('Duplicate HTTP fixture');
  });
  it('ties response bodies and methods to generated operations when fixtures are typechecked', async () => {
    const { apiResponse } = await import('../browser/HttpFixture.ts');
    const response = apiResponse({
      method: 'post',
      path: '/notifier-tests',
      status: 200,
      body: { isDelivered: true },
    });
    expect(response.path).toBe('/notifier-tests');
    apiResponse({
      method: 'get',
      // @ts-expect-error The delivery operation accepts POST only.
      path: '/notifier-tests',
      status: 200,
      // @ts-expect-error No GET response has a delivery response shape.
      body: { isDelivered: true },
    });
    // @ts-expect-error The generated delivery response requires isDelivered.
    apiResponse({ method: 'post', path: '/notifier-tests', status: 200, body: { delivery: true } });
  });
});

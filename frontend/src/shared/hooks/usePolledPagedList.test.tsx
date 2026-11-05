// @vitest-environment happy-dom
import { act, fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { waitForCondition } from '../lib/test/conditionWait';
import { mockApi, setUpMockApi } from '../lib/test/mockApi';
import { renderScreen } from '../lib/test/renderScreen';
import { PolledPagedListScreen } from './pagedListScreen.fixture';

const server = setUpMockApi();

describe('usePolledPagedList', () => {
  it('waits for the initial page before polling and stops its timer on unmount', async () => {
    const firstPage = mockApi.delayed({
      method: 'get',
      path: '/workspaces',
      status: 200,
      body: { items: [], nextCursor: 'next' },
    });
    server.use(firstPage.handler);
    vi.useFakeTimers();

    const controlledView = renderScreen(<PolledPagedListScreen />);
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(1));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3_000);
    });
    expect(server.getRequests()).toHaveLength(1);

    await act(async () => {
      firstPage.release();
    });
    await waitForCondition(() => expect(screen.getByText('Ready:Idle')).toBeTruthy());

    server.use(mockApi.get('/workspaces', 200, { items: [] }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    await waitForCondition(() => expect(server.getRequests()).toHaveLength(2));

    const removeListener = vi.spyOn(screen.getByTestId('scroll-list'), 'removeEventListener');
    controlledView.unmount();
    expect(removeListener).toHaveBeenCalledWith('scroll', expect.any(Function));
    expect(vi.getTimerCount()).toBe(0);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5_000);
    });
    expect(server.getRequests()).toHaveLength(2);
    fireEvent.scroll(document.body);
  });
});

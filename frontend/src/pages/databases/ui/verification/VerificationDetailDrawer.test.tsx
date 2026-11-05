// @vitest-environment happy-dom
import { screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { HttpResponse, http } from 'msw';
import { describe, expect, it, vi } from 'vitest';

import { API_BASE_PATH } from '../../../../shared/api';
import { mockApi, setUpMockApi } from '../../../../shared/lib/test/mockApi';
import { createPendingApiResponse } from '../../../../shared/lib/test/pendingApiResponse';
import { releaseAndSettle } from '../../../../shared/lib/test/releaseAndSettle';
import { renderScreen } from '../../../../shared/lib/test/renderScreen';
import { verificationRunsApi } from '../../api/verificationRunsApi';
import type { VerificationDetails } from '../../model/verification/VerificationDetails';
import { VerificationDetailDrawer } from './VerificationDetailDrawer';

dayjs.extend(utc);

const mockServer = setUpMockApi();

const buildDetails = (change: Partial<VerificationDetails> = {}): VerificationDetails => ({
  id: 'verification-id',
  databaseId: 'database-id',
  backupId: 'backup-id',
  trigger: 'MANUAL',
  status: 'COMPLETED',
  attemptCount: 1,
  createdAt: '2026-10-27T10:00:00Z',
  tableStats: [],
  ...change,
});

const renderDrawer = (verificationId: string) => (
  <VerificationDetailDrawer
    verificationId={verificationId}
    isOpen={true}
    onClose={() => {}}
    onClosed={() => {}}
  />
);

describe('VerificationDetailDrawer', () => {
  it('shows the newest verification when an earlier one answers after it', async () => {
    const heldFirst = createPendingApiResponse();
    const detailsCall = vi.spyOn(verificationRunsApi, 'getById');
    mockServer.use(
      http.get(`*${API_BASE_PATH}/verifications/:verificationId`, ({ params }) =>
        params.verificationId === 'first'
          ? heldFirst.response
          : HttpResponse.json(buildDetails({ schemaCount: 22 })),
      ),
    );
    const { rerender } = renderScreen(renderDrawer('first'));
    await waitFor(() => expect(detailsCall).toHaveBeenCalledOnce());

    rerender(renderDrawer('second'));
    expect(await screen.findByText('22')).toBeTruthy();

    await releaseAndSettle(
      detailsCall,
      () => heldFirst.respond(HttpResponse.json(buildDetails({ schemaCount: 11 }))),
      0,
    );

    expect(screen.getByText('22')).toBeTruthy();
    expect(screen.queryByText('11')).toBeNull();
  });

  it('gives each table its own row key when two schemas hold a table of the same name', async () => {
    const reportError = vi.spyOn(console, 'error').mockImplementation(() => {});
    mockServer.use(
      mockApi.get(
        '/verifications/{verificationId}',
        200,
        buildDetails({
          tableStats: [
            { schemaName: 'public', name: 'orders', rowCount: 20 },
            { schemaName: 'archive', name: 'orders', rowCount: 10 },
          ],
        }),
      ),
    );

    renderScreen(renderDrawer('first'));

    expect(await screen.findAllByText('orders')).toHaveLength(2);
    expect(screen.getByText('public')).toBeTruthy();
    expect(screen.getByText('archive')).toBeTruthy();
    expect(reportError.mock.calls.filter(([report]) => /\bkey\b/.test(String(report)))).toEqual([]);
  });
});

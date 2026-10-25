// @vitest-environment happy-dom
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';

import { API_BASE_PATH, type ApiComponents } from '../../../shared/api';
import { setUpMockApi } from '../../../shared/lib/test/mockApi';
import { renderScreen } from '../../../shared/lib/test/renderScreen';
import { UsersComponent } from './UsersComponent';

const mockServer = setUpMockApi();

const buildUser = (email: string): ApiComponents['schemas']['User'] => ({
  id: `${email}-id`,
  email,
  name: email,
  role: 'MEMBER',
  status: 'ACTIVE',
  createdAt: '2026-10-01T10:00:00Z',
});

describe('UsersComponent', () => {
  it('loads the next page with the cursor of the previous one when the list is scrolled down', async () => {
    const requestedCursors: (string | null)[] = [];
    mockServer.use(
      http.get(`*${API_BASE_PATH}/users`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('after');
        requestedCursors.push(cursor);

        const userPage: ApiComponents['schemas']['UserPage'] =
          cursor === null
            ? { items: [buildUser('first@example.com')], nextCursor: 'first-page-cursor' }
            : { items: [buildUser('second@example.com')] };

        return HttpResponse.json(userPage);
      }),
    );
    const { container } = renderScreen(<UsersComponent contentHeight={600} />);
    await screen.findByText(/first@example\.com/);

    fireEvent.scroll(container.querySelector('.overflow-y-auto') as Element);

    expect(await screen.findByText(/second@example\.com/)).toBeTruthy();
    await waitFor(() => expect(requestedCursors).toEqual([null, 'first-page-cursor']));
  });
});

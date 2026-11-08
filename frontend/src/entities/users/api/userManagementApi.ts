import type { User } from '@/entities/users/model/User';
import type { UserRole } from '@/entities/users/model/UserRole';
import type { ReadOptions } from '@/shared/api';
import { type ApiComponents, type ApiPaths, getApiClient, requireData } from '@/shared/api';

type UserListQuery = NonNullable<ApiPaths['/users']['get']['parameters']['query']>;

export const userManagementApi = {
  async getUsers(
    listQuery: UserListQuery,
    options: ReadOptions = {},
  ): Promise<ApiComponents['schemas']['UserPage']> {
    return requireData(await getApiClient(options).GET('/users', { params: { query: listQuery } }));
  },

  async getUserProfile(userId: string, options: ReadOptions = {}): Promise<User> {
    return requireData(
      await getApiClient(options).GET('/users/{userId}', { params: { path: { userId } } }),
    );
  },

  async deactivateUser(userId: string): Promise<User> {
    return requireData(
      await getApiClient().POST('/users/{userId}/deactivate', { params: { path: { userId } } }),
    );
  },

  async activateUser(userId: string): Promise<User> {
    return requireData(
      await getApiClient().POST('/users/{userId}/activate', { params: { path: { userId } } }),
    );
  },

  async changeUserRole(userId: string, role: UserRole): Promise<User> {
    return requireData(
      await getApiClient().PUT('/users/{userId}/role', {
        params: { path: { userId } },
        body: { role },
      }),
    );
  },
};

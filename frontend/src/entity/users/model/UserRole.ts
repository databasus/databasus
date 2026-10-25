import type { ApiComponents } from '../../../shared/api';

export type UserRole = ApiComponents['schemas']['UserRole'];

// The values the screens compare against. satisfies keeps them equal to the contract's enum.
export const UserRole = {
  ADMIN: 'ADMIN',
  MEMBER: 'MEMBER',
} as const satisfies Record<UserRole, UserRole>;

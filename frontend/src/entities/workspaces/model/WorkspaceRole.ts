import type { ApiComponents } from '../../../shared/api';

export type WorkspaceRole = ApiComponents['schemas']['WorkspaceRole'];

// The values the screens compare against. satisfies keeps them within the contract's enum.
export const WorkspaceRole = {
  OWNER: 'WORKSPACE_OWNER',
  ADMIN: 'WORKSPACE_ADMIN',
  MEMBER: 'WORKSPACE_MEMBER',
  VIEWER: 'WORKSPACE_VIEWER',
} as const satisfies Record<string, WorkspaceRole>;

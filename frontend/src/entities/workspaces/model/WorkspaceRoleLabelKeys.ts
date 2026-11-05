import type { TranslationKey } from '../../../shared/i18n';
import { WorkspaceRole } from './WorkspaceRole';

export const WORKSPACE_ROLE_LABEL_KEYS: Record<WorkspaceRole, TranslationKey> = {
  [WorkspaceRole.OWNER]: 'workspaces.roles.owner',
  [WorkspaceRole.ADMIN]: 'workspaces.roles.admin',
  [WorkspaceRole.MEMBER]: 'workspaces.roles.member',
  [WorkspaceRole.VIEWER]: 'workspaces.roles.viewer',
};

import type { TranslationKey } from '../../../shared/i18n';
import { UserRole } from './UserRole';

export const USER_ROLE_LABEL_KEYS: Record<UserRole, TranslationKey> = {
  [UserRole.ADMIN]: 'users.roles.admin',
  [UserRole.MEMBER]: 'users.roles.member',
};

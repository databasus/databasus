import { UserRole } from '@/entities/users/model/UserRole';
import type { TranslationKey } from '@/shared/i18n';

export const USER_ROLE_LABEL_KEYS: Record<UserRole, TranslationKey> = {
  [UserRole.ADMIN]: 'users.roles.admin',
  [UserRole.MEMBER]: 'users.roles.member',
};

export {
  getSessionSnapshot,
  subscribeSession,
  isCurrentSession,
  activateSession,
  invalidateSession,
  beginAuthIntent,
  isCurrentAuthIntent,
  abandonAuthIntent,
  acceptAuthSession,
} from '@/shared/auth/session';
export type { SessionSnapshot, AuthIntent } from '@/shared/auth/session';
export { useSession } from '@/shared/auth/useSession';
export { useAuthOperation } from '@/shared/auth/useAuthOperation';
export { useSessionGuard } from '@/shared/auth/useSessionGuard';

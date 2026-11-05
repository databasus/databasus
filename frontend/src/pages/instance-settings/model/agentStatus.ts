import type { TranslationKey } from '../../../shared/i18n';

// Agents are expected to heartbeat every 30 seconds; the 15-second list poll on
// the page gives a snappy transition between Online and Stale.
const ONLINE_WINDOW_MS = 90 * 1000;
const STALE_WINDOW_MS = 5 * 60 * 1000;

type AgentStatus = 'never-seen' | 'online' | 'stale' | 'offline';

export const getAgentStatus = (
  lastSeenAt: string | undefined,
  now: number = Date.now(),
): AgentStatus => {
  if (!lastSeenAt) return 'never-seen';

  const diff = now - new Date(lastSeenAt).getTime();
  if (diff <= ONLINE_WINDOW_MS) return 'online';
  if (diff <= STALE_WINDOW_MS) return 'stale';
  return 'offline';
};

export const AGENT_STATUS_TEXT_CLASSES: Record<AgentStatus, string> = {
  'never-seen': 'text-muted',
  online: 'text-success',
  stale: 'text-warning',
  offline: 'text-danger',
};

export const AGENT_STATUS_LABEL_KEYS: Record<AgentStatus, TranslationKey> = {
  'never-seen': 'status.agent.neverSeen',
  online: 'status.agent.online',
  stale: 'status.agent.stale',
  offline: 'status.agent.offline',
};

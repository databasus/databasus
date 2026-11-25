import type { ApiComponents } from '@/shared/api';

export type TwoFactorRequirement = ApiComponents['schemas']['TwoFactorRequirement'];

// The values the screens compare against. satisfies keeps them equal to the contract's enum.
export const TwoFactorRequirement = {
  OFF: 'OFF',
  EMAIL: 'EMAIL',
  AUTHENTICATOR_APP: 'AUTHENTICATOR_APP',
} as const satisfies Record<TwoFactorRequirement, TwoFactorRequirement>;

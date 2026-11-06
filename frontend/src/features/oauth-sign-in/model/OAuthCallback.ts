import { userApi } from '@/entities/users';
import { consumeOAuthSignInAttempt } from '@/features/oauth-sign-in/model/OAuthSignInAttempt';
import {
  beginAuthIntent,
  isCurrentAuthIntent,
  acceptAuthSession,
  type AuthIntent,
} from '@/shared/auth';

export class OAuthCallback {
  private exchange?: ReturnType<typeof userApi.handleGitHubOAuth>;
  private intent?: AuthIntent;
  constructor(
    private readonly parameters: {
      code: string | null;
      state: string | null;
      error: string | null;
    },
  ) {}

  Start() {
    if (!this.exchange) {
      this.intent = beginAuthIntent();
      this.exchange = this.exchangeCode();
    }
    return this.exchange;
  }

  IsCurrent() {
    return !!this.intent && isCurrentAuthIntent(this.intent);
  }
  Accept(session: Awaited<ReturnType<typeof userApi.handleGitHubOAuth>>) {
    return !!this.intent && acceptAuthSession({ intent: this.intent, session });
  }

  private async exchangeCode() {
    const attempt = consumeOAuthSignInAttempt(this.parameters.state);
    if (this.parameters.error || !this.parameters.code)
      throw new Error('OAuth authorization was not completed');
    const request = { code: this.parameters.code, redirectUri: attempt.redirectUri };
    return attempt.provider === 'github'
      ? userApi.handleGitHubOAuth(request)
      : userApi.handleGoogleOAuth(request);
  }
}

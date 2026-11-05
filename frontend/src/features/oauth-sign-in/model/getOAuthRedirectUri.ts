export function getOAuthRedirectUri(): string {
  return `${window.location.origin}/auth/callback`;
}

import { useSessionQuery } from '@/shared/api';
import { APP_VERSION } from '@/shared/config';
import { compareVersions, parseGitHubTag } from '@/shared/lib';

const GITHUB_RELEASES_API = 'https://api.github.com/repos/databasus/databasus/releases/latest';

export function useIsNewGitHubVersionAvailable(): boolean {
  const query = useSessionQuery({
    queryKey: ['github-release'],
    enabled: APP_VERSION !== 'dev',
    queryFn: async (signal) => {
      const response = await fetch(GITHUB_RELEASES_API, {
        signal,
        headers: { Accept: 'application/vnd.github+json' },
      });
      if (!response.ok) return null;

      const release = (await response.json()) as { tag_name?: string };

      return release.tag_name ? parseGitHubTag(release.tag_name) : null;
    },
  });

  return !!query.data && compareVersions(query.data, APP_VERSION) > 0;
}

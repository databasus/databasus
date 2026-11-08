export type ApplicationSection =
  | 'databases'
  | 'storages'
  | 'notifiers'
  | 'settings'
  | 'profile'
  | 'users'
  | 'databasus-settings';

export interface ApplicationAddress {
  section: ApplicationSection;
  workspaceId?: string;
  databaseId?: string;
  databaseTab?: string;
}

export const parseApplicationPath = (path: string): ApplicationAddress | undefined => {
  if (path === '/') return { section: 'databases' };

  if (path === '/profile' || path === '/users' || path === '/databasus-settings')
    return { section: path.slice(1) as ApplicationSection };

  const match =
    /^\/workspaces\/([a-zA-Z0-9_-]+)\/(databases|storages|notifiers|settings)(?:\/([a-zA-Z0-9_-]+)(?:\/([a-zA-Z0-9_-]+))?)?$/.exec(
      path,
    );
  if (!match || (match[3] && match[2] !== 'databases')) return undefined;

  return {
    section: match[2] as ApplicationSection,
    workspaceId: match[1],
    databaseId: match[3],
    databaseTab: match[4],
  };
};

export const getApplicationPath = (address: ApplicationAddress): string => {
  if (['profile', 'users', 'databasus-settings'].includes(address.section))
    return `/${address.section}`;

  if (!address.workspaceId) return '/';

  const sectionPath = `/workspaces/${address.workspaceId}/${address.section}`;

  return address.section === 'databases' && address.databaseId
    ? `${sectionPath}/${address.databaseId}/${address.databaseTab ?? 'backups'}`
    : sectionPath;
};

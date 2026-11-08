import { useEffect, useEffectEvent, useState } from 'react';

import type { Database } from '@/entities/databases';
import { getDatabaseTab } from '@/pages/databases/lib/getDatabaseTab';
import type { DatabaseDestination } from '@/pages/databases/model/DatabaseDestination';

interface Options {
  workspaceId: string;
  databases: Database[];
  requestedDatabaseId?: string;
  requestedTab?: string;
  isReady: boolean;
  isMobile: boolean;
  canSelectDefaultDatabase: boolean;
  onNavigate: (destination: DatabaseDestination) => void;
}

export function useDatabaseNavigation(options: Options) {
  const { databases, requestedDatabaseId, requestedTab, isReady, isMobile, workspaceId } = options;
  // eslint-disable-next-line i18next/no-literal-string -- browser preference key
  const preferenceKey = `selectedDatabaseId_${workspaceId}`;
  const readPreference = () => {
    try {
      return { workspaceId, databaseId: localStorage.getItem(preferenceKey) ?? undefined };
    } catch {
      return { workspaceId, databaseId: undefined };
    }
  };
  const [rememberedSelection, setRememberedSelection] = useState(readPreference);
  const currentSelection =
    rememberedSelection.workspaceId === workspaceId ? rememberedSelection : readPreference();
  if (rememberedSelection.workspaceId !== workspaceId) setRememberedSelection(currentSelection);

  const requestedDatabase = databases.find((database) => database.id === requestedDatabaseId);
  const defaultDatabase = requestedDatabaseId
    ? databases[0]
    : isMobile || !options.canSelectDefaultDatabase
      ? undefined
      : (databases.find((database) => database.id === currentSelection.databaseId) ?? databases[0]);
  const database = isReady ? (requestedDatabase ?? defaultDatabase) : undefined;
  const tab = database ? getDatabaseTab(database, requestedTab) : 'backups';
  const isResolved =
    isReady && requestedDatabaseId === database?.id && (!database || requestedTab === tab);
  const publishDestination = useEffectEvent(() =>
    options.onNavigate({ databaseId: database?.id, tab, shouldReplace: true }),
  );

  useEffect(() => {
    if (!isReady) return;

    if (!isResolved) publishDestination();

    setRememberedSelection((selection) =>
      selection.workspaceId === workspaceId && selection.databaseId === database?.id
        ? selection
        : { workspaceId, databaseId: database?.id },
    );

    try {
      if (database) localStorage.setItem(preferenceKey, database.id);
      else localStorage.removeItem(preferenceKey);
    } catch {}
  }, [isReady, isResolved, database, tab, preferenceKey, workspaceId]);

  return { database, tab, isResolved };
}

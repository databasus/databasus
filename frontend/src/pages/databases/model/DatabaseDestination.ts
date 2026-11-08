import type { DatabaseTab } from '@/pages/databases/model/DatabaseTab';

export interface DatabaseDestination {
  databaseId?: string;
  tab?: DatabaseTab;
  shouldReplace?: boolean;
}

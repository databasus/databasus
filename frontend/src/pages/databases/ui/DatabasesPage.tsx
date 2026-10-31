import type { Workspace } from '@/entities/workspaces';
import { DatabasesComponent } from '@/features/databases';

import { databaseResourceComposition } from './databaseResourceComposition';

interface Props {
  workspace: Workspace;
  contentHeight: number;
  isCanManageDBs: boolean;
}
export function DatabasesPage(props: Props) {
  return <DatabasesComponent {...props} resourceComposition={databaseResourceComposition} />;
}

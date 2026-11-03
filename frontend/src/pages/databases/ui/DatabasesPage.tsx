import type { Workspace } from '@/entities/workspaces';

import { DatabasesComponent } from './DatabasesComponent';

interface Props {
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageDBs: boolean;
}
export function DatabasesPage(props: Props) {
  return <DatabasesComponent {...props} />;
}

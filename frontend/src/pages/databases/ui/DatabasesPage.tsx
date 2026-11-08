import type { Workspace } from '@/entities/workspaces';
import type { DatabaseDestination } from '@/pages/databases/model/DatabaseDestination';
import { DatabasesComponent } from '@/pages/databases/ui/DatabasesComponent';

interface Props {
  navigationIntent: string;
  workspace: Workspace;
  onUnavailable?: (workspaceId: string) => void;
  isCanManageDBs: boolean;
  selectedDatabaseId?: string;
  selectedTab?: string;
  onNavigate: (destination: DatabaseDestination) => void;
}
export function DatabasesPage(props: Props) {
  return <DatabasesComponent {...props} />;
}

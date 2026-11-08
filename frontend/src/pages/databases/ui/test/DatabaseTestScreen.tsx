import { type ComponentProps, useState } from 'react';

import type { DatabaseTab } from '@/pages/databases/model/DatabaseTab';
import { DatabaseComponent } from '@/pages/databases/ui/DatabaseComponent';

type Props = Omit<
  ComponentProps<typeof DatabaseComponent>,
  'workspaceId' | 'currentTab' | 'onTabChange' | 'onUnavailable'
>;

export function DatabaseTestScreen(props: Props) {
  const [tab, setTab] = useState<DatabaseTab>('backups');

  return (
    <DatabaseComponent
      {...props}
      workspaceId="workspace-id"
      currentTab={tab}
      onTabChange={setTab}
      onUnavailable={() => {}}
    />
  );
}

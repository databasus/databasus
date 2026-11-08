import type { ComponentProps } from 'react';
import { useLocation, useNavigate } from 'react-router';

import { DatabasesComponent } from '@/pages/databases/ui/DatabasesComponent';
import { getApplicationPath, parseApplicationPath, useNavigationIntent } from '@/shared/navigation';

type Props = Omit<
  ComponentProps<typeof DatabasesComponent>,
  'selectedDatabaseId' | 'selectedTab' | 'onNavigate' | 'navigationIntent'
>;

export function DatabasesTestScreen(props: Props) {
  const location = useLocation();
  const navigationIntent = useNavigationIntent();
  const navigate = useNavigate();
  const address = parseApplicationPath(location.pathname);
  const isAddressedWorkspace = address?.workspaceId === props.workspace.id;

  return (
    <DatabasesComponent
      {...props}
      navigationIntent={navigationIntent}
      selectedDatabaseId={isAddressedWorkspace ? address.databaseId : undefined}
      selectedTab={isAddressedWorkspace ? address.databaseTab : undefined}
      onNavigate={(selection) => {
        const path = getApplicationPath({
          section: 'databases',
          workspaceId: props.workspace.id,
          databaseId: selection.databaseId,
          databaseTab: selection.tab,
        });
        if (path !== location.pathname) void navigate(path, { replace: selection.shouldReplace });
      }}
    />
  );
}

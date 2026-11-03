import { Button, Select } from 'antd';
import { useTranslation } from 'react-i18next';

import { type Workspace } from '@/entities/workspaces';
import { WRAPPING_SELECT_CLASS_NAME } from '@/shared/ui';

interface Props {
  canCreateWorkspace: boolean;
  workspaces: Workspace[];
  selectedWorkspace?: Workspace;
  onCreateWorkspace: () => void;
  onWorkspaceSelect: (workspace: Workspace) => void;
}

export const WorkspaceSelectionComponent = ({
  workspaces,
  selectedWorkspace,
  onCreateWorkspace,
  onWorkspaceSelect,
  canCreateWorkspace,
}: Props) => {
  const { t } = useTranslation();
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2">
      {workspaces.length > 0 && (
        <Select
          aria-label={t('app.workspaceSelection.selected')}
          className={`w-full min-w-0 md:w-[240px] ${WRAPPING_SELECT_CLASS_NAME}`}
          showSearch
          optionFilterProp="label"
          value={selectedWorkspace?.id}
          options={workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name }))}
          onChange={(id) => {
            const workspace = workspaces.find((item) => item.id === id);
            if (workspace) onWorkspaceSelect(workspace);
          }}
        />
      )}
      {canCreateWorkspace && (
        <Button
          size="small"
          className="!h-auto min-h-6 max-w-full !whitespace-normal"
          onClick={onCreateWorkspace}
        >
          {workspaces.length
            ? t('app.workspaceSelection.createNew')
            : t('app.workspaceSelection.create')}
        </Button>
      )}
    </div>
  );
};

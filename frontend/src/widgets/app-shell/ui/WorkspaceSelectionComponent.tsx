import { Button } from '@mantine/core';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Workspace } from '@/entities/workspaces';
import { ResourcePicker } from '@/shared/ui';

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
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const createWorkspace = () => {
    setIsDropdownOpen(false);
    workspaceSelectRef.current?.focus();
    onCreateWorkspace();
  };

  const { t } = useTranslation();
  const workspaceSelectRef = useRef<HTMLButtonElement | HTMLInputElement>(null);

  return (
    <div className="flex w-full min-w-0 flex-col">
      {workspaces.length > 0 ? (
        <ResourcePicker
          targetRef={workspaceSelectRef}
          label={t('app.workspaceSelection.selected')}
          className="w-full"
          isSearchable
          isOpen={isDropdownOpen}
          onOpenChange={setIsDropdownOpen}
          data={workspaces.map((workspace) => ({ value: workspace.id, label: workspace.name }))}
          selectedValues={selectedWorkspace ? [selectedWorkspace.id] : []}
          onSelect={(id) => {
            const workspace = workspaces.find((item) => item.id === id);
            if (workspace) onWorkspaceSelect(workspace);
          }}
          createLabel={t('app.workspaceSelection.createNew')}
          onCreate={canCreateWorkspace ? createWorkspace : undefined}
        />
      ) : canCreateWorkspace ? (
        <Button
          size="xs"
          className="!h-auto min-h-6 max-w-full !whitespace-normal"
          onClick={onCreateWorkspace}
        >
          {t('app.workspaceSelection.create')}
        </Button>
      ) : null}
    </div>
  );
};

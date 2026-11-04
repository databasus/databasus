import { Button, Select, type GetRef } from 'antd';
import { useRef, useState } from 'react';
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
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const createWorkspace = () => {
    setIsDropdownOpen(false);
    workspaceSelectRef.current?.focus();
    onCreateWorkspace();
  };

  const { t } = useTranslation();
  const workspaceSelectRef = useRef<GetRef<typeof Select>>(null);
  const createButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <div className="flex w-full min-w-0 flex-col">
      {workspaces.length > 0 ? (
        <Select
          ref={workspaceSelectRef}
          aria-label={t('app.workspaceSelection.selected')}
          className={`w-full min-w-0 ${WRAPPING_SELECT_CLASS_NAME}`}
          variant="filled"
          showSearch
          optionFilterProp="label"
          open={isDropdownOpen}
          onOpenChange={setIsDropdownOpen}
          onInputKeyDown={(event) => {
            if (event.key === 'Tab' && !event.shiftKey && isDropdownOpen && canCreateWorkspace) {
              event.preventDefault();
              event.stopPropagation();
              createButtonRef.current?.focus();
            }
          }}
          popupRender={(menu) => (
            <>
              {menu}
              {canCreateWorkspace && (
                <div className="mt-1 border-t border-border pt-1">
                  <Button
                    ref={createButtonRef}
                    type="text"
                    block
                    className="!h-auto min-h-8 !justify-start !whitespace-normal !text-accent"
                    onClick={createWorkspace}
                    onKeyDown={(event) => {
                      event.stopPropagation();

                      if (event.key === 'Escape' || (event.key === 'Tab' && event.shiftKey)) {
                        event.preventDefault();
                        if (event.key === 'Escape') setIsDropdownOpen(false);
                        workspaceSelectRef.current?.focus();
                      }
                    }}
                  >
                    {t('app.workspaceSelection.createNew')}
                  </Button>
                </div>
              )}
            </>
          )}
          value={selectedWorkspace?.id}
          options={workspaces.map((workspace) => ({
            value: workspace.id,
            label: workspace.name,
          }))}
          onChange={(id) => {
            const workspace = workspaces.find((item) => item.id === id);
            if (workspace) onWorkspaceSelect(workspace);
          }}
        />
      ) : canCreateWorkspace ? (
        <Button
          size="small"
          className="!h-auto min-h-6 max-w-full !whitespace-normal"
          onClick={onCreateWorkspace}
        >
          {t('app.workspaceSelection.create')}
        </Button>
      ) : null}
    </div>
  );
};

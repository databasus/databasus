import { Button, Radio } from '@mantine/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Storage } from '@/entities/storages';
import type { Workspace } from '@/entities/workspaces';
import { CreateWorkspaceForm } from '@/features/create-workspace';
import { EditStorageComponent } from '@/features/edit-storage';
import { useSessionGuard } from '@/shared/auth';
import { Dialog } from '@/shared/ui';

const NEW_WORKSPACE_CHOICE = 'new-workspace';

interface Props {
  workspaces: Workspace[];
  onWorkspaceCreated: (workspace: Workspace) => void;
  onStorageCreated: (storage: Storage, workspace: Workspace) => void;
  onClose: () => void;
}

// The workspace a new destination lives in is chosen or created first; a created workspace stays
// chosen when the storage step fails, so a retry never creates a second one.
export function CreateDestinationDialog({
  workspaces,
  onWorkspaceCreated,
  onStorageCreated,
  onClose,
}: Props) {
  const { t } = useTranslation();
  const isCurrent = useSessionGuard();
  const [createdWorkspaces, setCreatedWorkspaces] = useState<Workspace[]>([]);
  const availableWorkspaces = [
    ...workspaces.filter(
      (workspace) => !createdWorkspaces.some((created) => created.id === workspace.id),
    ),
    ...createdWorkspaces,
  ];
  const [workspaceChoice, setWorkspaceChoice] = useState<string>(
    availableWorkspaces[0]?.id ?? NEW_WORKSPACE_CHOICE,
  );
  const [storageWorkspace, setStorageWorkspace] = useState<Workspace>();

  const acceptWorkspace = (workspace: Workspace) => {
    if (!isCurrent()) return;

    setCreatedWorkspaces((current) => [...current, workspace]);
    setWorkspaceChoice(workspace.id);
    setStorageWorkspace(workspace);
    onWorkspaceCreated(workspace);
  };

  const continueWithChosenWorkspace = () => {
    const chosenWorkspace = availableWorkspaces.find(
      (workspace) => workspace.id === workspaceChoice,
    );
    if (chosenWorkspace) setStorageWorkspace(chosenWorkspace);
  };

  return (
    <Dialog
      title={t('instanceBackups.createDestination.title')}
      opened
      closeOnClickOutside={false}
      onClose={onClose}
    >
      {storageWorkspace ? (
        <>
          <p className="my-3 text-muted">
            {t('instanceBackups.createDestination.storageStep', {
              workspace: storageWorkspace.name,
            })}
          </p>
          <EditStorageComponent
            key={storageWorkspace.id}
            workspaceId={storageWorkspace.id}
            isShowName
            isShowClose={false}
            onClose={onClose}
            onChanged={(storage) => {
              if (!isCurrent()) return;

              onStorageCreated(storage, storageWorkspace);
              onClose();
            }}
          />
        </>
      ) : (
        <>
          <Radio.Group
            label={t('instanceBackups.createDestination.workspace')}
            value={workspaceChoice}
            onChange={setWorkspaceChoice}
          >
            <div className="mt-2 space-y-2">
              {availableWorkspaces.map((workspace) => (
                <Radio key={workspace.id} value={workspace.id} label={workspace.name} />
              ))}
              <Radio
                value={NEW_WORKSPACE_CHOICE}
                label={t('instanceBackups.createDestination.newWorkspace')}
              />
            </div>
          </Radio.Group>

          {workspaceChoice === NEW_WORKSPACE_CHOICE ? (
            <div className="mt-4">
              <CreateWorkspaceForm
                workspacesCount={availableWorkspaces.length}
                submitLabel={t('instanceBackups.createDestination.createWorkspace')}
                onCancel={onClose}
                onWorkspaceCreated={acceptWorkspace}
              />
            </div>
          ) : (
            <div className="mt-5 flex flex-wrap justify-end gap-2">
              <Button onClick={onClose}>{t('common.actions.cancel')}</Button>
              <Button variant="filled" onClick={continueWithChosenWorkspace}>
                {t('instanceBackups.createDestination.continue')}
              </Button>
            </div>
          )}
        </>
      )}
    </Dialog>
  );
}

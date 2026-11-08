import { useTranslation } from 'react-i18next';

import { databaseApi } from '@/entities/databases';
import { type Notifier, notifierApi } from '@/entities/notifiers';
import { workspaceApi } from '@/entities/workspaces';
import { ResourceTransferDialog } from '@/shared/ui';

interface Props {
  notifier: Notifier;
  onClose: () => void;
  onTransferred: () => void;
}

export const NotifierTransferDialogComponent = ({ notifier, onClose, onTransferred }: Props) => {
  const { t } = useTranslation();

  return (
    <ResourceTransferDialog
      contextKey={`${notifier.workspaceId}/${notifier.id}`}
      loadAvailability={async (signal) => {
        const notifierDatabases = await databaseApi.listAllDatabases(notifier.workspaceId, {
          notifierId: notifier.id,
          signal,
        });
        signal.throwIfAborted();

        if (notifierDatabases.length > 0) return { isBlocked: true };

        const workspaces = await workspaceApi.listAllWorkspaces({ signal });

        return {
          isBlocked: false,
          targets: workspaces
            .filter((workspace) => workspace.id !== notifier.workspaceId)
            .map((workspace) => ({ label: workspace.name, value: workspace.id })),
        };
      }}
      transfer={(targetWorkspaceId) => notifierApi.transferNotifier(notifier.id, targetWorkspaceId)}
      onClose={onClose}
      onTransferred={onTransferred}
      labels={{
        title: t('notifiers.transfer.title'),
        blocked: t('notifiers.transfer.blockedByDatabases'),
        description: t('notifiers.transfer.description'),
        target: t('notifiers.transfer.targetWorkspace'),
        placeholder: t('notifiers.transfer.selectWorkspace'),
        submit: t('notifiers.transfer.submit'),
      }}
    />
  );
};

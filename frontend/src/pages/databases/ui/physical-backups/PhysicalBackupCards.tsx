import { Spin } from 'antd';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { formatBackupSize } from '@/pages/databases/lib/formatBackupSize';
import { getPhysicalBackupDuration } from '@/pages/databases/lib/physical-backups/getPhysicalBackupDuration';
import { type PhysicalBackupListItem } from '@/pages/databases/model/physical-backups/PhysicalBackupListItem';
import {
  PhysicalBackupActions,
  type PhysicalBackupRowActions,
} from '@/pages/databases/ui/physical-backups/PhysicalBackupActions';
import { PhysicalBackupCreatedAt } from '@/pages/databases/ui/physical-backups/PhysicalBackupCreatedAt';
import { PhysicalBackupStatusBadge } from '@/pages/databases/ui/physical-backups/PhysicalBackupStatusBadge';
import { PhysicalBackupTypeCellComponent } from '@/pages/databases/ui/physical-backups/PhysicalBackupTypeCellComponent';
import { useLocale } from '@/shared/i18n';

interface Props {
  backups: PhysicalBackupListItem[];
  isLoading: boolean;
  actions: PhysicalBackupRowActions;
  onShowError: (backup: PhysicalBackupListItem) => void;
}

export const PhysicalBackupCards = ({
  backups,
  isLoading,
  actions,
  onShowError,
}: Props): JSX.Element => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Spin />
      </div>
    );
  }

  return (
    <div>
      {backups.map((backup) => (
        <div
          key={backup.id}
          className="mb-2 rounded-lg border border-border bg-surface p-4 shadow-sm"
        >
          <div className="space-y-3">
            <div className="flex items-start justify-between">
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.created')}</div>
                <div className="text-sm font-medium">
                  <PhysicalBackupCreatedAt backup={backup} />
                </div>
              </div>
              <div>
                <PhysicalBackupStatusBadge backup={backup} onShowError={onShowError} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-muted">{t('common.fields.type')}</div>
                <div className="text-sm font-medium">
                  <PhysicalBackupTypeCellComponent backup={backup} />
                </div>
              </div>
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.size')}</div>
                <div className="text-sm font-medium">
                  {formatBackupSize(backup.sizeMb, formatNumber)}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted">{t('backups.list.columns.duration')}</div>
                <div className="text-sm font-medium">{getPhysicalBackupDuration(backup)}</div>
              </div>
            </div>

            <div className="flex items-center justify-end border-t border-border pt-3">
              <PhysicalBackupActions backup={backup} actions={actions} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};

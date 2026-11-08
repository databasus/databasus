import { MultiSelect } from '@mantine/core';
import { DateInput } from '@mantine/dates';
import dayjs from 'dayjs';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupListFilter } from '@/pages/databases/logical-backups/model/LogicalBackupListFilter';
import {
  LogicalBackupStatus,
  LOGICAL_BACKUP_STATUS_LABEL_KEYS,
} from '@/pages/databases/logical-backups/model/LogicalBackupStatus';
import { useBackupDatePopover } from '@/pages/databases/ui/useBackupDatePopover';

interface Props {
  filter: LogicalBackupListFilter;
  onFilterChange: (filter: LogicalBackupListFilter) => void;
}

const FILTERABLE_STATUSES = [
  LogicalBackupStatus.IN_PROGRESS,
  LogicalBackupStatus.COMPLETED,
  LogicalBackupStatus.FAILED,
  LogicalBackupStatus.CANCELED,
];

export const LogicalBackupsFiltersPanelComponent = ({ filter, onFilterChange }: Props) => {
  const { t } = useTranslation();
  const statusFieldId = useId();
  const beforeFieldId = useId();
  const datePopover = useBackupDatePopover();

  const handleStatusChange = (statuses: LogicalBackupStatus[]) => {
    onFilterChange({ ...filter, statuses: statuses.length > 0 ? statuses : undefined });
  };

  const handleCreatedBeforeChange = (date: string | null) => {
    datePopover.closeAndReturnFocus();
    onFilterChange({
      ...filter,
      createdBefore: date ? dayjs(date).toISOString() : undefined,
    });
  };

  const statusOptions = FILTERABLE_STATUSES.map((status) => ({
    label: t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status]),
    value: status,
  }));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex min-w-0 flex-col items-start gap-2 sm:flex-row sm:items-center">
        <label className="min-w-[90px] pr-2 text-sm text-muted" htmlFor={statusFieldId}>
          {t('common.fields.status')}
        </label>
        <MultiSelect
          id={statusFieldId}
          value={filter.statuses ?? []}
          onChange={handleStatusChange}
          data={statusOptions}
          placeholder={t('backups.filters.allStatuses')}
          size="xs"
          variant="filled"
          className="w-full max-w-[200px]"
          clearable
        />
      </div>

      <div
        className="flex min-w-0 flex-col items-start gap-2 sm:flex-row sm:items-center"
        onKeyDownCapture={datePopover.onKeyDownCapture}
      >
        <label className="min-w-[90px] pr-2 text-sm text-muted" htmlFor={beforeFieldId}>
          {t('backups.filters.before')}
        </label>
        <DateInput
          {...datePopover.inputProps}
          popoverProps={{
            ...datePopover.popoverProps,
            attributes: { dropdown: { 'aria-label': t('backups.filters.before') } },
          }}
          id={beforeFieldId}
          value={filter.createdBefore ? dayjs(filter.createdBefore).format('YYYY-MM-DD') : null}
          onChange={handleCreatedBeforeChange}
          size="xs"
          variant="filled"
          className="w-full max-w-[200px] !rounded-md"
          clearable
          clearButtonProps={{ 'aria-label': t('common.actions.clear') }}
        />
      </div>
    </div>
  );
};

import { MultiSelect } from '@mantine/core';
import { DateInput } from '@mantine/dates';
import dayjs from 'dayjs';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupsFilters } from '@/pages/databases/physical-backups/model/PhysicalBackupsFilters';
import {
  PhysicalBackupStatus,
  PHYSICAL_BACKUP_STATUS_LABEL_KEYS,
} from '@/pages/databases/physical-backups/model/PhysicalBackupStatus';
import {
  PhysicalBackupType,
  PHYSICAL_BACKUP_TYPE_LABEL_KEYS,
} from '@/pages/databases/physical-backups/model/PhysicalBackupType';
import { useBackupDatePopover } from '@/pages/databases/ui/useBackupDatePopover';

interface Props {
  filters: PhysicalBackupsFilters;
  onFiltersChange: (filters: PhysicalBackupsFilters) => void;
}

export const PhysicalBackupsFiltersPanelComponent = ({ filters, onFiltersChange }: Props) => {
  const { t } = useTranslation();
  const typeFieldId = useId();
  const statusFieldId = useId();
  const beforeFieldId = useId();
  const datePopover = useBackupDatePopover();

  const handleTypeChange = (types: PhysicalBackupType[]) => {
    onFiltersChange({ ...filters, types: types.length > 0 ? types : undefined });
  };

  const handleStatusChange = (statuses: PhysicalBackupStatus[]) => {
    onFiltersChange({ ...filters, statuses: statuses.length > 0 ? statuses : undefined });
  };

  const handleBeforeDateChange = (date: string | null) => {
    datePopover.closeAndReturnFocus();
    onFiltersChange({
      ...filters,
      createdBefore: date ? dayjs(date).toISOString() : undefined,
    });
  };

  const typeOptions = Object.values(PhysicalBackupType).map((type) => ({
    label: t(PHYSICAL_BACKUP_TYPE_LABEL_KEYS[type]),
    value: type,
  }));

  const statusOptions = Object.values(PhysicalBackupStatus).map((status) => ({
    label: t(PHYSICAL_BACKUP_STATUS_LABEL_KEYS[status]),
    value: status,
  }));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <label className="min-w-[90px] pr-2 text-sm text-muted" htmlFor={typeFieldId}>
          {t('common.fields.type')}
        </label>
        <MultiSelect
          id={typeFieldId}
          value={filters.types ?? []}
          onChange={handleTypeChange}
          data={typeOptions}
          placeholder={t('backups.filters.allTypes')}
          size="xs"
          variant="filled"
          className="w-[200px]"
          clearable
        />
      </div>

      <div className="flex items-center gap-2">
        <label className="min-w-[90px] pr-2 text-sm text-muted" htmlFor={statusFieldId}>
          {t('common.fields.status')}
        </label>
        <MultiSelect
          id={statusFieldId}
          value={filters.statuses ?? []}
          onChange={handleStatusChange}
          data={statusOptions}
          placeholder={t('backups.filters.allStatuses')}
          size="xs"
          variant="filled"
          className="w-[200px]"
          clearable
        />
      </div>

      <div className="flex items-center gap-2" onKeyDownCapture={datePopover.onKeyDownCapture}>
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
          value={filters.createdBefore ? dayjs(filters.createdBefore).format('YYYY-MM-DD') : null}
          onChange={handleBeforeDateChange}
          size="xs"
          variant="filled"
          className="w-[200px] !rounded-md"
          clearable
          clearButtonProps={{ 'aria-label': t('common.actions.clear') }}
        />
      </div>
    </div>
  );
};

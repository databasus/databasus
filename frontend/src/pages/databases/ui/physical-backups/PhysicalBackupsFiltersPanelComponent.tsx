import { DatePicker, Select } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import { type PhysicalBackupsFilters } from '../../model/physical-backups/PhysicalBackupsFilters';
import { PhysicalBackupStatus } from '../../model/physical-backups/PhysicalBackupStatus';
import { PHYSICAL_BACKUP_STATUS_LABEL_KEYS } from '../../model/physical-backups/PhysicalBackupStatusLabelKeys';
import { PhysicalBackupType } from '../../model/physical-backups/PhysicalBackupType';
import { PHYSICAL_BACKUP_TYPE_LABEL_KEYS } from '../../model/physical-backups/PhysicalBackupTypeLabelKeys';

interface Props {
  filters: PhysicalBackupsFilters;
  onFiltersChange: (filters: PhysicalBackupsFilters) => void;
}

export const PhysicalBackupsFiltersPanelComponent = ({ filters, onFiltersChange }: Props) => {
  const { t } = useTranslation();

  const handleTypeChange = (types: PhysicalBackupType[]) => {
    onFiltersChange({ ...filters, types: types.length > 0 ? types : undefined });
  };

  const handleStatusChange = (statuses: PhysicalBackupStatus[]) => {
    onFiltersChange({ ...filters, statuses: statuses.length > 0 ? statuses : undefined });
  };

  const handleBeforeDateChange = (date: Dayjs | null) => {
    onFiltersChange({
      ...filters,
      createdBefore: date ? date.toISOString() : undefined,
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
        <span className="min-w-[90px] pr-2 text-sm text-gray-500 dark:text-gray-400">
          {t('common.fields.type')}
        </span>
        <Select
          mode="multiple"
          value={filters.types ?? []}
          onChange={handleTypeChange}
          options={typeOptions}
          placeholder={t('backups.filters.allTypes')}
          size="small"
          variant="filled"
          className="w-[200px] [&_.ant-select-selector]:!rounded-md"
          allowClear
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="min-w-[90px] pr-2 text-sm text-gray-500 dark:text-gray-400">
          {t('common.fields.status')}
        </span>
        <Select
          mode="multiple"
          value={filters.statuses ?? []}
          onChange={handleStatusChange}
          options={statusOptions}
          placeholder={t('backups.filters.allStatuses')}
          size="small"
          variant="filled"
          className="w-[200px] [&_.ant-select-selector]:!rounded-md"
          allowClear
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="min-w-[90px] pr-2 text-sm text-gray-500 dark:text-gray-400">
          {t('backups.filters.before')}
        </span>
        <DatePicker
          value={filters.createdBefore ? dayjs(filters.createdBefore) : null}
          onChange={handleBeforeDateChange}
          size="small"
          variant="filled"
          className="w-[200px] !rounded-md"
          allowClear
        />
      </div>
    </div>
  );
};

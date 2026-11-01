import { DatePicker, Select } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';

import type { LogicalBackupListFilter } from '../../model/LogicalBackupListFilter';
import { LogicalBackupStatus } from '../../model/LogicalBackupStatus';
import { LOGICAL_BACKUP_STATUS_LABEL_KEYS } from '../../model/LogicalBackupStatusLabelKeys';

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

  const handleStatusChange = (statuses: LogicalBackupStatus[]) => {
    onFilterChange({ ...filter, statuses: statuses.length > 0 ? statuses : undefined });
  };

  const handleCreatedBeforeChange = (date: Dayjs | null) => {
    onFilterChange({
      ...filter,
      createdBefore: date ? date.toISOString() : undefined,
    });
  };

  const statusOptions = FILTERABLE_STATUSES.map((status) => ({
    label: t(LOGICAL_BACKUP_STATUS_LABEL_KEYS[status]),
    value: status,
  }));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <span className="min-w-[90px] pr-2 text-sm text-muted">{t('common.fields.status')}</span>
        <Select
          mode="multiple"
          value={filter.statuses ?? []}
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
        <span className="min-w-[90px] pr-2 text-sm text-muted">{t('backups.filters.before')}</span>
        <DatePicker
          value={filter.createdBefore ? dayjs(filter.createdBefore) : null}
          onChange={handleCreatedBeforeChange}
          size="small"
          variant="filled"
          className="w-[200px] !rounded-md"
          allowClear
        />
      </div>
    </div>
  );
};

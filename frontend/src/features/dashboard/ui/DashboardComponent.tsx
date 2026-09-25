import { ExclamationCircleOutlined, InfoCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import { App, Spin, Table, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { HealthStatusBadgeComponent, getDatabaseLogoFromType } from '../../../entity/databases';
import { HealthcheckAttemptsStripComponent } from '../../../entity/healthcheck';
import { getStorageLogoFromType } from '../../../entity/storages';
import { type UserProfile, UserRole } from '../../../entity/users';
import type { WorkspaceResponse } from '../../../entity/workspaces';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { formatSizeMb } from '../../../shared/lib';
import { getUserTimeFormat } from '../../../shared/time';
import { dashboardApi } from '../api/dashboardApi';
import type { DashboardDatabase } from '../model/DashboardDatabase';
import type { DashboardTotals } from '../model/DashboardTotals';
import type { WorkspaceDashboard } from '../model/WorkspaceDashboard';

interface Props {
  workspace: WorkspaceResponse;
  user: UserProfile;
  contentHeight: number;
}

interface SummaryTile {
  label: string;
  value: string;
  hint?: string;
  details?: string;
}

const DASHBOARD_REFRESH_MS = 60_000;
const NO_VALUE = '-';

const getTimestampOrZero = (date?: Date): number => (date ? dayjs(date).valueOf() : 0);

const renderNoValue = () => <span className="text-gray-400 dark:text-gray-500">{NO_VALUE}</span>;

const renderSummaryTile = ({ label, value, hint, details }: SummaryTile) => (
  <div key={label} className="rounded bg-white p-3 shadow md:p-4 dark:bg-gray-800">
    <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
      {label}
      {hint && (
        <Tooltip title={hint}>
          <InfoCircleOutlined />
        </Tooltip>
      )}
    </div>
    <div className="mt-1 text-xl font-bold">{value}</div>
    {details && <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">{details}</div>}
  </div>
);

const renderDatabaseName = (database: DashboardDatabase) => (
  <div className="flex min-w-0 items-center gap-2">
    <img src={getDatabaseLogoFromType(database.type)} alt="" className="h-4 w-4 shrink-0" />
    <span className="font-medium break-all">{database.name}</span>
  </div>
);

const renderHealthStatus = (database: DashboardDatabase) =>
  database.healthStatus ? (
    <HealthStatusBadgeComponent healthStatus={database.healthStatus} />
  ) : (
    renderNoValue()
  );

const renderHealthcheckAttempts = (database: DashboardDatabase) =>
  database.recentHealthcheckAttempts.length > 0 ? (
    <div className="min-w-[120px]">
      <HealthcheckAttemptsStripComponent attempts={database.recentHealthcheckAttempts} />
    </div>
  ) : (
    renderNoValue()
  );

const renderStorage = (database: DashboardDatabase) =>
  database.storage ? (
    <span className="inline-flex items-center gap-1">
      {database.storage.name}
      <img src={getStorageLogoFromType(database.storage.type)} alt="" className="h-4 w-4" />
    </span>
  ) : (
    renderNoValue()
  );

const renderMobileField = (label: string, value: ReactNode) => (
  <div>
    <div className="text-xs text-gray-500 dark:text-gray-400">{label}</div>
    <div className="text-sm">{value}</div>
  </div>
);

export const DashboardComponent = ({ workspace, user, contentHeight }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();
  const { message } = App.useApp();

  const [workspaceDashboard, setWorkspaceDashboard] = useState<WorkspaceDashboard | undefined>();
  const [installationTotals, setInstallationTotals] = useState<DashboardTotals | undefined>();
  const [isLoading, setIsLoading] = useState(true);

  const isAdmin = user.role === UserRole.ADMIN;

  const loadDashboard = async (isSilent: boolean) => {
    if (!isSilent) {
      setIsLoading(true);
    }

    try {
      const [loadedWorkspaceDashboard, loadedInstallationTotals] = await Promise.all([
        dashboardApi.getWorkspaceDashboard(workspace.id),
        isAdmin ? dashboardApi.getInstallationDashboard() : Promise.resolve(undefined),
      ]);

      setWorkspaceDashboard(loadedWorkspaceDashboard);
      setInstallationTotals(loadedInstallationTotals);
    } catch (e) {
      message.error(translateApiError(e, t));
    }

    if (!isSilent) {
      setIsLoading(false);
    }
  };

  const renderBackupsCount = (database: DashboardDatabase) => (
    <div>
      <div>{formatNumber(database.backupsCount)}</div>
      {database.completedBackupsCount !== database.backupsCount && (
        <div className="text-xs text-gray-500 dark:text-gray-400">
          {t('dashboard.list.successfulBackups', {
            successfulBackupsCount: formatNumber(database.completedBackupsCount),
          })}
        </div>
      )}
      {database.failedBackupsCount > 0 && (
        <div className="text-xs text-red-600 dark:text-red-400">
          {t('dashboard.list.failedBackups', {
            failedBackupsCount: formatNumber(database.failedBackupsCount),
          })}
        </div>
      )}
    </div>
  );

  const renderMeanBackupSize = (database: DashboardDatabase) =>
    database.completedBackupsCount > 0
      ? formatSizeMb(database.meanBackupSizeMb, formatNumber)
      : renderNoValue();

  const renderLastBackup = (database: DashboardDatabase) => (
    <div className="flex items-center gap-1">
      {database.lastBackupTime ? (
        <Tooltip title={dayjs(database.lastBackupTime).format(getUserTimeFormat().format)}>
          <span>{formatRelativeTime(database.lastBackupTime)}</span>
        </Tooltip>
      ) : (
        <span className="text-gray-400 dark:text-gray-500">{t('dashboard.list.noBackups')}</span>
      )}
      {database.lastBackupErrorMessage && (
        <Tooltip title={t('databases.card.hasBackupError')}>
          <ExclamationCircleOutlined className="text-red-500" />
        </Tooltip>
      )}
    </div>
  );

  useEffect(() => {
    loadDashboard(false);

    const refreshInterval = setInterval(() => loadDashboard(true), DASHBOARD_REFRESH_MS);

    return () => clearInterval(refreshInterval);
  }, [workspace.id]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center" style={{ height: contentHeight }}>
        <Spin indicator={<LoadingOutlined spin />} size="large" />
      </div>
    );
  }

  if (!workspaceDashboard) {
    return null;
  }

  const { databases, totals } = workspaceDashboard;

  const summaryTiles: SummaryTile[] = [
    { label: t('dashboard.tiles.databases'), value: formatNumber(totals.databasesCount) },
    { label: t('dashboard.tiles.backups'), value: formatNumber(totals.backupsCount) },
    {
      label: t('dashboard.tiles.backupsSize'),
      value: formatSizeMb(totals.totalBackupSizeMb, formatNumber),
      hint: t('dashboard.tiles.backupsSizeHint'),
    },
  ];

  if (installationTotals) {
    summaryTiles.push({
      label: t('dashboard.tiles.installation'),
      value: formatSizeMb(installationTotals.totalBackupSizeMb, formatNumber),
      hint: t('dashboard.tiles.installationHint'),
      details: t('dashboard.tiles.installationDetails', {
        databasesCount: formatNumber(installationTotals.databasesCount),
        backupsCount: formatNumber(installationTotals.backupsCount),
      }),
    });
  }

  const columns: ColumnsType<DashboardDatabase> = [
    {
      title: t('dashboard.list.columns.database'),
      key: 'name',
      render: (_: unknown, database: DashboardDatabase) => renderDatabaseName(database),
      sorter: (a, b) => a.name.localeCompare(b.name),
    },
    {
      title: t('common.fields.status'),
      key: 'healthStatus',
      render: (_: unknown, database: DashboardDatabase) => renderHealthStatus(database),
    },
    {
      title: t('dashboard.list.columns.healthcheck'),
      key: 'recentHealthcheckAttempts',
      render: (_: unknown, database: DashboardDatabase) => renderHealthcheckAttempts(database),
    },
    {
      title: t('dashboard.list.columns.backups'),
      key: 'backupsCount',
      render: (_: unknown, database: DashboardDatabase) => renderBackupsCount(database),
      sorter: (a, b) => a.backupsCount - b.backupsCount,
    },
    {
      title: t('dashboard.list.columns.meanSize'),
      key: 'meanBackupSizeMb',
      render: (_: unknown, database: DashboardDatabase) => renderMeanBackupSize(database),
      sorter: (a, b) => a.meanBackupSizeMb - b.meanBackupSizeMb,
    },
    {
      title: t('dashboard.list.columns.totalSize'),
      key: 'totalBackupSizeMb',
      render: (_: unknown, database: DashboardDatabase) =>
        formatSizeMb(database.totalBackupSizeMb, formatNumber),
      sorter: (a, b) => a.totalBackupSizeMb - b.totalBackupSizeMb,
    },
    {
      title: t('dashboard.list.columns.lastBackup'),
      key: 'lastBackupTime',
      render: (_: unknown, database: DashboardDatabase) => renderLastBackup(database),
      sorter: (a, b) => getTimestampOrZero(a.lastBackupTime) - getTimestampOrZero(b.lastBackupTime),
    },
    {
      title: t('dashboard.list.columns.storage'),
      key: 'storage',
      render: (_: unknown, database: DashboardDatabase) => renderStorage(database),
    },
  ];

  return (
    <div className="overflow-y-auto" style={{ height: contentHeight }}>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-4 md:gap-3">
        {summaryTiles.map((summaryTile) => renderSummaryTile(summaryTile))}
      </div>

      <div className="mt-2 rounded bg-white p-3 shadow md:mt-3 md:p-5 dark:bg-gray-800">
        <h2 className="text-lg font-bold md:text-xl">{t('dashboard.list.title')}</h2>

        {databases.length === 0 ? (
          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            {t('dashboard.list.empty')}
          </div>
        ) : (
          <>
            <div className="mt-4 hidden md:block">
              <Table
                bordered
                columns={columns}
                dataSource={databases}
                rowKey="id"
                size="small"
                pagination={false}
              />
            </div>

            <div className="mt-3 md:hidden">
              {databases.map((database) => (
                <div
                  key={database.id}
                  className="mb-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                >
                  <div className="flex items-start justify-between gap-2">
                    {renderDatabaseName(database)}
                    {database.healthStatus && (
                      <HealthStatusBadgeComponent healthStatus={database.healthStatus} />
                    )}
                  </div>

                  {database.recentHealthcheckAttempts.length > 0 && (
                    <div className="mt-2">{renderHealthcheckAttempts(database)}</div>
                  )}

                  <div className="mt-3 grid grid-cols-2 gap-3">
                    {renderMobileField(
                      t('dashboard.list.columns.backups'),
                      renderBackupsCount(database),
                    )}
                    {renderMobileField(
                      t('dashboard.list.columns.lastBackup'),
                      renderLastBackup(database),
                    )}
                    {renderMobileField(
                      t('dashboard.list.columns.meanSize'),
                      renderMeanBackupSize(database),
                    )}
                    {renderMobileField(
                      t('dashboard.list.columns.totalSize'),
                      formatSizeMb(database.totalBackupSizeMb, formatNumber),
                    )}
                    {renderMobileField(
                      t('dashboard.list.columns.storage'),
                      renderStorage(database),
                    )}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

import { Button, Spin, Tooltip } from 'antd';
import { Fragment, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useApplicationStartup } from '@/app/model/useApplicationStartup';
import { useIsNewGitHubVersionAvailable } from '@/app/model/useIsNewGitHubVersionAvailable';
import { CreateWorkspaceDialogComponent } from '@/app/ui/CreateWorkspaceDialogComponent';
import { DatabasesPage } from '@/pages/databases';
import { InstanceSettingsPage } from '@/pages/instance-settings';
import { NotifiersComponent } from '@/pages/notifiers';
import { ProfilePage } from '@/pages/profile';
import { StoragesComponent } from '@/pages/storages';
import { UsersPage } from '@/pages/users';
import { WorkspaceSettingsComponent } from '@/pages/workspace-settings';
import { useSessionGuard } from '@/shared/auth';
import { APP_VERSION, CONTAINER_ARCH } from '@/shared/config';
import { useLocale } from '@/shared/i18n';
import { LoadFailure } from '@/shared/ui';
import {
  AppShell,
  SidebarComponent,
  WorkspaceSelectionComponent,
  getDiskUsageValues,
  type MainTab,
  type SidebarTab,
} from '@/widgets/app-shell';

export const ApplicationScreen = () => {
  const [selectedTab, setSelectedTab] = useState<MainTab>('databases');
  const [isCreateWorkspaceDialogOpen, setShowCreateWorkspaceDialog] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const handleCreateWorkspace = () => {
    if (!canCreateWorkspace || !isCurrent()) return;

    workspaceCreationInvoker.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setShowCreateWorkspaceDialog(true);
  };
  const handleCloseCreateWorkspace = () => {
    if (!isCurrent()) return;

    setShowCreateWorkspaceDialog(false);
    queueMicrotask(() => {
      if (isCurrent() && workspaceCreationInvoker.current?.isConnected)
        workspaceCreationInvoker.current.focus({ preventScroll: true });
    });
  };

  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const startup = useApplicationStartup();
  const isCurrent = useSessionGuard();
  const workspaceCreationInvoker = useRef<HTMLElement | null>(null);
  const {
    user,
    settings: globalSettings,
    workspaces,
    selectedWorkspace,
    diskUsage,
    isLoading,
    isReady,
    isAdmin,
    canCreateWorkspace,
    canManageWorkspace,
  } = startup;

  const isNewGitHubVersionAvailable = useIsNewGitHubVersionAvailable();
  useEffect(() => {
    if (!isAdmin && (selectedTab === 'users' || selectedTab === 'databasus-settings'))
      setSelectedTab('databases');
  }, [isAdmin, selectedTab]);

  const isUsedMoreThan85Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.85;
  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  const tabs: SidebarTab[] = [
    {
      text: t('app.navigation.databases'),
      name: 'databases',
      icon: '/icons/menu/database-gray.svg',
      selectedIcon: '/icons/menu/database-white.svg',
      onClick: () => setSelectedTab('databases'),
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: true,
    },
    {
      text: t('app.navigation.storages'),
      name: 'storages',
      icon: '/icons/menu/storage-gray.svg',
      selectedIcon: '/icons/menu/storage-white.svg',
      onClick: () => setSelectedTab('storages'),
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.notifiers'),
      name: 'notifiers',
      icon: '/icons/menu/notifier-gray.svg',
      selectedIcon: '/icons/menu/notifier-white.svg',
      onClick: () => setSelectedTab('notifiers'),
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.workspaceSettings'),
      name: 'settings',
      icon: '/icons/menu/workspace-settings-gray.svg',
      selectedIcon: '/icons/menu/workspace-settings-white.svg',
      onClick: () => setSelectedTab('settings'),
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.profile'),
      name: 'profile',
      icon: '/icons/menu/profile-gray.svg',
      selectedIcon: '/icons/menu/profile-white.svg',
      onClick: () => setSelectedTab('profile'),
      isAdminOnly: false,
      marginTop: '25px',
      isVisible: true,
    },
    {
      text: t('app.navigation.databasusSettings'),
      name: 'databasus-settings',
      icon: '/icons/menu/global-settings-gray.svg',
      selectedIcon: '/icons/menu/global-settings-white.svg',
      onClick: () => setSelectedTab('databasus-settings'),
      isAdminOnly: true,
      marginTop: '0px',
      isVisible: true,
    },
    {
      text: t('app.navigation.users'),
      name: 'users',
      icon: '/icons/menu/user-card-gray.svg',
      selectedIcon: '/icons/menu/user-card-white.svg',
      onClick: () => setSelectedTab('users'),
      isAdminOnly: true,
      marginTop: '0px',
      isVisible: true,
    },
  ];

  return (
    <>
      <AppShell
        workspaceSelector={
          isReady && (
            <WorkspaceSelectionComponent
              workspaces={workspaces}
              selectedWorkspace={selectedWorkspace}
              onCreateWorkspace={handleCreateWorkspace}
              onWorkspaceSelect={startup.selectWorkspace}
              canCreateWorkspace={canCreateWorkspace}
            />
          )
        }
        auxiliary={
          isUsedMoreThan85Percent && (
            <Tooltip title={t('app.diskUsage.hint')}>
              <span className={`text-xs ${isUsedMoreThan95Percent ? 'text-danger' : 'text-muted'}`}>
                {t('app.diskUsage.summary', getDiskUsageValues(diskUsage, formatNumber))}
              </span>
            </Tooltip>
          )
        }
        sidebar={
          isReady && (
            <SidebarComponent
              isOpen={isSidebarOpen}
              onClose={() => setIsSidebarOpen(false)}
              selectedTab={selectedTab}
              tabs={tabs}
              user={user}
              diskUsage={diskUsage}
            />
          )
        }
        isSidebarOpen={isSidebarOpen}
        onOpenSidebar={() => setIsSidebarOpen(true)}
      >
        {isLoading ? (
          <div className="flex flex-1 items-center justify-center">
            <Spin />
          </div>
        ) : startup.error !== undefined ? (
          <LoadFailure error={startup.error} onRetry={() => void startup.retry()} />
        ) : (
          isReady && (
            <>
              {selectedTab === 'profile' && <ProfilePage onUserChanged={startup.acceptUser} />}
              {selectedTab === 'databasus-settings' && isAdmin && (
                <InstanceSettingsPage onSettingsChanged={startup.acceptSettings} />
              )}
              {selectedTab === 'users' && isAdmin && <UsersPage />}
              {['databases', 'storages', 'notifiers', 'settings'].includes(selectedTab) &&
                (selectedWorkspace ? (
                  <Fragment key={selectedWorkspace.id}>
                    {selectedTab === 'databases' && (
                      <DatabasesPage
                        onUnavailable={startup.reconcileUnavailableWorkspace}
                        workspace={selectedWorkspace}
                        isCanManageDBs={canManageWorkspace}
                      />
                    )}
                    {selectedTab === 'storages' && (
                      <StoragesComponent
                        onUnavailable={startup.reconcileUnavailableWorkspace}
                        workspace={selectedWorkspace}
                        isCanManageStorages={canManageWorkspace}
                      />
                    )}
                    {selectedTab === 'notifiers' && (
                      <NotifiersComponent
                        onUnavailable={startup.reconcileUnavailableWorkspace}
                        workspace={selectedWorkspace}
                        isCanManageNotifiers={canManageWorkspace}
                      />
                    )}
                    {selectedTab === 'settings' && user && (
                      <WorkspaceSettingsComponent
                        onRenamed={startup.updateWorkspace}
                        onDeleted={startup.removeWorkspace}
                        onUnavailable={startup.reconcileUnavailableWorkspace}
                        selectedWorkspace={selectedWorkspace}
                        user={user}
                      />
                    )}
                  </Fragment>
                ) : (
                  <div className="flex flex-1 items-center justify-center">
                    {canCreateWorkspace && (
                      <Button type="primary" onClick={handleCreateWorkspace}>
                        {t('app.workspaceSelection.create')}
                      </Button>
                    )}
                  </div>
                ))}
            </>
          )
        )}
        <div className="pointer-events-none absolute bottom-1 left-2 hidden text-xs text-muted md:block">
          {`v${APP_VERSION}`}
          <br />
          {CONTAINER_ARCH}
          {isNewGitHubVersionAvailable && (
            <Tooltip title={t('app.navigation.newVersionAvailable')} trigger={['hover', 'focus']}>
              <a
                className="pointer-events-auto ml-1 inline-flex"
                aria-label={t('app.navigation.newVersionAvailable')}
                href="https://github.com/databasus/databasus/releases/latest"
                target="_blank"
                rel="noreferrer"
              >
                <span className="inline-flex h-2 w-2 rounded-full bg-success-fill" />
              </a>
            </Tooltip>
          )}
        </div>
      </AppShell>
      {isCreateWorkspaceDialogOpen && user && globalSettings && (
        <CreateWorkspaceDialogComponent
          user={user}
          globalSettings={globalSettings}
          workspacesCount={workspaces.length}
          onClose={handleCloseCreateWorkspace}
          onWorkspaceCreated={(workspace) => {
            startup.acceptWorkspace(workspace);
            setSelectedTab('databases');
          }}
        />
      )}
    </>
  );
};

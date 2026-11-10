import { Button, Loader, Tooltip } from '@mantine/core';
import { HardDrive } from 'lucide-react';
import { Fragment, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';

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
import { getApplicationPath, parseApplicationPath, useNavigationIntent } from '@/shared/navigation';
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
  const location = useLocation();
  const navigationIntent = useNavigationIntent();
  const navigate = useNavigate();
  const address = parseApplicationPath(location.pathname) ?? { section: 'databases' as const };
  const selectedTab = address.section;
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
  const startup = useApplicationStartup(address.workspaceId);
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
    isReconcilingWorkspace,
    isAdmin,
    canCreateWorkspace,
    canManageWorkspace,
  } = startup;

  const isNewGitHubVersionAvailable = useIsNewGitHubVersionAvailable();
  const getTabPath = (section: MainTab) =>
    getApplicationPath({ section, workspaceId: selectedWorkspace?.id });
  const isWorkspaceSection = ['databases', 'storages', 'notifiers', 'settings'].includes(
    selectedTab,
  );
  const isUnavailableSection =
    !isAdmin && (selectedTab === 'users' || selectedTab === 'databasus-settings');
  const isUnavailableWorkspace =
    isWorkspaceSection && address.workspaceId !== selectedWorkspace?.id;
  const canonicalPath =
    isUnavailableSection || isUnavailableWorkspace
      ? getApplicationPath({ section: 'databases', workspaceId: selectedWorkspace?.id })
      : isWorkspaceSection && !selectedWorkspace
        ? '/'
        : location.pathname;
  const isNavigationReady =
    isReady && (isReconcilingWorkspace || canonicalPath === location.pathname);
  useEffect(() => {
    if (isReady && !isReconcilingWorkspace && canonicalPath !== location.pathname)
      void navigate(canonicalPath, { replace: true });
  }, [isReady, isReconcilingWorkspace, canonicalPath, location.pathname, navigate]);

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
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: true,
    },
    {
      text: t('app.navigation.storages'),
      name: 'storages',
      icon: HardDrive,
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.notifiers'),
      name: 'notifiers',
      icon: '/icons/menu/notifier-gray.svg',
      selectedIcon: '/icons/menu/notifier-white.svg',
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.workspaceSettings'),
      name: 'settings',
      icon: '/icons/menu/workspace-settings-gray.svg',
      selectedIcon: '/icons/menu/workspace-settings-white.svg',
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: !!selectedWorkspace,
    },
    {
      text: t('app.navigation.profile'),
      name: 'profile',
      icon: '/icons/menu/profile-gray.svg',
      selectedIcon: '/icons/menu/profile-white.svg',
      isAdminOnly: false,
      marginTop: '25px',
      isVisible: true,
    },
    {
      text: t('app.navigation.databasusSettings'),
      name: 'databasus-settings',
      icon: '/icons/menu/global-settings-gray.svg',
      selectedIcon: '/icons/menu/global-settings-white.svg',
      isAdminOnly: true,
      marginTop: '0px',
      isVisible: true,
    },
    {
      text: t('app.navigation.users'),
      name: 'users',
      icon: '/icons/menu/user-card-gray.svg',
      selectedIcon: '/icons/menu/user-card-white.svg',
      isAdminOnly: true,
      marginTop: '0px',
      isVisible: true,
    },
  ].map((tab) => ({ ...tab, href: getTabPath(tab.name as MainTab) })) as SidebarTab[];

  return (
    <>
      <AppShell
        workspaceSelector={
          isNavigationReady && (
            <WorkspaceSelectionComponent
              workspaces={workspaces}
              selectedWorkspace={selectedWorkspace}
              onCreateWorkspace={handleCreateWorkspace}
              onWorkspaceSelect={(workspace) => {
                if (selectedWorkspace?.id === workspace.id && !isReconcilingWorkspace) return;

                startup.selectWorkspace(workspace);
                if (isWorkspaceSection)
                  void navigate(
                    getApplicationPath({ section: selectedTab, workspaceId: workspace.id }),
                  );
              }}
              canCreateWorkspace={canCreateWorkspace}
            />
          )
        }
        auxiliary={
          isUsedMoreThan85Percent && (
            <Tooltip label={t('app.diskUsage.hint')}>
              <span className={`text-xs ${isUsedMoreThan95Percent ? 'text-danger' : 'text-muted'}`}>
                {t('app.diskUsage.summary', getDiskUsageValues(diskUsage, formatNumber))}
              </span>
            </Tooltip>
          )
        }
        sidebar={
          isNavigationReady && (
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
        {isLoading || isReconcilingWorkspace || (isReady && !isNavigationReady) ? (
          <div className="flex flex-1 items-center justify-center">
            <Loader role="status" />
          </div>
        ) : startup.error !== undefined ? (
          <LoadFailure error={startup.error} onRetry={() => void startup.retry()} />
        ) : (
          isNavigationReady && (
            <>
              {selectedTab === 'profile' && <ProfilePage onUserChanged={startup.acceptUser} />}
              {selectedTab === 'databasus-settings' && isAdmin && (
                <InstanceSettingsPage onSettingsChanged={startup.acceptSettings} />
              )}
              {selectedTab === 'users' && isAdmin && <UsersPage />}
              {['databases', 'storages', 'notifiers', 'settings'].includes(selectedTab) &&
                (selectedWorkspace && address.workspaceId === selectedWorkspace.id ? (
                  <Fragment key={selectedWorkspace.id}>
                    {selectedTab === 'databases' && (
                      <DatabasesPage
                        navigationIntent={navigationIntent}
                        selectedDatabaseId={address.databaseId}
                        selectedTab={address.databaseTab}
                        onNavigate={(selection) => {
                          const path = getApplicationPath({
                            section: 'databases',
                            workspaceId: selectedWorkspace.id,
                            databaseId: selection.databaseId,
                            databaseTab: selection.tab,
                          });
                          if (path !== window.location.pathname)
                            void navigate(path, { replace: selection.shouldReplace });
                        }}
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
                      <Button variant="filled" onClick={handleCreateWorkspace}>
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
            <Tooltip
              label={t('app.navigation.newVersionAvailable')}
              events={{ hover: true, focus: true, touch: false }}
            >
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
            void navigate(getApplicationPath({ section: 'databases', workspaceId: workspace.id }));
          }}
        />
      )}
    </>
  );
};

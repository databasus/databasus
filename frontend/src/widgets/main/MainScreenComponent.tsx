import { LoadingOutlined, MenuOutlined } from '@ant-design/icons';
import { App, Button, Spin, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { APP_VERSION, CONTAINER_ARCH } from '../../constants';
import { type DiskUsage, diskApi } from '../../entity/disk';
import {
  type UserProfile,
  type UsersSettings,
  WorkspaceRole,
  settingsApi,
  userApi,
} from '../../entity/users';
import { type WorkspaceResponse, workspaceApi } from '../../entity/workspaces';
import { DashboardComponent } from '../../features/dashboard';
import { DatabasesComponent } from '../../features/databases/ui/DatabasesComponent';
import { NotifiersComponent } from '../../features/notifiers/ui/NotifiersComponent';
import { SettingsComponent } from '../../features/settings';
import { StoragesComponent } from '../../features/storages/ui/StoragesComponent';
import { ProfileComponent } from '../../features/users';
import { UsersComponent } from '../../features/users/ui/UsersComponent';
import {
  CreateWorkspaceDialogComponent,
  WorkspaceSettingsComponent,
} from '../../features/workspaces';
import { useIsMobile, useIsNewGitHubVersionAvailable, useScreenHeight } from '../../shared/hooks';
import { getWebsitePageUrl, translateApiError, useLocale } from '../../shared/i18n';
import { LanguageThemeControlComponent, SponsorshipLinkComponent } from '../../shared/ui';
import { StarButtonComponent } from '../../shared/ui/StarButtonComponent';
import { type MainTab, SidebarComponent, type SidebarTab } from './SidebarComponent';
import { WorkspaceSelectionComponent } from './WorkspaceSelectionComponent';
import { getDiskUsageValues } from './getDiskUsageValues';

export const MainScreenComponent = () => {
  const { t } = useTranslation();
  const { locale, formatNumber } = useLocale();
  const { message } = App.useApp();
  const screenHeight = useScreenHeight();
  const isMobile = useIsMobile();
  const isNewGitHubVersionAvailable = useIsNewGitHubVersionAvailable();
  const contentHeight = screenHeight - (isMobile ? 70 : 95);

  const [selectedTab, setSelectedTab] = useState<MainTab>('dashboard');
  const [diskUsage, setDiskUsage] = useState<DiskUsage | undefined>(undefined);
  const [user, setUser] = useState<UserProfile | undefined>(undefined);
  const [globalSettings, setGlobalSettings] = useState<UsersSettings | undefined>(undefined);

  const [workspaces, setWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [selectedWorkspace, setSelectedWorkspace] = useState<WorkspaceResponse | undefined>(
    undefined,
  );

  const [isLoading, setIsLoading] = useState(false);
  const [showCreateWorkspaceDialog, setShowCreateWorkspaceDialog] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const [diskUsage, user, workspaces, settings] = await Promise.all([
        diskApi.getDiskUsage(),
        userApi.getCurrentUser(),
        workspaceApi.getWorkspaces(),
        settingsApi.getSettings(),
      ]);

      setDiskUsage(diskUsage);
      setUser(user);
      setWorkspaces(workspaces.workspaces);
      setGlobalSettings(settings);
    } catch (e) {
      message.error(translateApiError(e, t));
    }

    setIsLoading(false);
  };

  useEffect(() => {
    loadData();
  }, []);

  // Set selected workspace if none selected and workspaces available
  useEffect(() => {
    if (!selectedWorkspace && workspaces.length > 0) {
      const previouslySelectedWorkspaceId = localStorage.getItem('selected_workspace_id');
      const previouslySelectedWorkspace = workspaces.find(
        (workspace) => workspace.id === previouslySelectedWorkspaceId,
      );
      const workspaceToSelect = previouslySelectedWorkspace || workspaces[0];
      setSelectedWorkspace(workspaceToSelect);
    }
  }, [workspaces, selectedWorkspace]);

  // Save selected workspace to localStorage
  useEffect(() => {
    if (selectedWorkspace) {
      localStorage.setItem('selected_workspace_id', selectedWorkspace.id);
    }
  }, [selectedWorkspace]);

  const handleCreateWorkspace = () => {
    setShowCreateWorkspaceDialog(true);
  };

  const handleWorkspaceCreated = async (newWorkspace: WorkspaceResponse) => {
    try {
      const workspacesResponse = await workspaceApi.getWorkspaces();
      setWorkspaces(workspacesResponse.workspaces);
      setSelectedWorkspace(newWorkspace);
      setSelectedTab('dashboard');
    } catch (e) {
      message.error(translateApiError(e, t));
    }
  };

  const isUsedMoreThan95Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.95;

  const isUsedMoreThan85Percent =
    diskUsage && diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes > 0.85;

  const isCanManageDBs = selectedWorkspace?.userRole !== WorkspaceRole.VIEWER;

  const tabs: SidebarTab[] = [
    {
      text: t('app.navigation.dashboard'),
      name: 'dashboard',
      icon: '/icons/menu/dashboard-gray.svg',
      selectedIcon: '/icons/menu/dashboard-white.svg',
      onClick: () => setSelectedTab('dashboard'),
      isAdminOnly: false,
      marginTop: '0px',
      isVisible: true,
    },
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
    <div style={{ height: screenHeight }} className="bg-[#f5f5f5] p-2 md:p-3 dark:bg-gray-900">
      <div className="mb-2 flex h-[50px] items-center rounded bg-white px-2 py-2 shadow md:mb-3 md:h-[60px] md:p-3 dark:bg-gray-800">
        <div className="flex items-center gap-2 hover:opacity-80 md:gap-3">
          <a href={getWebsitePageUrl('home', locale)} target="_blank" rel="noreferrer">
            <img className="h-[30px] w-[30px] p-1 md:h-[40px] md:w-[40px]" src="/logo.svg" />
          </a>
        </div>

        <div className="ml-2 flex-1 pr-2 md:ml-4 md:flex-initial md:pr-0">
          {!isLoading && (
            <WorkspaceSelectionComponent
              workspaces={workspaces}
              selectedWorkspace={selectedWorkspace}
              onCreateWorkspace={handleCreateWorkspace}
              onWorkspaceSelect={setSelectedWorkspace}
            />
          )}
        </div>

        <div className="ml-auto hidden items-center gap-5 md:flex">
          <a
            className="!text-black hover:opacity-80 dark:!text-gray-200"
            href={getWebsitePageUrl('installation', locale)}
            target="_blank"
            rel="noreferrer"
          >
            {t('app.navigation.docs')}
          </a>

          <a
            className="!text-black hover:opacity-80 dark:!text-gray-200"
            href="https://t.me/databasus_community"
            target="_blank"
            rel="noreferrer"
          >
            {t('app.navigation.community')}
          </a>

          <SponsorshipLinkComponent className="!text-black hover:opacity-80 dark:!text-gray-200" />

          {isUsedMoreThan85Percent && (
            <Tooltip title={t('app.diskUsage.hint')}>
              <div
                className={`cursor-pointer text-center text-xs ${isUsedMoreThan95Percent ? 'text-red-500' : 'text-gray-500 dark:text-gray-400'}`}
              >
                <Trans
                  i18nKey="app.diskUsage.compactSummary"
                  values={getDiskUsageValues(diskUsage, formatNumber)}
                  components={{ lineBreak: <br /> }}
                />
              </div>
            </Tooltip>
          )}

          <div className="flex items-center gap-2">
            <StarButtonComponent />

            <LanguageThemeControlComponent />
          </div>
        </div>

        <div className="ml-auto flex items-center gap-2 md:hidden">
          <Button
            type="text"
            icon={<MenuOutlined style={{ fontSize: '20px' }} />}
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="mt-1"
          />
        </div>
      </div>

      {isLoading || !user ? (
        <div className="flex items-center justify-center py-2" style={{ height: contentHeight }}>
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div className="relative flex">
          <SidebarComponent
            isOpen={isSidebarOpen}
            onClose={() => setIsSidebarOpen(false)}
            selectedTab={selectedTab}
            tabs={tabs}
            user={user}
            diskUsage={diskUsage}
            contentHeight={contentHeight}
          />

          {selectedTab === 'profile' && (
            <div className="flex-1 md:pl-4">
              <ProfileComponent contentHeight={contentHeight} />
            </div>
          )}

          {selectedTab === 'databasus-settings' && (
            <div className="flex-1 md:pl-4">
              <SettingsComponent contentHeight={contentHeight} />
            </div>
          )}

          {selectedTab === 'users' && (
            <div className="flex-1 md:pl-4">
              <UsersComponent contentHeight={contentHeight} />
            </div>
          )}

          {(selectedTab === 'dashboard' ||
            selectedTab === 'databases' ||
            selectedTab === 'storages' ||
            selectedTab === 'notifiers' ||
            selectedTab === 'settings') && (
            <>
              {workspaces.length === 0 ? (
                <div className="flex-1 md:pl-3">
                  <div
                    className="flex grow items-center justify-center rounded"
                    style={{ height: contentHeight }}
                  >
                    <Button
                      type="primary"
                      size="large"
                      onClick={handleCreateWorkspace}
                      className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                    >
                      {t('app.workspaceSelection.create')}
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="flex-1 md:pl-1">
                    {selectedTab === 'notifiers' && selectedWorkspace && (
                      <NotifiersComponent
                        contentHeight={contentHeight}
                        workspace={selectedWorkspace}
                        isCanManageNotifiers={isCanManageDBs}
                        key={`notifiers-${selectedWorkspace.id}`}
                      />
                    )}
                    {selectedTab === 'storages' && selectedWorkspace && (
                      <StoragesComponent
                        contentHeight={contentHeight}
                        workspace={selectedWorkspace}
                        isCanManageStorages={isCanManageDBs}
                        key={`storages-${selectedWorkspace.id}`}
                      />
                    )}
                    {selectedTab === 'databases' && selectedWorkspace && (
                      <DatabasesComponent
                        contentHeight={contentHeight}
                        workspace={selectedWorkspace}
                        isCanManageDBs={isCanManageDBs}
                        key={`databases-${selectedWorkspace.id}`}
                      />
                    )}

                    <div className="flex-1 md:pl-3">
                      {selectedTab === 'dashboard' && selectedWorkspace && user && (
                        <DashboardComponent
                          workspace={selectedWorkspace}
                          user={user}
                          contentHeight={contentHeight}
                          key={`dashboard-${selectedWorkspace.id}`}
                        />
                      )}
                      {selectedTab === 'settings' && selectedWorkspace && user && (
                        <WorkspaceSettingsComponent
                          workspaceResponse={selectedWorkspace}
                          contentHeight={contentHeight}
                          user={user}
                          key={`settings-${selectedWorkspace.id}`}
                        />
                      )}
                    </div>
                  </div>
                </>
              )}
            </>
          )}

          <div className="absolute bottom-1 left-2 mb-[0px] hidden text-sm text-gray-400 md:block">
            {/* eslint-disable-next-line i18next/no-literal-string -- version prefix */}
            {`v${APP_VERSION}`}
            <br />
            <span className="inline-flex items-center gap-1.5">
              {CONTAINER_ARCH}
              {isNewGitHubVersionAvailable && (
                <Tooltip title={t('app.navigation.newVersionAvailable')}>
                  <a
                    href="https://github.com/databasus/databasus/releases/latest"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex"
                  >
                    <span className="inline-flex h-2 w-2 rounded-full bg-green-500" />
                  </a>
                </Tooltip>
              )}
            </span>
          </div>
        </div>
      )}

      {/* Create Workspace Dialog */}
      {showCreateWorkspaceDialog && user && globalSettings && (
        <CreateWorkspaceDialogComponent
          user={user}
          globalSettings={globalSettings}
          onClose={() => setShowCreateWorkspaceDialog(false)}
          onWorkspaceCreated={handleWorkspaceCreated}
          workspacesCount={workspaces.length}
        />
      )}
    </div>
  );
};

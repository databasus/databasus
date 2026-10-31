import { useEffect, useRef, useState } from 'react';

import { diskApi, type DiskUsage } from '@/entities/disk';
import { settingsApi, userApi, UserRole, type User, type InstanceSettings } from '@/entities/users';
import { workspaceApi, WorkspaceRole, type Workspace } from '@/entities/workspaces';
import { useSessionGuard } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

export function useApplicationStartup() {
  const [user, setUser] = useState<User>();
  const [settings, setSettings] = useState<InstanceSettings>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string>();
  const [diskUsage, setDiskUsage] = useState<DiskUsage>();
  const [error, setError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);

  const load = async () => {
    const requestRevision = revision.current.Invalidate();
    const isCurrent = () => isSessionCurrent() && revision.current.IsCurrent(requestRevision);
    setIsLoading(true);
    setError(undefined);
    void diskApi
      .getDiskUsage()
      .then((disk) => {
        if (isCurrent()) setDiskUsage(disk);
      })
      .catch(() => {});
    try {
      const [currentUser, currentSettings, accessibleWorkspaces] = await Promise.all([
        userApi.getCurrentUser(),
        settingsApi.getSettings(),
        workspaceApi.listAllWorkspaces(),
      ]);
      if (!isCurrent()) return;
      setUser(currentUser);
      setSettings(currentSettings);
      setWorkspaces(accessibleWorkspaces);
      const remembered = localStorage.getItem('selected_workspace_id');
      setSelectedWorkspaceId(
        (current) =>
          accessibleWorkspaces.find((workspace) => workspace.id === (current ?? remembered))?.id ??
          accessibleWorkspaces[0]?.id,
      );
    } catch (failure) {
      if (isCurrent()) setError(failure);
    } finally {
      if (isCurrent()) setIsLoading(false);
    }
  };

  const acceptUser = (changedUser: User) => {
    if (isSessionCurrent()) setUser(changedUser);
  };
  const acceptSettings = (changedSettings: InstanceSettings) => {
    if (isSessionCurrent()) setSettings(changedSettings);
  };
  const acceptWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;
    setWorkspaces((current) => [...current.filter((item) => item.id !== workspace.id), workspace]);
    setSelectedWorkspaceId(workspace.id);
  };
  const isSessionCurrent = useSessionGuard();
  const revision = useRef(new RequestRevision());

  useEffect(() => {
    void load();
    return () => {
      revision.current.Invalidate();
    };
  }, []);
  useEffect(() => {
    if (selectedWorkspaceId) localStorage.setItem('selected_workspace_id', selectedWorkspaceId);
  }, [selectedWorkspaceId]);

  const selectedWorkspace = workspaces.find((workspace) => workspace.id === selectedWorkspaceId);
  const isReady = !isLoading && error === undefined && !!user && !!settings;
  const isAdmin = isReady && user.role === UserRole.ADMIN;
  const canCreateWorkspace = isReady && (isAdmin || settings.isMemberAllowedToCreateWorkspaces);
  const canManageWorkspace =
    isReady &&
    !!selectedWorkspace &&
    (isAdmin ||
      (!!selectedWorkspace.currentUserRole &&
        selectedWorkspace.currentUserRole !== WorkspaceRole.VIEWER));
  return {
    user,
    settings,
    workspaces,
    selectedWorkspace,
    diskUsage,
    error,
    isLoading,
    isReady,
    isAdmin,
    canCreateWorkspace,
    canManageWorkspace,
    retry: load,
    acceptUser,
    acceptSettings,
    acceptWorkspace,
    selectWorkspace: (workspace: Workspace) => setSelectedWorkspaceId(workspace.id),
  };
}

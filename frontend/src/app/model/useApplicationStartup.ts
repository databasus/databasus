import { useEffect, useEffectEvent, useState } from 'react';

import { diskApi, type DiskUsage } from '@/entities/disk';
import { settingsApi, userApi, UserRole, type User, type InstanceSettings } from '@/entities/users';
import { workspaceApi, WorkspaceRole, type Workspace } from '@/entities/workspaces';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';

export function useApplicationStartup() {
  const [user, setUser] = useState<User>();
  const [settings, setSettings] = useState<InstanceSettings>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string>();
  const [diskUsage, setDiskUsage] = useState<DiskUsage>();
  const [error, setError] = useState<unknown>();
  const [isLoading, setIsLoading] = useState(true);

  const load = async () => {
    const requestRevision = revision.Invalidate();
    accessRevision.Invalidate();
    const isCurrent = () => isSessionCurrent() && revision.IsCurrent(requestRevision);
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
      let remembered: string | null = null;
      try {
        remembered = localStorage.getItem('selected_workspace_id');
      } catch {}
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

    accessRevision.Invalidate();
    setWorkspaces((current) => [...current.filter((item) => item.id !== workspace.id), workspace]);
    setSelectedWorkspaceId(workspace.id);
  };
  const updateWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;

    accessRevision.Invalidate();
    setWorkspaces((current) =>
      current.map((item) => (item.id === workspace.id ? workspace : item)),
    );
  };
  const removeWorkspace = (workspaceId: string) => {
    if (!isSessionCurrent()) return;

    accessRevision.Invalidate();
    setWorkspaces((current) => current.filter((workspace) => workspace.id !== workspaceId));
    setSelectedWorkspaceId((current) =>
      current === workspaceId
        ? workspaces.find((workspace) => workspace.id !== workspaceId)?.id
        : current,
    );
  };
  const selectWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;

    accessRevision.Invalidate();
    setSelectedWorkspaceId(workspace.id);
  };
  const reconcileUnavailableWorkspace = async (workspaceId: string) => {
    if (!isSessionCurrent()) return;

    removeWorkspace(workspaceId);
    const requestRevision = accessRevision.Invalidate();

    try {
      const accessible = await workspaceApi.listAllWorkspaces();
      if (!isSessionCurrent() || !accessRevision.IsCurrent(requestRevision)) return;

      setWorkspaces(accessible);
      setSelectedWorkspaceId((current) =>
        accessible.some((workspace) => workspace.id === current) ? current : accessible[0]?.id,
      );
    } catch (failure) {
      if (isSessionCurrent() && accessRevision.IsCurrent(requestRevision)) setError(failure);
    }
  };
  const isSessionCurrent = useSessionGuard();
  const revision = useRequestRevision();
  const accessRevision = useRequestRevision();

  const startApplicationLoad = useEffectEvent(() => {
    void load();
  });

  useEffect(() => {
    startApplicationLoad();

    return () => {
      revision.Invalidate();
      accessRevision.Invalidate();
    };
  }, [revision, accessRevision]);
  useEffect(() => {
    if (isLoading || error !== undefined) return;

    try {
      if (selectedWorkspaceId) localStorage.setItem('selected_workspace_id', selectedWorkspaceId);
      else localStorage.removeItem('selected_workspace_id');
    } catch {}
  }, [selectedWorkspaceId, isLoading, error]);

  const selectedWorkspace =
    workspaces.find((workspace) => workspace.id === selectedWorkspaceId) ?? workspaces[0];
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
    updateWorkspace,
    removeWorkspace,
    reconcileUnavailableWorkspace,
    selectWorkspace,
  };
}

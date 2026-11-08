import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useEffectEvent, useRef, useState } from 'react';

import { diskApi } from '@/entities/disk';
import { settingsApi, userApi, UserRole, type User, type InstanceSettings } from '@/entities/users';
import { workspaceApi, WorkspaceRole, type Workspace } from '@/entities/workspaces';
import { useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';
import { useNavigationIntent } from '@/shared/navigation';

export function useApplicationStartup(requestedWorkspaceId?: string) {
  const client = useQueryClient();
  const navigationIntent = useNavigationIntent();
  const isSessionCurrent = useSessionGuard();
  const accessRevision = useRequestRevision();
  const rejectedWorkspaceIds = useRef(new Set<string>());
  const [user, setUser] = useState<User>();
  const [settings, setSettings] = useState<InstanceSettings>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string>();
  const [startupReadError, setStartupReadError] = useState<unknown>();
  const query = useSessionQuery({
    queryKey: ['application-startup'],
    queryFn: async (signal) => {
      setStartupReadError(undefined);

      const reportFailure = (error: unknown): never => {
        if (!signal.aborted && isSessionCurrent())
          setStartupReadError((current: unknown) => current ?? error);

        throw error;
      };
      const reads = [
        userApi.getCurrentUser({ signal }).catch(reportFailure),
        settingsApi.getSettings({ signal }).catch(reportFailure),
        workspaceApi.listAllWorkspaces({ signal }).catch(reportFailure),
      ] as const;

      try {
        const [user, settings, workspaces] = await Promise.all(reads);

        return { user, settings, workspaces };
      } finally {
        // A failed essential read must not release still-running sibling requests.
        await Promise.allSettled(reads);
      }
    },
  });
  const disk = useSessionQuery({
    queryKey: ['disk-usage'],
    queryFn: (signal) => diskApi.getDiskUsage({ signal }),
  });
  const access = useSessionQuery({
    queryKey: ['workspace-access'],
    queryFn: (signal) => workspaceApi.listAllWorkspaces({ signal }),
    enabled: false,
  });
  const diskUsage = disk.data;
  const [accessError, setError] = useState<unknown>();
  const [reconcilingWorkspaceId, setReconcilingWorkspaceId] = useState<string>();
  const isReconcilingWorkspace = reconcilingWorkspaceId !== undefined && accessError === undefined;
  const error = accessError ?? startupReadError ?? query.error ?? undefined;
  const isLoading =
    startupReadError === undefined && (query.isPending || (query.isFetching && query.isError));

  const discardWorkspaceRead = () => {
    accessRevision.Invalidate();
    void client.cancelQueries({ queryKey: access.queryKey, exact: true });
  };

  const load = () => {
    rejectedWorkspaceIds.current.clear();

    if (reconcilingWorkspaceId) return refreshWorkspaceAccess();

    discardWorkspaceRead();
    setError(undefined);
    void disk.refetch();

    return query.refetch();
  };
  const discardNavigatedWorkspaceRead = useEffectEvent(() => {
    rejectedWorkspaceIds.current.clear();
    discardWorkspaceRead();
    setReconcilingWorkspaceId(undefined);
    setError(undefined);
  });
  useEffect(() => {
    discardNavigatedWorkspaceRead();
  }, [navigationIntent]);
  useEffect(() => {
    if (!query.data) return;

    const { user, settings, workspaces } = query.data;
    setUser(user);
    setSettings(settings);
    setWorkspaces(workspaces);

    let remembered: string | null = null;
    try {
      remembered = localStorage.getItem('selected_workspace_id');
    } catch {}

    setSelectedWorkspaceId(
      (current) =>
        workspaces.find((workspace) => workspace.id === (current ?? remembered))?.id ??
        workspaces[0]?.id,
    );
  }, [query.data, query.dataRevision]);

  const acceptUser = (changedUser: User) => {
    if (isSessionCurrent()) setUser(changedUser);
  };
  const acceptSettings = (changedSettings: InstanceSettings) => {
    if (isSessionCurrent()) setSettings(changedSettings);
  };
  const acceptWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;

    discardWorkspaceRead();
    setWorkspaces((current) => [...current.filter((item) => item.id !== workspace.id), workspace]);
    setSelectedWorkspaceId(workspace.id);
  };
  const updateWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;

    discardWorkspaceRead();
    setWorkspaces((current) =>
      current.map((item) => (item.id === workspace.id ? workspace : item)),
    );
  };
  const removeWorkspace = (workspaceId: string) => {
    if (!isSessionCurrent()) return;

    discardWorkspaceRead();
    setWorkspaces((current) => current.filter((workspace) => workspace.id !== workspaceId));
    setSelectedWorkspaceId((current) =>
      current === workspaceId
        ? workspaces.find((workspace) => workspace.id !== workspaceId)?.id
        : current,
    );
  };
  const selectWorkspace = (workspace: Workspace) => {
    if (!isSessionCurrent()) return;

    discardWorkspaceRead();
    setSelectedWorkspaceId(workspace.id);
  };

  useEffect(() => {
    if (
      !requestedWorkspaceId ||
      !workspaces.some((workspace) => workspace.id === requestedWorkspaceId)
    )
      return;

    setSelectedWorkspaceId(requestedWorkspaceId);
  }, [requestedWorkspaceId, workspaces]);
  const refreshWorkspaceAccess = async () => {
    if (!isSessionCurrent()) return;

    setError(undefined);
    const requestRevision = accessRevision.Invalidate();

    try {
      const { data: accessible, error } = await access.refetch();
      if (error) throw error;

      if (!accessible) return;

      if (!isSessionCurrent() || !accessRevision.IsCurrent(requestRevision)) return;

      const availableWorkspaces = accessible.filter(
        (workspace) => !rejectedWorkspaceIds.current.has(workspace.id),
      );
      setWorkspaces(availableWorkspaces);
      setSelectedWorkspaceId((current) =>
        availableWorkspaces.some((workspace) => workspace.id === current)
          ? current
          : availableWorkspaces[0]?.id,
      );
      setReconcilingWorkspaceId(undefined);
    } catch (failure) {
      if (isSessionCurrent() && accessRevision.IsCurrent(requestRevision)) setError(failure);
    }
  };
  const reconcileUnavailableWorkspace = (workspaceId: string) => {
    if (!isSessionCurrent()) return;

    rejectedWorkspaceIds.current.add(workspaceId);
    removeWorkspace(workspaceId);
    setReconcilingWorkspaceId(workspaceId);

    return refreshWorkspaceAccess();
  };

  useEffect(() => {
    if (isLoading || error !== undefined) return;

    try {
      if (selectedWorkspaceId) localStorage.setItem('selected_workspace_id', selectedWorkspaceId);
      else localStorage.removeItem('selected_workspace_id');
    } catch {}
  }, [selectedWorkspaceId, isLoading, error]);

  const selectedWorkspace =
    workspaces.find(
      (workspace) => workspace.id === (requestedWorkspaceId ?? selectedWorkspaceId),
    ) ?? workspaces[0];
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
    isReconcilingWorkspace,
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

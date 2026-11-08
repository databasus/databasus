import { Alert, Button, CloseButton, Drawer, Loader, Switch, TextInput } from '@mantine/core';
import dayjs from 'dayjs';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { USER_ROLE_LABEL_KEYS, type User, UserRole, userManagementApi } from '@/entities/users';
import { UserAuditLogsSidebar } from '@/pages/users/ui/UserAuditLogsSidebar';
import { useSessionMutation } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useIsMobile, usePagedList, useRequestRevision } from '@/shared/hooks';
import { translateApiError, useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { getUserTimeFormat } from '@/shared/time';
import { closeOverlayOnEscape } from '@/shared/ui';
import { OverlayFocusReturn } from '@/shared/ui';
import { WrappingSelect } from '@/shared/ui';
import { LoadFailure } from '@/shared/ui';
import tableStyles from '@/shared/ui/nativeTable.module.css';

const isUserActive = (user: User): boolean => user.status === 'ACTIVE';

const withUserId = (userIds: Set<string>, userId: string): Set<string> =>
  new Set(userIds).add(userId);

const withoutUserId = (userIds: Set<string>, userId: string): Set<string> => {
  const remainingUserIds = new Set(userIds);
  remainingUserIds.delete(userId);

  return remainingUserIds;
};

export function UsersPage() {
  const deactivateUserMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userManagementApi.deactivateUser>) =>
      userManagementApi.deactivateUser(...parameters),
  });
  const activateUserMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userManagementApi.activateUser>) =>
      userManagementApi.activateUser(...parameters),
  });
  const changeUserRoleMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof userManagementApi.changeUserRole>) =>
      userManagementApi.changeUserRole(...parameters),
  });

  const [searchQuery, setSearchQuery] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [processingUsers, setProcessingUsers] = useState<Set<string>>(new Set());
  const [changingRoleUsers, setChangingRoleUsers] = useState<Set<string>>(new Set());
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [failure, setFailure] = useState<unknown>();
  const [listFailure, setListFailure] = useState<unknown>();

  const retryUsers = () => {
    if (isLoading) return;
    queryRevision.Invalidate();
    acknowledgedUsers.clear();
    setFailure(undefined);
    void reload();
  };

  const handleActivationToggle = async (toggledUser: User) => {
    if (actionUsers.current.has(toggledUser.id)) return;
    actionUsers.current.add(toggledUser.id);
    const requestQueryRevision = queryRevision.GetCurrent();
    setProcessingUsers((current) => withUserId(current, toggledUser.id));
    setFailure(undefined);

    try {
      const changedUser = isUserActive(toggledUser)
        ? await deactivateUserMutation.mutateAsync([toggledUser.id])
        : await activateUserMutation.mutateAsync([toggledUser.id]);
      if (!isCurrent() || !queryRevision.IsCurrent(requestQueryRevision)) return;
      acknowledgedUsers.set(changedUser.id, changedUser);
      replaceOrPrependItem(changedUser);
      notify({
        kind: 'success',
        title: i18n.t(
          isUserActive(changedUser) ? 'users.list.activated' : 'users.list.deactivated',
        ),
      });
    } catch (error) {
      if (isCurrent() && queryRevision.IsCurrent(requestQueryRevision)) setFailure(error);
    } finally {
      actionUsers.current.delete(toggledUser.id);
      if (isCurrent()) setProcessingUsers((current) => withoutUserId(current, toggledUser.id));
    }
  };

  const handleRoleChange = async (userId: string, newRole: UserRole) => {
    if (actionUsers.current.has(userId)) return;
    actionUsers.current.add(userId);
    const requestQueryRevision = queryRevision.GetCurrent();
    setChangingRoleUsers((current) => withUserId(current, userId));
    setFailure(undefined);

    try {
      const changedUser = await changeUserRoleMutation.mutateAsync([userId, newRole]);
      if (!isCurrent() || !queryRevision.IsCurrent(requestQueryRevision)) return;
      acknowledgedUsers.set(changedUser.id, changedUser);
      replaceOrPrependItem(changedUser);
      notify({ kind: 'success', title: i18n.t('users.list.roleChanged') });
    } catch (error) {
      if (isCurrent() && queryRevision.IsCurrent(requestQueryRevision)) setFailure(error);
    } finally {
      actionUsers.current.delete(userId);
      if (isCurrent()) setChangingRoleUsers((current) => withoutUserId(current, userId));
    }
  };
  const handleRowClick = (user: User, invoker: HTMLElement) => {
    drawerInvoker.current = invoker;
    setSelectedUser(user);
    setIsDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setIsDrawerOpen(false);
    setSelectedUser(null);
    queueMicrotask(() => {
      if (isCurrent() && drawerInvoker.current?.isConnected)
        drawerInvoker.current.focus({ preventScroll: true });
    });
  };

  const renderUserCard = (user: User) => {
    const date = dayjs(user.createdAt);
    const timeFormat = getUserTimeFormat();

    return (
      <div key={user.id} className="mb-3 rounded-lg border border-border bg-surface p-4 shadow-sm">
        <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <div className="font-medium text-content">{user.name}</div>
            <div className="text-sm text-muted">{user.email}</div>
          </div>
          <div className="text-right text-xs text-muted">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-muted">{formatRelativeTime(date)}</div>
          </div>
        </div>

        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted">{t('users.list.roleLabel')}</span>
            <WrappingSelect
              aria-label={`${t('users.list.columns.systemRole')} (${user.email})`}
              value={user.role}
              onChange={(value) => handleRoleChange(user.id, value)}
              isLoading={changingRoleUsers.has(user.id)}
              disabled={processingUsers.has(user.id) || changingRoleUsers.has(user.id)}
              size="xs"
              className="w-36"
              data={roleOptions}
            />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted">{t('users.list.activeLabel')}</span>
            <Switch
              aria-label={`${t('users.list.columns.isActive')} (${user.email})`}
              checked={isUserActive(user)}
              onChange={() => handleActivationToggle(user)}
              aria-busy={processingUsers.has(user.id)}
              disabled={processingUsers.has(user.id) || changingRoleUsers.has(user.id)}
              size="xs"
            />
          </div>
        </div>

        <Button
          variant="outline"
          size="xs"
          onClick={(event) => handleRowClick(user, event.currentTarget)}
          className="!h-auto min-h-8 w-full max-w-full min-w-0 !whitespace-normal"
        >
          {t('users.list.viewAuditLogs')}
        </Button>
      </div>
    );
  };

  const { t, i18n } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const notify = useFeedback();
  const isMobile = useIsMobile();
  const isCurrent = useSessionGuard();
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const actionUsers = useRef(new Set<string>());
  const drawerInvoker = useRef<HTMLElement | null>(null);
  const queryRevision = useRequestRevision();
  const [acknowledgedUsers] = useState(() => new Map<string, User>());

  // A layout effect runs before any late answer can arrive, so an answer of the old query cannot
  // pass its revision check.
  useLayoutEffect(() => {
    queryRevision.Invalidate();
    acknowledgedUsers.clear();
  }, [searchQuery, queryRevision, acknowledgedUsers]);

  const {
    items: users,
    isLoading,
    isLoadingMore,
    hasMore,
    reload,
    replaceOrPrependItem,
  } = usePagedList<User, string>({
    fetchPage: async (after) => {
      if (isCurrent()) setListFailure(undefined);
      const page = await userManagementApi.getUsers({
        limit: 20,
        after,
        query: searchQuery || undefined,
      });
      const pageUsers = new Map(
        page.items.map((user) => [user.id, acknowledgedUsers.get(user.id) ?? user]),
      );
      return { ...page, items: [...pageUsers.values()] };
    },
    getItemKey: (user) => user.id,
    onError: (error) => {
      if (isCurrent()) setListFailure(error);
    },
    scrollContainerRef,
    resetKey: searchQuery,
  });

  useEffect(() => {
    if (inputValue === searchQuery) return;
    const timer = setTimeout(() => {
      setSearchQuery(inputValue);
      setFailure(undefined);
    }, 500);
    return () => clearTimeout(timer);
  }, [inputValue, searchQuery]);

  const roleOptions = Object.values(UserRole).map((role) => ({
    label: t(USER_ROLE_LABEL_KEYS[role]),
    value: role,
  }));

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <div className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div
          ref={scrollContainerRef}
          className="min-h-0 min-w-0 flex-1 overflow-y-auto rounded bg-surface p-4 wrap-anywhere shadow sm:p-5"
        >
          <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
            <h1 className="text-2xl font-bold">{t('users.list.title')}</h1>
            <div className="text-sm text-muted">
              {isLoading
                ? t('common.states.loading')
                : listFailure === undefined && t('users.list.loaded', { count: users.length })}
            </div>
          </div>

          <div className="mb-4">
            <TextInput
              aria-label={t('users.list.searchPlaceholder')}
              placeholder={t('users.list.searchPlaceholder')}
              rightSection={
                inputValue ? (
                  <CloseButton
                    aria-label={t('common.actions.clear')}
                    onClick={() => setInputValue('')}
                  />
                ) : null
              }
              rightSectionPointerEvents="all"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              style={{ width: isMobile ? '100%' : 400 }}
            />
          </div>

          {failure !== undefined && <Alert color="danger" title={translateApiError(failure, t)} />}
          {listFailure !== undefined && <LoadFailure error={listFailure} onRetry={retryUsers} />}
          {isLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Loader size="md" role="status" />
            </div>
          ) : users.length === 0 ? (
            listFailure === undefined && (
              <div className="flex h-32 items-center justify-center text-muted">
                {t('users.list.empty')}
              </div>
            )
          ) : (
            <>
              {isMobile ? (
                <div>{users.map(renderUserCard)}</div>
              ) : (
                <div className="relative mb-4 overflow-x-auto">
                  <table className={tableStyles.table}>
                    <thead>
                      <tr>
                        <th scope="col" style={{ width: 350 }}>
                          {t('users.list.columns.user')}
                        </th>
                        <th scope="col" style={{ width: 200 }}>
                          {t('users.list.columns.systemRole')}
                        </th>
                        <th scope="col" style={{ width: 200 }}>
                          {t('users.list.columns.isActive')}
                        </th>
                        <th scope="col" style={{ width: 300 }}>
                          {t('users.list.columns.created')}
                        </th>
                        <th scope="col">{''}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {users.map((user) => (
                        <tr key={user.id}>
                          <td>
                            <div>
                              {user.name} ({user.email})
                            </div>
                          </td>
                          <td>
                            <WrappingSelect
                              aria-label={`${t('users.list.columns.systemRole')} (${user.email})`}
                              value={user.role}
                              onChange={(value) => handleRoleChange(user.id, value)}
                              isLoading={changingRoleUsers.has(user.id)}
                              disabled={
                                processingUsers.has(user.id) || changingRoleUsers.has(user.id)
                              }
                              size="xs"
                              className="w-36"
                              data={roleOptions}
                            />
                          </td>
                          <td>
                            <Switch
                              aria-label={`${t('users.list.columns.isActive')} (${user.email})`}
                              checked={isUserActive(user)}
                              onChange={() => handleActivationToggle(user)}
                              aria-busy={processingUsers.has(user.id)}
                              disabled={
                                processingUsers.has(user.id) || changingRoleUsers.has(user.id)
                              }
                              size="xs"
                            />
                          </td>
                          <td>
                            {(() => {
                              const date = dayjs(user.createdAt);
                              const timeFormat = getUserTimeFormat();
                              return (
                                <div className="text-sm text-muted">
                                  <div>{date.format(timeFormat.format)}</div>
                                  <div className="text-xs text-muted">
                                    {formatRelativeTime(date)}
                                  </div>
                                </div>
                              );
                            })()}
                          </td>
                          <td>
                            <div>
                              <Button
                                variant="outline"
                                size="xs"
                                className="min-w-max"
                                onClick={(event) => handleRowClick(user, event.currentTarget)}
                              >
                                {t('users.list.viewAuditLogs')}
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {isLoadingMore && (
                <div className="flex justify-center py-4">
                  <Loader role="status" />
                </div>
              )}

              {!hasMore && users.length > 0 && (
                <div className="py-4 text-center text-sm text-muted">
                  {t('users.list.allLoaded', { count: users.length })}
                </div>
              )}
            </>
          )}
        </div>
      </div>
      <Drawer
        closeOnEscape={false}
        onKeyDown={(event) =>
          closeOverlayOnEscape(event, { isOpen: isDrawerOpen, onClose: handleDrawerClose })
        }
        title={
          <div>
            <div className="text-lg font-semibold text-content">{t('users.auditLogs.title')}</div>
            <div className="text-sm text-muted">{selectedUser?.email}</div>
          </div>
        }
        position="right"
        size={isMobile ? '100%' : 900}
        onClose={handleDrawerClose}
        opened={isDrawerOpen}
        closeButtonProps={{ 'aria-label': t('common.actions.close') }}
      >
        {isDrawerOpen && <OverlayFocusReturn />}
        {selectedUser && <UserAuditLogsSidebar key={selectedUser.id} user={selectedUser} />}
      </Drawer>
    </div>
  );
}

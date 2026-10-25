import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Drawer, Input, Select, Spin, Switch, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  USER_ROLE_LABEL_KEYS,
  type User,
  UserRole,
  userManagementApi,
} from '../../../entity/users';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { getUserTimeFormat } from '../../../shared/time';
import { UserAuditLogsSidebarComponent } from './UserAuditLogsSidebarComponent';

interface Props {
  contentHeight: number;
}

const isUserActive = (user: User): boolean => user.status === 'ACTIVE';

const withUserId = (userIds: Set<string>, userId: string): Set<string> =>
  new Set(userIds).add(userId);

const withoutUserId = (userIds: Set<string>, userId: string): Set<string> => {
  const remainingUserIds = new Set(userIds);
  remainingUserIds.delete(userId);

  return remainingUserIds;
};

const getRoleColor = (role: UserRole): string => {
  switch (role) {
    case UserRole.ADMIN:
      return '#3b82f6';
    case UserRole.MEMBER:
      return '#10b981';
    default:
      return '#6b7280';
  }
};

export function UsersComponent({ contentHeight }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<string | undefined>(undefined);
  const [hasMore, setHasMore] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [inputValue, setInputValue] = useState('');

  const pageSize = 20;

  const [processingUsers, setProcessingUsers] = useState<Set<string>>(new Set());
  const [changingRoleUsers, setChangingRoleUsers] = useState<Set<string>>(new Set());

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  useEffect(() => {
    loadUsers(true);
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (inputValue !== searchQuery) {
        setSearchQuery(inputValue);
        setHasMore(true);
        loadUsers(true, inputValue);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [inputValue]);

  const handleScroll = useCallback(() => {
    if (!scrollContainerRef.current || isLoadingMore || !hasMore || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
    const threshold = 100;

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadUsers(false);
    }
  }, [isLoadingMore, hasMore, nextCursor, searchQuery]);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll);
      return () => scrollContainer.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

  const loadUsers = async (isInitialLoad = false, query?: string) => {
    if (!isInitialLoad && loadingRef.current) {
      return;
    }

    loadingRef.current = true;

    if (isInitialLoad) {
      setIsLoading(true);
      setUsers([]);
    } else {
      setIsLoadingMore(true);
    }

    try {
      const currentQuery = query !== undefined ? query : searchQuery;
      const response = await userManagementApi.getUsers({
        limit: pageSize,
        after: isInitialLoad ? undefined : nextCursor,
        query: currentQuery || undefined,
      });

      if (isInitialLoad) {
        setUsers(response.items);
      } else {
        setUsers((prev) => {
          const existingIds = new Set(prev.map((user) => user.id));
          const newUsers = response.items.filter((user) => !existingIds.has(user.id));
          return [...prev, ...newUsers];
        });
      }

      setNextCursor(response.nextCursor);
      setHasMore(response.nextCursor !== undefined);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  };

  const replaceUser = (changedUser: User) => {
    setUsers((prev) => prev.map((user) => (user.id === changedUser.id ? changedUser : user)));
  };

  const handleActivationToggle = async (toggledUser: User) => {
    const userId = toggledUser.id;
    const isActive = isUserActive(toggledUser);

    replaceUser({ ...toggledUser, status: isActive ? 'INACTIVE' : 'ACTIVE' });
    setProcessingUsers((prev) => withUserId(prev, userId));

    try {
      if (isActive) {
        replaceUser(await userManagementApi.deactivateUser(userId));
        message.success(t('users.list.deactivated'));
      } else {
        replaceUser(await userManagementApi.activateUser(userId));
        message.success(t('users.list.activated'));
      }
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
      replaceUser(toggledUser);
    } finally {
      setProcessingUsers((prev) => withoutUserId(prev, userId));
    }
  };

  const handleRoleChange = async (userId: string, newRole: UserRole) => {
    const currentUser = users.find((user) => user.id === userId);
    const originalRole = currentUser?.role;

    setUsers((prev) =>
      prev.map((user) => (user.id === userId ? { ...user, role: newRole } : user)),
    );

    setChangingRoleUsers((prev) => withUserId(prev, userId));

    try {
      await userManagementApi.changeUserRole(userId, newRole);
      message.success(t('users.list.roleChanged'));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));

      if (originalRole) {
        setUsers((prev) =>
          prev.map((user) => (user.id === userId ? { ...user, role: originalRole } : user)),
        );
      }
    } finally {
      setChangingRoleUsers((prev) => withoutUserId(prev, userId));
    }
  };

  const handleRowClick = (user: User) => {
    setSelectedUser(user);
    setIsDrawerOpen(true);
  };

  const handleDrawerClose = () => {
    setIsDrawerOpen(false);
    setSelectedUser(null);
  };

  const roleOptions = Object.values(UserRole).map((role) => ({
    label: <span style={{ color: getRoleColor(role) }}>{t(USER_ROLE_LABEL_KEYS[role])}</span>,
    value: role,
  }));

  const columns: ColumnsType<User> = [
    {
      title: t('users.list.columns.user'),
      key: 'user',
      width: 350,
      render: (_, record: User) => (
        <div>
          {record.name} ({record.email})
        </div>
      ),
    },
    {
      title: t('users.list.columns.systemRole'),
      dataIndex: 'role',
      key: 'role',
      width: 200,
      render: (role: UserRole, record: User) => (
        <Select
          value={role}
          onChange={(value) => handleRoleChange(record.id, value)}
          loading={changingRoleUsers.has(record.id)}
          disabled={changingRoleUsers.has(record.id)}
          size="small"
          className="w-36"
          style={{
            color: getRoleColor(role),
          }}
          options={roleOptions}
        />
      ),
    },
    {
      title: t('users.list.columns.isActive'),
      key: 'isActive',
      width: 200,
      render: (_, record: User) => (
        <Switch
          checked={isUserActive(record)}
          onChange={() => handleActivationToggle(record)}
          loading={processingUsers.has(record.id)}
          disabled={processingUsers.has(record.id)}
          size="small"
          style={{
            backgroundColor: isUserActive(record) ? '#155dfc' : undefined,
          }}
        />
      ),
    },
    {
      title: t('users.list.columns.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 300,
      render: (createdAt: string) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserTimeFormat();
        return (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-xs text-gray-400 dark:text-gray-500">
              {formatRelativeTime(date)}
            </div>
          </div>
        );
      },
    },
    {
      title: '',
      key: 'empty',
      render: (_, record: User) => (
        <div>
          <Button type="primary" ghost size="small" onClick={() => handleRowClick(record)}>
            {t('users.list.viewAuditLogs')}
          </Button>
        </div>
      ),
    },
  ];

  const renderUserCard = (user: User) => {
    const date = dayjs(user.createdAt);
    const timeFormat = getUserTimeFormat();

    return (
      <div
        key={user.id}
        className="mb-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
      >
        <div className="mb-3 flex items-start justify-between">
          <div className="flex-1">
            <div className="font-medium text-gray-900 dark:text-white">{user.name}</div>
            <div className="text-sm text-gray-500 dark:text-gray-400">{user.email}</div>
          </div>
          <div className="text-right text-xs text-gray-500 dark:text-gray-400">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-gray-400">{formatRelativeTime(date)}</div>
          </div>
        </div>

        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {t('users.list.roleLabel')}
            </span>
            <Select
              value={user.role}
              onChange={(value) => handleRoleChange(user.id, value)}
              loading={changingRoleUsers.has(user.id)}
              disabled={changingRoleUsers.has(user.id)}
              size="small"
              className="w-36"
              style={{
                color: getRoleColor(user.role),
              }}
              options={roleOptions}
            />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {t('users.list.activeLabel')}
            </span>
            <Switch
              checked={isUserActive(user)}
              onChange={() => handleActivationToggle(user)}
              loading={processingUsers.has(user.id)}
              disabled={processingUsers.has(user.id)}
              size="small"
              style={{
                backgroundColor: isUserActive(user) ? '#155dfc' : undefined,
              }}
            />
          </div>
        </div>

        <Button
          type="primary"
          ghost
          size="small"
          onClick={() => handleRowClick(user)}
          className="w-full"
        >
          {t('users.list.viewAuditLogs')}
        </Button>
      </div>
    );
  };

  return (
    <div className="flex grow">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <div className="mb-4 flex items-center justify-between">
            <h1 className="text-2xl font-bold dark:text-white">{t('users.list.title')}</h1>
            <div className="text-sm text-gray-500 dark:text-gray-400">
              {isLoading
                ? t('common.states.loading')
                : t('users.list.loaded', { count: users.length })}
            </div>
          </div>

          <div className="mb-4">
            <Input
              placeholder={t('users.list.searchPlaceholder')}
              allowClear
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              style={{ width: isMobile ? '100%' : 400 }}
            />
          </div>

          {isLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Spin indicator={<LoadingOutlined spin />} size="large" />
            </div>
          ) : users.length === 0 ? (
            <div className="flex h-32 items-center justify-center text-gray-500 dark:text-gray-400">
              {t('users.list.empty')}
            </div>
          ) : (
            <>
              {isMobile ? (
                <div>{users.map(renderUserCard)}</div>
              ) : (
                <Table
                  columns={columns}
                  dataSource={users}
                  pagination={false}
                  rowKey="id"
                  size="small"
                  className="mb-4"
                />
              )}

              {isLoadingMore && (
                <div className="flex justify-center py-4">
                  <Spin indicator={<LoadingOutlined spin />} />
                </div>
              )}

              {!hasMore && users.length > 0 && (
                <div className="py-4 text-center text-sm text-gray-500 dark:text-gray-400">
                  {t('users.list.allLoaded', { count: users.length })}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Audit logs drawer */}
      <Drawer
        title={
          <div>
            <div className="text-lg font-semibold text-gray-900 dark:text-white">
              {t('users.auditLogs.title')}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">{selectedUser?.email}</div>
          </div>
        }
        placement="right"
        width={isMobile ? '100%' : 900}
        onClose={handleDrawerClose}
        open={isDrawerOpen}
      >
        {selectedUser && <UserAuditLogsSidebarComponent user={selectedUser} />}
      </Drawer>
    </div>
  );
}

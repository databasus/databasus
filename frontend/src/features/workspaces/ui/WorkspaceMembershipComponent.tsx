import {
  DeleteOutlined,
  LoadingOutlined,
  PlusOutlined,
  SwapOutlined,
  UserAddOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  App,
  AutoComplete,
  Button,
  Input,
  Modal,
  Popconfirm,
  Select,
  Spin,
  Table,
  Tag,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type User, WORKSPACE_ROLE_LABEL_KEYS } from '../../../entity/users';
import { userManagementApi } from '../../../entity/users';
import { UserRole } from '../../../entity/users';
import { WorkspaceRole } from '../../../entity/users';
import type {
  AddMemberRequest,
  AddMemberResponse,
  ChangeMemberRoleRequest,
  GetMembersResponse,
  TransferOwnershipRequest,
  WorkspaceMemberResponse,
  WorkspaceResponse,
} from '../../../entity/workspaces';
import { AddMemberStatusEnum, workspaceMembershipApi } from '../../../entity/workspaces';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { getUserShortTimeFormat } from '../../../shared/time';

interface Props {
  workspaceResponse: WorkspaceResponse;
  user: User;
}

type RoleTagColor = 'purple' | 'orange' | 'blue' | 'green' | 'default';

export function WorkspaceMembershipComponent({ workspaceResponse, user }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const isMobile = useIsMobile();

  const [members, setMembers] = useState<WorkspaceMemberResponse[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(true);

  const [isAddMemberModalOpen, setIsAddMemberModalOpen] = useState(false);
  const [addMemberForm, setAddMemberForm] = useState({ email: '', role: WorkspaceRole.MEMBER });
  const [isAddingMember, setIsAddingMember] = useState(false);
  const [addMemberEmailError, setAddMemberEmailError] = useState(false);

  const [isInviteDialogOpen, setIsInviteDialogOpen] = useState(false);
  const [invitedEmail, setInvitedEmail] = useState('');

  const [changingRoleFor, setChangingRoleFor] = useState<string | null>(null);
  const [isChangingRole, setIsChangingRole] = useState(false);

  const [isTransferOwnershipModalOpen, setIsTransferOwnershipModalOpen] = useState(false);
  const [transferForm, setTransferForm] = useState({ selectedMemberId: '' });
  const [isTransferringOwnership, setIsTransferringOwnership] = useState(false);
  const [transferMemberError, setTransferMemberError] = useState(false);

  const [removingMembers, setRemovingMembers] = useState<Set<string>>(new Set());

  const [userSearchResults, setUserSearchResults] = useState<User[]>([]);
  const [isSearchingUsers, setIsSearchingUsers] = useState(false);
  const [searchInputValue, setSearchInputValue] = useState('');

  // Only OWNER and ADMIN can manage members
  // MEMBER and VIEWER cannot manage members
  const canManageMembers =
    user.role === UserRole.ADMIN ||
    workspaceResponse.userRole === WorkspaceRole.OWNER ||
    workspaceResponse.userRole === WorkspaceRole.ADMIN;

  const canTransferOwnership =
    user.role === UserRole.ADMIN || workspaceResponse.userRole === WorkspaceRole.OWNER;

  const eligibleMembers = members.filter((member) => {
    if (member.role === WorkspaceRole.OWNER) return false;

    if (member.userId === user.id || member.email === user.email) {
      return user.role === UserRole.ADMIN && workspaceResponse.userRole !== WorkspaceRole.OWNER;
    }

    return true;
  });

  const loadMembers = async () => {
    setIsLoadingMembers(true);
    try {
      const response: GetMembersResponse = await workspaceMembershipApi.getMembers(
        workspaceResponse.id,
      );
      setMembers(response.members);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsLoadingMembers(false);
    }
  };

  const searchUsers = async (query: string) => {
    if (user.role !== UserRole.ADMIN) return;

    setIsSearchingUsers(true);
    try {
      const response = await userManagementApi.getUsers({
        limit: 10,
        query: query || undefined,
      });
      const activeUsers = response.items.filter((foundUser) => foundUser.status === 'ACTIVE');
      setUserSearchResults(activeUsers);
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
      setUserSearchResults([]);
    } finally {
      setIsSearchingUsers(false);
    }
  };

  useEffect(() => {
    if (user.role !== UserRole.ADMIN || !isAddMemberModalOpen) return;

    const timer = setTimeout(() => {
      searchUsers(searchInputValue);
    }, 300);

    return () => clearTimeout(timer);
  }, [searchInputValue, isAddMemberModalOpen]);

  const handleAddMember = async () => {
    if (!addMemberForm.email.trim()) {
      setAddMemberEmailError(true);
      message.error(t('workspaces.members.add.emailRequired'));
      return;
    }
    setAddMemberEmailError(false);
    setIsAddingMember(true);

    try {
      const request: AddMemberRequest = {
        email: addMemberForm.email.trim(),
        role: addMemberForm.role,
      };
      const response: AddMemberResponse = await workspaceMembershipApi.addMember(
        workspaceResponse.id,
        request,
      );

      const emailToRemember = request.email;
      setAddMemberForm({ email: '', role: WorkspaceRole.MEMBER });
      setIsAddMemberModalOpen(false);

      if (response.status === AddMemberStatusEnum.ADDED) {
        message.success(t('workspaces.members.add.added'));
        loadMembers();
      } else if (response.status === AddMemberStatusEnum.INVITED) {
        setInvitedEmail(emailToRemember);
        setIsInviteDialogOpen(true);
        loadMembers();
      }
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsAddingMember(false);
    }
  };

  const handleChangeRole = async (userId: string, newRole: WorkspaceRole) => {
    setChangingRoleFor(userId);
    setIsChangingRole(true);

    try {
      const request: ChangeMemberRoleRequest = { role: newRole };
      await workspaceMembershipApi.changeMemberRole(workspaceResponse.id, userId, request);

      setMembers((prev) =>
        prev.map((member) => (member.userId === userId ? { ...member, role: newRole } : member)),
      );

      message.success(t('workspaces.members.roleChanged'));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setChangingRoleFor(null);
      setIsChangingRole(false);
    }
  };

  const handleRemoveMember = async (userId: string, memberEmail: string) => {
    setRemovingMembers((prev) => new Set(prev).add(userId));

    try {
      await workspaceMembershipApi.removeMember(workspaceResponse.id, userId);
      setMembers((prev) => prev.filter((member) => member.userId !== userId));
      message.success(t('workspaces.members.remove.removed', { email: memberEmail }));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setRemovingMembers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(userId);
        return newSet;
      });
    }
  };

  const handleTransferOwnership = async () => {
    if (!transferForm.selectedMemberId) {
      setTransferMemberError(true);
      message.error(t('workspaces.transferOwnership.memberRequired'));
      return;
    }

    const selectedMember = members.find(
      (member) => member.userId === transferForm.selectedMemberId,
    );
    if (!selectedMember) {
      message.error(t('workspaces.transferOwnership.memberNotFound'));
      return;
    }

    setTransferMemberError(false);
    setIsTransferringOwnership(true);

    try {
      const request: TransferOwnershipRequest = {
        newOwnerEmail: selectedMember.email,
      };
      await workspaceMembershipApi.transferOwnership(workspaceResponse.id, request);

      setTransferForm({ selectedMemberId: '' });
      setIsTransferOwnershipModalOpen(false);
      message.success(t('workspaces.transferOwnership.transferred'));
      loadMembers();
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsTransferringOwnership(false);
    }
  };

  const getRoleColor = (role: WorkspaceRole): RoleTagColor => {
    switch (role) {
      case WorkspaceRole.OWNER:
        return 'purple';
      case WorkspaceRole.ADMIN:
        return 'orange';
      case WorkspaceRole.MEMBER:
        return 'blue';
      case WorkspaceRole.VIEWER:
        return 'green';
      default:
        return 'default';
    }
  };

  const toRoleOptions = (roles: WorkspaceRole[]) =>
    roles.map((role) => ({ label: t(WORKSPACE_ROLE_LABEL_KEYS[role]), value: role }));

  useEffect(() => {
    loadMembers();
  }, [workspaceResponse.id]);

  const memberRoleOptions = toRoleOptions([
    WorkspaceRole.ADMIN,
    WorkspaceRole.MEMBER,
    WorkspaceRole.VIEWER,
  ]);

  const newMemberRoleOptions = toRoleOptions([
    WorkspaceRole.VIEWER,
    WorkspaceRole.MEMBER,
    WorkspaceRole.ADMIN,
  ]);

  const columns: ColumnsType<WorkspaceMemberResponse> = [
    {
      title: t('workspaces.members.fields.member'),
      key: 'member',
      width: 300,
      render: (_, record: WorkspaceMemberResponse) => (
        <div className="flex items-center">
          <UserOutlined className="mr-2 text-gray-400 dark:text-gray-500" />
          <div>
            <div className="font-medium dark:text-white">{record.name}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{record.email}</div>
          </div>
        </div>
      ),
    },
    {
      title: t('workspaces.members.fields.role'),
      dataIndex: 'role',
      key: 'role',
      width: 150,
      render: (role: WorkspaceRole, record: WorkspaceMemberResponse) => {
        const isCurrentUser = record.userId === user.id || record.email === user.email;

        if (canManageMembers && role !== WorkspaceRole.OWNER && !isCurrentUser) {
          return (
            <Select
              value={role}
              onChange={(newRole) => handleChangeRole(record.userId, newRole)}
              loading={changingRoleFor === record.userId && isChangingRole}
              disabled={changingRoleFor === record.userId && isChangingRole}
              size="small"
              style={{ width: 150 }}
              options={memberRoleOptions}
            />
          );
        }
        return <Tag color={getRoleColor(role)}>{t(WORKSPACE_ROLE_LABEL_KEYS[role])}</Tag>;
      },
    },
    {
      title: t('workspaces.members.fields.joined'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 200,
      render: (createdAt: Date) => {
        const date = dayjs(createdAt);
        const timeFormat = getUserShortTimeFormat();
        return (
          <div className="text-sm text-gray-600 dark:text-gray-300">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-xs text-gray-400 dark:text-gray-500">
              {formatRelativeTime(date)}
            </div>
          </div>
        );
      },
    },
    {
      title: t('workspaces.members.fields.actions'),
      key: 'actions',
      width: 120,
      render: (_, record: WorkspaceMemberResponse) => {
        const isCurrentUser = record.userId === user.id || record.email === user.email;

        if (!canManageMembers || record.role === WorkspaceRole.OWNER || isCurrentUser) return null;

        return (
          <div className="flex items-center space-x-2">
            <Tooltip title={t('workspaces.members.remove.tooltip')}>
              <Popconfirm
                title={t('workspaces.members.remove.title')}
                description={t('workspaces.members.remove.question', { email: record.email })}
                onConfirm={() => handleRemoveMember(record.userId, record.email)}
                okText={t('common.actions.remove')}
                cancelText={t('common.actions.cancel')}
                okButtonProps={{ danger: true }}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<DeleteOutlined />}
                  danger
                  loading={removingMembers.has(record.userId)}
                  disabled={removingMembers.has(record.userId)}
                />
              </Popconfirm>
            </Tooltip>
          </div>
        );
      },
    },
  ];

  const renderMemberCard = (member: WorkspaceMemberResponse) => {
    const isCurrentUser = member.userId === user.id || member.email === user.email;
    const date = dayjs(member.createdAt);
    const timeFormat = getUserShortTimeFormat();

    return (
      <div
        key={member.id}
        className="mb-3 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800"
      >
        <div className="flex items-start justify-between">
          <div className="flex items-center">
            <UserOutlined className="mr-2 text-gray-400 dark:text-gray-500" />
            <div>
              <div className="font-medium dark:text-white">{member.name}</div>
              <div className="text-xs text-gray-500 dark:text-gray-400">{member.email}</div>
            </div>
          </div>
          {canManageMembers && member.role !== WorkspaceRole.OWNER && !isCurrentUser && (
            <Popconfirm
              title={t('workspaces.members.remove.title')}
              description={t('workspaces.members.remove.question', { email: member.email })}
              onConfirm={() => handleRemoveMember(member.userId, member.email)}
              okText={t('common.actions.remove')}
              cancelText={t('common.actions.cancel')}
              okButtonProps={{ danger: true }}
            >
              <Button
                type="text"
                size="small"
                icon={<DeleteOutlined />}
                danger
                loading={removingMembers.has(member.userId)}
                disabled={removingMembers.has(member.userId)}
              />
            </Popconfirm>
          )}
        </div>

        <div className="mt-3 flex items-center justify-between">
          <div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {t('workspaces.members.fields.role')}
            </div>
            {canManageMembers && member.role !== WorkspaceRole.OWNER && !isCurrentUser ? (
              <Select
                value={member.role}
                onChange={(newRole) => handleChangeRole(member.userId, newRole)}
                loading={changingRoleFor === member.userId && isChangingRole}
                disabled={changingRoleFor === member.userId && isChangingRole}
                size="small"
                style={{ width: 150 }}
                options={memberRoleOptions}
              />
            ) : (
              <Tag color={getRoleColor(member.role)}>
                {t(WORKSPACE_ROLE_LABEL_KEYS[member.role])}
              </Tag>
            )}
          </div>
          <div className="text-right">
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {t('workspaces.members.fields.joined')}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-300">
              {date.format(timeFormat.format)}
            </div>
            <div className="text-xs text-gray-400 dark:text-gray-500">
              {formatRelativeTime(date)}
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="max-w-[850px]">
      <div className="mb-6 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <h2 className="text-xl font-bold text-gray-900 dark:text-white">
          {t('workspaces.members.title')}
        </h2>

        <div className="flex flex-col gap-2 md:flex-row md:space-x-2">
          {canTransferOwnership && (
            <Button
              icon={<SwapOutlined />}
              onClick={() => setIsTransferOwnershipModalOpen(true)}
              disabled={isLoadingMembers || eligibleMembers.length === 0}
              className="w-full md:w-auto"
            >
              {t('workspaces.transferOwnership.button')}
            </Button>
          )}
          {canManageMembers && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setIsAddMemberModalOpen(true)}
              disabled={isLoadingMembers}
              className="w-full border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700 md:w-auto"
            >
              {t('workspaces.members.add.button')}
            </Button>
          )}
        </div>
      </div>

      {isLoadingMembers ? (
        <div className="flex h-64 items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div>
          <div className="mb-4 text-sm text-gray-500 dark:text-gray-400">
            {members.length === 0
              ? t('workspaces.members.empty')
              : t('workspaces.members.count', { count: members.length })}
          </div>

          {isMobile ? (
            members.length === 0 ? (
              <div className="py-8 text-center text-gray-500 dark:text-gray-400">
                <div className="mb-2">{t('workspaces.members.empty')}</div>
                {canManageMembers && (
                  <div className="text-sm">{t('workspaces.members.emptyHint')}</div>
                )}
              </div>
            ) : (
              <div>{members.map(renderMemberCard)}</div>
            )
          ) : (
            <Table
              columns={columns}
              dataSource={members}
              pagination={false}
              rowKey="id"
              size="small"
              locale={{
                emptyText: (
                  <div className="py-8 text-center text-gray-500 dark:text-gray-400">
                    <div className="mb-2">{t('workspaces.members.empty')}</div>
                    {canManageMembers && (
                      <div className="text-sm">{t('workspaces.members.emptyHint')}</div>
                    )}
                  </div>
                ),
              }}
            />
          )}
        </div>
      )}

      {/* Add Member Modal */}
      <Modal
        title={t('workspaces.members.add.title')}
        open={isAddMemberModalOpen}
        onOk={handleAddMember}
        onCancel={() => {
          setIsAddMemberModalOpen(false);
          setAddMemberForm({ email: '', role: WorkspaceRole.MEMBER });
          setAddMemberEmailError(false);
          setSearchInputValue('');
          setUserSearchResults([]);
        }}
        confirmLoading={isAddingMember}
        okText={t('workspaces.members.add.submit')}
        cancelText={t('common.actions.cancel')}
        okButtonProps={{
          className: 'border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700',
        }}
      >
        <div className="py-4">
          <div className="mb-4">
            <div className="mb-2 font-medium text-gray-900 dark:text-white">
              {t('workspaces.members.fields.email')}
            </div>
            {user.role === UserRole.ADMIN ? (
              <AutoComplete
                value={addMemberForm.email}
                onChange={(value) => {
                  setAddMemberEmailError(false);
                  setAddMemberForm({
                    ...addMemberForm,
                    email: value.toLowerCase().trim(),
                  });
                  setSearchInputValue(value);
                }}
                onSelect={(value) => {
                  setAddMemberForm({
                    ...addMemberForm,
                    email: value.toLowerCase().trim(),
                  });
                }}
                onFocus={() => {
                  searchUsers('');
                }}
                placeholder={t('workspaces.members.fields.emailPlaceholder')}
                status={addMemberEmailError ? 'error' : undefined}
                options={userSearchResults.map((user) => ({
                  value: user.email,
                  label: `${user.name} (${user.email})`,
                }))}
                notFoundContent={
                  isSearchingUsers ? (
                    <div className="flex justify-center py-2">
                      <Spin indicator={<LoadingOutlined spin />} size="small" />
                    </div>
                  ) : null
                }
                style={{ width: '100%' }}
              />
            ) : (
              <Input
                value={addMemberForm.email}
                onChange={(e) => {
                  setAddMemberEmailError(false);
                  setAddMemberForm({
                    ...addMemberForm,
                    email: e.target.value.toLowerCase().trim(),
                  });
                }}
                placeholder={t('workspaces.members.fields.emailPlaceholder')}
                status={addMemberEmailError ? 'error' : undefined}
              />
            )}
            <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {t('workspaces.members.add.hint')}
            </div>
          </div>

          <div className="mb-4">
            <div className="mb-2 font-medium text-gray-900 dark:text-white">
              {t('workspaces.members.fields.role')}
            </div>
            <Select
              value={addMemberForm.role}
              onChange={(role) => setAddMemberForm({ ...addMemberForm, role })}
              style={{ width: '100%' }}
              options={newMemberRoleOptions}
            />
          </div>
        </div>
      </Modal>

      {/* Invite Dialog */}
      <Modal
        title={t('workspaces.members.invited.title')}
        open={isInviteDialogOpen}
        onOk={() => setIsInviteDialogOpen(false)}
        onCancel={() => setIsInviteDialogOpen(false)}
        okText={t('common.actions.ok')}
        cancelButtonProps={{ style: { display: 'none' } }}
        okButtonProps={{
          className: 'border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700',
        }}
      >
        <div className="py-4">
          <div className="flex items-center">
            <UserAddOutlined className="mr-3 text-2xl text-blue-600 dark:text-blue-400" />
            <div>
              <div className="font-medium text-gray-900 dark:text-white">
                {t('workspaces.members.invited.sentTo', { email: invitedEmail })}
              </div>
              <div className="mt-1 text-sm text-gray-600 dark:text-gray-300">
                {t('workspaces.members.invited.description')}
              </div>
            </div>
          </div>
        </div>
      </Modal>

      {/* Transfer Ownership Modal */}
      <Modal
        title={t('workspaces.transferOwnership.title')}
        open={isTransferOwnershipModalOpen}
        onOk={handleTransferOwnership}
        onCancel={() => {
          setIsTransferOwnershipModalOpen(false);
          setTransferForm({ selectedMemberId: '' });
          setTransferMemberError(false);
        }}
        confirmLoading={isTransferringOwnership}
        okText={t('workspaces.transferOwnership.submit')}
        cancelText={t('common.actions.cancel')}
        okButtonProps={{
          danger: true,
          disabled: eligibleMembers.length === 0,
        }}
      >
        <div className="py-4">
          <div className="mb-4 rounded-md bg-yellow-50 p-3 dark:bg-yellow-900/30">
            <div className="text-sm text-yellow-800 dark:text-yellow-200">
              <Trans
                i18nKey="workspaces.transferOwnership.warning"
                components={{ bold: <strong /> }}
              />
            </div>
          </div>

          {eligibleMembers.length === 0 ? (
            <div className="rounded-md bg-gray-50 p-4 text-center dark:bg-gray-700">
              <div className="text-sm text-gray-600 dark:text-gray-300">
                {t('workspaces.transferOwnership.noEligibleMembers')}
              </div>
            </div>
          ) : (
            <div className="mb-4">
              <div className="mb-2 font-medium text-gray-900 dark:text-white">
                {t('workspaces.transferOwnership.newOwner')}
              </div>
              <Select
                value={transferForm.selectedMemberId || undefined}
                onChange={(memberId) => {
                  setTransferMemberError(false);
                  setTransferForm({ selectedMemberId: memberId });
                }}
                placeholder={t('workspaces.transferOwnership.newOwnerPlaceholder')}
                style={{ width: '100%' }}
                status={transferMemberError ? 'error' : undefined}
                options={eligibleMembers.map((member) => ({
                  label: (
                    <div className="flex items-center">
                      <UserOutlined className="mr-2 text-gray-400 dark:text-gray-500" />
                      <div>
                        {member.name} ({member.email})
                      </div>
                    </div>
                  ),
                  value: member.userId,
                }))}
              />
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {t('workspaces.transferOwnership.newOwnerHint')}
              </div>
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}

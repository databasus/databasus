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

import { type User, UserRole, userManagementApi } from '../../../entities/users';
import {
  type AssignableWorkspaceRole,
  WORKSPACE_ROLE_LABEL_KEYS,
  type Workspace,
  type WorkspaceMember,
  WorkspaceRole,
  workspaceApi,
  workspaceMemberApi,
} from '../../../entities/workspaces';
import { useIsMobile } from '../../../shared/hooks';
import { translateApiError, useLocale } from '../../../shared/i18n';
import { getUserShortTimeFormat } from '../../../shared/time';

interface Props {
  selectedWorkspace: Workspace;
  user: User;
}

interface AddMemberForm {
  email: string;
  role: AssignableWorkspaceRole;
}

type RoleTagColor = 'purple' | 'orange' | 'blue' | 'green' | 'default';

const EMPTY_ADD_MEMBER_FORM: AddMemberForm = { email: '', role: WorkspaceRole.MEMBER };

export function WorkspaceMembersComponent({ selectedWorkspace, user }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const isMobile = useIsMobile();

  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(true);

  const [isAddMemberModalOpen, setIsAddMemberModalOpen] = useState(false);
  const [addMemberForm, setAddMemberForm] = useState<AddMemberForm>(EMPTY_ADD_MEMBER_FORM);
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

  const isInstanceAdmin = user.role === UserRole.ADMIN;
  const isOwner = selectedWorkspace.currentUserRole === WorkspaceRole.OWNER;

  const canManageMembers =
    isInstanceAdmin || isOwner || selectedWorkspace.currentUserRole === WorkspaceRole.ADMIN;

  // Admins are managed, and ownership is transferred, only by the owner and an instance admin.
  const canManageAdmins = isInstanceAdmin || isOwner;
  const canTransferOwnership = canManageAdmins;

  const eligibleMembers = members.filter((member) => {
    if (member.role === WorkspaceRole.OWNER) return false;

    if (member.userId === user.id) {
      return isInstanceAdmin && !isOwner;
    }

    return true;
  });

  const isMemberManageable = (member: WorkspaceMember) =>
    canManageMembers &&
    member.role !== WorkspaceRole.OWNER &&
    member.userId !== user.id &&
    (member.role !== WorkspaceRole.ADMIN || canManageAdmins);

  const loadMembers = async () => {
    setIsLoadingMembers(true);
    try {
      setMembers(await workspaceMemberApi.listAllMembers(selectedWorkspace.id));
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
      const addedMember = await workspaceMemberApi.addMember({
        workspaceId: selectedWorkspace.id,
        email: addMemberForm.email.trim(),
        role: addMemberForm.role,
      });

      setAddMemberForm(EMPTY_ADD_MEMBER_FORM);
      setIsAddMemberModalOpen(false);

      if (addedMember.outcome === 'INVITED') {
        setInvitedEmail(addedMember.email);
        setIsInviteDialogOpen(true);
      } else {
        message.success(t('workspaces.members.add.added'));
      }

      loadMembers();
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setIsAddingMember(false);
    }
  };

  const handleChangeRole = async (memberId: string, newRole: AssignableWorkspaceRole) => {
    setChangingRoleFor(memberId);
    setIsChangingRole(true);

    try {
      const changedMember = await workspaceMemberApi.changeMemberRole(memberId, newRole);

      setMembers((prev) => prev.map((member) => (member.id === memberId ? changedMember : member)));

      message.success(t('workspaces.members.roleChanged'));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setChangingRoleFor(null);
      setIsChangingRole(false);
    }
  };

  const handleRemoveMember = async (memberId: string, memberEmail: string) => {
    setRemovingMembers((prev) => new Set(prev).add(memberId));

    try {
      await workspaceMemberApi.removeMember(memberId);
      setMembers((prev) => prev.filter((member) => member.id !== memberId));
      message.success(t('workspaces.members.remove.removed', { email: memberEmail }));
    } catch (error: unknown) {
      message.error(translateApiError(error, t));
    } finally {
      setRemovingMembers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(memberId);
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

    const selectedMember = members.find((member) => member.id === transferForm.selectedMemberId);
    if (!selectedMember) {
      message.error(t('workspaces.transferOwnership.memberNotFound'));
      return;
    }

    setTransferMemberError(false);
    setIsTransferringOwnership(true);

    try {
      await workspaceApi.transferOwnership(selectedWorkspace.id, selectedMember.id);

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

  const toRoleOptions = (roles: AssignableWorkspaceRole[]) =>
    roles.map((role) => ({ label: t(WORKSPACE_ROLE_LABEL_KEYS[role]), value: role }));

  const renderMemberName = (member: WorkspaceMember) => (
    <div>
      <div className="font-medium dark:text-white">
        {member.name}
        {member.isInvited && (
          <Tag className="ml-2" color="gold">
            {t('workspaces.members.invitedMark')}
          </Tag>
        )}
      </div>
      <div className="text-xs text-gray-500 dark:text-gray-400">{member.email}</div>
    </div>
  );

  useEffect(() => {
    loadMembers();
  }, [selectedWorkspace.id]);

  const assignableRoles: AssignableWorkspaceRole[] = canManageAdmins
    ? [WorkspaceRole.VIEWER, WorkspaceRole.MEMBER, WorkspaceRole.ADMIN]
    : [WorkspaceRole.VIEWER, WorkspaceRole.MEMBER];

  const memberRoleOptions = toRoleOptions(assignableRoles);

  const columns: ColumnsType<WorkspaceMember> = [
    {
      title: t('workspaces.members.fields.member'),
      key: 'member',
      width: 300,
      render: (_, record: WorkspaceMember) => (
        <div className="flex items-center">
          <UserOutlined className="mr-2 text-gray-400 dark:text-gray-500" />
          {renderMemberName(record)}
        </div>
      ),
    },
    {
      title: t('workspaces.members.fields.role'),
      dataIndex: 'role',
      key: 'role',
      width: 150,
      render: (role: WorkspaceRole, record: WorkspaceMember) => {
        if (isMemberManageable(record)) {
          return (
            <Select<AssignableWorkspaceRole>
              aria-label={t('workspaces.members.fields.role')}
              value={role as AssignableWorkspaceRole}
              onChange={(newRole) => handleChangeRole(record.id, newRole)}
              loading={changingRoleFor === record.id && isChangingRole}
              disabled={changingRoleFor === record.id && isChangingRole}
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
      render: (createdAt: string) => {
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
      render: (_, record: WorkspaceMember) => {
        if (!isMemberManageable(record)) return null;

        return (
          <div className="flex items-center space-x-2">
            <Tooltip title={t('workspaces.members.remove.tooltip')}>
              <Popconfirm
                title={t('workspaces.members.remove.title')}
                description={t('workspaces.members.remove.question', { email: record.email })}
                onConfirm={() => handleRemoveMember(record.id, record.email)}
                okText={t('common.actions.remove')}
                cancelText={t('common.actions.cancel')}
                okButtonProps={{ danger: true }}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<DeleteOutlined />}
                  danger
                  loading={removingMembers.has(record.id)}
                  disabled={removingMembers.has(record.id)}
                />
              </Popconfirm>
            </Tooltip>
          </div>
        );
      },
    },
  ];

  const renderMemberCard = (member: WorkspaceMember) => {
    const isManageable = isMemberManageable(member);
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
            {renderMemberName(member)}
          </div>
          {isManageable && (
            <Popconfirm
              title={t('workspaces.members.remove.title')}
              description={t('workspaces.members.remove.question', { email: member.email })}
              onConfirm={() => handleRemoveMember(member.id, member.email)}
              okText={t('common.actions.remove')}
              cancelText={t('common.actions.cancel')}
              okButtonProps={{ danger: true }}
            >
              <Button
                type="text"
                size="small"
                icon={<DeleteOutlined />}
                danger
                loading={removingMembers.has(member.id)}
                disabled={removingMembers.has(member.id)}
              />
            </Popconfirm>
          )}
        </div>

        <div className="mt-3 flex items-center justify-between">
          <div>
            <div className="text-xs text-gray-500 dark:text-gray-400">
              {t('workspaces.members.fields.role')}
            </div>
            {isManageable ? (
              <Select<AssignableWorkspaceRole>
                aria-label={t('workspaces.members.fields.role')}
                value={member.role as AssignableWorkspaceRole}
                onChange={(newRole) => handleChangeRole(member.id, newRole)}
                loading={changingRoleFor === member.id && isChangingRole}
                disabled={changingRoleFor === member.id && isChangingRole}
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
          setAddMemberForm(EMPTY_ADD_MEMBER_FORM);
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
            <Select<AssignableWorkspaceRole>
              aria-label={t('workspaces.members.fields.role')}
              value={addMemberForm.role}
              onChange={(role) => setAddMemberForm({ ...addMemberForm, role })}
              style={{ width: '100%' }}
              options={memberRoleOptions}
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
                  value: member.id,
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

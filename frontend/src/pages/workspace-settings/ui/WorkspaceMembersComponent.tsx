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
import { useEffect, useEffectEvent, useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { ApiError } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useRequestRevision } from '@/shared/hooks';

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
import { LoadFailure } from '../../../shared/ui';

interface Props {
  selectedWorkspace: Workspace;
  user: User;
  onUnavailable?: (error: unknown) => void;
}

interface AddMemberForm {
  email: string;
  role: AssignableWorkspaceRole;
}

const EMPTY_ADD_MEMBER_FORM: AddMemberForm = { email: '', role: WorkspaceRole.MEMBER };

export function WorkspaceMembersComponent({ selectedWorkspace, user, onUnavailable }: Props) {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const { message } = App.useApp();
  const isMobile = useIsMobile();

  const [loadError, setLoadError] = useState<unknown>();
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(true);
  const [hasLoadedMembers, setHasLoadedMembers] = useState(false);

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
    const readRevision = memberReadRevision.Invalidate();
    const isActive = () => isCurrent() && memberReadRevision.IsCurrent(readRevision);
    setIsLoadingMembers(true);
    setLoadError(undefined);

    try {
      const loadedMembers = await workspaceMemberApi.listAllMembers(selectedWorkspace.id);
      if (!isActive()) return;

      setMembers(loadedMembers);
      setHasLoadedMembers(true);
    } catch (error: unknown) {
      if (!isActive()) return;

      setLoadError(error);

      if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
        setMembers([]);
        setHasLoadedMembers(false);
        onUnavailable?.(error);
      }
    } finally {
      if (isActive()) {
        setIsLoadingMembers(false);
      }
    }
  };

  const searchUsers = async (query: string) => {
    const requestRevision = searchRevision.Invalidate();
    const isActive = () => isCurrent() && searchRevision.IsCurrent(requestRevision);

    if (user.role !== UserRole.ADMIN) return;

    setIsSearchingUsers(true);

    try {
      const response = await userManagementApi.getUsers({
        limit: 10,
        query: query || undefined,
      });
      if (!isActive()) return;

      const activeUsers = response.items.filter((foundUser) => foundUser.status === 'ACTIVE');
      setUserSearchResults(activeUsers);
    } catch (error: unknown) {
      if (!isActive()) return;

      message.error(translateApiError(error, t));
      setUserSearchResults([]);
    } finally {
      if (isActive()) {
        setIsSearchingUsers(false);
      }
    }
  };

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
      if (!isCurrent()) return;

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
      if (!isCurrent()) return;

      message.error(translateApiError(error, t));
    } finally {
      if (isCurrent()) {
        setIsAddingMember(false);
      }
    }
  };

  const handleChangeRole = async (memberId: string, newRole: AssignableWorkspaceRole) => {
    setChangingRoleFor(memberId);
    setIsChangingRole(true);

    try {
      const changedMember = await workspaceMemberApi.changeMemberRole(memberId, newRole);
      if (!isCurrent()) return;

      setMembers((prev) => prev.map((member) => (member.id === memberId ? changedMember : member)));

      message.success(t('workspaces.members.roleChanged'));
    } catch (error: unknown) {
      if (!isCurrent()) return;

      message.error(translateApiError(error, t));
    } finally {
      if (isCurrent()) {
        setChangingRoleFor(null);
        setIsChangingRole(false);
      }
    }
  };

  const handleRemoveMember = async (memberId: string, memberEmail: string) => {
    setRemovingMembers((prev) => new Set(prev).add(memberId));

    try {
      await workspaceMemberApi.removeMember(memberId);
      if (!isCurrent()) return;

      setMembers((prev) => prev.filter((member) => member.id !== memberId));
      message.success(t('workspaces.members.remove.removed', { email: memberEmail }));
    } catch (error: unknown) {
      if (!isCurrent()) return;

      message.error(translateApiError(error, t));
    } finally {
      if (isCurrent()) {
        setRemovingMembers((prev) => {
          const newSet = new Set(prev);
          newSet.delete(memberId);
          return newSet;
        });
      }
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
      if (!isCurrent()) return;

      setTransferForm({ selectedMemberId: '' });
      setIsTransferOwnershipModalOpen(false);
      message.success(t('workspaces.transferOwnership.transferred'));
      loadMembers();
    } catch (error: unknown) {
      if (!isCurrent()) return;

      message.error(translateApiError(error, t));
    } finally {
      if (isCurrent()) {
        setIsTransferringOwnership(false);
      }
    }
  };

  const toRoleOptions = (roles: AssignableWorkspaceRole[]) =>
    roles.map((role) => ({ label: t(WORKSPACE_ROLE_LABEL_KEYS[role]), value: role }));

  const renderMemberName = (member: WorkspaceMember) => (
    <div>
      <div className="font-medium">
        {member.name}
        {member.isInvited && <Tag className="ml-2">{t('workspaces.members.invitedMark')}</Tag>}
      </div>
      <div className="text-xs text-muted">{member.email}</div>
    </div>
  );

  const renderMemberCard = (member: WorkspaceMember) => {
    const isManageable = isMemberManageable(member);
    const date = dayjs(member.createdAt);
    const timeFormat = getUserShortTimeFormat();

    return (
      <div
        key={member.id}
        className="mb-3 rounded-lg border border-border bg-surface p-3 shadow-sm"
      >
        <div className="flex items-start justify-between">
          <div className="flex items-center">
            <UserOutlined className="mr-2 text-muted" />
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
                aria-label={t('workspaces.members.remove.tooltip')}
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
            <div className="text-xs text-muted">{t('workspaces.members.fields.role')}</div>
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
              <Tag>{t(WORKSPACE_ROLE_LABEL_KEYS[member.role])}</Tag>
            )}
          </div>
          <div className="text-right">
            <div className="text-xs text-muted">{t('workspaces.members.fields.joined')}</div>
            <div className="text-sm text-muted">{date.format(timeFormat.format)}</div>
            <div className="text-xs text-muted">{formatRelativeTime(date)}</div>
          </div>
        </div>
      </div>
    );
  };

  const memberEmailControlId = useId();
  const isCurrent = useSessionGuard();
  const searchRevision = useRequestRevision();
  const memberReadRevision = useRequestRevision();

  const startMembersLoad = useEffectEvent(() => {
    void loadMembers();
  });

  useEffect(() => {
    startMembersLoad();
  }, []);

  const canSearchUsers = useEffectEvent(() => user.role === UserRole.ADMIN);
  const startUserSearch = useEffectEvent((query: string) => {
    searchUsers(query);
  });

  useEffect(() => {
    if (!canSearchUsers() || !isAddMemberModalOpen) return;

    const timer = setTimeout(() => {
      startUserSearch(searchInputValue);
    }, 300);

    return () => {
      clearTimeout(timer);
      searchRevision.Invalidate();
    };
  }, [searchInputValue, isAddMemberModalOpen, searchRevision]);

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
          <UserOutlined className="mr-2 text-muted" />
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

        return <Tag>{t(WORKSPACE_ROLE_LABEL_KEYS[role])}</Tag>;
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
          <div className="text-sm text-muted">
            <div>{date.format(timeFormat.format)}</div>
            <div className="text-xs text-muted">{formatRelativeTime(date)}</div>
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
                  aria-label={t('workspaces.members.remove.tooltip')}
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

  return (
    <div className="max-w-212.5">
      {loadError !== undefined && (
        <LoadFailure error={loadError} onRetry={() => void loadMembers()} />
      )}
      <div className="mb-6 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <h2 className="text-xl font-bold text-muted">{t('workspaces.members.title')}</h2>

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
              className="w-full border-border bg-accent hover:border-border hover:bg-accent md:w-auto"
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
      ) : hasLoadedMembers ? (
        <div>
          <div className="mb-4 text-sm text-muted">
            {members.length === 0
              ? t('workspaces.members.empty')
              : t('workspaces.members.count', { count: members.length })}
          </div>

          {isMobile ? (
            members.length === 0 ? (
              <div className="py-8 text-center text-muted">
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
                  <div className="py-8 text-center text-muted">
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
      ) : null}

      <Modal
        width={420}
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
          className: 'border-border bg-accent hover:border-border hover:bg-accent',
        }}
      >
        <div className="py-2">
          <div className="mb-3">
            <label htmlFor={memberEmailControlId} className="mb-1 block text-content">
              {t('workspaces.members.fields.email')}
            </label>
            {user.role === UserRole.ADMIN ? (
              <AutoComplete
                id={memberEmailControlId}
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
                id={memberEmailControlId}
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
            <div className="mt-1 text-xs text-muted">{t('workspaces.members.add.hint')}</div>
          </div>

          <div className="mb-0">
            <div className="mb-1 text-content">{t('workspaces.members.fields.role')}</div>
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
          className: 'border-border bg-accent hover:border-border hover:bg-accent',
        }}
      >
        <div className="py-4">
          <div className="flex items-center">
            <UserAddOutlined className="mr-3 text-2xl text-accent" />
            <div>
              <div className="font-medium text-muted">
                {t('workspaces.members.invited.sentTo', { email: invitedEmail })}
              </div>
              <div className="mt-1 text-sm text-muted">
                {t('workspaces.members.invited.description')}
              </div>
            </div>
          </div>
        </div>
      </Modal>

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
          className: '!border-danger !bg-danger !text-surface',
          disabled: eligibleMembers.length === 0,
        }}
      >
        <div className="py-4">
          <div className="mb-4 rounded-md border border-warning bg-surface p-3">
            <div className="text-sm text-warning">
              <Trans
                i18nKey="workspaces.transferOwnership.warning"
                components={{ bold: <strong /> }}
              />
            </div>
          </div>

          {eligibleMembers.length === 0 ? (
            <div className="rounded-md bg-surface p-4 text-center">
              <div className="text-sm text-muted">
                {t('workspaces.transferOwnership.noEligibleMembers')}
              </div>
            </div>
          ) : (
            <div className="mb-4">
              <div className="mb-2 font-medium text-muted">
                {t('workspaces.transferOwnership.newOwner')}
              </div>
              <Select
                aria-label={t('workspaces.transferOwnership.newOwner')}
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
                      <UserOutlined className="mr-2 text-muted" />
                      <div>
                        {member.name} ({member.email})
                      </div>
                    </div>
                  ),
                  value: member.id,
                }))}
              />
              <div className="mt-1 text-xs text-muted">
                {t('workspaces.transferOwnership.newOwnerHint')}
              </div>
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}

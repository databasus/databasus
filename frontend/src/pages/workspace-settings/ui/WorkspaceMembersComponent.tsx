import { ActionIcon, Autocomplete, Badge, Button, Loader, TextInput, Tooltip } from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { ArrowRightLeft, Plus, Trash2, UserPlus, UserRound } from 'lucide-react';
import { useEffect, useEffectEvent, useId, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { type User, UserRole, userManagementApi } from '@/entities/users';
import {
  type AssignableWorkspaceRole,
  WORKSPACE_ROLE_LABEL_KEYS,
  type Workspace,
  type WorkspaceMember,
  WorkspaceRole,
  workspaceApi,
  workspaceMemberApi,
} from '@/entities/workspaces';
import { useSessionMutation } from '@/shared/api';
import { ApiError, useSessionQuery } from '@/shared/api';
import { useSessionGuard } from '@/shared/auth';
import { useResourceCollection } from '@/shared/hooks';
import { useIsMobile } from '@/shared/hooks';
import { translateApiError, useLocale } from '@/shared/i18n';
import { useFeedback } from '@/shared/lib/feedback';
import { getUserShortTimeFormat } from '@/shared/time';
import { AnchoredConfirmation } from '@/shared/ui';
import { WrappingSelect } from '@/shared/ui';
import { Dialog } from '@/shared/ui';
import { LoadFailure } from '@/shared/ui';
import tableStyles from '@/shared/ui/nativeTable.module.css';
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
  const addMemberMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceMemberApi.addMember>) =>
      workspaceMemberApi.addMember(...parameters),
  });
  const changeMemberRoleMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceMemberApi.changeMemberRole>) =>
      workspaceMemberApi.changeMemberRole(...parameters),
  });
  const removeMemberMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceMemberApi.removeMember>) =>
      workspaceMemberApi.removeMember(...parameters),
  });
  const transferOwnershipMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof workspaceApi.transferOwnership>) =>
      workspaceApi.transferOwnership(...parameters),
  });

  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();
  const notify = useFeedback();
  const isMobile = useIsMobile();

  const client = useQueryClient();
  const collection = useResourceCollection<WorkspaceMember>({
    queryKey: ['workspace-members', selectedWorkspace.id],
    read: (signal) => workspaceMemberApi.listAllMembers(selectedWorkspace.id, { signal }),
    onError: (error) => {
      if (error instanceof ApiError && [403, 404].includes(error.status ?? 0)) {
        collection.clear();
        onUnavailable?.(error);
      }
    },
  });
  const members = collection.items;
  const setMembers = collection.update;
  const loadError = collection.loadError;
  const isLoadingMembers = collection.isLoading;
  const hasLoadedMembers = collection.isReady;
  const loadMembers = collection.reload;
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
  const isTransferringOwnership = transferOwnershipMutation.isPending;
  const [transferMemberError, setTransferMemberError] = useState(false);

  const [removingMembers, setRemovingMembers] = useState<Set<string>>(new Set());

  const [searchedTerm, setSearchedTerm] = useState<string>();
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

  const search = useSessionQuery({
    queryKey: ['workspace-member-search', selectedWorkspace.id, searchedTerm],
    enabled: isAddMemberModalOpen && user.role === UserRole.ADMIN && searchedTerm !== undefined,
    queryFn: async (signal) => {
      const response = await userManagementApi.getUsers(
        { limit: 10, query: searchedTerm || undefined },
        { signal },
      );

      return response.items.filter((user) => user.status === 'ACTIVE');
    },
  });
  const userSearchResults = search.data ?? [];
  const isSearchingUsers = search.isFetching;
  const searchUsers = (term: string) => {
    if (term === searchedTerm) void search.refetch();
    else setSearchedTerm(term);
  };
  const reportSearchFailure = useEffectEvent((error: unknown) =>
    notify({ kind: 'error', title: translateApiError(error, t) }),
  );
  useEffect(() => {
    if (search.error) reportSearchFailure(search.error);
  }, [search.error]);

  const handleAddMember = async () => {
    if (!addMemberForm.email.trim()) {
      setAddMemberEmailError(true);
      notify({ kind: 'error', title: t('workspaces.members.add.emailRequired') });
      return;
    }
    setAddMemberEmailError(false);
    setIsAddingMember(true);

    try {
      const addedMember = await addMemberMutation.mutateAsync([
        {
          workspaceId: selectedWorkspace.id,
          email: addMemberForm.email.trim(),
          role: addMemberForm.role,
        },
      ]);
      if (!isCurrent()) return;

      collection.accept(addedMember);
      setAddMemberForm(EMPTY_ADD_MEMBER_FORM);
      setIsAddMemberModalOpen(false);

      if (addedMember.outcome === 'INVITED') {
        setInvitedEmail(addedMember.email);
        setIsInviteDialogOpen(true);
      } else {
        notify({ kind: 'success', title: t('workspaces.members.add.added') });
      }

      loadMembers();
    } catch (error: unknown) {
      if (!isCurrent()) return;

      notify({ kind: 'error', title: translateApiError(error, t) });
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
      const changedMember = await changeMemberRoleMutation.mutateAsync([memberId, newRole]);
      if (!isCurrent()) return;

      setMembers((prev) => prev.map((member) => (member.id === memberId ? changedMember : member)));
      notify({ kind: 'success', title: t('workspaces.members.roleChanged') });
    } catch (error: unknown) {
      if (!isCurrent()) return;

      notify({ kind: 'error', title: translateApiError(error, t) });
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
      await removeMemberMutation.mutateAsync([memberId]);
      if (!isCurrent()) return;

      setMembers((prev) => prev.filter((member) => member.id !== memberId));
      notify({
        kind: 'success',
        title: t('workspaces.members.remove.removed', { email: memberEmail }),
      });
    } catch (error: unknown) {
      if (!isCurrent()) return;

      notify({ kind: 'error', title: translateApiError(error, t) });
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
      notify({ kind: 'error', title: t('workspaces.transferOwnership.memberRequired') });
      return;
    }

    const selectedMember = members.find((member) => member.id === transferForm.selectedMemberId);

    if (!selectedMember) {
      notify({ kind: 'error', title: t('workspaces.transferOwnership.memberNotFound') });
      return;
    }

    setTransferMemberError(false);

    try {
      await transferOwnershipMutation.mutateAsync([selectedWorkspace.id, selectedMember.id]);
      if (!isCurrent()) return;

      setTransferForm({ selectedMemberId: '' });
      setIsTransferOwnershipModalOpen(false);
      notify({ kind: 'success', title: t('workspaces.transferOwnership.transferred') });
      loadMembers();
    } catch (error: unknown) {
      if (!isCurrent()) return;

      notify({ kind: 'error', title: translateApiError(error, t) });
    } finally {
      if (isCurrent()) {
        transferOwnershipMutation.reset();
      }
    }
  };

  const toRoleOptions = (roles: AssignableWorkspaceRole[]) =>
    roles.map((role) => ({ label: t(WORKSPACE_ROLE_LABEL_KEYS[role]), value: role }));

  const renderMemberName = (member: WorkspaceMember) => (
    <div className="min-w-0 [overflow-wrap:anywhere]">
      <div className="font-medium">
        {member.name}
        {member.isInvited && (
          <Badge variant="default" radius="sm" size="sm" className="ml-2">
            {t('workspaces.members.invitedMark')}
          </Badge>
        )}
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
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 items-center">
            <UserRound className="mr-2 inline-block shrink-0 align-middle text-muted" size="1em" />
            {renderMemberName(member)}
          </div>
          {isManageable && (
            <AnchoredConfirmation
              title={t('workspaces.members.remove.title')}
              description={t('workspaces.members.remove.question', { email: member.email })}
              onConfirm={() => handleRemoveMember(member.id, member.email)}
              confirmLabel={t('common.actions.remove')}
              isConfirming={removingMembers.has(member.id)}
            >
              <ActionIcon
                variant="subtle"
                size="xs"
                aria-label={t('workspaces.members.remove.tooltip')}
                color="danger"
                loading={removingMembers.has(member.id)}
                disabled={removingMembers.has(member.id)}
                aria-busy={removingMembers.has(member.id)}
              >
                <Trash2 size="1em" className="inline-block shrink-0 align-middle" />
              </ActionIcon>
            </AnchoredConfirmation>
          )}
        </div>

        <div className="mt-3 flex items-center justify-between">
          <div>
            <div className="text-xs text-muted">{t('workspaces.members.fields.role')}</div>
            {isManageable ? (
              <WrappingSelect<AssignableWorkspaceRole>
                aria-label={t('workspaces.members.fields.role')}
                value={member.role as AssignableWorkspaceRole}
                onChange={(newRole) => handleChangeRole(member.id, newRole)}
                isLoading={changingRoleFor === member.id && isChangingRole}
                disabled={changingRoleFor === member.id && isChangingRole}
                size="xs"
                style={{ width: 150 }}
                data={memberRoleOptions}
              />
            ) : (
              <Badge
                variant="default"
                radius="sm"
                size="sm"
                styles={{ label: { textTransform: 'none', fontWeight: 400 } }}
              >
                {t(WORKSPACE_ROLE_LABEL_KEYS[member.role])}
              </Badge>
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
  const cancelUserSearch = useEffectEvent(() => {
    void client.cancelQueries({ queryKey: search.queryKey, exact: true });
  });
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
      cancelUserSearch();
    };
  }, [searchInputValue, isAddMemberModalOpen, client]);

  const assignableRoles: AssignableWorkspaceRole[] = canManageAdmins
    ? [WorkspaceRole.VIEWER, WorkspaceRole.MEMBER, WorkspaceRole.ADMIN]
    : [WorkspaceRole.VIEWER, WorkspaceRole.MEMBER];

  const memberRoleOptions = toRoleOptions(assignableRoles);
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
              leftSection={
                <ArrowRightLeft size="1em" className="inline-block shrink-0 align-middle" />
              }
              onClick={() => setIsTransferOwnershipModalOpen(true)}
              disabled={isLoadingMembers || eligibleMembers.length === 0}
              className="w-full md:w-auto"
            >
              {t('workspaces.transferOwnership.button')}
            </Button>
          )}
          {canManageMembers && (
            <Button
              variant="filled"
              leftSection={<Plus size="1em" className="inline-block shrink-0 align-middle" />}
              onClick={() => setIsAddMemberModalOpen(true)}
              disabled={isLoadingMembers}
              className="w-full md:w-auto"
            >
              {t('workspaces.members.add.button')}
            </Button>
          )}
        </div>
      </div>

      {isLoadingMembers ? (
        <div className="flex h-64 items-center justify-center">
          <Loader size="md" role="status" />
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
            <div className="relative overflow-x-auto">
              <table className={tableStyles.table}>
                <thead>
                  <tr>
                    <th scope="col" style={{ width: 300 }}>
                      {t('workspaces.members.fields.member')}
                    </th>
                    <th scope="col" style={{ width: 150 }}>
                      {t('workspaces.members.fields.role')}
                    </th>
                    <th scope="col" style={{ width: 200 }}>
                      {t('workspaces.members.fields.joined')}
                    </th>
                    <th scope="col" style={{ width: 120 }}>
                      {t('workspaces.members.fields.actions')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {members.map((member) => (
                    <tr key={member.id}>
                      <td>
                        <div className="flex items-center">
                          <UserRound
                            className="mr-2 inline-block shrink-0 align-middle text-muted"
                            size="1em"
                          />
                          {renderMemberName(member)}
                        </div>
                      </td>
                      <td>
                        {(() => {
                          if (isMemberManageable(member)) {
                            return (
                              <WrappingSelect<AssignableWorkspaceRole>
                                aria-label={t('workspaces.members.fields.role')}
                                value={member.role as AssignableWorkspaceRole}
                                onChange={(newRole) => handleChangeRole(member.id, newRole)}
                                isLoading={changingRoleFor === member.id && isChangingRole}
                                disabled={changingRoleFor === member.id && isChangingRole}
                                size="xs"
                                style={{ width: 150 }}
                                data={memberRoleOptions}
                              />
                            );
                          }
                          return (
                            <Badge
                              variant="default"
                              radius="sm"
                              size="sm"
                              styles={{ label: { textTransform: 'none', fontWeight: 400 } }}
                            >
                              {t(WORKSPACE_ROLE_LABEL_KEYS[member.role])}
                            </Badge>
                          );
                        })()}
                      </td>
                      <td>
                        {(() => {
                          const date = dayjs(member.createdAt);
                          const timeFormat = getUserShortTimeFormat();
                          return (
                            <div className="text-sm text-muted">
                              <div>{date.format(timeFormat.format)}</div>
                              <div className="text-xs text-muted">{formatRelativeTime(date)}</div>
                            </div>
                          );
                        })()}
                      </td>
                      <td>
                        {(() => {
                          if (!isMemberManageable(member)) return null;
                          return (
                            <div className="flex items-center space-x-2">
                              <AnchoredConfirmation
                                title={t('workspaces.members.remove.title')}
                                description={t('workspaces.members.remove.question', {
                                  email: member.email,
                                })}
                                onConfirm={() => handleRemoveMember(member.id, member.email)}
                                confirmLabel={t('common.actions.remove')}
                                isConfirming={removingMembers.has(member.id)}
                              >
                                <Tooltip label={t('workspaces.members.remove.tooltip')}>
                                  <ActionIcon
                                    variant="subtle"
                                    size="xs"
                                    aria-label={t('workspaces.members.remove.tooltip')}
                                    color="danger"
                                    loading={removingMembers.has(member.id)}
                                    disabled={removingMembers.has(member.id)}
                                    aria-busy={removingMembers.has(member.id)}
                                  >
                                    <Trash2
                                      size="1em"
                                      className="inline-block shrink-0 align-middle"
                                    />
                                  </ActionIcon>
                                </Tooltip>
                              </AnchoredConfirmation>
                            </div>
                          );
                        })()}
                      </td>
                    </tr>
                  ))}
                  {members.length === 0 && (
                    <tr>
                      <td colSpan={4}>
                        <div className="py-8 text-center text-muted">
                          <div className="mb-2">{t('workspaces.members.empty')}</div>
                          {canManageMembers && (
                            <div className="text-sm">{t('workspaces.members.emptyHint')}</div>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      ) : null}

      <Dialog
        size={420}
        title={t('workspaces.members.add.title')}
        opened={isAddMemberModalOpen}
        onClose={() => {
          setIsAddMemberModalOpen(false);
          setAddMemberForm(EMPTY_ADD_MEMBER_FORM);
          setAddMemberEmailError(false);
          setSearchInputValue('');
          setSearchedTerm(undefined);
        }}
      >
        <div className="py-2">
          <div className="mb-3">
            <label htmlFor={memberEmailControlId} className="mb-1 block text-content">
              {t('workspaces.members.fields.email')}
            </label>
            {user.role === UserRole.ADMIN ? (
              <Autocomplete
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
                onOptionSubmit={(value) => {
                  setAddMemberForm({
                    ...addMemberForm,
                    email: value.toLowerCase().trim(),
                  });
                }}
                onFocus={() => {
                  searchUsers('');
                }}
                placeholder={t('workspaces.members.fields.emailPlaceholder')}
                error={(addMemberEmailError ? 'error' : undefined) === 'error'}
                data={userSearchResults.map((user) => ({
                  value: user.email,
                  label: `${user.name} (${user.email})`,
                }))}
                loading={isSearchingUsers}
                aria-busy={isSearchingUsers}
                filter={({ options }) => options}
                renderOption={({ option }) => {
                  const user = userSearchResults.find((user) => user.email === option.value);
                  return user ? `${user.name} (${user.email})` : option.value;
                }}
                style={{ width: '100%' }}
              />
            ) : (
              <TextInput
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
                error={(addMemberEmailError ? 'error' : undefined) === 'error'}
              />
            )}
            <div className="mt-1 text-xs text-muted">{t('workspaces.members.add.hint')}</div>
          </div>

          <div className="mb-0">
            <div className="mb-1 text-content">{t('workspaces.members.fields.role')}</div>
            <WrappingSelect<AssignableWorkspaceRole>
              aria-label={t('workspaces.members.fields.role')}
              value={addMemberForm.role}
              onChange={(role) => setAddMemberForm({ ...addMemberForm, role })}
              style={{ width: '100%' }}
              data={memberRoleOptions}
            />
          </div>
        </div>
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <div>
            <Button
              onClick={() => {
                setIsAddMemberModalOpen(false);
                setAddMemberForm(EMPTY_ADD_MEMBER_FORM);
                setAddMemberEmailError(false);
                setSearchInputValue('');
                setSearchedTerm(undefined);
              }}
            >
              {t('common.actions.cancel')}
            </Button>
            <Button
              variant={'filled'}
              onClick={handleAddMember}
              loading={isAddingMember}
              aria-busy={isAddingMember}
            >
              {t('workspaces.members.add.submit')}
            </Button>
          </div>
        </div>
      </Dialog>

      {/* Invite Dialog */}
      <Dialog
        title={t('workspaces.members.invited.title')}
        opened={isInviteDialogOpen}
        onClose={() => setIsInviteDialogOpen(false)}
      >
        <div className="py-4">
          <div className="flex items-center">
            <UserPlus
              className="mr-3 inline-block shrink-0 align-middle text-2xl text-accent"
              size="1em"
            />
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
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <div>
            <Button onClick={() => setIsInviteDialogOpen(false)} style={{ display: 'none' }}>
              {t('common.actions.cancel')}
            </Button>
            <Button variant={'filled'} onClick={() => setIsInviteDialogOpen(false)}>
              {t('common.actions.ok')}
            </Button>
          </div>
        </div>
      </Dialog>

      <Dialog
        title={t('workspaces.transferOwnership.title')}
        opened={isTransferOwnershipModalOpen}
        onClose={() => {
          setIsTransferOwnershipModalOpen(false);
          setTransferForm({ selectedMemberId: '' });
          setTransferMemberError(false);
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
              <WrappingSelect
                aria-label={t('workspaces.transferOwnership.newOwner')}
                value={transferForm.selectedMemberId || undefined}
                onChange={(memberId) => {
                  setTransferMemberError(false);
                  setTransferForm({ selectedMemberId: memberId });
                }}
                placeholder={t('workspaces.transferOwnership.newOwnerPlaceholder')}
                style={{ width: '100%' }}
                hasError={(transferMemberError ? 'error' : undefined) === 'error'}
                renderOption={(memberId) => {
                  const member = eligibleMembers.find((member) => member.id === memberId);
                  return (
                    <span className="inline-flex min-w-0 items-center gap-2">
                      <UserRound aria-hidden size="1em" />
                      <span>
                        {member?.name} ({member?.email})
                      </span>
                    </span>
                  );
                }}
                data={eligibleMembers.map((member) => ({
                  label: `${member.name} (${member.email})`,
                  value: member.id,
                }))}
              />
              <div className="mt-1 text-xs text-muted">
                {t('workspaces.transferOwnership.newOwnerHint')}
              </div>
            </div>
          )}
        </div>
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <div>
            <Button
              onClick={() => {
                setIsTransferOwnershipModalOpen(false);
                setTransferForm({ selectedMemberId: '' });
                setTransferMemberError(false);
              }}
            >
              {t('common.actions.cancel')}
            </Button>
            <Button
              variant={'filled'}
              onClick={handleTransferOwnership}
              loading={isTransferringOwnership}
              color={'danger'}
              className={'!border-danger !bg-danger !text-surface'}
              disabled={eligibleMembers.length === 0}
              aria-busy={isTransferringOwnership}
            >
              {t('workspaces.transferOwnership.submit')}
            </Button>
          </div>
        </div>
      </Dialog>
    </div>
  );
}

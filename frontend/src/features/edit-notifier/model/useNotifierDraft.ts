import { useLayoutEffect, useRef, useState } from 'react';

import {
  type Notifier,
  type NotifierSettings,
  NotifierType,
  notifierApi,
} from '@/entities/notifiers';
import { createDefaultNotifierSettings } from '@/features/edit-notifier/lib/createDefaultNotifierSettings';
import { isNotifierSettingsFilled } from '@/features/edit-notifier/lib/isNotifierSettingsFilled';
import { acknowledgeNotifierSettings } from '@/features/edit-notifier/model/acknowledgeNotifierSettings';
import {
  type NotifierDraft,
  isNotifierDraftUnchanged,
} from '@/features/edit-notifier/model/NotifierDraft';
import { useSessionMutation } from '@/shared/api';
import { getSessionSnapshot, isCurrentSession, subscribeSession } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

interface Options {
  workspaceId: string;
  editingNotifier?: Notifier;
  onPersisted: (notifier: Notifier) => void;
  onChanged: (notifier: Notifier) => void;
  onTestSucceeded: () => void;
}

type NotifierError =
  | { kind: 'api'; error: unknown }
  | { kind: 'delivery'; diagnostic: string; isSlack: boolean };

const createDraft = (notifier?: Notifier): NotifierDraft => ({
  name: notifier?.name ?? '',
  settings: notifier
    ? structuredClone(notifier.settings)
    : createDefaultNotifierSettings(NotifierType.TELEGRAM),
});

export const useNotifierDraft = ({
  workspaceId,
  editingNotifier,
  onPersisted,
  onChanged,
  onTestSucceeded,
}: Options) => {
  const testNotifierSettingsMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.testNotifierSettings>) =>
      notifierApi.testNotifierSettings(...parameters),
  });
  const { reset: resetRequest } = testNotifierSettingsMutation;
  const updateNotifierMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.updateNotifier>) =>
      notifierApi.updateNotifier(...parameters),
  });
  const createNotifierMutation = useSessionMutation({
    mutationFn: (parameters: Parameters<typeof notifierApi.createNotifier>) =>
      notifierApi.createNotifier(...parameters),
  });

  const [draft, setDraft] = useState(() => createDraft(editingNotifier));
  const [baseline, setBaseline] = useState(draft);
  const [savedNotifierId, setSavedNotifierId] = useState(editingNotifier?.id);
  const isTesting = testNotifierSettingsMutation.isPending;
  const [isSaving, setIsSaving] = useState(false);
  const [testedSettingsRevision, setTestedSettingsRevision] = useState<number>();
  const [error, setError] = useState<NotifierError>();
  const revisions = useRef({
    session: new RequestRevision(),
    settings: new RequestRevision(),
    draft: new RequestRevision(),
    test: new RequestRevision(),
  });
  const latestDraft = useRef(draft);
  const baselineRef = useRef(baseline);
  const savedIdRef = useRef(savedNotifierId);
  const context = useRef({ workspaceId, inputId: editingNotifier?.id });
  const activeSave = useRef<symbol | undefined>(undefined);
  const activeTest = useRef<symbol | undefined>(undefined);
  const callbacks = useRef({ onPersisted, onChanged, onTestSucceeded });

  const changeName = (name: string) => {
    revisions.current.draft.Invalidate();
    const changedDraft = { ...latestDraft.current, name };
    latestDraft.current = changedDraft;
    setDraft(changedDraft);
  };

  const changeSettings = (settings: NotifierSettings) => {
    if (
      (savedIdRef.current || activeSave.current) &&
      settings.type !== latestDraft.current.settings.type
    )
      return;

    revisions.current.draft.Invalidate();
    revisions.current.settings.Invalidate();
    revisions.current.test.Invalidate();
    activeTest.current = undefined;
    resetRequest();
    setTestedSettingsRevision(undefined);
    setError(undefined);

    const changedDraft = { ...latestDraft.current, settings };
    latestDraft.current = changedDraft;
    setDraft(changedDraft);
  };

  const testSettings = async () => {
    if (
      activeTest.current ||
      activeSave.current ||
      !latestDraft.current.name.trim() ||
      !isNotifierSettingsFilled(
        latestDraft.current.settings,
        !!savedIdRef.current,
        savedIdRef.current ? baselineRef.current.settings : undefined,
      )
    )
      return;

    const authSession = getSessionSnapshot();
    const { session, settings, test } = revisions.current;
    const sessionRevision = session.GetCurrent();
    const settingsRevision = settings.GetCurrent();
    const testRevision = test.Invalidate();

    const request = Symbol();
    activeTest.current = request;

    setError(undefined);

    try {
      const response = await testNotifierSettingsMutation.mutateAsync([
        workspaceId,
        latestDraft.current.settings,
        savedIdRef.current,
      ]);
      if (
        !isCurrentSession(authSession) ||
        !session.IsCurrent(sessionRevision) ||
        !settings.IsCurrent(settingsRevision) ||
        !test.IsCurrent(testRevision)
      )
        return;

      if (response.isDelivered) {
        setTestedSettingsRevision(settingsRevision);
        callbacks.current.onTestSucceeded();
      } else {
        setError({
          kind: 'delivery',
          diagnostic: response.error ?? '',
          isSlack: latestDraft.current.settings.type === NotifierType.SLACK,
        });
      }
    } catch (error) {
      if (
        isCurrentSession(authSession) &&
        session.IsCurrent(sessionRevision) &&
        settings.IsCurrent(settingsRevision) &&
        test.IsCurrent(testRevision)
      )
        setError({ kind: 'api', error });
    } finally {
      if (activeTest.current === request) {
        activeTest.current = undefined;
        resetRequest();
      }
    }
  };

  const save = async () => {
    if (activeSave.current || testedSettingsRevision !== revisions.current.settings.GetCurrent())
      return;
    const submittedDraft = structuredClone(latestDraft.current);

    if (
      !submittedDraft.name.trim() ||
      !isNotifierSettingsFilled(
        submittedDraft.settings,
        !!savedIdRef.current,
        savedIdRef.current ? baselineRef.current.settings : undefined,
      )
    )
      return;

    const authSession = getSessionSnapshot();
    const session = revisions.current.session;
    const sessionRevision = session.GetCurrent();
    const submittedRevision = revisions.current.draft.GetCurrent();

    const request = Symbol();
    activeSave.current = request;
    setIsSaving(true);
    setError(undefined);

    try {
      const savedNotifier = savedIdRef.current
        ? await updateNotifierMutation.mutateAsync([
            savedIdRef.current,
            submittedDraft.name,
            submittedDraft.settings,
          ])
        : await createNotifierMutation.mutateAsync([
            workspaceId,
            submittedDraft.name,
            submittedDraft.settings,
          ]);
      if (!isCurrentSession(authSession) || !session.IsCurrent(sessionRevision)) return;

      savedIdRef.current = savedNotifier.id;
      setSavedNotifierId(savedNotifier.id);

      const acknowledgedDraft = { name: savedNotifier.name, settings: savedNotifier.settings };
      baselineRef.current = acknowledgedDraft;
      setBaseline(acknowledgedDraft);

      const currentDraft = revisions.current.draft.IsCurrent(submittedRevision)
        ? acknowledgedDraft
        : {
            ...latestDraft.current,
            name:
              latestDraft.current.name === submittedDraft.name
                ? savedNotifier.name
                : latestDraft.current.name,
            settings: acknowledgeNotifierSettings({
              settings: latestDraft.current.settings,
              submittedSettings: submittedDraft.settings,
              savedSettings: savedNotifier.settings,
            }),
          };

      latestDraft.current = currentDraft;
      setDraft(currentDraft);
      callbacks.current.onPersisted(savedNotifier);

      if (isNotifierDraftUnchanged({ draft: currentDraft, baseline: acknowledgedDraft }))
        callbacks.current.onChanged(savedNotifier);
    } catch (error) {
      if (isCurrentSession(authSession) && session.IsCurrent(sessionRevision))
        setError({ kind: 'api', error });
    } finally {
      if (activeSave.current === request) {
        activeSave.current = undefined;
        setIsSaving(false);
      }
    }
  };

  useLayoutEffect(() => {
    callbacks.current = { onPersisted, onChanged, onTestSucceeded };
  }, [onPersisted, onChanged, onTestSucceeded]);

  useLayoutEffect(() => {
    const hasNewContext =
      context.current.workspaceId !== workspaceId ||
      (context.current.inputId !== editingNotifier?.id &&
        savedIdRef.current !== editingNotifier?.id);
    context.current = { workspaceId, inputId: editingNotifier?.id };
    const isClean = isNotifierDraftUnchanged({
      draft: latestDraft.current,
      baseline: baselineRef.current,
    });

    if (!hasNewContext && (!editingNotifier || !isClean || activeSave.current)) return;

    revisions.current.session.Invalidate();
    revisions.current.settings.Invalidate();
    revisions.current.test.Invalidate();
    revisions.current.draft.Invalidate();
    activeSave.current = undefined;
    activeTest.current = undefined;

    const initialDraft = createDraft(editingNotifier);
    latestDraft.current = initialDraft;
    baselineRef.current = initialDraft;
    savedIdRef.current = editingNotifier?.id;
    setDraft(initialDraft);
    setBaseline(initialDraft);
    setSavedNotifierId(editingNotifier?.id);
    setTestedSettingsRevision(undefined);
    setError(undefined);
    setIsSaving(false);
    resetRequest();
  }, [workspaceId, editingNotifier, resetRequest]);

  useLayoutEffect(() => {
    const { session: sessionRevision, test: testRevision } = revisions.current;
    const invalidateSession = () => {
      sessionRevision.Invalidate();
      testRevision.Invalidate();
      activeSave.current = undefined;
      activeTest.current = undefined;
      setIsSaving(false);
      resetRequest();
      setTestedSettingsRevision(undefined);
      setError(undefined);
    };
    const unsubscribe = subscribeSession(invalidateSession);

    return () => {
      unsubscribe();
      sessionRevision.Invalidate();
      activeSave.current = undefined;
      activeTest.current = undefined;
    };
  }, [resetRequest]);

  return {
    ...draft,
    savedNotifierId,
    savedSettings: savedNotifierId !== undefined ? baseline.settings : undefined,
    isSaving,
    isTesting,
    error,
    isUnsaved: !isNotifierDraftUnchanged({ draft, baseline }),
    // eslint-disable-next-line react/refs -- every invalidation comes with a state update, so this read cannot go stale
    isTested: testedSettingsRevision === revisions.current.settings.GetCurrent(),
    changeName,
    changeSettings,
    testSettings,
    save,
  };
};

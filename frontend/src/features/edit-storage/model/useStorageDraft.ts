import { useEffectEvent, useLayoutEffect, useRef, useState } from 'react';

import { type Storage, type StorageSettings, storageApi } from '@/entities/storages';
import { getSessionSnapshot, isCurrentSession, subscribeSession } from '@/shared/auth';
import { RequestRevision } from '@/shared/lib/RequestRevision';

import { createDefaultStorageSettings } from '../lib/createDefaultStorageSettings';
import { isStorageSettingsFilled } from '../lib/isStorageSettingsFilled';
import { type StorageDraft, isStorageDraftUnchanged } from '../lib/StorageDraft';
import { acknowledgeStorageSettings } from './acknowledgeStorageSettings';

interface Options {
  workspaceId: string;
  editingStorage?: Storage;
  draft?: StorageDraft;
  onPersisted?: (storage: Storage) => void;
  onChanged: (storage: Storage) => void;
  onTestSucceeded: () => void;
}

type StorageError = { kind: 'api'; error: unknown } | { kind: 'connection'; diagnostic: string };
const createDraft = ({
  editingStorage,
  draft,
}: Pick<Options, 'editingStorage' | 'draft'>): StorageDraft =>
  draft
    ? structuredClone(draft)
    : {
        storageId: editingStorage?.id,
        name: editingStorage?.name ?? '',
        settings: editingStorage
          ? structuredClone(editingStorage.settings)
          : createDefaultStorageSettings('LOCAL'),
      };

export function useStorageDraft(options: Options) {
  const [draft, setDraft] = useState(() => createDraft(options));
  const [baseline, setBaseline] = useState(draft);
  const [storageId, setStorageId] = useState(draft.storageId);
  const [savedSettings, setSavedSettings] = useState(options.editingStorage?.settings);
  const [isSaving, setIsSaving] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testedRevision, setTestedRevision] = useState<number>();
  const [error, setError] = useState<StorageError>();

  const changeName = (name: string) => {
    currentDraft.current = { ...currentDraft.current, name };
    setDraft(currentDraft.current);
  };
  const invalidateTest = () => {
    revisions.current.test.Invalidate();
    activeTest.current = undefined;
    setIsTesting(false);
    setTestedRevision(undefined);
    setError(undefined);
  };
  const changeSettings = (settings: StorageSettings) => {
    if (
      (persistedId.current || activeSave.current) &&
      settings.type !== currentDraft.current.settings.type
    )
      return;

    revisions.current.settings.Invalidate();
    invalidateTest();
    currentDraft.current = { ...currentDraft.current, settings };
    setDraft(currentDraft.current);
  };
  const testConnection = async () => {
    if (
      activeTest.current ||
      !isStorageSettingsFilled(currentDraft.current.settings, !!persistedId.current, savedSettings)
    )
      return;

    const session = getSessionSnapshot();
    const contextRevision = revisions.current.context.GetCurrent();
    const settingsRevision = revisions.current.settings.GetCurrent();
    const testRevision = revisions.current.test.Invalidate();
    const request = Symbol();

    const isActive = () =>
      isCurrentSession(session) &&
      revisions.current.context.IsCurrent(contextRevision) &&
      revisions.current.settings.IsCurrent(settingsRevision) &&
      revisions.current.test.IsCurrent(testRevision);

    activeTest.current = request;
    setIsTesting(true);
    setError(undefined);

    try {
      const response = await storageApi.testStorageSettings(
        options.workspaceId,
        structuredClone(currentDraft.current.settings),
        persistedId.current,
      );
      if (!isActive()) return;

      if (response.isConnected) {
        setTestedRevision(settingsRevision);
        callbacks.current.onTestSucceeded();
      } else setError({ kind: 'connection', diagnostic: response.error ?? '' });
    } catch (failure) {
      if (isActive()) setError({ kind: 'api', error: failure });
    } finally {
      if (activeTest.current === request) {
        activeTest.current = undefined;
        setIsTesting(false);
      }
    }
  };
  const save = async () => {
    if (activeSave.current || testedRevision !== revisions.current.settings.GetCurrent()) return;

    const submitted = structuredClone(currentDraft.current);
    const submittedTestRevision = revisions.current.test.GetCurrent();

    if (
      !submitted.name.trim() ||
      !isStorageSettingsFilled(submitted.settings, !!persistedId.current, savedSettings)
    )
      return;

    const session = getSessionSnapshot();
    const contextRevision = revisions.current.context.GetCurrent();

    const request = Symbol();
    activeSave.current = request;
    setIsSaving(true);
    setError(undefined);

    try {
      const saved = persistedId.current
        ? await storageApi.updateStorage(persistedId.current, submitted.name, submitted.settings)
        : await storageApi.createStorage(options.workspaceId, submitted.name, submitted.settings);
      if (!isCurrentSession(session) || !revisions.current.context.IsCurrent(contextRevision))
        return;

      persistedId.current = saved.id;
      setStorageId(saved.id);
      setSavedSettings(saved.settings);

      const acknowledged: StorageDraft = {
        storageId: saved.id,
        name: saved.name,
        settings: saved.settings,
      };

      const merged: StorageDraft = {
        storageId: saved.id,
        name: currentDraft.current.name === submitted.name ? saved.name : currentDraft.current.name,
        settings: acknowledgeStorageSettings({
          current: currentDraft.current.settings,
          submitted: submitted.settings,
          saved: saved.settings,
        }),
      };

      if (!revisions.current.test.IsCurrent(submittedTestRevision)) invalidateTest();

      currentDraft.current = merged;
      baselineRef.current = acknowledged;
      setBaseline(acknowledged);
      setDraft(merged);

      callbacks.current.onPersisted?.(saved);

      if (isStorageDraftUnchanged({ draft: merged, baseline: acknowledged }))
        callbacks.current.onChanged(saved);
    } catch (failure) {
      if (isCurrentSession(session) && revisions.current.context.IsCurrent(contextRevision))
        setError({ kind: 'api', error: failure });
    } finally {
      if (activeSave.current === request) {
        activeSave.current = undefined;
        setIsSaving(false);
      }
    }
  };

  const revisions = useRef({
    context: new RequestRevision(),
    settings: new RequestRevision(),
    test: new RequestRevision(),
  });
  const currentDraft = useRef(draft);
  const baselineRef = useRef(baseline);
  const persistedId = useRef(storageId);
  const activeTest = useRef<symbol | undefined>(undefined);
  const activeSave = useRef<symbol | undefined>(undefined);
  const callbacks = useRef(options);
  const context = useRef({
    workspaceId: options.workspaceId,
    inputId: options.editingStorage?.id,
    draft: options.draft,
  });
  useLayoutEffect(() => {
    callbacks.current = options;
  });

  const createInitialDraft = useEffectEvent(() => createDraft(options));

  useLayoutEffect(() => {
    const hasNewContext =
      context.current.workspaceId !== options.workspaceId ||
      (context.current.inputId !== options.editingStorage?.id &&
        persistedId.current !== options.editingStorage?.id) ||
      context.current.draft !== options.draft;
    context.current = {
      workspaceId: options.workspaceId,
      inputId: options.editingStorage?.id,
      draft: options.draft,
    };

    if (
      !hasNewContext &&
      (!options.editingStorage ||
        !isStorageDraftUnchanged({ draft: currentDraft.current, baseline: baselineRef.current }) ||
        activeSave.current)
    )
      return;

    revisions.current.context.Invalidate();
    revisions.current.settings.Invalidate();
    invalidateTest();
    activeSave.current = undefined;

    const initial = createInitialDraft();
    currentDraft.current = initial;
    baselineRef.current = initial;
    persistedId.current = initial.storageId;
    setStorageId(initial.storageId);
    setDraft(initial);
    setBaseline(initial);
    setSavedSettings(options.editingStorage?.settings);
    setIsSaving(false);
  }, [options.workspaceId, options.editingStorage, options.draft]);
  useLayoutEffect(() => {
    const { context: contextRevision, test: testRevision } = revisions.current;
    const cancel = () => {
      contextRevision.Invalidate();
      testRevision.Invalidate();
      activeTest.current = undefined;
      activeSave.current = undefined;
      setIsSaving(false);
      setIsTesting(false);
      setTestedRevision(undefined);
    };
    const unsubscribe = subscribeSession(cancel);

    return () => {
      unsubscribe();
      contextRevision.Invalidate();
      testRevision.Invalidate();
      activeTest.current = undefined;
      activeSave.current = undefined;
    };
  }, []);

  return {
    name: draft.name,
    settings: draft.settings,
    storageId,
    savedSettings,
    isSaving,
    isTesting,
    error,
    isUnsaved: !storageId || !isStorageDraftUnchanged({ draft, baseline }),
    // eslint-disable-next-line react/refs -- every invalidation comes with a state update, so this read cannot go stale
    isTested: testedRevision === revisions.current.settings.GetCurrent(),
    changeName,
    changeSettings,
    testConnection,
    save,
  };
}

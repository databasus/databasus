import { useEffect, useEffectEvent, useRef } from 'react';

import { getApiClient, requireData } from '../api';
import type { ApiComponents } from '../api';
import type { PagedListParams } from './usePagedList';
import { usePagedList } from './usePagedList';
import { usePolledPagedList } from './usePolledPagedList';

type Workspace = ApiComponents['schemas']['Workspace'];
interface Props {
  resetKey?: string;
  onError?: (error: unknown) => void;
  onPublish?: (list: ReturnType<typeof usePagedList<Workspace, string>>) => void;
}

const useListParameters = ({
  resetKey = 'first',
  onError = () => {},
}: Props): PagedListParams<Workspace, string> => {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  return {
    resetKey,
    scrollContainerRef,
    onError,
    getItemKey: (workspace) => workspace.id,
    fetchPage: async (after) =>
      requireData(await getApiClient().GET('/workspaces', { params: { query: { after } } })),
  };
};

const ListScreen = ({
  list,
  parameters,
  onPublish,
}: {
  list: ReturnType<typeof usePagedList<Workspace, string>>;
  parameters: PagedListParams<Workspace, string>;
  onPublish?: Props['onPublish'];
}) => {
  const { scrollContainerRef } = parameters;

  const publishList = useEffectEvent(() => {
    onPublish?.(list);
  });

  useEffect(() => {
    publishList();
  }, [
    list.items,
    list.isLoading,
    list.isLoadingMore,
    onPublish,
    list.reload,
    list.refreshFirstPage,
    list.replaceOrPrependItem,
  ]);
  return (
    <div ref={scrollContainerRef} data-testid="scroll-list">
      <output>
        {list.isLoading ? 'Loading' : 'Ready'}:{list.isLoadingMore ? 'More' : 'Idle'}
      </output>
      {list.items.map((workspace) => (
        <p key={workspace.id}>{workspace.name}</p>
      ))}
      <button onClick={() => void list.reload()}>Reload</button>
      <button onClick={() => void list.refreshFirstPage()}>Refresh</button>
    </div>
  );
};

export const PagedListScreen = (props: Props) => {
  const parameters = useListParameters(props);
  const list = usePagedList(parameters);
  return <ListScreen list={list} parameters={parameters} onPublish={props.onPublish} />;
};

export const PolledPagedListScreen = (props: Props) => {
  const parameters = useListParameters(props);
  const list = usePolledPagedList({ ...parameters, refreshIntervalMs: 1_000 });
  return <ListScreen list={list} parameters={parameters} onPublish={props.onPublish} />;
};

import { CancelledError, useMutation, type MutationFunction } from '@tanstack/react-query';
import { useCallback } from 'react';

import { getSessionSnapshot } from '@/shared/auth';

export const useSessionMutation = <Data, Variables>(options: {
  mutationFn: MutationFunction<Data, Variables>;
}) => {
  const mutation = useMutation({
    mutationFn: (
      { generation, variables }: { generation: number; variables: Variables },
      context,
    ) => {
      if (getSessionSnapshot().generation !== generation) throw new CancelledError();

      return options.mutationFn(variables, context);
    },
  });
  const { mutate, mutateAsync } = mutation;
  const run = useCallback(
    (variables: Variables) =>
      mutateAsync({ generation: getSessionSnapshot().generation, variables }),
    [mutateAsync],
  );
  const dispatch = useCallback(
    (variables: Variables) => mutate({ generation: getSessionSnapshot().generation, variables }),
    [mutate],
  );

  return {
    ...mutation,
    variables: mutation.variables?.variables,
    mutate: dispatch,
    mutateAsync: run,
  };
};

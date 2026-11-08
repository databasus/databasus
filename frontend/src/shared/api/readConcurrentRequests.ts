export const readConcurrentRequests = async <Data>(
  signal: AbortSignal,
  read: (signal: AbortSignal) => Promise<Data>,
): Promise<Data> => {
  const groupCancellation = new AbortController();
  const groupSignal = AbortSignal.any([signal, groupCancellation.signal]);

  try {
    return await read(groupSignal);
  } catch (error) {
    groupCancellation.abort();
    throw error;
  }
};

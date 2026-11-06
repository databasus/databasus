import { useState } from 'react';

import { RequestRevision } from '@/shared/lib/RequestRevision';

export const useRequestRevision = (): RequestRevision => {
  const [requestRevision] = useState(() => new RequestRevision());

  return requestRevision;
};

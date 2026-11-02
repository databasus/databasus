import { useState } from 'react';

import { RequestRevision } from '../lib/RequestRevision';

export const useRequestRevision = (): RequestRevision => {
  const [requestRevision] = useState(() => new RequestRevision());

  return requestRevision;
};

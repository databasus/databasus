import { HttpResponse } from 'msw';

const pendingResponses = new Set<() => void>();

export const createPendingApiResponse = () => {
  let resolveResponse: (response: Response) => void = () => {};
  const response = new Promise<Response>((resolve) => {
    resolveResponse = resolve;
  });
  const respond = (value: Response) => {
    pendingResponses.delete(cancel);
    resolveResponse(value);
  };
  const cancel = () => respond(HttpResponse.error());
  pendingResponses.add(cancel);
  return { response, respond, cancel };
};

export const cancelPendingApiResponses = () => {
  for (const cancel of pendingResponses) cancel();
};

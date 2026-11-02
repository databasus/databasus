import { type RefObject, useLayoutEffect, useRef } from 'react';

export const useLatestRef = <Value>(value: Value): RefObject<Value> => {
  const latestValue = useRef(value);

  useLayoutEffect(() => {
    latestValue.current = value;
  });

  return latestValue;
};

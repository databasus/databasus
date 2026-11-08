import { useEffect, useState } from 'react';

// The age of the newest sample is shown to the second, so the clock it is measured against ticks
// every second.
export const useCurrentTime = (): Date => {
  const [currentTime, setCurrentTime] = useState(() => new Date());

  useEffect(() => {
    const interval = setInterval(() => setCurrentTime(new Date()), 1000);

    return () => clearInterval(interval);
  }, []);

  return currentTime;
};

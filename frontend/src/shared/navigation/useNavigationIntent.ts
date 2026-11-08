import { useState } from 'react';
import { useLocation, useNavigationType } from 'react-router';

export function useNavigationIntent(): string {
  const { key } = useLocation();
  const navigationType = useNavigationType();
  const [navigation, setNavigation] = useState({ locationKey: key, intentKey: key });
  const intentKey = navigationType === 'REPLACE' ? navigation.intentKey : key;
  if (navigation.locationKey !== key) setNavigation({ locationKey: key, intentKey });

  return intentKey;
}

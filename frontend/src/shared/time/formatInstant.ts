import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

import { getUserTimeFormat } from '@/shared/time/getUserTimeFormat';

dayjs.extend(utc);

export const formatInstant = (instant: string): string =>
  dayjs.utc(instant).local().format(getUserTimeFormat().format);

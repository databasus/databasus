import { Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { ReasonTexts } from '@/pages/databases/wal-monitoring/model/ReasonTexts';
import { MonitoringTextComponent } from '@/pages/databases/wal-monitoring/ui/MonitoringTextComponent';

interface Props {
  tips: ReasonTexts[];
}

// Tips never change the verdict, so they sit among the technical details as quiet notes instead of
// competing with it.
export const MonitoringTipsComponent = ({ tips }: Props) => {
  const { t } = useTranslation();

  if (tips.length === 0) {
    return null;
  }

  return (
    <ul className="space-y-3 text-sm">
      {tips.map((tip) => (
        <li key={tip.titleKey} className="flex gap-2">
          <span className="shrink-0 pt-0.5 text-muted" aria-hidden>
            <Info size={14} />
          </span>
          <div>
            <div className="font-semibold">{t(tip.titleKey)}</div>
            <div>
              <MonitoringTextComponent text={tip.whatHappened} />
            </div>
            <div className="text-muted">
              <MonitoringTextComponent text={tip.whatToDo} />
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
};

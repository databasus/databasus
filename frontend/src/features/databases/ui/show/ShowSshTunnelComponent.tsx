import { useTranslation } from 'react-i18next';

import {
  SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS,
  type SshTunnelSettings,
} from '../../../../entities/databases';

interface Props {
  sshTunnel: SshTunnelSettings;
}

export const ShowSshTunnelComponent = ({ sshTunnel }: Props) => {
  const { t } = useTranslation();

  if (!sshTunnel?.isEnabled) return null;

  return (
    <>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.host')}</div>
        <div>{sshTunnel.host}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.port')}</div>
        <div>{sshTunnel.port}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.username')}</div>
        <div>{sshTunnel.username}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.authType')}</div>
        <div>{sshTunnel.authType && t(SSH_TUNNEL_AUTH_TYPE_LABEL_KEYS[sshTunnel.authType])}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.sshTunnel.credentials')}</div>
        <div>*************</div>
      </div>
    </>
  );
};

/* eslint-disable i18next/no-literal-string -- shell commands and their token placeholder, identical in every language */
export type AgentArchitecture = 'amd64' | 'arm64';

export const TOKEN_PLACEHOLDER = '<YOUR_AGENT_TOKEN>';

export const buildInstallCommand = (arch: AgentArchitecture): string => {
  const host = window.location.origin;
  return `curl -L -o verification-agent "${host}/api/v1/system/verification-agent?arch=${arch}" && chmod +x verification-agent`;
};

export const buildLaunchCommand = (agentId: string, token: string): string => {
  const host = window.location.origin;

  return [
    './verification-agent start \\',
    `  --databasus-host=${host} \\`,
    `  --agent-id=${agentId} \\`,
    `  --token=${token} \\`,
    '  --max-cpu=2 \\',
    '  --max-ram-mb=2048 \\',
    '  --max-disk-gb=20 \\',
    '  --max-concurrent-jobs=1',
  ].join('\n');
};

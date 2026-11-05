export type SESAuthType = 'KEYS' | 'IAM_ROLE';

export interface SESNotifier {
  fromEmail: string;
  targetEmail: string;
  region: string;
  authType: SESAuthType;
  accessKey?: string;
  secretKey?: string;
  roleArn?: string;
  externalId?: string;
}

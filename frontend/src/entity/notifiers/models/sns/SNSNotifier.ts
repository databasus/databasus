export type SNSAuthType = 'KEYS' | 'IAM_ROLE';

export interface SNSNotifier {
  topicArn: string;
  region: string;
  authType: SNSAuthType;
  accessKey?: string;
  secretKey?: string;
  roleArn?: string;
  externalId?: string;
}

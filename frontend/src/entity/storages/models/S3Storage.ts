export type S3AuthType = 'KEYS' | 'IAM_ROLE';

export interface S3Storage {
  s3Bucket: string;
  s3Region: string;
  authType?: S3AuthType;
  s3AccessKey?: string;
  s3SecretKey?: string;
  roleArn?: string;
  externalId?: string;
  s3Endpoint?: string;
  s3Prefix?: string;
  s3UseVirtualHostedStyle?: boolean;
  skipTLSVerify?: boolean;
  s3StorageClass?: string;
}

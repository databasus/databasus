import { EmailNotifierSecurity } from './EmailNotifierSecurity';

const IMPLICIT_TLS_PORT = 465;

export const getDefaultEmailNotifierSecurity = (port: number): EmailNotifierSecurity =>
  port === IMPLICIT_TLS_PORT ? EmailNotifierSecurity.TLS : EmailNotifierSecurity.STARTTLS;

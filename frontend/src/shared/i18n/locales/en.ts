// The source of truth for the set of keys. Every other dictionary is typed `typeof en`.
// Keep it one object literal: see frontend/AGENTS.md - Interface languages (i18n).
export const en = {
  language: {
    nativeName: 'English',
  },
  common: {
    actions: {
      add: 'Add',
      back: 'Back',
      cancel: 'Cancel',
      close: 'Close',
      confirm: 'Confirm',
      continue: 'Continue',
      copy: 'Copy',
      create: 'Create',
      delete: 'Delete',
      download: 'Download',
      edit: 'Edit',
      ok: 'OK',
      refresh: 'Refresh',
      remove: 'Remove',
      retry: 'Retry',
      save: 'Save',
    },
    answers: {
      no: 'No',
      yes: 'Yes',
    },
    fields: {
      host: 'Host',
      name: 'Name',
      password: 'Password',
      port: 'Port',
      status: 'Status',
      type: 'Type',
      username: 'Username',
    },
    states: {
      disabled: 'Disabled',
      enabled: 'Enabled',
      loading: 'Loading...',
    },
    messages: {
      copiedToClipboard: 'Copied to clipboard',
    },
    // Joins translated words into one list (Chinese uses 、)
    listSeparator: ', ',
    confirmation: {
      title: 'Confirmation',
    },
  },
  app: {
    documentTitle: 'Databasus - databases backups (PostgreSQL, MySQL, MongoDB)',
    languageControl: {
      title: 'Language: {{language}}',
    },
    theme: {
      title: 'Theme: {{theme}}',
      light: 'Light',
      dark: 'Dark',
      system: 'System',
    },
    navigation: {
      dashboard: 'Dashboard',
      databases: 'Databases',
      storages: 'Storages',
      notifiers: 'Notifiers',
      workspaceSettings: 'Settings',
      profile: 'Profile',
      databasusSettings: 'Databasus settings',
      users: 'Users',
      docs: 'Docs',
      documentation: 'Documentation',
      community: 'Community',
      newVersionAvailable: 'New version available',
    },
    diskUsage: {
      title: 'Disk Usage',
      hint: 'To make backups locally and restore them, you need to have enough space on your disk. For restore, you need to have same amount of space that the backup size.',
      compactSummary: '{{used}} of {{total}} GB<lineBreak/>ROM used ({{percent}}%)',
      summary: '{{used}} of {{total}} GB used ({{percent}}%)',
    },
    workspaceSelection: {
      createShort: 'Create',
      create: 'Create workspace',
      selected: 'Selected workspace',
      placeholder: 'Select a workspace',
      search: 'Search workspaces...',
      notFound: 'No workspaces found',
      createNew: '+ Create new workspace',
    },
    sponsorship: {
      link: 'Sponsorship',
      hint: 'Databasus saved your data? You can support Databasus as well to keep it free, maintained and independent',
    },
    starButton: {
      label: 'Star on GitHub',
      ariaLabel: 'Star databasus/databasus on GitHub',
    },
    clipboardPaste: {
      title: 'Paste from clipboard',
      submit: 'Submit',
      description: 'Automatic clipboard access is not available. Please paste your content below.',
      placeholder: 'Paste your connection string here...',
    },
    auth: {
      accountsCheckFailed: 'Failed to check whether the instance has any account: {{error}}',
    },
    oauthCallback: {
      failedTitle: 'Authentication Failed',
      codeNotFound: 'Authorization code not found',
      stateMissing: 'OAuth state parameter missing',
      invalidProvider: 'Invalid OAuth provider',
      returnToSignIn: 'Return to sign in',
      completing: 'Completing authentication...',
    },
    oauthStorage: {
      configNotFound: 'Google Drive storage configuration not found',
      exchangeFailed: 'Failed to exchange OAuth code',
      unsupportedType: 'Unsupported storage type for OAuth',
      invalidState: 'OAuth state parameter is invalid',
      paramNotFound: 'OAuth param not found. Ensure the redirect URL is configured correctly.',
      addStorageTitle: 'Add storage',
      storageDescription:
        'Storage - is a place where backups will be stored (local disk, S3, etc.)',
    },
  },
  errors: {
    unknown: 'Something went wrong. Please try again.',
    networkUnreachable: 'Cannot reach the Databasus server. Check your connection and try again.',
    requestFailed:
      'The server could not complete the request (HTTP {{status}}). Please try again later.',
    signInCodeIncorrect: 'The code is incorrect.',
    pendingSignInNotUsable: 'This sign-in can no longer be completed. Sign in again.',
    signInCodeNotSent:
      'The instance could not send the sign-in code. Try again later or ask an administrator.',
    tooManySignInCodes:
      'Too many sign-in codes were requested for this account in the last hour. Try again later.',
    signInCodeResentTooSoon:
      'A code was sent less than a minute ago. Wait a minute before requesting another.',
    rateLimitExceeded: 'Too many attempts. Wait a little and try again.',
    emailNotConfigured: 'The instance has no mail server configured.',
    adminEmailMissing: 'Your account has no valid email address to send a test email to.',
  },
  status: {
    logicalBackup: {
      inProgress: 'In progress',
      completed: 'Successful',
      failed: 'Failed',
      deleted: 'Deleted',
      canceled: 'Canceled',
    },
    physicalBackup: {
      inProgress: 'In progress',
      completed: 'Successful',
      error: 'Error',
      chainBroken: 'Chain broken',
      canceled: 'Canceled',
    },
    restoreVerification: {
      verifiedSuccessful: 'Verified',
      verificationFailed: 'Verification failed',
    },
    restore: {
      inProgress: 'In progress',
      completed: 'Successful',
      failed: 'Failed',
      canceled: 'Canceled',
    },
    verification: {
      pending: 'Pending',
      running: 'Running',
      completed: 'Successful',
      failed: 'Failed',
      canceled: 'Canceled',
    },
    health: {
      available: 'Available',
      unavailable: 'Unavailable',
    },
    agent: {
      neverSeen: 'Never seen',
      online: 'Online',
      stale: 'Stale',
      offline: 'Offline',
    },
  },
  intervals: {
    types: {
      hourly: 'Hourly',
      daily: 'Daily',
      weekly: 'Weekly',
      monthly: 'Monthly',
      cron: 'Cron',
    },
    weekdays: {
      mon: 'Mon',
      tue: 'Tue',
      wed: 'Wed',
      thu: 'Thu',
      fri: 'Fri',
      sat: 'Sat',
      sun: 'Sun',
    },
    periods: {
      day: '1 day',
      week: '1 week',
      month: '1 month',
      threeMonths: '3 months',
      sixMonths: '6 months',
      year: '1 year',
      twoYears: '2 years',
      threeYears: '3 years',
      fourYears: '4 years',
      fiveYears: '5 years',
      forever: 'Forever',
    },
  },
  auditLogs: {
    title: 'Audit Logs',
    columns: {
      user: 'User',
      message: 'Message',
      workspace: 'Workspace',
      created: 'Created',
    },
    systemUser: 'System',
    loadedOfTotal: 'Loaded: {{loaded}} of {{total}}',
    empty: 'No audit logs found.',
    loadingMore: 'Loading more logs...',
    allLoaded: 'All logs loaded ({{count}} total)',
  },
  settings: {
    title: 'Databasus settings',
    externalRegistrations: {
      title: 'Allow external registrations',
      description:
        'When enabled, new users can register accounts in Databasus. If disabled, new users can only register via invitation',
    },
    memberInvitations: {
      title: 'Allow member invitations',
      description:
        'When enabled, existing members can invite new users to join Databasus. If not - only admins can invite users.',
    },
    memberWorkspaceCreation: {
      title: 'Members can create workspaces',
      description:
        'When enabled, members (non-admin users) can create new workspaces. If not - only admins can create workspaces.',
    },
    twoFactorAuth: {
      title: 'Enable two-factor sign-in',
      description:
        "When enabled, a password sign-in also has to present a six-digit code sent to the account's email address. Sign-in through Google or GitHub is not covered.",
      mailServerRequired: 'This needs a mail server. <docsLink>Configure SMTP</docsLink> first.',
    },
    saving: 'Saving...',
    saveChanges: 'Save Changes',
    reset: 'Reset',
    updated: 'Settings updated successfully',
    readMore: 'Read more about settings you can <docsLink>here</docsLink>',
    mailServer: {
      title: 'Mail server',
      configured: 'The mail server is <status>configured</status>.',
      notConfigured: 'The mail server is <status>not configured</status>.',
      description:
        'It sends invitations, password reset codes and sign-in codes. Email notifiers have SMTP settings of their own.',
      docs: 'See <docsLink>the SMTP documentation</docsLink> for how to connect a mail server.',
      sendTestEmail: 'Send test email',
      testEmailSent: 'Test email sent to {{email}}',
    },
    healthcheck: {
      title: 'Health-check',
      openInNewTab: 'Click to open in new tab',
      copied: 'Health-check endpoint copied to clipboard',
      description: "Use this endpoint to monitor your Databasus system's availability",
    },
  },
  dashboard: {
    tiles: {
      databases: 'Databases',
      backups: 'Backups',
      backupsSize: 'Total backup size',
      backupsSizeHint:
        'Counts successful backups only. For physical databases, WAL segments are included.',
      installation: 'All workspaces',
      installationHint:
        'Totals across every workspace of this Databasus instance. Only admins see them.',
      installationDetails: 'Databases: {{databasesCount}}, backups: {{backupsCount}}',
    },
    list: {
      title: 'Databases',
      empty: 'This workspace has no databases yet',
      columns: {
        database: 'Database',
        healthcheck: 'Healthcheck',
        backups: 'Backups',
        meanSize: 'Mean size',
        totalSize: 'Total size',
        lastBackup: 'Last backup',
        storage: 'Storage',
      },
      successfulBackups: 'Successful: {{successfulBackupsCount}}',
      failedBackups: 'Failed: {{failedBackupsCount}}',
      noBackups: 'No backups yet',
    },
  },
  databases: {
    types: {
      postgresql: 'PostgreSQL',
      mysql: 'MySQL',
      mariadb: 'MariaDB',
      mongodb: 'MongoDB',
    },
    sslModes: {
      disable: 'Disable',
      require: 'Require',
      verifyCa: 'Verify CA',
      verifyFull: 'Verify full',
    },
    physicalBackupTypes: {
      full: 'Full backups only',
      fullIncremental: 'Full + incremental',
      fullIncrementalWalStream: 'Full + incremental + WAL streaming',
    },
    list: {
      addDatabase: 'Add database',
      searchPlaceholder: 'Search database',
      noSearchResults: 'No databases found matching "{{query}}"',
      description: 'Database - is a thing we are backing up',
      backToList: '← Back to databases',
    },
    card: {
      storage: 'Storage: <storageName/>',
      lastBackup: 'Last backup {{relativeTime}}',
      hasBackupError: 'Has backup error',
    },
    tabs: {
      config: 'Config',
      backups: 'Backups',
      verifications: 'Verifications',
    },
    create: {
      title: 'Add database for backup',
      namePlaceholder: 'My favourite DB',
      creating: 'Creating database...',
      complete: 'Complete',
      backupType: {
        title: 'Choose backup type',
        logical: {
          title: 'Logical',
          description: 'Recommended for databases under 50 GB. Simpler to set up.',
        },
        physical: {
          title: 'Physical',
          description:
            'For databases over 50 GB. Enables point-in-time recovery and better RPO/RTO, but needs extra setup.',
        },
      },
    },
    config: {
      namePlaceholder: 'Enter name...',
      removing: 'Removing database...',
      lastBackupError: {
        title: 'Last backup error',
        errorLabel: 'The error:',
        clearHint: 'To clean this error (choose any):',
        clearByTestingConnection:
          '- test connection via button below (even if you updated settings);',
        clearByNextBackup: '- wait until the next backup is done without errors;',
      },
      sections: {
        database: 'Database settings',
        backupConfig: 'Backup config',
        healthcheck: 'Healthcheck settings',
        restoreVerification: 'Restore verification',
        notifiers: 'Notifiers settings',
      },
      copyConfirmation:
        'Are you sure you want to copy this database? A new database with the same settings will be created.',
      removeConfirmation:
        'Are you sure you want to remove this database? This action cannot be undone.',
      copied: {
        title: 'Database copied successfully!',
        description: '"{{name}}" has been created successfully',
      },
      connectionTestSucceeded: {
        title: 'Connection test successful!',
        description: 'Database connection tested successfully',
      },
    },
    actions: {
      testConnection: 'Test connection',
      replace: 'Replace',
    },
    fields: {
      databaseType: 'Database type',
      version: '{{engine}} version',
      databaseName: 'DB name',
      sslMode: 'SSL mode',
      useHttps: 'Use HTTPS',
      cpuCount: 'CPU count',
      backupType: 'Backup type',
      clientCertificate: 'Client certificate',
      clientKey: 'Client key',
      serverCaCertificate: 'Server CA certificate',
      includeSchemas: 'Include schemas',
      excludeTables: 'Exclude tables',
      skipUserMappings: 'Skip user mappings',
      excludeExtensions: 'Exclude extensions',
      restoreOwnership: 'Restore ownership',
      restorePrivileges: 'Restore privileges',
      excludeEvents: 'Exclude events',
      galeraReplication: 'Galera replication',
      useSrvConnection: 'Use SRV connection',
      directConnection: 'Direct connection',
      authDatabase: 'Auth database',
      excludeCollections: 'Exclude collections',
    },
    edit: {
      parseFromClipboard: 'Parse from clipboard',
      clipboardEmpty: 'Clipboard is empty',
      clipboardReadFailed: 'Failed to read clipboard. Please check browser permissions.',
      connectionStringParsed: 'Connection string parsed successfully',
      connectionStringParsedWithoutPassword:
        'Connection string parsed successfully. Please enter the password manually.',
      connectionTestPassed: {
        title: 'Connection test passed',
        description: 'You can continue with the next step',
      },
      ipWhitelistHint:
        'If your database uses IP whitelist, make sure Databasus server IP is added to the allowed list.',
      localhostHint:
        'Please <docsLink>read this document</docsLink> to study how to backup local database',
      advancedSettings: 'Advanced settings',
      placeholders: {
        host: 'Enter {{engine}} host',
        port: 'Enter {{engine}} port',
        username: 'Enter {{engine}} username',
        password: 'Enter {{engine}} password',
        databaseName: 'Enter {{engine}} database name',
      },
      cpuCountTooltip:
        'Number of CPU cores to use for backup and restore operations. Higher values may speed up operations but use more resources.',
      excludeTablesPlaceholder: 'No tables excluded',
      excludeTablesTooltip:
        'Table names to exclude from the backup. You can paste a list separated by commas or new lines.',
      serverCaCertificateTooltip:
        'Optional. When provided, the server certificate is verified against this CA (verify-ca / verify-full).',
      copyFailed: 'Failed to copy',
      postgresql: {
        supabaseHint:
          'Please <docsLink>read this document</docsLink> to study how to backup Supabase database',
        includeSchemasPlaceholder: 'All schemas (default)',
        excludeTablesTooltip:
          "Tables to exclude from the backup. Use 'tablename' or 'schema.tablename'. Glob patterns are supported (e.g. 'logs_*'). You can paste a list separated by commas or new lines.",
        skipUserMappingsTooltip:
          'Skip restoring user mappings (CREATE USER MAPPING statements). Enable this when the backup role cannot read the mapping credentials - otherwise they are dumped without options and break restore for FDWs like oracle_fdw.',
        excludeExtensionsTooltip:
          "Skip restoring extension definitions (CREATE EXTENSION statements). Enable this if you're restoring to a managed PostgreSQL service where extensions are managed by the provider.",
        restoreOwnershipTooltip:
          'Apply ALTER OWNER statements from the dump so restored objects keep their original owner. The connection user must be able to assign these roles - typically a superuser.',
        restorePrivilegesTooltip:
          'Apply GRANT and REVOKE statements from the dump so restored objects keep their original ACLs. The connection user must be able to grant to the referenced roles - typically a superuser.',
        backupTypeTooltips: {
          full: 'Periodic standalone full backups. Each backup is self-contained.',
          fullIncremental:
            'Full backups plus incremental ones that store only the changes since the previous backup. Smaller and faster.',
          fullIncrementalWalStream:
            'Adds continuous WAL streaming on top of full and incremental backups. Only this option enables point-in-time recovery (PITR), but it requires more space and slower in restore.',
        },
      },
      mariadb: {
        skipEvents: 'Skip events',
        skipEventsTooltip:
          'Skip backing up database events. Enable this if the event scheduler is disabled on your MariaDB server.',
        skipGaleraDisable: 'Skip disabling on restore',
        skipGaleraDisableTooltip:
          'By default Databasus runs SET SESSION wsrep_on=OFF during restore to avoid Galera writeset-size errors. That requires the SUPER privilege. Enable this to skip it if your managed provider denies SUPER - large restores may then hit Galera writeset limits.',
      },
      mongodb: {
        srvTooltip:
          'Enable for MongoDB Atlas SRV connections (mongodb+srv://). Port is not required for SRV connections.',
        srvOverTunnelTooltip:
          'Not available over an SSH tunnel - SRV resolves its own host list and bypasses the forwarded port.',
        directConnectionTooltip:
          'Connect directly to a single server, skipping replica set discovery. Useful when the server is behind a load balancer, proxy or tunnel.',
        directConnectionOverTunnelTooltip:
          'Forced on while the SSH tunnel is enabled - a tunnel exposes a single address, so replica set discovery cannot be used.',
        excludeCollectionsPlaceholder: 'No collections excluded',
        excludeCollectionsTooltip:
          'Collection names to exclude from the backup. You can paste a list separated by commas or new lines.',
      },
    },
    show: {
      galeraDisableSkipped: 'Skip disabling on restore',
    },
    sshTunnel: {
      authTypes: {
        password: 'Password',
        privateKey: 'Private key',
      },
      title: 'SSH tunnel',
      tooltip:
        'For a database inside a closed network. Databasus connects to the SSH host below, and that host reaches the database using the host and port above.',
      host: 'SSH host',
      hostTooltip:
        'The database host above is resolved by the SSH host, not by Databasus. Use 127.0.0.1 when the database runs on the SSH host itself.',
      port: 'SSH port',
      username: 'SSH username',
      usernamePlaceholder: 'Enter SSH username',
      authType: 'SSH auth',
      credentials: 'SSH credentials',
      password: 'SSH password',
      passwordPlaceholder: 'Enter SSH password',
      privateKey: 'SSH private key',
      keyPassphrase: 'Key passphrase',
      keyPassphrasePlaceholder: 'Only for an encrypted key',
    },
    notifiers: {
      title: 'Notifiers',
      description:
        'Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)',
      multipleHint: 'You can select several notifiers, notifications will be sent to all of them.',
      selectPlaceholder: 'Select notifiers',
      createNew: 'Create new notifier',
      addTitle: 'Add notifier',
      notifyTo: 'Notify to',
      empty: 'No notifiers configured',
    },
    readOnlyUser: {
      dialogTitle: 'Create read-only user',
      readOnly: {
        checking: 'Checking read-only user...',
        title: 'Create a read-only user for Databasus?',
        description:
          'A read-only user is a {{databaseType}} user with limited permissions that can only read data from your database, not modify it. This is recommended for backup operations because:',
        noWriteCredentials:
          '<bold>A read-only user allows to avoid storing credentials with write access at all</bold>. Even in the worst case of hacking, nobody will be able to corrupt your data.',
        create: 'Yes, create read-only user',
        skipTitle: 'Skip read-only user creation?',
        skipQuestion: 'Are you sure you want to skip creating a read-only user?',
      },
      replicationOnly: {
        checking: 'Checking replication-only user...',
        title: 'Create a replication-only user for Databasus?',
        description:
          'A replication-only user is a {{databaseType}} user with limited permissions that can only read data from your database, not modify it. This is recommended for backup operations because:',
        noWriteCredentials:
          '<bold>A replication-only user allows to avoid storing credentials with write access at all</bold>. Even in the worst case of hacking, nobody will be able to corrupt your data.',
        create: 'Yes, create replication-only user',
        skipTitle: 'Skip replication-only user creation?',
        skipQuestion: 'Are you sure you want to skip creating a replication-only user?',
      },
      reasons: {
        preventsModifications: 'it prevents accidental data modifications during backup',
        leastPrivilege: 'it follows the principle of least privilege',
        bestPractice: "it's a security best practice",
      },
      securityNote:
        'Databasus enforce enterprise-grade security (<docsLink>read in details here</docsLink>). However, it is not possible to be covered from all possible risks.',
      noWritePrivileges: 'Current user has <bold>no write privileges</bold>.',
      noWriteRoles: 'Current user has <bold>no write roles</bold>.',
      writePrivileges: 'Current user has the following write privileges:',
      writeRoles: 'Current user has the following write roles:',
      expand: '(expand)',
      collapse: '(collapse)',
      skip: 'Skip',
      skipRisk:
        'Using a user with full permissions for backups is not recommended and may pose security risks. Databasus is highly recommending you to not skip this step.',
      skipAdvice:
        "100% protection is never possible. It's better to be safe in case of 0.01% risk of full hacking. So it is better to follow the secure way with read-only user.",
      acceptRisks: 'Yes, I accept risks',
      continueSecurely: "Let's continue with the secure way",
      forcedWalRotationUnavailable:
        'This source would not grant EXECUTE on pg_switch_wal() to the new user, which continuous WAL streaming needs to keep the recovery point close to the present. Full and incremental backups work normally with these credentials.',
    },
    transfer: {
      title: 'Transfer database to another workspace',
      targetWorkspace: 'Target workspace',
      selectWorkspace: 'Select workspace',
      storage: {
        title: 'Storage',
        transferExisting: 'Transfer with existing storage',
        selectFromTarget: 'Select storage from target workspace',
        blocked:
          'Databases using this storage: {{count}}. Transfer is blocked because other databases depend on it.',
        canTransfer: 'Storage can be transferred',
        selectPlaceholder: 'Select storage',
        createNew: 'Create new storage',
        addTitle: 'Add storage',
        description:
          'Storage - is a place where backups will be stored (local disk, S3, Google Drive, etc.)',
      },
      notifiers: {
        title: 'Notifiers (optional)',
        transferWithDatabase: 'Transfer notifiers with database',
        selectFromTarget: 'Select notifiers from target workspace',
        willTransfer: 'Will be transferred:',
        willNotTransfer: 'Will NOT be transferred (used by other databases):',
        usedByDatabases: '{{name}} (used by databases: {{count}})',
        noneTransferred:
          'No notifiers will be transferred. You can select notifiers from the target workspace after transfer.',
        selectPlaceholder: 'Select notifiers (optional)',
      },
      submit: 'Transfer',
      transferred: {
        title: 'Database transferred successfully!',
        description: '"{{name}}" has been transferred to the new workspace',
      },
    },
    connectionString: {
      errors: {
        empty: 'Connection string is empty',
        unrecognizedFormat: 'Unrecognized connection string format',
        hostMissing: 'Host is missing from connection string',
        usernameMissing: 'Username is missing from connection string',
        passwordMissing: 'Password is missing from connection string',
        databaseMissing: 'Database name is missing from connection string',
        parseFailed: 'Failed to parse connection string',
        jdbc: {
          invalidFormat: 'Invalid JDBC connection string format. Expected: {{expectedFormat}}',
          queryParametersMissing:
            'JDBC connection string is missing query parameters (user and password)',
          usernameMissing: 'Username (user parameter) is missing from JDBC connection string',
          passwordMissing: 'Password parameter is missing from JDBC connection string',
          parseFailed: 'Failed to parse JDBC connection string',
        },
        keyValue: {
          hostMissing: 'Host is missing from connection string. Use host=hostname',
          usernameMissing: 'Username is missing from connection string. Use user=username',
          passwordMissing: 'Password is missing from connection string. Use password=yourpassword',
          databaseMissing: 'Database name is missing from connection string. Use database=database',
          parseFailed: 'Failed to parse key-value connection string',
        },
        libpq: {
          databaseMissing: 'Database name is missing from connection string. Use dbname=database',
          parseFailed: 'Failed to parse libpq connection string',
        },
      },
    },
    physicalConnectionErrors: {
      grantAsSuperuserNote: 'Grant it with the command below (run as a superuser).',
      applyAndRestartNote: 'Apply the change below, then restart PostgreSQL.',
      pgHbaNoEntry: {
        title: 'Replication is not allowed from this host',
        summary:
          'PostgreSQL has no pg_hba.conf entry that permits a replication connection from Databasus. Physical backups stream via replication, which needs a "host replication" rule - an ordinary "host all" rule does not cover it.',
        addLineNote:
          'Add the line below to pg_hba.conf. The first "all" matches any user and the second any host - narrow the user to the specific replication role and/or the host to a CIDR range if you want to restrict it.',
        reloadNote: 'After updating pg_hba.conf, <bold>reload PG</bold>.',
        managedNote:
          'On managed PostgreSQL (RDS / Azure / GCP) you cannot edit pg_hba.conf - enable external or replication access in the provider console instead.',
      },
      badCredentials: {
        title: 'Wrong username or password',
        summary:
          'PostgreSQL rejected the credentials. Double-check the username and password for this database.',
      },
      noReplicationPrivilege: {
        title: 'User cannot run replication',
        summary:
          'The user connected but lacks the REPLICATION privilege that physical backups require.',
        managedNote:
          'On AWS RDS use GRANT rds_replication TO {{username}}; on Azure / GCP enable replication for the role in the provider console.',
      },
      noWalSwitchPrivilege: {
        title: 'User cannot force a WAL segment switch',
        summary:
          'Continuous WAL streaming only uploads a segment once the source closes it, so on a rarely-written database the newest WAL stays on the source until the 16 MB segment fills. Forcing a switch on a timer is what keeps the recovery point between backups close to the present, and it needs EXECUTE on pg_switch_wal(). Full and incremental backups do not need it: they restore to the point each backup finished and replay no archived WAL.',
        switchBackupTypeNote:
          'Or switch the backup type to <bold>Full + incremental</bold>, which does not replay archived WAL.',
        managedNote:
          'On managed PostgreSQL (RDS / Azure / GCP) the function belongs to the platform superuser and a customer role cannot be granted EXECUTE on it, so continuous WAL streaming is unavailable there - use full + incremental backups instead.',
      },
      walLevelInvalid: {
        title: 'wal_level is too low',
        summary:
          'wal_level must be "replica" or "logical" for physical backups. It is currently set lower.',
        applyNote:
          'Apply the change below, then restart PostgreSQL (wal_level only takes effect after a restart).',
        managedNote: 'On managed PostgreSQL set wal_level in the provider parameter group.',
      },
      noWalSenders: {
        title: 'No WAL sender processes available',
        summary: 'max_wal_senders is 0, so PostgreSQL cannot stream WAL for backups.',
        managedNote: 'On managed PostgreSQL set max_wal_senders in the provider parameter group.',
      },
      noReplicationSlots: {
        title: 'No replication slots available',
        summary: 'max_replication_slots is 0, so PostgreSQL cannot allocate a slot for backups.',
        managedNote:
          'On managed PostgreSQL set max_replication_slots in the provider parameter group.',
      },
      walSummaryDisabled: {
        title: 'WAL summarization is off',
        summary:
          'summarize_wal must be on for incremental backups. It is currently off on this server.',
        applyNote: 'Apply the change below (no restart needed).',
        thenNote: 'and then',
        managedNote: 'On managed PostgreSQL set summarize_wal in the provider parameter group.',
      },
      customTablespaces: {
        title: 'Custom tablespaces are not supported',
        summary:
          'This cluster has tablespaces outside pg_default / pg_global, which physical backups cannot stream. Drop the custom tablespaces, or switch this database to logical backups.',
      },
      systemIdentifierMismatch: {
        title: 'This is a different cluster',
        summary:
          'The cluster at this address has a different system identifier than the one this backup was set up for. Point Databasus at the original cluster, or create a new database entry.',
      },
      connectionFailed: {
        title: 'Could not connect',
        summary:
          'Databasus could not open a replication connection. Check the host, port, SSL mode and that a firewall is not blocking the connection.',
      },
    },
  },
  restores: {
    list: {
      empty: 'No restores yet',
    },
    targetDatabase: {
      select: 'Select database to restore to',
      description:
        'Enter info of the database we will restore backup to. <underline>The empty database for restore should be created before the restore</underline>. During the restore, all the current data will be cleared',
      notInUseHint:
        'Make sure the database is not used right now (most likely you do not want to restore the data to the same DB where the backup was made)',
      submit: 'Restore to this DB',
    },
    card: {
      startedAt: 'Started at',
      duration: 'Duration',
      errorDetailsTooltip: 'Click to see error details',
      cancelTooltip: 'Cancel restore',
      expectedDurationHint:
        'Expected restoration time usually 3x-5x longer than the backup duration (sometimes less, sometimes more depending on data type)',
      expectedDurationLimit:
        'So it is expected to take up to {{duration}} (usually significantly faster)',
    },
    errorDetails: {
      title: 'Restore error details',
      copied: 'Error message copied to clipboard',
      excludeExtensionsTip:
        '<bold>💡 Tip:</bold> This error typically occurs when restoring to managed PostgreSQL services (like Yandex Cloud, AWS RDS or similar). Try enabling <bold>"Exclude extensions"</bold> in Advanced settings before restoring.',
    },
    cancelConfirmation: {
      warning:
        '<bold>⚠️ Warning:</bold> Cancelling this restore will likely leave your database in a corrupted or incomplete state. You will need to recreate the database before attempting another restore.',
      question: 'Are you sure you want to cancel?',
      confirm: 'Yes, cancel restore',
    },
  },
  healthcheck: {
    attempts: {
      title: 'Healthcheck attempts',
      period: 'Period',
      periods: {
        today: 'Today',
        sevenDays: '7 days',
        thirtyDays: '30 days',
        allTime: 'All time',
      },
      empty: 'No data yet',
    },
    config: {
      enable: 'Enable healthcheck',
      isEnabled: 'Is health check enabled',
      notifyWhenUnavailable: 'Notify when unavailable',
      checkIntervalMinutes: 'Check interval (minutes)',
      attemptsBeforeDown: 'Attempts before down',
      storeAttemptsDays: 'Store attempts (days)',
      tooltips: {
        enable: 'Enable or disable healthcheck monitoring for this database',
        notifyWhenUnavailable: 'Send notifications when database becomes unavailable',
        checkIntervalMinutes: 'How often to check database health (in minutes)',
        attemptsBeforeDown: 'Number of failed attempts before marking database as down',
        storeAttemptsDays: 'How many days to store health check attempt history',
      },
    },
  },
  workspaces: {
    roles: {
      owner: 'Owner',
      admin: 'Admin',
      member: 'Member',
      viewer: 'Viewer',
    },
    fields: {
      name: 'Workspace name',
      namePlaceholder: 'Enter workspace name',
    },
    create: {
      title: 'Create workspace',
      submit: 'Create workspace',
      defaultName: 'My workspace',
      description:
        'Workspace is a place where you group:<lineBreak/>- your databases;<lineBreak/>- storages (like local drive, S3, Google Drive, etc.)<lineBreak/>- notifiers (like email, Slack, Telegram, etc.);<lineBreak/>- access control (if you have team);',
      nameRequired: 'Please enter a workspace name',
      created: 'Workspace created successfully',
      permissionDenied: {
        title: 'Permission denied',
        description:
          "You don't have permission to create workspaces. Please ask the administrator to create the workspace for you.",
      },
    },
    settings: {
      title: 'Workspace settings',
      readOnly: "You don't have permission to modify these settings",
      nameRequired: 'Workspace name is required',
      saving: 'Saving...',
      saveChanges: 'Save Changes',
      reset: 'Reset',
      updated: 'Basic information updated successfully',
    },
    delete: {
      dangerZone: 'Danger Zone',
      title: 'Delete this workspace',
      description:
        'Once you delete a workspace, there is no going back. All data and resources associated with this workspace will be permanently removed.',
      submit: 'Delete workspace',
      deleting: 'Deleting...',
      notFound: 'Workspace not found',
      noPermission: 'You do not have permission to delete this workspace',
      hasDatabases:
        'Cannot delete workspace. Please remove all databases first. Databases found: {{count}}.',
      deleted: 'Workspace deleted successfully',
      confirmation: {
        title: 'Delete Workspace',
        question: 'Are you sure you want to delete the workspace <workspaceName/>?',
        warning:
          '<bold>This action cannot be undone.</bold> All data and associated resources will be permanently removed.',
        submit: 'Delete Workspace',
      },
    },
    members: {
      title: 'Users',
      count: 'Members: {{count}}',
      empty: 'No members found',
      emptyHint: 'Click "Add member" to get started',
      fields: {
        member: 'Member',
        role: 'Role',
        joined: 'Joined',
        actions: 'Actions',
        email: 'Email address',
        emailPlaceholder: 'Enter email address',
      },
      add: {
        button: 'Add member',
        title: 'Add member',
        submit: 'Add member',
        emailRequired: 'Email is required',
        hint: 'If the user exists, they will be added directly. Otherwise, an invitation will be sent.',
        added: 'Member added successfully',
      },
      invited: {
        title: 'User invited',
        sentTo: 'Invitation sent to {{email}}',
        description:
          'The user is not present in the system yet, but has been invited to the workspace. After the user signs up via specified email, they will automatically become a member of the workspace.',
      },
      roleChanged: 'Member role updated successfully',
      remove: {
        tooltip: 'Remove member',
        title: 'Remove member',
        question: 'Are you sure you want to remove "{{email}}" from this workspace?',
        removed: 'Member "{{email}}" removed successfully',
      },
    },
    transferOwnership: {
      button: 'Transfer ownership',
      title: 'Transfer workspace ownership',
      submit: 'Transfer ownership',
      warning:
        '<bold>Warning:</bold> This action cannot be undone. You will lose ownership of this workspace and the new owner will have full control.',
      noEligibleMembers:
        'No members available to transfer ownership to. You need to have at least one other member in the workspace to transfer ownership.',
      newOwner: 'Select new owner',
      newOwnerPlaceholder: 'Select a member to transfer ownership to',
      newOwnerHint: 'The selected member will become the workspace owner',
      memberRequired: 'Please select a member to transfer ownership to',
      memberNotFound: 'Selected member not found',
      transferred: 'Ownership transferred successfully',
    },
    auditLogs: {
      empty: 'No audit logs found for this workspace.',
    },
  },
  users: {
    roles: {
      admin: 'Admin',
      member: 'Member',
    },
    fields: {
      name: 'Name',
      email: 'Email',
      yourName: 'Your name',
      yourEmail: 'Your email',
      confirmPassword: 'Confirm password',
      newPassword: 'New Password',
    },
    validation: {
      nameRequired: 'Name is required',
      emailRequired: 'Email is required',
      passwordTooShort: 'Password must be at least {{minLength}} characters long',
      passwordsDoNotMatch: 'Passwords do not match',
    },
    oauth: {
      divider: 'or continue',
      continueWithGithub: 'Continue with GitHub',
      continueWithGoogle: 'Continue with Google',
      invalidConfiguration: 'Invalid OAuth configuration',
    },
    signIn: {
      title: 'Sign in',
      submit: 'Sign in',
      noAccount: "Don't have an account? <signUpLink>Sign up</signUpLink>",
      forgotPassword: 'Forgot password?',
    },
    signInCode: {
      title: 'Enter the code',
      description:
        'We sent a six-digit code to {{email}}. It stops working 10 minutes after it was sent.',
      code: 'Sign-in code',
      submit: 'Sign in',
      resend: 'Send another code',
      backToSignIn: 'Back to sign in',
      invalidCode: 'Code must be 6 digits',
      codeResent: 'A new code is on its way. The previous one no longer works.',
    },
    signUp: {
      title: 'Sign up',
      namePlaceholder: 'John Doe',
      submit: 'Sign up',
      administersInstance: 'This is the first account on this instance, so it will administer it.',
      hasAccount: 'Already have an account? <signInLink>Sign in</signInLink>',
    },
    requestPasswordReset: {
      title: 'Reset password',
      description: "Enter your email address and we'll send you a reset code.",
      submit: 'Send reset code',
      codeSent: 'If the email exists, a reset code has been sent',
      rememberPassword: 'Remember your password? <signInLink>Sign in</signInLink>',
    },
    resetPassword: {
      title: 'Reset Password',
      description: 'Enter the code sent to your email and your new password.',
      code: 'Reset Code',
      confirmPassword: 'Confirm Password',
      submit: 'Reset password',
      invalidCode: 'Code must be 6 digits',
      succeeded: 'Password reset successfully! Redirecting to sign in...',
      noCode: "Didn't receive a code? <requestCodeLink>Request new code</requestCodeLink>",
      backToSignIn: 'Back to sign in',
    },
    profile: {
      title: 'Profile',
      information: 'Profile Information',
      userId: 'User ID',
      namePlaceholder: 'Enter your name',
      emailPlaceholder: 'Enter your email',
      role: 'Role',
      saveChanges: 'Save changes',
      noChanges: 'No changes to save',
      updated: 'Profile updated successfully',
      logout: 'Logout',
      changePassword: {
        title: 'Change Password',
        newPasswordPlaceholder: 'Enter new password',
        confirmPassword: 'Confirm New Password',
        confirmPasswordPlaceholder: 'Confirm new password',
        passwordsDoNotMatch: 'New passwords do not match',
        changing: 'Changing password...',
        submit: 'Change password',
        signedInWithNewPassword: 'Successfully signed in with new password',
      },
    },
    list: {
      title: 'Databasus users',
      loadedOfTotal: 'Users: {{loaded}} of {{total}}',
      searchPlaceholder: 'Search by email or name...',
      empty: 'No users found.',
      allLoaded: 'All users loaded ({{count}} total)',
      columns: {
        user: 'User',
        systemRole: 'System role',
        isActive: 'Is active?',
        created: 'Created',
      },
      roleLabel: 'Role:',
      activeLabel: 'Active:',
      viewAuditLogs: 'View audit logs',
      activated: 'User activated successfully',
      deactivated: 'User deactivated successfully',
      roleChanged: 'User role changed successfully',
    },
    auditLogs: {
      title: 'User Audit Logs',
      empty: 'No audit logs found for this user.',
    },
  },
  verification: {
    triggers: {
      manual: 'Manual',
      scheduled: 'Scheduled',
    },
    notificationTypes: {
      verificationSuccess: 'Verification success',
      verificationFailed: 'Verification failed',
    },
    config: {
      scheduledVerification: 'Scheduled verification',
      scheduledVerificationTooltip:
        'Periodically restore the latest backup into an ephemeral Postgres container to verify it is restorable. <docsLink>Read here how it works</docsLink>',
      interval: 'Verification interval',
      afterBackup: 'After backup',
      afterBackupHint:
        'Runs automatically after each successful backup. If there are pending in the queue, they are canceled to not cause infinite queue when backups are faster than verifications.',
      weekday: 'Verification weekday',
      dayOfMonth: 'Verification day of month',
      timeOfDay: 'Verification time of day',
      cron: {
        expression: 'Cron expression (UTC)',
        format: 'Cron format: minute hour day month weekday (UTC)',
        examples: 'Examples:',
        exampleEverySunday: '- {{expression}} - Every Sunday at 4:00 AM UTC',
        exampleEverySixHours: '- {{expression}} - Every 6 hours',
        nextRun: 'Next run {{dateTime}}<lineBreak/>({{relativeTime}})',
        invalid: 'Invalid cron expression',
      },
      notifications: 'Notifications',
      noNotifications: 'None',
    },
    runs: {
      title: 'Restore verifications',
      description:
        'Each row is one attempt to restore a backup of this database into a temporary copy',
      noAgentsWarning:
        'No verification agents registered - please add it in Databasus settings tab in "Verification agents" section. <docsLink>Learn more</docsLink>',
      empty: 'No restore checks yet',
      allLoaded: 'All restore checks loaded ({{total}} total)',
      attempt: 'attempt {{attempt}}',
      restoreDuration: 'restore {{duration}}',
      durationTooltip: 'Restore + verify time reported by the agent.',
      fields: {
        createdAt: 'Created at',
        trigger: 'Trigger',
        duration: 'Duration',
        actions: 'Actions',
      },
      actions: {
        cancel: 'Cancel verification',
        viewDetails: 'View details',
      },
      details: {
        title: 'Restore check details',
        loadFailed: 'Could not load restore check details.',
        cancellationReason: 'Cancellation reason',
        failure: 'Failure',
        sections: {
          status: 'Status',
          timeline: 'Timeline',
          results: 'Results & diagnostics',
        },
        fields: {
          attempt: 'Attempt',
          startedAt: 'Started at',
          finishedAt: 'Finished at',
          restoreDuration: 'Restore duration',
          verifyDuration: 'Verify duration',
          restoredDbSize: 'Restored DB size',
          schemas: 'Schemas',
          tables: 'Tables',
          pgRestoreExitCode: 'pg_restore exit code',
        },
        tableStats: {
          title: 'Per-table row counts',
          empty: 'No per-table stats reported.',
          schema: 'Schema',
          table: 'Table',
          rows: 'Rows',
        },
      },
    },
    agents: {
      title: 'Verification agents',
      description:
        'Agents that run restore verifications to confirm a backup is restorable (<docsLink>read more</docsLink>)',
      empty: 'No agents registered yet.',
      columns: {
        capacity: 'Capacity',
        created: 'Created',
      },
      capacity: {
        cpu: '{{count}} CPU',
        ram: '{{count}} GB RAM',
        disk: '{{count}} GB disk',
        jobs: 'jobs: {{count}}',
        notReported: 'not yet reported',
      },
      actions: {
        viewInstallCommands: 'View install commands',
        rotateToken: 'Rotate token',
      },
      deleteConfirmation: 'Delete this agent?',
      deleted: 'Agent "{{name}}" deleted',
      copyFailed: 'Failed to copy',
      agentId: 'Agent ID:',
      create: {
        title: 'Create verification agent',
        tokenNotice: 'A token will be generated and shown exactly once on the next screen.',
        namePlaceholder: 'Agent name',
        nameRequired: 'Name is required',
      },
      rotate: {
        title: 'Rotate token',
        description:
          'Rotating the token for <agentName/> invalidates the existing token immediately. Any worker still using the old token will be rejected on its next heartbeat.',
      },
      token: {
        title: 'Agent token',
        tokenFor: 'Token for <agentName/>:',
        shownOnce: "Shown once. Store it securely - you won't be able to retrieve it again.",
        confirmSaved: "I've saved the token",
      },
      install: {
        title: 'Install commands',
        description:
          'Install commands for <agentName/>. Replace <tokenPlaceholder/> with the token you saved when you created or last rotated this agent.',
        architecture: 'Architecture',
        stepInstall: 'Step 1 - Install',
        stepLaunch: 'Step 2 - Launch',
        capacityDefaults:
          'The capacity values below are starting defaults - tune <code>--max-cpu</code>, <code>--max-ram-mb</code>, <code>--max-disk-gb</code> and <code>--max-concurrent-jobs</code> to the machine running the agent.',
        afterInstallation: 'After installation',
        runsInBackground: 'The agent runs in the background after <code>start</code>',
        checkStatus: 'Check status: <code>./verification-agent status</code>',
        viewLogs: 'View logs: <code>databasus-verification.log</code> in the working directory',
        stop: 'Stop the agent: <code>./verification-agent stop</code>',
      },
    },
  },
  notifiers: {
    types: {
      email: 'Email',
      telegram: 'Telegram',
      webhook: 'Webhook',
      slack: 'Slack',
      discord: 'Discord',
      teams: 'Teams',
      mattermost: 'Mattermost',
    },
    notificationTypes: {
      all: 'All',
      backupSuccess: 'Backup success',
      backupFailed: 'Backup failed',
      healthcheckSuccess: 'Healthcheck success',
      healthcheckFailed: 'Healthcheck failed',
      verificationSuccess: 'Verification success',
      verificationFailed: 'Verification failed',
    },
    mattermostDeliveryModes: {
      webhook: 'Incoming webhook',
      bot: 'Bot account',
    },
    list: {
      addNotifier: 'Add notifier',
      searchPlaceholder: 'Search notifier',
      noSearchResults: 'No notifiers found matching "{{query}}"',
      description:
        'Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)',
      backToList: '← Back to notifiers',
    },
    card: {
      notifyTo: {
        email: 'Notify to Email',
        telegram: 'Notify to Telegram',
        webhook: 'Notify to Webhook',
        slack: 'Notify to Slack',
        discord: 'Notify to Discord',
        teams: 'Notify to Teams',
        mattermost: 'Notify to Mattermost',
      },
      hasSendError: 'Has send error',
    },
    create: {
      title: 'Add notifier',
      namePlaceholder: 'Chat with me',
    },
    config: {
      namePlaceholder: 'Enter name...',
      settingsTitle: 'Notifier settings',
      sendError: {
        title: 'Send error',
        errorLabel: 'The error:',
        clearHint: 'To clean this error (choose any):',
        clearBySendingTest:
          '- send test notification via button below (even if you updated settings);',
        clearByNextNotification: '- wait until the next notification is sent without errors;',
      },
      removeConfirmation:
        'Are you sure you want to remove this notifier? This action cannot be undone.',
      removeBlockedByDatabases:
        'Notifier is used by some databases. Please remove the notifier from databases first.',
    },
    testNotification: {
      send: 'Send test notification',
      sent: {
        title: 'Test notification sent!',
        description: 'Test notification sent successfully',
      },
    },
    transfer: {
      title: 'Transfer notifier to another workspace',
      blockedByDatabases:
        'This notifier is used by some databases. Please transfer or remove related databases first.',
      description: 'Select a workspace to transfer this notifier to.',
      targetWorkspace: 'Target workspace',
      selectWorkspace: 'Select workspace',
      submit: 'Transfer',
    },
    fields: {
      botToken: 'Bot token',
      targetChatId: 'Target chat ID',
      webhookUrl: 'Webhook URL',
      channelId: 'Channel ID',
      skipTlsVerify: 'Skip TLS verify',
      skipTls: 'Skip TLS',
    },
    email: {
      targetEmail: 'Target email',
      targetEmailTooltip: 'The email where you want to receive the message',
      smtpHost: 'SMTP host',
      smtpPort: 'SMTP port',
      smtpUser: 'SMTP user',
      smtpPassword: 'SMTP password',
      smtpPasswordPlaceholder: 'password',
      from: 'From',
      fromTooltip:
        'Optional. Sender address, with or without a name: Acme Backups <noreply@example.com>. If empty, the SMTP user is used when it is an email address, otherwise noreply@ and the SMTP host',
      fromAuto: '(auto)',
      advancedSettings: 'Advanced settings',
      skipTlsTooltip:
        'Skip TLS certificate verification. Enable this if your SMTP server uses a self-signed certificate. Warning: this reduces security.',
      security: 'Connection security',
      securityTooltip:
        'TLS encrypts the connection from the start and is usual on port 465. STARTTLS upgrades the connection and fails if the server does not offer it. None sends the password and messages unencrypted - use it only for a trusted relay without STARTTLS.',
      securityModes: {
        tls: 'TLS',
        starttls: 'STARTTLS',
        none: 'None (unencrypted)',
      },
      heloName: 'Greeting name',
      heloNamePlaceholder: 'Automatic',
      heloNameTooltip:
        'The name Databasus introduces itself with to the mail server (EHLO). If empty, the host from DATABASUS_URL is used, or the machine host name.',
    },
    telegram: {
      botTokenHelpLink: 'How to get Telegram bot API token?',
      targetChatIdTooltip:
        'The chat where you want to receive the message (it can be your private chat or a group)',
      chatIdHelp: {
        toggle: 'How to get Telegram chat ID?',
        privateChat:
          'To get your chat ID, message <botLink>@getmyid_bot</botLink> in Telegram. <underline>Make sure you started chat with the bot</underline>',
        group:
          'If you want to get chat ID of a group, add your bot with <botLink>@getmyid_bot</botLink> to the group and write /start (you will see chat ID)',
      },
      useProxy: 'Use proxy',
      useProxyTooltip: 'Use a proxy for Telegram API requests',
      proxy: 'Proxy',
      proxyUrl: 'Proxy URL',
      proxyUrlTooltip: 'Supports http, https, socks5, socks5h. May include username and password',
      sendToGroupTopic: 'Send to group topic',
      sendToGroupTopicTooltip: 'Enable this to send messages to a specific thread in a group chat',
      threadId: 'Thread ID',
      threadIdTooltip: 'The ID of the thread where messages should be sent',
      topicId: 'Topic ID',
      threadIdHelp: {
        steps:
          'To get the thread ID, go to the thread in your Telegram group, tap on the thread name at the top, then tap “Thread Info”. Copy the thread link and take the last number from the URL.',
        example:
          '<bold>Example:</bold> If the thread link is <code>https://t.me/c/2831948048/3</code>, the thread ID is <code>3</code>',
        note: '<bold>Note:</bold> Thread functionality only works in group chats, not in private chats.',
      },
    },
    webhook: {
      method: 'Method',
      sendOn: 'Send on',
      sendOnPlaceholder: 'Select notification types',
      customHeaders: 'Custom headers',
      customHeadersTooltip:
        'Add custom HTTP headers to the webhook request (e.g., Authorization, X-API-Key)',
      savedHeadersHidden: '*Saved headers hidden for security reasons',
      headers: 'Headers',
      headerNamePlaceholder: 'Header name',
      headerValuePlaceholder: 'Header value',
      addHeader: 'Add header',
      hiddenHeaderValue: '(hidden)',
      bodyTemplate: {
        title: 'Body template',
        headingVariable: '<variable/> - notification title',
        messageVariable: '<variable/> - notification message',
        invalidJsonFormat: 'Invalid JSON format',
        invalidJson: 'Invalid JSON',
      },
      exampleRequest: {
        title: 'Example request',
        headers: 'Headers:',
      },
    },
    slack: {
      howToConnect: 'How to connect Slack (how to get bot token and chat ID)?',
      testFailedHint:
        'Make sure channel is public or bot is added to the private channel (via @invite) or group. For direct messages use User ID from Slack profile.',
    },
    discord: {
      channelWebhookUrl: 'Channel webhook URL',
      howTo: {
        title: 'How to get Discord webhook URL:',
        createChannel: '1. Create or select a Discord channel',
        openChannelSettings: '2. Go to channel settings (gear icon)',
        openIntegrations: '3. Navigate to Integrations',
        createWebhook: '4. Create a new webhook',
        copyWebhookUrl: '5. Copy the webhook URL',
        privacyNote: 'Note: make sure make channel private if needed',
      },
    },
    teams: {
      howToConnect: 'How to connect Microsoft Teams?',
      powerAutomateUrl: 'Power Automate URL',
      powerAutomateUrlLabel: 'Power Automate URL:',
      powerAutomateUrlTooltip:
        'HTTP endpoint from your Power Automate flow (When an HTTP request is received)',
      showFullUrl: 'Show',
      hideFullUrl: 'Hide',
    },
    mattermost: {
      howToConnect: 'How to connect Mattermost?',
      connectVia: 'Connect via',
      incomingWebhookUrl: 'Incoming webhook URL',
      serverUrl: 'Server URL',
      channelIdTooltip:
        '26-character channel ID, not the channel name. Open the channel, click its name and choose View Info.',
      channel: 'Channel',
      showOptionalSettings: 'Show optional settings',
      hideOptionalSettings: 'Hide optional settings',
      channelOverride: 'Channel override',
      channelOverrideTooltip:
        'Post to this channel instead of the one the webhook was created for. Mattermost ignores it when the webhook is locked to a channel.',
      postAsUsername: 'Post as username',
      postAsUsernameTooltip:
        'Requires Enable integrations to override usernames in the Mattermost system console, otherwise it is ignored.',
      postAsIconUrl: 'Post as icon URL',
      postAsIconUrlTooltip:
        'Requires Enable integrations to override profile picture icons in the Mattermost system console, otherwise it is ignored.',
      skipTlsTooltip:
        'Skip TLS certificate verification. Enable this if your Mattermost server uses a self-signed certificate. Warning: this reduces security.',
      webhookHowTo: {
        title: 'How to get an incoming webhook URL:',
        openIncomingWebhooks: '1. Main menu - Integrations - Incoming Webhooks',
        addWebhook: '2. Add Incoming Webhook, pick the channel',
        copyUrl: '3. Copy the generated URL',
      },
      botHowTo: {
        title: 'How to get a bot token:',
        addBotAccount: '1. Integrations - Bot Accounts - Add Bot Account',
        copyToken: '2. Copy the token shown once after creation',
        addBotToChannel: '3. Add the bot to the team and to the channel',
      },
    },
  },
  backups: {
    encryption: {
      none: 'None',
      encrypted: 'Enabled',
    },
    encryptionOptions: {
      none: 'None',
      encrypted: 'Encrypt backup files',
    },
    list: {
      title: 'Backups',
      scheduledBackupsDisabled:
        'Scheduled backups are disabled (you can enable it back in the backup configuration)',
      empty: 'No backups yet',
      allLoaded: 'All backups loaded ({{count}} total)',
      clickToSeeErrorDetails: 'Click to see error details',
      errorDetailsTitle: 'Backup error details',
      restoreTitle: 'Restore from backup',
      columns: {
        createdAt: 'Created at',
        created: 'Created',
        size: 'Size',
        duration: 'Duration',
        actions: 'Actions',
      },
      actions: {
        cancelBackup: 'Cancel backup',
        deleteBackup: 'Delete backup',
      },
    },
    filters: {
      allTypes: 'All types',
      allStatuses: 'All statuses',
      before: 'Before',
    },
    config: {
      backupsEnabled: 'Backups enabled',
      storage: 'Storage',
      selectStorage: 'Select storage',
      createNewStorage: 'Create new storage',
      addStorageTitle: 'Add storage',
      storageDescription:
        'Storage - is a place where backups will be stored (local disk, S3, Google Drive, etc.)',
      storageChangeWarning:
        'If you change the storage, all backups in this storage will be deleted.',
      storageChangeAcknowledge: 'I understand',
      encryption: 'Encryption',
      notifications: 'Notifications',
      noNotifications: 'None',
      schedule: {
        weekday: 'Weekday',
        dayOfMonth: 'Day of month',
        cronExpression: 'Cron expression (UTC)',
        timeOfDay: 'Time of day',
        nextRun: 'Next run {{time}}',
        invalidCron: 'Invalid cron expression',
        cronHelp: {
          format: 'Cron format: minute hour day month weekday (UTC)',
          examples: 'Examples:',
          dailyAt2am: 'Daily at 2:00 AM UTC',
          every6Hours: 'Every 6 hours',
          mondaysAt3am: 'Every Monday at 3:00 AM UTC',
          twiceAMonth: '1st and 15th at 4:30 AM UTC',
        },
      },
      gfs: {
        hourly: 'Hourly: {{count}}',
        daily: 'Daily: {{count}}',
        weekly: 'Weekly: {{count}}',
        monthly: 'Monthly: {{count}}',
        yearly: 'Yearly: {{count}}',
        notConfigured: 'Not configured',
      },
    },
    logical: {
      retentionPolicyTypes: {
        gfs: 'GFS (keep last N hourly, daily, weekly, monthly and yearly backups)',
        timePeriod: 'Time period (last N days)',
        count: 'Count (N last backups)',
      },
      notificationTypes: {
        backupSuccess: 'Backup success',
        backupFailed: 'Backup failed',
      },
      list: {
        backupNow: 'Backup now',
        makeBackupNow: 'Make backup right now',
        encrypted: 'Encrypted',
        sizeTooltip:
          'Top: the compressed backup file size we actually store in the storage (local, S3, Google Drive, etc.). Bottom (grayed): the original uncompressed database size at backup time. Backups are usually compressed ~5x.',
        databaseSize: '{{size}} (DB size)',
        deleteConfirmation: 'Are you sure you want to delete this backup?',
        actions: {
          restore: 'Restore from backup',
          verifyRestore:
            'Verify restore - queue an agent to restore this backup to a temporary database and report row counts',
          download: {
            postgresql:
              'Download backup file. It can be restored manually via pg_restore (from custom format)',
            mysql:
              'Download backup file. It can be restored manually via mysql client (from SQL dump)',
            mariadb:
              'Download backup file. It can be restored manually via mariadb client (from SQL dump)',
            mongodb:
              'Download backup file. It can be restored manually via mongorestore (from archive)',
            other: 'Download backup file',
          },
        },
        verifyRestore: {
          title: 'Verify restore?',
          description:
            'An agent will restore this backup to a temporary database and report row counts. This may take a while depending on backup size. <docsLink>How it works?</docsLink>',
          queue: 'Queue check',
          queued: 'Restore check queued',
        },
      },
      config: {
        interval: 'Backup interval',
        weekday: 'Backup weekday',
        dayOfMonth: 'Backup day of month',
        timeOfDay: 'Backup time of day',
        encryptionTooltip:
          "If backup is encrypted, backup files in your storage (S3, local, etc.) cannot be used directly. You can restore backups through Databasus or download them unencrypted via the 'Download' button.",
        retentionPolicy: 'Retention policy',
        retention: {
          timePeriodTooltip:
            'How long to keep the backups. Backups older than this period are automatically deleted.',
          count: 'Most recent backups',
          countTooltip:
            'Keep only the specified number of most recent backups. Older backups beyond this count are automatically deleted.',
          countValue: 'Most recent backups: {{count}}',
          showGfsHint: 'What is GFS (Grandfather-Father-Son)?',
          hideGfsHint: 'Hide',
          gfsHint:
            'GFS (Grandfather-Father-Son) rotation: keep the last N hourly, daily, weekly, monthly and yearly backups. This allows keeping backups over long periods of time within a reasonable storage space.',
          gfsTooltip:
            'Grandfather-Father-Son rotation: keep the last N hourly, daily, weekly, monthly and yearly backups.',
          gfsFields: {
            hourly: 'Hourly backups',
            daily: 'Daily backups',
            weekly: 'Weekly backups',
            monthly: 'Monthly backups',
            yearly: 'Yearly backups',
          },
        },
        advancedSettings: 'Advanced settings',
        retryIfFailed: 'Retry backup if failed',
        retryIfFailedShort: 'Retry if failed',
        retryIfFailedTooltip:
          'Automatically retry failed backups. Backups can fail due to network failures, storage issues or temporary database unavailability.',
        maxFailedTries: 'Max failed tries count',
        maxFailedTriesTooltip:
          'Maximum number of retry attempts for failed backups. You will receive a notification when all tries have failed.',
      },
    },
    physical: {
      types: {
        full: 'Full',
        incremental: 'Incremental',
        wal: 'WAL',
      },
      retentions: {
        chains: 'Chains',
        fullBackups: 'Full backups',
        chainsAndFullBackups: 'Chains and full backups',
      },
      fullBackupsPolicies: {
        lastN: 'Count',
        gfs: 'GFS',
      },
      notificationTypes: {
        backupSuccess: 'Backup success',
        backupFailed: 'Backup failed',
        chainBroken: 'Chain broken',
        walGap: 'WAL gap',
      },
      list: {
        totalUsage: 'Total usage: {{size}}',
        backUpNow: 'Back up now',
        fullBackup: 'Full backup',
        incrementalBackup: 'Incremental backup',
        restore: 'Restore',
        restoreFromThisBackup: 'Restore from this backup',
        pointInTimeRestoreTitle: 'Point-in-time restore',
        deleteFullConfirmation:
          'Deleting this full backup removes its entire chain (all incrementals that depend on it). This cannot be undone. Continue?',
        deleteIncrementalConfirmation:
          'Deleting this incremental backup removes all later incrementals that depend on it. This cannot be undone. Continue?',
      },
      config: {
        fullBackupCadence: 'Full backup cadence',
        incrementalBackupCadence: 'Incremental backup cadence',
        incrementalMoreFrequent: 'Incremental backups must run more frequently than full backups.',
        encryptionTooltip:
          'If backups are encrypted, the files in your storage cannot be used directly. You can restore them through Databasus or download them unencrypted.',
        retention: 'Retention',
        fullBackupsOnlyRetention:
          'This database keeps full backups only, so retention applies to full backups.',
        retentionOptions: {
          chains: 'Keep last N chains',
          chainsAndFullBackups: 'Keep last N chains and full backups',
        },
        retentionHelp: {
          chain:
            'A chain is one full backup plus the incrementals (and streamed WAL) that depend on it. Restoring needs the full backup and every incremental in its chain.',
          chains:
            'Keeps only the last N chains. When a chain rolls off, its full backup and all its incrementals are deleted - you can restore only within the kept chains.',
          chainsAndFullBackups:
            'Keeps the last N chains for recent point-in-time recovery, and additionally retains older standalone full backups by the policy below (GFS or last N) for long-term restore points after their incrementals are gone.',
        },
        chainsCount: 'Chains',
        chainsCountTooltip:
          'Number of most recent backup chains to keep. A chain is a full backup plus its incrementals.',
        fullBackups: 'Full backups',
        fullBackupsCount: 'Most recent full backups',
        gfsFields: {
          hourly: 'Hourly',
          daily: 'Daily',
          weekly: 'Weekly',
          monthly: 'Monthly',
          yearly: 'Yearly',
        },
        chainsKept: 'Chains kept',
        fullBackupsKept: 'Full backups kept',
        fullBackupsCountValue: 'Most recent: {{count}}',
        gfsTooltip:
          'Grandfather-Father-Son rotation: keep the last N hourly, daily, weekly, monthly and yearly full backups.',
      },
      restore: {
        errorHints: {
          downloadInProgress:
            'A restore download is already in progress for this database. Wait for it to finish, then try again.',
          targetTimeUnreachable:
            'The requested target time cannot be reached - there is a WAL gap or the time is out of the available range. Pick a different time or restore the latest available point.',
        },
        backupIntro:
          'Restore command for this backup. A full backup needs only its own file; an incremental one also needs its full backup and all earlier incrementals.',
        pointInTimeIntro:
          'Point-in-time restore. Pick a target time, or leave it empty to restore the latest available point.',
        targetTime: 'Target time',
        latestAvailable: 'Latest available',
        preparing: 'Preparing restore command...',
        linkExpiry:
          'The download link is single-use and expires in 15 minutes. If a command fails with a 401, reopen this dialog to mint a fresh one.',
        copied: 'Copied',
        methods: {
          script: 'Via script',
          manual: 'Manual',
        },
        environments: {
          host: 'Host PostgreSQL',
        },
        fields: {
          restoreDirectory: 'Restore directory',
          pgBinPath: 'PostgreSQL bin path',
          pgBinPathPlaceholder: '{{path}} (optional)',
          pgImage: 'PostgreSQL image',
        },
        scriptTitles: {
          host: 'Run on your restore host',
          docker: 'Run where Docker is available',
        },
        manualSteps: {
          download: 'Download the bundle',
          extract: 'Extract it',
          verify: 'Verify the transfer',
          reconstruct: 'Reconstruct the data directory',
          checkConfig: "Check the cluster's configuration files",
          wireUpRecovery: 'Decompress WAL and wire up recovery',
          recoveryParameters: 'If recovery stops on parameter settings',
        },
        beforeYouRun: {
          title: 'Before you run',
          host: {
            clientTools:
              'Install the PostgreSQL {{version}} client tools - set the bin path above if they are not on PATH.',
            zstd: 'zstd is required to decompress the backup.',
            zstdWithWal: 'zstd is required to decompress the backup and its WAL.',
            emptyDirectory: 'The restore directory must be empty and not an in-use cluster.',
          },
          docker: {
            image: 'Use a postgres:{{version}} image - the major version must match the source.',
            hostTools:
              'The host needs zstd, tar and curl: the download and all decompression - the base backup - run on the host, and only pg_combinebackup runs in the container, so the postgres image needs no extra tools.',
            hostToolsWithWal:
              'The host needs zstd, tar and curl: the download and all decompression - the base backup and the WAL - run on the host, and only pg_combinebackup runs in the container, so the postgres image needs no extra tools.',
          },
        },
        afterItFinishes: {
          title: 'After it finishes',
          host: {
            clusterLocation: 'The cluster is at <dataDir/>.',
            changeOwner: 'Own it as root: <command/>.',
            start: 'Start it as postgres: <command/>, or point data_directory at it.',
            walReplay:
              'PostgreSQL replays WAL and promotes - watch the log for recovery completion.',
          },
          docker: {
            clusterLocation: '<dataDir/> on the host is the restored cluster.',
            volumeMount:
              'PostgreSQL {{version}} keeps its data directory at <dataDir/>, under the volume <volumeDir/>. Mount the output dir as that volume root - this is what a docker-compose <code>./pgdata:/var/lib/postgresql</code> expects:',
            dataDirectoryMount: "Bind-mount the cluster at the image's data directory <dataDir/>:",
            ownership:
              'Ownership must match the postgres uid inside the image (999 in the official one).',
          },
          missingConfigFiles:
            'A base backup copies only the data directory. If your source PostgreSQL keeps its configuration outside it - the Debian/Ubuntu layout, <configDir/> - then <code>postgresql.conf</code>, <code>pg_hba.conf</code> and <code>pg_ident.conf</code> are not in the backup and the server will not start. Ask the source for their real paths with <code>psql -Atc "SHOW config_file"</code>, copy all three in, then comment out <code>data_directory</code>, <code>hba_file</code>, <code>ident_file</code>, <code>external_pid_file</code> and the <code>ssl</code> / <code>ssl_*</code> lines, create <code>conf.d</code>, and set <code>listen_addresses = \'*\'</code> for a container. If those files are present and you still get the same error, it is the mounted directory that is wrong, not the configuration.',
        },
      },
    },
  },
  storages: {
    types: {
      local: 'Local storage',
      s3: 'S3',
      googleDrive: 'Google Drive',
      nas: 'NAS',
      azureBlob: 'Azure Blob Storage',
      ftp: 'FTP',
      sftp: 'SFTP',
      rclone: 'Rclone',
    },
    s3StorageClasses: {
      default: 'Default (Standard)',
      standard: 'Standard',
      standardIa: 'Standard - Infrequent Access',
      onezoneIa: 'One Zone - Infrequent Access',
      intelligentTiering: 'Intelligent Tiering',
      reducedRedundancy: 'Reduced Redundancy',
      glacierIr: 'Glacier Instant Retrieval',
    },
    azureBlobAuthMethods: {
      accountKey: 'Account key',
      connectionString: 'Connection string',
    },
    sftpAuthMethods: {
      password: 'Password',
      privateKey: 'Private key',
    },
    list: {
      addStorage: 'Add storage',
      description: 'Storage - is a place where backups will be stored (local disk, S3, etc.)',
      backToList: '← Back to storages',
    },
    card: {
      type: {
        local: 'Type: local storage',
        s3: 'Type: S3',
        googleDrive: 'Type: Google Drive',
        nas: 'Type: NAS',
        azureBlob: 'Type: Azure Blob Storage',
        ftp: 'Type: FTP',
        sftp: 'Type: SFTP',
        rclone: 'Type: Rclone',
      },
      hasSaveError: 'Has save error',
    },
    config: {
      namePlaceholder: 'Enter name...',
      settingsTitle: 'Storage settings',
      lastSaveError: {
        title: 'Save error',
        errorLabel: 'The error:',
        clearHint: 'To clean this error (choose any):',
        clearByTestingConnection:
          '- test connection via button below (even if you updated settings);',
        clearByNextSave: '- wait until the next save is done without errors;',
      },
      removeConfirmation:
        'Are you sure you want to remove this storage? This action cannot be undone. If some backups are using this storage, they will be removed too.',
      removeBlockedByDatabases:
        'Storage is used by some databases. Please remove the storage from databases first.',
    },
    actions: {
      testConnection: 'Test connection',
    },
    connectionTestSucceeded: {
      title: 'Connection test successful!',
      description: 'Storage connection tested successfully',
    },
    transfer: {
      title: 'Transfer storage to another workspace',
      blockedByDatabases:
        'This storage is used by some databases. Please transfer or remove related databases first.',
      description: 'Select a workspace to transfer this storage to.',
      targetWorkspace: 'Target workspace',
      selectWorkspace: 'Select workspace',
      submit: 'Transfer',
    },
    fields: {
      path: 'Path',
      remotePath: 'Remote path',
      share: 'Share',
      domain: 'Domain',
      useSsl: 'Use SSL',
      useSslTls: 'Use SSL/TLS',
      skipTls: 'Skip TLS',
      skipTlsVerify: 'Skip TLS verify',
      skipHostKey: 'Skip host key',
      authMethod: 'Auth method',
      privateKey: 'Private key',
      credentials: 'Credentials',
      s3Bucket: 'S3 Bucket',
      region: 'Region',
      accessKey: 'Access key',
      secretKey: 'Secret key',
      endpoint: 'Endpoint',
      folderPrefix: 'Folder prefix',
      prefix: 'Prefix',
      blobPrefix: 'Blob prefix',
      virtualHost: 'Virtual host',
      storageClass: 'Storage class',
      connection: 'Connection',
      connectionString: 'Connection string',
      accountName: 'Account name',
      accountKey: 'Account key',
      containerName: 'Container name',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      userToken: 'User Token',
      config: 'Config',
    },
    edit: {
      namePlaceholder: 'My Storage',
      advancedSettings: 'Advanced settings',
      optionalPlaceholder: '{{example}} (optional)',
      usernamePlaceholder: 'username',
      passwordPlaceholder: 'password',
      remotePathTooltip: 'Remote directory path for storing backups (optional)',
      local: {
        diskSpaceWarning:
          'Be careful: with local storage you may run out of ROM memory. It is recommended to use S3 or unlimited storages',
      },
      s3: {
        cloudflareR2Guide: 'How to use with Cloudflare R2?',
        endpointTooltip: 'Custom S3-compatible endpoint URL (optional, leave empty for AWS S3)',
        prefixTooltip:
          "Optional prefix for all object keys (e.g., 'backups/' or 'my_team/'). May not work with some S3-compatible storages. Cannot be changed after creation (otherwise backups will be lost).",
        useVirtualHostedStyle: 'Use virtual-hosted style',
        virtualHostTooltip:
          'Use virtual-hosted-style URLs (bucket.s3.region.amazonaws.com) instead of path-style (s3.region.amazonaws.com/bucket). May be required if you see COS errors.',
        skipTls: 'Skip TLS',
        skipTlsVerifyTooltip:
          'Skip TLS certificate verification. Enable this if your S3-compatible storage uses a self-signed certificate. Warning: this reduces security.',
        storageClassTooltip:
          'S3 storage class for uploaded objects. Leave as default for Standard. Some providers offer cheaper classes like One Zone IA. Do not use Glacier/Deep Archive - files must be immediately accessible for restores.',
      },
      googleDrive: {
        guideLink: 'How to connect Google Drive?',
        authorize: 'Authorize',
      },
      nas: {
        shareWithSlashWarning:
          'Share must be a single share name. Use the Path field for subdirectories (e.g. Share: Databasus, Path: DB1)',
        useSslTooltip: 'Enable SSL/TLS encryption for secure connection',
        domainTooltip:
          'Windows domain name (optional, leave empty if not using domain authentication)',
        pathPlaceholder: '{{example}} (optional, no leading slash)',
        pathTooltip: 'Subdirectory path within the share (optional)',
      },
      azureBlob: {
        connectionStringTooltip: 'Azure Storage connection string from Azure Portal',
        endpointTooltip: 'Custom endpoint URL (optional, leave empty for standard Azure)',
        prefixTooltip: "Optional prefix for all blob names (e.g., 'backups/' or 'my_team/')",
      },
      ftp: {
        enableFtps: 'Enable FTPS',
        useSslTooltip: 'Use explicit TLS encryption (FTPS) for secure file transfer',
        skipCertificateVerification: 'Skip certificate verification',
        skipTlsVerifyTooltip:
          'Skip TLS certificate verification. Enable this if your FTP server uses a self-signed certificate. Warning: this reduces security.',
      },
      sftp: {
        privateKeyTooltip:
          'Paste your SSH private key (PEM format). Supports RSA, DSA, ECDSA, and Ed25519 keys.',
        skipHostKeyVerification: 'Skip host key verification',
        skipHostKeyTooltip:
          'Skip SSH host key verification. Enable this if you trust the server. Warning: this reduces security.',
      },
      rclone: {
        configTooltip:
          "Paste your rclone.conf content here. You can get it by running 'rclone config file' and copying the contents. This config supports 70+ cloud storage providers.",
        configHiddenNote:
          '*content is hidden to not expose sensitive data. If you want to update existing config, put a new one here',
        docsLink: 'Rclone documentation',
        remotePathTooltip:
          "Optional path prefix on the remote where backups will be stored (e.g., '/backups' or 'my-folder/backups')",
      },
    },
  },
};

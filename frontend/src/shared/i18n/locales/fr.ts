import type { en } from './en';

export const fr: typeof en = {
  language: {
    nativeName: 'Français',
  },
  common: {
    actions: {
      add: 'Ajouter',
      back: 'Retour',
      cancel: 'Annuler',
      close: 'Fermer',
      confirm: 'Confirmer',
      continue: 'Continuer',
      copy: 'Copier',
      create: 'Créer',
      delete: 'Supprimer',
      download: 'Télécharger',
      edit: 'Modifier',
      ok: 'OK',
      refresh: 'Actualiser',
      remove: 'Supprimer',
      retry: 'Réessayer',
      save: 'Enregistrer',
    },
    answers: {
      no: 'Non',
      yes: 'Oui',
    },
    fields: {
      host: 'Hôte',
      name: 'Nom',
      password: 'Mot de passe',
      port: 'Port',
      status: 'Statut',
      type: 'Type',
      username: "Nom d'utilisateur",
    },
    states: {
      disabled: 'Désactivé',
      enabled: 'Activé',
      loading: 'Chargement...',
    },
    messages: {
      copiedToClipboard: 'Copié dans le presse-papiers',
    },
    // Joins translated words into one list (Chinese uses 、)
    listSeparator: ', ',
    confirmation: {
      title: 'Confirmation',
    },
  },
  app: {
    documentTitle: 'Databasus - sauvegarde de bases de données (PostgreSQL, MySQL, MongoDB)',
    languageControl: {
      title: 'Langue : {{language}}',
    },
    theme: {
      title: 'Thème : {{theme}}',
      light: 'Clair',
      dark: 'Sombre',
      system: 'Système',
    },
    navigation: {
      dashboard: 'Tableau de bord',
      databases: 'Bases de données',
      storages: 'Stockages',
      notifiers: 'Canaux de notification',
      workspaceSettings: 'Paramètres',
      profile: 'Profil',
      databasusSettings: 'Paramètres de Databasus',
      users: 'Utilisateurs',
      docs: 'Documentation',
      documentation: 'Documentation',
      community: 'Communauté',
      newVersionAvailable: 'Nouvelle version disponible',
    },
    diskUsage: {
      title: 'Espace disque',
      hint: "Pour créer des sauvegardes en local et les restaurer, il faut assez d'espace sur le disque. Une restauration demande autant d'espace libre que la taille de la sauvegarde.",
      compactSummary: '{{used}} sur {{total}} GB<lineBreak/>utilisés ({{percent}} %)',
      summary: '{{used}} sur {{total}} GB utilisés ({{percent}} %)',
    },
    workspaceSelection: {
      createShort: 'Créer',
      create: 'Créer un espace de travail',
      selected: 'Espace de travail',
      placeholder: 'Choisissez un espace de travail',
      search: 'Rechercher un espace de travail...',
      notFound: 'Aucun espace de travail trouvé',
      createNew: '+ Nouvel espace de travail',
    },
    sponsorship: {
      link: 'Soutenir',
      hint: "Databasus a sauvé vos données ? Soutenez le projet à votre tour pour qu'il reste gratuit, maintenu et indépendant.",
    },
    starButton: {
      label: 'Mettre une étoile',
      ariaLabel: 'Mettre une étoile à databasus/databasus sur GitHub',
    },
    clipboardPaste: {
      title: 'Coller depuis le presse-papiers',
      submit: 'Valider',
      description:
        "L'accès automatique au presse-papiers n'est pas disponible. Collez votre contenu ci-dessous.",
      placeholder: 'Collez votre chaîne de connexion ici...',
    },
    auth: {
      accountsCheckFailed: "Impossible de vérifier si l'instance possède un compte : {{error}}",
    },
    oauthCallback: {
      failedTitle: "Échec de l'authentification",
      codeNotFound: "Code d'autorisation introuvable",
      stateMissing: 'Paramètre state OAuth manquant',
      invalidProvider: 'Fournisseur OAuth non valide',
      returnToSignIn: 'Retour à la connexion',
      completing: "Finalisation de l'authentification...",
    },
    oauthStorage: {
      configNotFound: 'Configuration du stockage Google Drive introuvable',
      exchangeFailed: "Impossible d'échanger le code OAuth",
      unsupportedType: 'Ce type de stockage ne prend pas en charge OAuth',
      invalidState: 'Paramètre state OAuth non valide',
      paramNotFound:
        "Paramètre OAuth introuvable. Vérifiez que l'URL de redirection est correctement configurée.",
      addStorageTitle: 'Ajouter un stockage',
      storageDescription:
        "Un stockage est l'endroit où les sauvegardes sont conservées (disque local, S3, etc.)",
    },
  },
  errors: {
    unknown: "Une erreur s'est produite. Réessayez.",
    networkUnreachable:
      'Impossible de joindre le serveur Databasus. Vérifiez votre connexion et réessayez.',
    requestFailed:
      "Le serveur n'a pas pu traiter la requête (HTTP {{status}}). Réessayez plus tard.",
    signInCodeIncorrect: 'Le code est incorrect.',
    pendingSignInNotUsable: 'Cette connexion ne peut plus aboutir. Reconnectez-vous.',
    signInCodeNotSent:
      "L'instance n'a pas pu envoyer le code de connexion. Réessayez plus tard ou prévenez un administrateur.",
    tooManySignInCodes:
      'Trop de codes de connexion ont été demandés pour ce compte au cours de la dernière heure. Réessayez plus tard.',
    signInCodeResentTooSoon:
      "Un code a été envoyé il y a moins d'une minute. Attendez une minute avant d'en demander un autre.",
    rateLimitExceeded: 'Trop de tentatives. Patientez un peu et réessayez.',
    emailNotConfigured: "L'instance n'a pas de serveur de messagerie configuré.",
    adminEmailMissing:
      "Votre compte n'a pas d'adresse e-mail valide à laquelle envoyer un e-mail de test.",
  },
  status: {
    logicalBackup: {
      inProgress: 'En cours',
      completed: 'Réussie',
      failed: 'Échouée',
      deleted: 'Supprimée',
      canceled: 'Annulée',
    },
    physicalBackup: {
      inProgress: 'En cours',
      completed: 'Réussie',
      error: 'Erreur',
      chainBroken: 'Chaîne rompue',
      canceled: 'Annulée',
    },
    restoreVerification: {
      verifiedSuccessful: 'Vérifiée',
      verificationFailed: 'Vérification échouée',
    },
    restore: {
      inProgress: 'En cours',
      completed: 'Réussie',
      failed: 'Échouée',
      canceled: 'Annulée',
    },
    verification: {
      pending: 'En attente',
      running: 'En cours',
      completed: 'Réussie',
      failed: 'Échouée',
      canceled: 'Annulée',
    },
    health: {
      available: 'Disponible',
      unavailable: 'Indisponible',
    },
    agent: {
      neverSeen: 'Jamais connecté',
      online: 'En ligne',
      stale: 'Réponse tardive',
      offline: 'Hors ligne',
    },
  },
  intervals: {
    types: {
      hourly: 'Toutes les heures',
      daily: 'Tous les jours',
      weekly: 'Toutes les semaines',
      monthly: 'Tous les mois',
      cron: 'Cron',
    },
    weekdays: {
      mon: 'Lun.',
      tue: 'Mar.',
      wed: 'Mer.',
      thu: 'Jeu.',
      fri: 'Ven.',
      sat: 'Sam.',
      sun: 'Dim.',
    },
    periods: {
      day: '1 jour',
      week: '1 semaine',
      month: '1 mois',
      threeMonths: '3 mois',
      sixMonths: '6 mois',
      year: '1 an',
      twoYears: '2 ans',
      threeYears: '3 ans',
      fourYears: '4 ans',
      fiveYears: '5 ans',
      forever: 'Sans limite',
    },
  },
  auditLogs: {
    title: "Journaux d'audit",
    columns: {
      user: 'Utilisateur',
      message: 'Événement',
      workspace: 'Espace de travail',
      created: 'Date',
    },
    systemUser: 'Système',
    loadedOfTotal: 'Chargées : {{loaded}} sur {{total}}',
    empty: "Aucune entrée d'audit trouvée.",
    loadingMore: "Chargement d'autres entrées...",
    allLoaded: 'Toutes les entrées sont chargées ({{count}} au total)',
  },
  settings: {
    title: 'Paramètres de Databasus',
    externalRegistrations: {
      title: 'Autoriser les inscriptions externes',
      description:
        "Si l'option est activée, les nouveaux utilisateurs peuvent créer un compte dans Databasus. Sinon, ils ne peuvent s'inscrire que sur invitation.",
    },
    memberInvitations: {
      title: 'Autoriser les invitations par les membres',
      description:
        "Si l'option est activée, les membres peuvent inviter de nouveaux utilisateurs dans Databasus. Sinon, seuls les administrateurs peuvent inviter des utilisateurs.",
    },
    memberWorkspaceCreation: {
      title: 'Les membres peuvent créer des espaces de travail',
      description:
        "Si l'option est activée, les membres (utilisateurs non administrateurs) peuvent créer des espaces de travail. Sinon, seuls les administrateurs peuvent en créer.",
    },
    twoFactorAuth: {
      title: 'Activer la connexion en deux étapes',
      description:
        "Une fois activé, la connexion par mot de passe demande aussi un code à six chiffres envoyé à l'adresse du compte. La connexion par Google ou GitHub n'est pas concernée.",
      mailServerRequired:
        "Il faut un serveur de messagerie. <docsLink>Configurez le SMTP</docsLink> d'abord.",
    },
    saving: 'Enregistrement...',
    saveChanges: 'Enregistrer les modifications',
    reset: 'Réinitialiser',
    updated: 'Paramètres enregistrés',
    readMore:
      'Pour en savoir plus sur ces paramètres, <docsLink>consultez la documentation</docsLink>',
    mailServer: {
      title: 'Serveur de messagerie',
      configured: 'Le serveur de messagerie <status>est configuré</status>.',
      notConfigured: "Le serveur de messagerie <status>n'est pas configuré</status>.",
      description:
        'Il envoie les invitations, les codes de réinitialisation du mot de passe et les codes de connexion. Les canaux de notification par e-mail ont leurs propres paramètres SMTP.',
      docs: 'La <docsLink>documentation SMTP</docsLink> explique comment connecter un serveur de messagerie.',
      sendTestEmail: 'Envoyer un e-mail de test',
      testEmailSent: 'E-mail de test envoyé à {{email}}',
    },
    healthcheck: {
      title: 'Vérification de disponibilité',
      openInNewTab: 'Cliquez pour ouvrir dans un nouvel onglet',
      copied: 'URL de vérification de disponibilité copiée dans le presse-papiers',
      description:
        'Utilisez cette URL pour surveiller la disponibilité de votre instance Databasus',
    },
  },
  dashboard: {
    tiles: {
      databases: 'Bases de données',
      backups: 'Sauvegardes',
      backupsSize: 'Taille totale des sauvegardes',
      backupsSizeHint:
        'Seules les sauvegardes réussies sont comptées. Pour les bases de données physiques, les segments WAL sont inclus.',
      installation: 'Tous les espaces de travail',
      installationHint:
        'Totaux de tous les espaces de travail de cette instance Databasus. Seuls les administrateurs les voient.',
      installationDetails: 'Bases de données : {{databasesCount}}, sauvegardes : {{backupsCount}}',
    },
    list: {
      title: 'Bases de données',
      empty: "Cet espace de travail n'a pas encore de base de données",
      columns: {
        database: 'Base de données',
        healthcheck: 'Disponibilité',
        backups: 'Sauvegardes',
        meanSize: 'Taille moyenne',
        totalSize: 'Taille totale',
        lastBackup: 'Dernière sauvegarde',
        storage: 'Stockage',
      },
      successfulBackups: 'Réussies : {{successfulBackupsCount}}',
      failedBackups: 'Échouées : {{failedBackupsCount}}',
      noBackups: 'Aucune sauvegarde pour le moment',
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
      disable: 'Désactivé',
      require: 'Requis',
      verifyCa: 'Vérifier la CA',
      verifyFull: 'Vérification complète',
    },
    physicalBackupTypes: {
      full: 'Sauvegardes complètes uniquement',
      fullIncremental: 'Complète + incrémentale',
      fullIncrementalWalStream: 'Complète + incrémentale + streaming WAL',
    },
    list: {
      addDatabase: 'Ajouter une base de données',
      searchPlaceholder: 'Rechercher une base de données',
      noSearchResults: 'Aucune base de données ne correspond à « {{query}} »',
      description: "Une base de données, c'est ce que nous sauvegardons",
      backToList: '← Retour aux bases de données',
    },
    card: {
      storage: 'Stockage : <storageName/>',
      lastBackup: 'Dernière sauvegarde {{relativeTime}}',
      hasBackupError: 'Erreur de sauvegarde',
    },
    tabs: {
      config: 'Configuration',
      backups: 'Sauvegardes',
      verifications: 'Vérifications',
    },
    create: {
      title: 'Ajouter une base de données à sauvegarder',
      namePlaceholder: 'Ma base préférée',
      creating: 'Création de la base de données...',
      complete: 'Terminer',
      backupType: {
        title: 'Choisissez le type de sauvegarde',
        logical: {
          title: 'Logique',
          description: 'Recommandée pour les bases de moins de 50 GB. Plus simple à configurer.',
        },
        physical: {
          title: 'Physique',
          description:
            'Pour les bases de plus de 50 GB. Permet la restauration à un instant donné et améliore le RPO/RTO, mais demande une configuration supplémentaire.',
        },
      },
    },
    config: {
      namePlaceholder: 'Saisissez un nom...',
      removing: 'Suppression de la base de données...',
      lastBackupError: {
        title: 'Erreur de la dernière sauvegarde',
        errorLabel: 'Erreur :',
        clearHint: 'Pour effacer cette erreur (au choix) :',
        clearByTestingConnection:
          '- testez la connexion avec le bouton ci-dessous (même si vous avez déjà modifié les paramètres) ;',
        clearByNextBackup: '- attendez que la prochaine sauvegarde se termine sans erreur ;',
      },
      sections: {
        database: 'Paramètres de la base de données',
        backupConfig: 'Configuration des sauvegardes',
        healthcheck: 'Vérification de disponibilité',
        restoreVerification: 'Vérification de restauration',
        notifiers: 'Canaux de notification',
      },
      copyConfirmation:
        'Voulez-vous vraiment copier cette base de données ? Une nouvelle base de données avec les mêmes paramètres sera créée.',
      removeConfirmation:
        'Voulez-vous vraiment supprimer cette base de données ? Cette action est irréversible.',
      copied: {
        title: 'Base de données copiée !',
        description: '« {{name}} » a été créée',
      },
      connectionTestSucceeded: {
        title: 'Connexion réussie !',
        description: 'La connexion à la base de données fonctionne',
      },
    },
    actions: {
      testConnection: 'Tester la connexion',
      replace: 'Remplacer',
    },
    fields: {
      databaseType: 'Type de base de données',
      version: 'Version de {{engine}}',
      databaseName: 'Nom de la base',
      sslMode: 'Mode SSL',
      useHttps: 'Utiliser HTTPS',
      cpuCount: 'Nombre de CPU',
      backupType: 'Type de sauvegarde',
      clientCertificate: 'Certificat client',
      clientKey: 'Clé client',
      serverCaCertificate: 'Certificat CA du serveur',
      includeSchemas: 'Schémas inclus',
      excludeTables: 'Tables exclues',
      skipUserMappings: "Ignorer les correspondances d'utilisateurs",
      excludeExtensions: 'Exclure les extensions',
      restoreOwnership: 'Restaurer les propriétaires',
      restorePrivileges: 'Restaurer les privilèges',
      excludeEvents: 'Exclure les événements',
      galeraReplication: 'Réplication Galera',
      useSrvConnection: 'Connexion SRV',
      directConnection: 'Connexion directe',
      authDatabase: "Base d'authentification",
      excludeCollections: 'Collections exclues',
    },
    edit: {
      parseFromClipboard: 'Remplir depuis le presse-papiers',
      clipboardEmpty: 'Le presse-papiers est vide',
      clipboardReadFailed:
        'Impossible de lire le presse-papiers. Vérifiez les autorisations du navigateur.',
      connectionStringParsed: 'Champs remplis à partir de la chaîne de connexion',
      connectionStringParsedWithoutPassword:
        'Champs remplis à partir de la chaîne de connexion. Saisissez le mot de passe manuellement.',
      connectionTestPassed: {
        title: 'Test de connexion réussi',
        description: "Vous pouvez passer à l'étape suivante",
      },
      ipWhitelistHint:
        "Si votre base de données n'accepte que certaines adresses IP, ajoutez l'adresse IP du serveur Databasus à la liste autorisée.",
      localhostHint:
        '<docsLink>Lisez cette page</docsLink> pour savoir comment sauvegarder une base de données locale',
      advancedSettings: 'Paramètres avancés',
      placeholders: {
        host: "Saisissez l'hôte {{engine}}",
        port: 'Saisissez le port {{engine}}',
        username: "Saisissez le nom d'utilisateur {{engine}}",
        password: 'Saisissez le mot de passe {{engine}}',
        databaseName: 'Saisissez le nom de la base {{engine}}',
      },
      cpuCountTooltip:
        'Nombre de cœurs CPU utilisés pour la sauvegarde et la restauration. Une valeur plus élevée peut accélérer ces opérations, mais consomme plus de ressources.',
      excludeTablesPlaceholder: 'Aucune table exclue',
      excludeTablesTooltip:
        'Noms des tables à exclure de la sauvegarde. Vous pouvez coller une liste séparée par des virgules ou des retours à la ligne.',
      serverCaCertificateTooltip:
        'Facultatif. Si ce certificat est fourni, celui du serveur est vérifié avec cette CA (verify-ca / verify-full).',
      copyFailed: 'Échec de la copie',
      postgresql: {
        supabaseHint:
          '<docsLink>Lisez cette page</docsLink> pour savoir comment sauvegarder une base de données Supabase',
        includeSchemasPlaceholder: 'Tous les schémas (par défaut)',
        excludeTablesTooltip:
          "Tables à exclure de la sauvegarde. Utilisez 'tablename' ou 'schema.tablename'. Les motifs glob sont acceptés (par ex. 'logs_*'). Vous pouvez coller une liste séparée par des virgules ou des retours à la ligne.",
        skipUserMappingsTooltip:
          "Ne pas restaurer les correspondances d'utilisateurs (user mappings, instructions CREATE USER MAPPING). Activez cette option si le rôle de sauvegarde ne peut pas lire les identifiants de ces correspondances : sinon, elles sont exportées sans options et font échouer la restauration des FDW comme oracle_fdw.",
        excludeExtensionsTooltip:
          "Ne pas restaurer les définitions d'extensions (instructions CREATE EXTENSION). Activez cette option si vous restaurez vers un service PostgreSQL managé où le fournisseur gère les extensions.",
        restoreOwnershipTooltip:
          "Appliquer les instructions ALTER OWNER du dump pour que les objets restaurés gardent leur propriétaire d'origine. L'utilisateur de connexion doit pouvoir attribuer ces rôles, ce qui demande en général un superutilisateur.",
        restorePrivilegesTooltip:
          "Appliquer les instructions GRANT et REVOKE du dump pour que les objets restaurés gardent leurs ACL d'origine. L'utilisateur de connexion doit pouvoir accorder des droits aux rôles concernés, ce qui demande en général un superutilisateur.",
        backupTypeTooltips: {
          full: 'Sauvegardes complètes périodiques. Chaque sauvegarde se suffit à elle-même.',
          fullIncremental:
            'Sauvegardes complètes, plus des sauvegardes incrémentales qui ne stockent que les changements depuis la sauvegarde précédente. Plus légères et plus rapides.',
          fullIncrementalWalStream:
            'Ajoute un streaming WAL continu aux sauvegardes complètes et incrémentales. Seule cette option permet la restauration à un instant donné (PITR), mais elle prend plus de place et la restauration est plus lente.',
        },
      },
      mariadb: {
        skipEvents: 'Ignorer les événements',
        skipEventsTooltip:
          "Ne pas sauvegarder les événements de la base de données. Activez cette option si le planificateur d'événements est désactivé sur votre serveur MariaDB.",
        skipGaleraDisable: 'Ne pas désactiver à la restauration',
        skipGaleraDisableTooltip:
          "Par défaut, Databasus exécute SET SESSION wsrep_on=OFF pendant la restauration pour éviter les erreurs de taille de writeset Galera. Cela demande le privilège SUPER. Activez cette option pour sauter cette étape si votre fournisseur managé refuse SUPER : les grosses restaurations risquent alors d'atteindre les limites de writeset de Galera.",
      },
      mongodb: {
        srvTooltip:
          "Activez cette option pour les connexions SRV de MongoDB Atlas (mongodb+srv://). Le port n'est pas nécessaire pour une connexion SRV.",
        srvOverTunnelTooltip:
          "Indisponible via un tunnel SSH : SRV résout sa propre liste d'hôtes et contourne le port redirigé.",
        directConnectionTooltip:
          'Se connecter directement à un seul serveur, sans découverte du replica set. Utile quand le serveur est derrière un répartiteur de charge, un proxy ou un tunnel.',
        directConnectionOverTunnelTooltip:
          "Toujours activée tant que le tunnel SSH est actif : un tunnel n'expose qu'une seule adresse, la découverte du replica set est donc impossible.",
        excludeCollectionsPlaceholder: 'Aucune collection exclue',
        excludeCollectionsTooltip:
          'Noms des collections à exclure de la sauvegarde. Vous pouvez coller une liste séparée par des virgules ou des retours à la ligne.',
      },
    },
    show: {
      galeraDisableSkipped: 'Non désactivée à la restauration',
    },
    sshTunnel: {
      authTypes: {
        password: 'Mot de passe',
        privateKey: 'Clé privée',
      },
      title: 'Tunnel SSH',
      tooltip:
        "Pour une base de données dans un réseau fermé. Databasus se connecte à l'hôte SSH ci-dessous, et cet hôte joint la base de données avec l'hôte et le port indiqués plus haut.",
      host: 'Hôte SSH',
      hostTooltip:
        "L'hôte de la base de données indiqué plus haut est résolu par l'hôte SSH, pas par Databasus. Utilisez 127.0.0.1 si la base de données tourne sur l'hôte SSH lui-même.",
      port: 'Port SSH',
      username: 'Utilisateur SSH',
      usernamePlaceholder: "Saisissez l'utilisateur SSH",
      authType: 'Authentification SSH',
      credentials: 'Identifiants SSH',
      password: 'Mot de passe SSH',
      passwordPlaceholder: 'Saisissez le mot de passe SSH',
      privateKey: 'Clé privée SSH',
      keyPassphrase: 'Phrase secrète de la clé',
      keyPassphrasePlaceholder: 'Uniquement pour une clé chiffrée',
    },
    notifiers: {
      title: 'Canaux de notification',
      description:
        "Un canal de notification est l'endroit où les notifications sont envoyées (e-mail, Slack, Telegram, etc.)",
      multipleHint:
        'Vous pouvez choisir plusieurs canaux : les notifications seront envoyées à chacun.',
      selectPlaceholder: 'Choisissez des canaux de notification',
      createNew: 'Créer un canal de notification',
      addTitle: 'Ajouter un canal de notification',
      notifyTo: 'Notifier via',
      empty: 'Aucun canal de notification configuré',
    },
    readOnlyUser: {
      dialogTitle: 'Créer un utilisateur en lecture seule',
      readOnly: {
        checking: "Vérification de l'utilisateur en lecture seule...",
        title: 'Créer un utilisateur en lecture seule pour Databasus ?',
        description:
          "Un utilisateur en lecture seule est un utilisateur {{databaseType}} aux droits limités : il peut lire les données de votre base, mais pas les modifier. C'est recommandé pour les sauvegardes, car :",
        noWriteCredentials:
          "<bold>Avec un utilisateur en lecture seule, aucun identifiant avec droits d'écriture n'est stocké</bold>. Même en cas de piratage, personne ne pourra corrompre vos données.",
        create: 'Oui, créer un utilisateur en lecture seule',
        skipTitle: "Ne pas créer d'utilisateur en lecture seule ?",
        skipQuestion: "Voulez-vous vraiment passer la création d'un utilisateur en lecture seule ?",
      },
      replicationOnly: {
        checking: "Vérification de l'utilisateur réservé à la réplication...",
        title: 'Créer un utilisateur réservé à la réplication pour Databasus ?',
        description:
          "Un utilisateur réservé à la réplication est un utilisateur {{databaseType}} aux droits limités : il peut lire les données de votre base, mais pas les modifier. C'est recommandé pour les sauvegardes, car :",
        noWriteCredentials:
          "<bold>Avec un utilisateur réservé à la réplication, aucun identifiant avec droits d'écriture n'est stocké</bold>. Même en cas de piratage, personne ne pourra corrompre vos données.",
        create: 'Oui, créer un utilisateur réservé à la réplication',
        skipTitle: "Ne pas créer d'utilisateur réservé à la réplication ?",
        skipQuestion:
          "Voulez-vous vraiment passer la création d'un utilisateur réservé à la réplication ?",
      },
      reasons: {
        preventsModifications:
          'il évite toute modification accidentelle des données pendant la sauvegarde',
        leastPrivilege: 'il respecte le principe du moindre privilège',
        bestPractice: "c'est une bonne pratique de sécurité",
      },
      securityNote:
        'Databasus applique une sécurité de niveau entreprise (<docsLink>en savoir plus</docsLink>). Mais aucune protection ne couvre tous les risques possibles.',
      noWritePrivileges: "L'utilisateur actuel <bold>n'a aucun privilège d'écriture</bold>.",
      noWriteRoles: "L'utilisateur actuel <bold>n'a aucun rôle d'écriture</bold>.",
      writePrivileges: "L'utilisateur actuel a les privilèges d'écriture suivants :",
      writeRoles: "L'utilisateur actuel a les rôles d'écriture suivants :",
      expand: '(afficher)',
      collapse: '(masquer)',
      skip: 'Passer',
      skipRisk:
        "Utiliser un utilisateur avec tous les droits pour les sauvegardes n'est pas recommandé et peut poser des risques de sécurité. Databasus vous recommande vivement de ne pas passer cette étape.",
      skipAdvice:
        "Une protection à 100 % n'existe pas. Mieux vaut se prémunir même contre un risque de piratage de 0,01 %. Il est donc préférable de choisir la voie sûre avec un utilisateur en lecture seule.",
      acceptRisks: "Oui, j'accepte les risques",
      continueSecurely: 'Continuer de façon sécurisée',
      forcedWalRotationUnavailable:
        "Cette source n'accorderait pas EXECUTE sur pg_switch_wal() au nouvel utilisateur, alors que le streaming WAL continu en a besoin pour garder le point de restauration proche du présent. Les sauvegardes complètes et incrémentales fonctionnent normalement avec ces identifiants.",
    },
    transfer: {
      title: 'Transférer la base de données vers un autre espace de travail',
      targetWorkspace: 'Espace de travail cible',
      selectWorkspace: 'Choisissez un espace de travail',
      storage: {
        title: 'Stockage',
        transferExisting: 'Transférer avec le stockage actuel',
        selectFromTarget: "Choisir un stockage de l'espace de travail cible",
        blocked:
          "Bases de données utilisant ce stockage : {{count}}. Le transfert est bloqué, car d'autres bases de données en dépendent.",
        canTransfer: 'Le stockage peut être transféré',
        selectPlaceholder: 'Choisissez un stockage',
        createNew: 'Créer un stockage',
        addTitle: 'Ajouter un stockage',
        description:
          "Un stockage est l'endroit où les sauvegardes sont conservées (disque local, S3, Google Drive, etc.)",
      },
      notifiers: {
        title: 'Canaux de notification (facultatif)',
        transferWithDatabase: 'Transférer les canaux de notification avec la base de données',
        selectFromTarget: "Choisir des canaux de notification de l'espace de travail cible",
        willTransfer: 'Seront transférés :',
        willNotTransfer: "Ne seront PAS transférés (utilisés par d'autres bases de données) :",
        usedByDatabases: '{{name}} (bases de données : {{count}})',
        noneTransferred:
          "Aucun canal de notification ne sera transféré. Après le transfert, vous pourrez choisir des canaux dans l'espace de travail cible.",
        selectPlaceholder: 'Choisissez des canaux de notification (facultatif)',
      },
      submit: 'Transférer',
      transferred: {
        title: 'Base de données transférée !',
        description: '« {{name}} » a été transférée dans le nouvel espace de travail',
      },
    },
    connectionString: {
      errors: {
        empty: 'La chaîne de connexion est vide',
        unrecognizedFormat: 'Format de chaîne de connexion non reconnu',
        hostMissing: "La chaîne de connexion ne contient pas l'hôte",
        usernameMissing: "La chaîne de connexion ne contient pas le nom d'utilisateur",
        passwordMissing: 'La chaîne de connexion ne contient pas le mot de passe',
        databaseMissing: 'La chaîne de connexion ne contient pas le nom de la base de données',
        parseFailed: "Impossible d'analyser la chaîne de connexion",
        jdbc: {
          invalidFormat:
            'Format de chaîne de connexion JDBC non valide. Format attendu : {{expectedFormat}}',
          queryParametersMissing:
            'La chaîne de connexion JDBC ne contient pas les paramètres de requête (user et password)',
          usernameMissing:
            "La chaîne de connexion JDBC ne contient pas le nom d'utilisateur (paramètre user)",
          passwordMissing: 'La chaîne de connexion JDBC ne contient pas le paramètre password',
          parseFailed: "Impossible d'analyser la chaîne de connexion JDBC",
        },
        keyValue: {
          hostMissing: "La chaîne de connexion ne contient pas l'hôte. Utilisez host=hostname",
          usernameMissing:
            "La chaîne de connexion ne contient pas le nom d'utilisateur. Utilisez user=username",
          passwordMissing:
            'La chaîne de connexion ne contient pas le mot de passe. Utilisez password=yourpassword',
          databaseMissing:
            'La chaîne de connexion ne contient pas le nom de la base de données. Utilisez database=database',
          parseFailed: "Impossible d'analyser la chaîne de connexion au format clé=valeur",
        },
        libpq: {
          databaseMissing:
            'La chaîne de connexion ne contient pas le nom de la base de données. Utilisez dbname=database',
          parseFailed: "Impossible d'analyser la chaîne de connexion libpq",
        },
      },
    },
    physicalConnectionErrors: {
      grantAsSuperuserNote:
        'Accordez-le avec la commande ci-dessous (en tant que superutilisateur).',
      applyAndRestartNote: 'Appliquez la modification ci-dessous, puis redémarrez PostgreSQL.',
      pgHbaNoEntry: {
        title: 'La réplication depuis cet hôte est refusée',
        summary:
          "PostgreSQL n'a aucune entrée pg_hba.conf qui autorise une connexion de réplication depuis Databasus. Les sauvegardes physiques passent par la réplication, qui exige une règle « host replication » : une règle « host all » ordinaire ne la couvre pas.",
        addLineNote:
          "Ajoutez la ligne ci-dessous à pg_hba.conf. Le premier « all » correspond à n'importe quel utilisateur et le second à n'importe quel hôte : pour restreindre l'accès, remplacez l'utilisateur par le rôle de réplication concerné et/ou l'hôte par une plage CIDR.",
        reloadNote:
          'Après avoir modifié pg_hba.conf, <bold>rechargez la configuration de PG</bold>.',
        managedNote:
          "Sur un PostgreSQL managé (RDS / Azure / GCP), pg_hba.conf n'est pas modifiable : activez plutôt l'accès externe ou la réplication dans la console du fournisseur.",
      },
      badCredentials: {
        title: "Nom d'utilisateur ou mot de passe incorrect",
        summary:
          "PostgreSQL a refusé les identifiants. Vérifiez le nom d'utilisateur et le mot de passe de cette base de données.",
      },
      noReplicationPrivilege: {
        title: "L'utilisateur n'a pas le droit de réplication",
        summary:
          "L'utilisateur s'est connecté, mais il n'a pas le privilège REPLICATION exigé par les sauvegardes physiques.",
        managedNote:
          'Sur AWS RDS, exécutez GRANT rds_replication TO {{username}}. Sur Azure / GCP, activez la réplication pour le rôle dans la console du fournisseur.',
      },
      noWalSwitchPrivilege: {
        title: "L'utilisateur ne peut pas forcer un changement de segment WAL",
        summary:
          "Le streaming WAL continu n'envoie un segment qu'une fois que la source l'a fermé : sur une base peu sollicitée en écriture, les WAL les plus récents restent donc sur la source jusqu'à ce que le segment de 16 MB soit plein. Forcer un changement de segment à intervalle régulier garde le point de restauration entre deux sauvegardes proche du présent, et cela demande EXECUTE sur pg_switch_wal(). Les sauvegardes complètes et incrémentales n'en ont pas besoin : elles restaurent l'état à la fin de chaque sauvegarde et ne rejouent aucun WAL archivé.",
        switchBackupTypeNote:
          'Ou passez le type de sauvegarde à <bold>Complète + incrémentale</bold>, qui ne rejoue pas de WAL archivé.',
        managedNote:
          'Sur un PostgreSQL managé (RDS / Azure / GCP), la fonction appartient au superutilisateur de la plateforme et un rôle client ne peut pas obtenir EXECUTE dessus : le streaming WAL continu y est donc indisponible. Utilisez plutôt des sauvegardes complètes + incrémentales.',
      },
      walLevelInvalid: {
        title: 'wal_level est trop bas',
        summary:
          'wal_level doit valoir « replica » ou « logical » pour les sauvegardes physiques. Sa valeur actuelle est inférieure.',
        applyNote:
          "Appliquez la modification ci-dessous, puis redémarrez PostgreSQL (wal_level ne prend effet qu'après un redémarrage).",
        managedNote:
          'Sur un PostgreSQL managé, définissez wal_level dans le groupe de paramètres du fournisseur.',
      },
      noWalSenders: {
        title: 'Aucun processus walsender disponible',
        summary:
          'max_wal_senders vaut 0 : PostgreSQL ne peut donc pas diffuser les WAL pour les sauvegardes.',
        managedNote:
          'Sur un PostgreSQL managé, définissez max_wal_senders dans le groupe de paramètres du fournisseur.',
      },
      noReplicationSlots: {
        title: 'Aucun slot de réplication disponible',
        summary:
          'max_replication_slots vaut 0 : PostgreSQL ne peut donc pas allouer de slot pour les sauvegardes.',
        managedNote:
          'Sur un PostgreSQL managé, définissez max_replication_slots dans le groupe de paramètres du fournisseur.',
      },
      walSummaryDisabled: {
        title: 'summarize_wal est désactivé',
        summary:
          'summarize_wal doit être activé pour les sauvegardes incrémentales. Il est actuellement désactivé sur ce serveur.',
        applyNote: 'Appliquez la modification ci-dessous (sans redémarrage).',
        thenNote: 'puis',
        managedNote:
          'Sur un PostgreSQL managé, définissez summarize_wal dans le groupe de paramètres du fournisseur.',
      },
      customTablespaces: {
        title: 'Les tablespaces personnalisés ne sont pas pris en charge',
        summary:
          'Ce cluster contient des tablespaces autres que pg_default / pg_global, que les sauvegardes physiques ne peuvent pas diffuser. Supprimez les tablespaces personnalisés ou passez cette base de données en sauvegardes logiques.',
      },
      systemIdentifierMismatch: {
        title: "Il s'agit d'un autre cluster",
        summary:
          "Le cluster à cette adresse n'a pas le même identifiant système que celui pour lequel cette sauvegarde a été configurée. Pointez Databasus vers le cluster d'origine ou créez une nouvelle entrée de base de données.",
      },
      connectionFailed: {
        title: 'Connexion impossible',
        summary:
          "Databasus n'a pas pu ouvrir de connexion de réplication. Vérifiez l'hôte, le port et le mode SSL, et qu'aucun pare-feu ne bloque la connexion.",
      },
    },
  },
  restores: {
    list: {
      empty: 'Aucune restauration pour le moment',
    },
    targetDatabase: {
      select: 'Choisir la base de données de destination',
      description:
        "Saisissez les informations de la base de données où la sauvegarde sera restaurée. <underline>Créez d'abord une base de données vide pour la restauration</underline>. Pendant la restauration, toutes ses données actuelles seront effacées",
      notInUseHint:
        "Assurez-vous que la base de données n'est pas utilisée en ce moment (vous ne voulez probablement pas restaurer les données dans la base d'où vient la sauvegarde)",
      submit: 'Restaurer dans cette base',
    },
    card: {
      startedAt: 'Début',
      duration: 'Durée',
      errorDetailsTooltip: "Cliquez pour voir le détail de l'erreur",
      cancelTooltip: 'Annuler la restauration',
      expectedDurationHint:
        'Une restauration prend en général 3 à 5 fois plus de temps que la sauvegarde (parfois moins, parfois plus selon le type de données)',
      expectedDurationLimit:
        "Elle peut donc prendre jusqu'à {{duration}} (en général beaucoup moins)",
    },
    errorDetails: {
      title: "Détail de l'erreur de restauration",
      copied: "Message d'erreur copié dans le presse-papiers",
      excludeExtensionsTip:
        "<bold>💡 Astuce :</bold> cette erreur survient en général lors d'une restauration vers un service PostgreSQL managé (Yandex Cloud, AWS RDS ou similaire). Essayez d'activer <bold>« Exclure les extensions »</bold> dans les paramètres avancés avant de restaurer.",
    },
    cancelConfirmation: {
      warning:
        '<bold>⚠️ Attention :</bold> si vous annulez cette restauration, votre base de données restera probablement corrompue ou incomplète. Vous devrez la recréer avant de tenter une nouvelle restauration.',
      question: 'Voulez-vous vraiment annuler ?',
      confirm: 'Oui, annuler la restauration',
    },
  },
  healthcheck: {
    attempts: {
      title: 'Vérifications de disponibilité',
      period: 'Période',
      periods: {
        today: "Aujourd'hui",
        sevenDays: '7 jours',
        thirtyDays: '30 jours',
        allTime: 'Depuis le début',
      },
      empty: 'Aucune donnée pour le moment',
    },
    config: {
      enable: 'Vérifier la disponibilité',
      isEnabled: 'Vérification activée',
      notifyWhenUnavailable: "Notifier en cas d'indisponibilité",
      checkIntervalMinutes: 'Intervalle de vérification (min)',
      attemptsBeforeDown: 'Tentatives avant indisponibilité',
      storeAttemptsDays: "Conserver l'historique (jours)",
      tooltips: {
        enable: 'Active ou désactive la vérification de disponibilité de cette base de données',
        notifyWhenUnavailable:
          'Envoyer des notifications quand la base de données devient indisponible',
        checkIntervalMinutes:
          'Fréquence de vérification de la disponibilité de la base de données (en minutes)',
        attemptsBeforeDown:
          'Nombre de tentatives échouées avant de considérer la base de données comme indisponible',
        storeAttemptsDays:
          "Nombre de jours pendant lesquels l'historique des vérifications de disponibilité est conservé",
      },
    },
  },
  workspaces: {
    roles: {
      owner: 'Propriétaire',
      admin: 'Administrateur',
      member: 'Membre',
      viewer: 'Lecteur',
    },
    fields: {
      name: "Nom de l'espace de travail",
      namePlaceholder: "Saisissez le nom de l'espace de travail",
    },
    create: {
      title: 'Créer un espace de travail',
      submit: "Créer l'espace de travail",
      defaultName: 'Mon espace de travail',
      description:
        "Un espace de travail regroupe :<lineBreak/>- vos bases de données ;<lineBreak/>- les stockages (disque local, S3, Google Drive, etc.) ;<lineBreak/>- les canaux de notification (e-mail, Slack, Telegram, etc.) ;<lineBreak/>- le contrôle d'accès (si vous travaillez en équipe).",
      nameRequired: "Saisissez un nom d'espace de travail",
      created: 'Espace de travail créé',
      permissionDenied: {
        title: 'Accès refusé',
        description:
          "Vous n'avez pas le droit de créer des espaces de travail. Demandez à l'administrateur de le créer pour vous.",
      },
    },
    settings: {
      title: "Paramètres de l'espace de travail",
      readOnly: "Vous n'avez pas le droit de modifier ces paramètres",
      nameRequired: "Le nom de l'espace de travail est obligatoire",
      saving: 'Enregistrement...',
      saveChanges: 'Enregistrer les modifications',
      reset: 'Réinitialiser',
      updated: 'Informations générales enregistrées',
    },
    delete: {
      dangerZone: 'Zone de danger',
      title: 'Supprimer cet espace de travail',
      description:
        "La suppression d'un espace de travail est définitive. Toutes les données et ressources associées à cet espace de travail seront supprimées pour toujours.",
      submit: "Supprimer l'espace de travail",
      deleting: 'Suppression...',
      notFound: 'Espace de travail introuvable',
      noPermission: "Vous n'avez pas le droit de supprimer cet espace de travail",
      hasDatabases:
        "Impossible de supprimer l'espace de travail. Supprimez d'abord toutes ses bases de données. Bases de données trouvées : {{count}}.",
      deleted: 'Espace de travail supprimé',
      confirmation: {
        title: "Supprimer l'espace de travail",
        question: "Voulez-vous vraiment supprimer l'espace de travail <workspaceName/> ?",
        warning:
          '<bold>Cette action est irréversible.</bold> Toutes les données et les ressources associées seront supprimées définitivement.',
        submit: "Supprimer l'espace de travail",
      },
    },
    members: {
      title: 'Utilisateurs',
      count: 'Membres : {{count}}',
      empty: 'Aucun membre',
      emptyHint: 'Cliquez sur « Ajouter un membre » pour commencer',
      fields: {
        member: 'Membre',
        role: 'Rôle',
        joined: 'Ajouté le',
        actions: 'Actions',
        email: 'Adresse e-mail',
        emailPlaceholder: "Saisissez l'adresse e-mail",
      },
      add: {
        button: 'Ajouter un membre',
        title: 'Ajouter un membre',
        submit: 'Ajouter',
        emailRequired: "L'adresse e-mail est obligatoire",
        hint: "Si l'utilisateur existe déjà, il est ajouté directement. Sinon, une invitation lui est envoyée.",
        added: 'Membre ajouté',
      },
      invited: {
        title: 'Utilisateur invité',
        sentTo: 'Invitation envoyée à {{email}}',
        description:
          "Cet utilisateur n'a pas encore de compte, mais il a été invité dans l'espace de travail. Dès qu'il s'inscrira avec cette adresse e-mail, il en deviendra automatiquement membre.",
      },
      roleChanged: 'Rôle du membre modifié',
      remove: {
        tooltip: 'Supprimer le membre',
        title: 'Supprimer le membre',
        question: 'Voulez-vous vraiment supprimer « {{email}} » de cet espace de travail ?',
        removed: 'Membre « {{email}} » supprimé',
      },
    },
    transferOwnership: {
      button: 'Transférer la propriété',
      title: "Transférer la propriété de l'espace de travail",
      submit: 'Transférer la propriété',
      warning:
        '<bold>Attention :</bold> cette action est irréversible. Vous ne serez plus propriétaire de cet espace de travail et le nouveau propriétaire en aura le contrôle total.',
      noEligibleMembers:
        "Aucun membre à qui transférer la propriété. L'espace de travail doit compter au moins un autre membre.",
      newOwner: 'Nouveau propriétaire',
      newOwnerPlaceholder: 'Choisissez le membre qui deviendra propriétaire',
      newOwnerHint: "Le membre choisi deviendra propriétaire de l'espace de travail",
      memberRequired: 'Choisissez le membre à qui transférer la propriété',
      memberNotFound: 'Membre introuvable',
      transferred: 'Propriété transférée',
    },
    auditLogs: {
      empty: "Aucune entrée d'audit pour cet espace de travail.",
    },
  },
  users: {
    roles: {
      admin: 'Administrateur',
      member: 'Membre',
    },
    fields: {
      name: 'Nom',
      email: 'E-mail',
      yourName: 'Votre nom',
      yourEmail: 'Votre e-mail',
      confirmPassword: 'Confirmez le mot de passe',
      newPassword: 'Nouveau mot de passe',
    },
    validation: {
      nameRequired: 'Le nom est obligatoire',
      emailRequired: "L'e-mail est obligatoire",
      passwordTooShort: 'Mot de passe trop court. Longueur minimale : {{minLength}}',
      passwordsDoNotMatch: 'Les mots de passe ne correspondent pas',
    },
    oauth: {
      divider: 'ou par e-mail',
      continueWithGithub: 'Continuer avec GitHub',
      continueWithGoogle: 'Continuer avec Google',
      invalidConfiguration: 'Configuration OAuth non valide',
    },
    signIn: {
      title: 'Connexion',
      submit: 'Se connecter',
      noAccount: 'Pas encore de compte ? <signUpLink>Inscrivez-vous</signUpLink>',
      forgotPassword: 'Mot de passe oublié ?',
    },
    signInCode: {
      title: 'Saisissez le code',
      description:
        'Nous avons envoyé un code à six chiffres à {{email}}. Il cesse de fonctionner 10 minutes après son envoi.',
      code: 'Code de connexion',
      submit: 'Se connecter',
      resend: 'Envoyer un autre code',
      backToSignIn: 'Retour à la connexion',
      invalidCode: 'Le code comporte 6 chiffres',
      codeResent: 'Un nouveau code est parti. Le précédent ne fonctionne plus.',
    },
    signUp: {
      title: 'Inscription',
      namePlaceholder: 'Jean Dupont',
      submit: "S'inscrire",
      administersInstance:
        "Il s'agit du premier compte de cette instance : il en sera l'administrateur.",
      hasAccount: 'Vous avez déjà un compte ? <signInLink>Connectez-vous</signInLink>',
    },
    requestPasswordReset: {
      title: 'Réinitialiser le mot de passe',
      description:
        'Saisissez votre adresse e-mail et nous vous enverrons un code de réinitialisation.',
      submit: 'Envoyer le code',
      codeSent: 'Si cette adresse e-mail existe, un code de réinitialisation a été envoyé',
      rememberPassword:
        'Vous vous souvenez de votre mot de passe ? <signInLink>Connectez-vous</signInLink>',
    },
    resetPassword: {
      title: 'Réinitialiser le mot de passe',
      description: 'Saisissez le code reçu par e-mail et votre nouveau mot de passe.',
      code: 'Code de réinitialisation',
      confirmPassword: 'Confirmez le mot de passe',
      submit: 'Réinitialiser le mot de passe',
      invalidCode: 'Le code doit comporter 6 chiffres',
      succeeded: 'Mot de passe réinitialisé ! Redirection vers la connexion...',
      noCode:
        "Vous n'avez pas reçu de code ? <requestCodeLink>Demander un nouveau code</requestCodeLink>",
      backToSignIn: 'Retour à la connexion',
    },
    profile: {
      title: 'Profil',
      information: 'Informations du profil',
      userId: 'ID utilisateur',
      namePlaceholder: 'Saisissez votre nom',
      emailPlaceholder: 'Saisissez votre e-mail',
      role: 'Rôle',
      saveChanges: 'Enregistrer les modifications',
      noChanges: 'Aucune modification à enregistrer',
      updated: 'Profil mis à jour',
      logout: 'Se déconnecter',
      changePassword: {
        title: 'Changer le mot de passe',
        newPasswordPlaceholder: 'Saisissez le nouveau mot de passe',
        confirmPassword: 'Confirmez le nouveau mot de passe',
        confirmPasswordPlaceholder: 'Ressaisissez le nouveau mot de passe',
        passwordsDoNotMatch: 'Les nouveaux mots de passe ne correspondent pas',
        changing: 'Changement du mot de passe...',
        submit: 'Changer le mot de passe',
        signedInWithNewPassword: 'Vous êtes connecté avec votre nouveau mot de passe',
      },
    },
    list: {
      title: 'Utilisateurs de Databasus',
      loadedOfTotal: 'Utilisateurs : {{loaded}} sur {{total}}',
      searchPlaceholder: 'Rechercher par e-mail ou par nom...',
      empty: 'Aucun utilisateur trouvé.',
      allLoaded: 'Tous les utilisateurs sont chargés ({{count}} au total)',
      columns: {
        user: 'Utilisateur',
        systemRole: 'Rôle système',
        isActive: 'Actif',
        created: 'Créé le',
      },
      roleLabel: 'Rôle :',
      activeLabel: 'Actif :',
      viewAuditLogs: "Voir les journaux d'audit",
      activated: 'Utilisateur activé',
      deactivated: 'Utilisateur désactivé',
      roleChanged: "Rôle de l'utilisateur modifié",
    },
    auditLogs: {
      title: "Journaux d'audit de l'utilisateur",
      empty: "Aucune entrée d'audit pour cet utilisateur.",
    },
  },
  verification: {
    triggers: {
      manual: 'Manuelle',
      scheduled: 'Planifiée',
    },
    notificationTypes: {
      verificationSuccess: 'Vérification réussie',
      verificationFailed: 'Vérification échouée',
    },
    config: {
      scheduledVerification: 'Vérification planifiée',
      scheduledVerificationTooltip:
        "Restaure régulièrement la dernière sauvegarde dans un conteneur Postgres éphémère pour vérifier qu'elle est restaurable. <docsLink>Voir comment ça marche</docsLink>",
      interval: 'Intervalle de vérification',
      afterBackup: 'Après chaque sauvegarde',
      afterBackupHint:
        'Se lance automatiquement après chaque sauvegarde réussie. Les vérifications encore en attente dans la file sont annulées, pour que la file ne grossisse pas sans fin quand les sauvegardes vont plus vite que les vérifications.',
      weekday: 'Jour de la semaine',
      dayOfMonth: 'Jour du mois',
      timeOfDay: 'Heure de vérification',
      cron: {
        expression: 'Expression cron (UTC)',
        format: 'Format cron : minute, heure, jour, mois, jour de la semaine (UTC)',
        examples: 'Exemples :',
        exampleEverySunday: '- {{expression}} : tous les dimanches à 4 h 00 UTC',
        exampleEverySixHours: '- {{expression}} : toutes les 6 heures',
        nextRun: 'Prochaine exécution : {{dateTime}}<lineBreak/>({{relativeTime}})',
        invalid: 'Expression cron non valide',
      },
      notifications: 'Notifications',
      noNotifications: 'Aucune',
    },
    runs: {
      title: 'Vérifications de restauration',
      description:
        "Chaque ligne correspond à une tentative de restauration d'une sauvegarde de cette base de données dans une copie temporaire",
      noAgentsWarning:
        'Aucun agent de vérification enregistré : ajoutez-en un dans les paramètres de Databasus, section « Agents de vérification ». <docsLink>En savoir plus</docsLink>',
      empty: 'Aucune vérification de restauration pour le moment',
      allLoaded: 'Toutes les vérifications sont chargées ({{total}} au total)',
      attempt: 'tentative {{attempt}}',
      restoreDuration: 'restauration : {{duration}}',
      durationTooltip: "Durée de restauration + vérification communiquée par l'agent.",
      fields: {
        createdAt: 'Créée le',
        trigger: 'Déclenchement',
        duration: 'Durée',
        actions: 'Actions',
      },
      actions: {
        cancel: 'Annuler la vérification',
        viewDetails: 'Voir les détails',
      },
      details: {
        title: 'Détails de la vérification de restauration',
        loadFailed: 'Impossible de charger les détails de la vérification.',
        cancellationReason: "Motif de l'annulation",
        failure: 'Échec',
        sections: {
          status: 'Statut',
          timeline: 'Chronologie',
          results: 'Résultats et diagnostic',
        },
        fields: {
          attempt: 'Tentative',
          startedAt: 'Début',
          finishedAt: 'Fin',
          restoreDuration: 'Durée de restauration',
          verifyDuration: 'Durée de vérification',
          restoredDbSize: 'Taille de la base restaurée',
          schemas: 'Schémas',
          tables: 'Tables',
          pgRestoreExitCode: 'Code de sortie de pg_restore',
        },
        tableStats: {
          title: 'Nombre de lignes par table',
          empty: 'Aucune statistique par table.',
          schema: 'Schéma',
          table: 'Table',
          rows: 'Lignes',
        },
      },
    },
    agents: {
      title: 'Agents de vérification',
      description:
        "Les agents exécutent les vérifications de restauration pour confirmer qu'une sauvegarde est restaurable (<docsLink>en savoir plus</docsLink>)",
      empty: 'Aucun agent enregistré pour le moment.',
      columns: {
        capacity: 'Ressources',
        created: 'Créé le',
      },
      capacity: {
        cpu: '{{count}} CPU',
        ram: '{{count}} GB de RAM',
        disk: '{{count}} GB de disque',
        jobs: 'tâches : {{count}}',
        notReported: 'pas encore communiquées',
      },
      actions: {
        viewInstallCommands: "Voir les commandes d'installation",
        rotateToken: 'Renouveler le jeton',
      },
      deleteConfirmation: 'Supprimer cet agent ?',
      deleted: 'Agent « {{name}} » supprimé',
      copyFailed: 'Échec de la copie',
      agentId: "ID de l'agent :",
      create: {
        title: 'Créer un agent de vérification',
        tokenNotice: "Un jeton sera généré et affiché une seule fois sur l'écran suivant.",
        namePlaceholder: "Nom de l'agent",
        nameRequired: 'Le nom est obligatoire',
      },
      rotate: {
        title: 'Renouveler le jeton',
        description:
          "Renouveler le jeton de <agentName/> invalide immédiatement le jeton actuel. Un agent qui utilise encore l'ancien jeton sera refusé à son prochain heartbeat.",
      },
      token: {
        title: "Jeton de l'agent",
        tokenFor: 'Jeton pour <agentName/> :',
        shownOnce:
          'Affiché une seule fois. Conservez-le en lieu sûr : vous ne pourrez plus le récupérer.',
        confirmSaved: "J'ai enregistré le jeton",
      },
      install: {
        title: "Commandes d'installation",
        description:
          "Commandes d'installation pour <agentName/>. Remplacez <tokenPlaceholder/> par le jeton que vous avez enregistré à la création de l'agent ou lors du dernier renouvellement du jeton.",
        architecture: 'Architecture',
        stepInstall: 'Étape 1 : installation',
        stepLaunch: 'Étape 2 : lancement',
        capacityDefaults:
          "Les valeurs de ressources ci-dessous sont des valeurs de départ : ajustez <code>--max-cpu</code>, <code>--max-ram-mb</code>, <code>--max-disk-gb</code> et <code>--max-concurrent-jobs</code> à la machine qui exécute l'agent.",
        afterInstallation: "Après l'installation",
        runsInBackground: "Après <code>start</code>, l'agent tourne en arrière-plan",
        checkStatus: 'Vérifier le statut : <code>./verification-agent status</code>',
        viewLogs:
          'Voir les logs : <code>databasus-verification.log</code> dans le répertoire de travail',
        stop: "Arrêter l'agent : <code>./verification-agent stop</code>",
      },
    },
  },
  notifiers: {
    types: {
      email: 'E-mail',
      telegram: 'Telegram',
      webhook: 'Webhook',
      slack: 'Slack',
      discord: 'Discord',
      teams: 'Teams',
      mattermost: 'Mattermost',
    },
    notificationTypes: {
      all: 'Toutes',
      backupSuccess: 'Sauvegarde réussie',
      backupFailed: 'Sauvegarde échouée',
      healthcheckSuccess: 'Vérification de disponibilité réussie',
      healthcheckFailed: 'Vérification de disponibilité échouée',
      verificationSuccess: 'Vérification de restauration réussie',
      verificationFailed: 'Vérification de restauration échouée',
    },
    mattermostDeliveryModes: {
      webhook: 'Webhook entrant',
      bot: 'Compte bot',
    },
    list: {
      addNotifier: 'Ajouter un canal de notification',
      searchPlaceholder: 'Rechercher un canal de notification',
      noSearchResults: 'Aucun canal de notification ne correspond à « {{query}} »',
      description:
        "Un canal de notification est l'endroit où les notifications sont envoyées (e-mail, Slack, Telegram, etc.)",
      backToList: '← Retour aux canaux de notification',
    },
    card: {
      notifyTo: {
        email: 'Notifications par e-mail',
        telegram: 'Notifications sur Telegram',
        webhook: 'Notifications par webhook',
        slack: 'Notifications sur Slack',
        discord: 'Notifications sur Discord',
        teams: 'Notifications sur Teams',
        mattermost: 'Notifications sur Mattermost',
      },
      hasSendError: "Erreur d'envoi",
    },
    create: {
      title: 'Ajouter un canal de notification',
      namePlaceholder: 'Mes alertes',
    },
    config: {
      namePlaceholder: 'Saisissez un nom...',
      settingsTitle: 'Paramètres du canal de notification',
      sendError: {
        title: "Erreur d'envoi",
        errorLabel: 'Erreur :',
        clearHint: 'Pour effacer cette erreur (au choix) :',
        clearBySendingTest:
          '- envoyez une notification de test avec le bouton ci-dessous (même si vous avez déjà modifié les paramètres) ;',
        clearByNextNotification: '- attendez que la prochaine notification parte sans erreur ;',
      },
      removeConfirmation:
        'Voulez-vous vraiment supprimer ce canal de notification ? Cette action est irréversible.',
      removeBlockedByDatabases:
        "Ce canal de notification est utilisé par des bases de données. Retirez-le d'abord de ces bases de données.",
    },
    testNotification: {
      send: 'Envoyer une notification de test',
      sent: {
        title: 'Notification de test envoyée !',
        description: 'La notification de test est bien partie',
      },
    },
    transfer: {
      title: 'Transférer le canal de notification vers un autre espace de travail',
      blockedByDatabases:
        "Ce canal de notification est utilisé par des bases de données. Transférez ou supprimez d'abord ces bases de données.",
      description:
        "Choisissez l'espace de travail vers lequel transférer ce canal de notification.",
      targetWorkspace: 'Espace de travail cible',
      selectWorkspace: 'Choisissez un espace de travail',
      submit: 'Transférer',
    },
    fields: {
      botToken: 'Jeton du bot',
      targetChatId: 'ID du chat',
      webhookUrl: 'URL du webhook',
      channelId: 'ID du canal',
      skipTlsVerify: 'Ignorer la vérification TLS',
      skipTls: 'Sans vérification TLS',
    },
    email: {
      targetEmail: 'E-mail du destinataire',
      targetEmailTooltip: "L'adresse e-mail à laquelle vous voulez recevoir les messages",
      smtpHost: 'Hôte SMTP',
      smtpPort: 'Port SMTP',
      smtpUser: 'Utilisateur SMTP',
      smtpPassword: 'Mot de passe SMTP',
      smtpPasswordPlaceholder: 'mot de passe',
      from: 'Expéditeur',
      fromTooltip:
        "Facultatif. Adresse de l'expéditeur, avec ou sans nom : Acme Backups <noreply@example.com>. Si le champ est vide, l'utilisateur SMTP est utilisé s'il s'agit d'une adresse e-mail, sinon noreply@ suivi de l'hôte SMTP",
      fromAuto: '(automatique)',
      advancedSettings: 'Paramètres avancés',
      skipTlsTooltip:
        'Ne pas vérifier le certificat TLS. Activez cette option si votre serveur SMTP utilise un certificat auto-signé. Attention : cela réduit la sécurité.',
      security: 'Sécurité de la connexion',
      securityTooltip:
        "TLS chiffre la connexion dès le départ, généralement sur le port 465. STARTTLS chiffre la connexion après son ouverture et échoue si le serveur ne le propose pas. « Sans chiffrement » envoie le mot de passe et les messages en clair : ne l'utilisez qu'avec un relais de confiance sans STARTTLS.",
      securityModes: {
        tls: 'TLS',
        starttls: 'STARTTLS',
        none: 'Sans chiffrement',
      },
      heloName: 'Nom annoncé',
      heloNamePlaceholder: 'Automatique',
      heloNameTooltip:
        "Le nom sous lequel Databasus se présente au serveur de messagerie (EHLO). S'il est vide, l'hôte de DATABASUS_URL est utilisé, ou à défaut le nom de la machine.",
    },
    telegram: {
      botTokenHelpLink: "Comment obtenir le jeton d'API d'un bot Telegram ?",
      targetChatIdTooltip:
        'Le chat où vous voulez recevoir les messages (votre chat privé ou un groupe)',
      chatIdHelp: {
        toggle: "Comment obtenir l'ID d'un chat Telegram ?",
        privateChat:
          "Pour obtenir l'ID de votre chat, écrivez à <botLink>@getmyid_bot</botLink> dans Telegram. <underline>Assurez-vous d'avoir démarré une conversation avec le bot</underline>",
        group:
          "Pour obtenir l'ID d'un groupe, ajoutez votre bot et <botLink>@getmyid_bot</botLink> au groupe, puis écrivez /start (l'ID du chat s'affichera)",
      },
      useProxy: 'Utiliser un proxy',
      useProxyTooltip: "Passer par un proxy pour les requêtes vers l'API Telegram",
      proxy: 'Proxy',
      proxyUrl: 'URL du proxy',
      proxyUrlTooltip:
        "Prend en charge http, https, socks5 et socks5h. Peut inclure un nom d'utilisateur et un mot de passe",
      sendToGroupTopic: 'Envoyer dans un sujet du groupe',
      sendToGroupTopicTooltip:
        "Activez cette option pour envoyer les messages dans un sujet précis d'un groupe",
      threadId: 'ID du sujet',
      threadIdTooltip: "L'ID du sujet où envoyer les messages",
      topicId: 'ID du sujet',
      threadIdHelp: {
        steps:
          "Pour obtenir l'ID du sujet, ouvrez-le dans votre groupe Telegram, puis appuyez sur son nom en haut de l'écran pour afficher ses informations. Copiez le lien du sujet et prenez le dernier nombre de l'URL.",
        example:
          "<bold>Exemple :</bold> si le lien du sujet est <code>https://t.me/c/2831948048/3</code>, l'ID du sujet est <code>3</code>",
        note: '<bold>Remarque :</bold> les sujets ne fonctionnent que dans les groupes, pas dans les chats privés.',
      },
    },
    webhook: {
      method: 'Méthode',
      sendOn: 'Envoyer pour',
      sendOnPlaceholder: 'Choisissez les types de notification',
      customHeaders: 'En-têtes personnalisés',
      customHeadersTooltip:
        'Ajoute des en-têtes HTTP personnalisés à la requête du webhook (par ex. Authorization, X-API-Key)',
      savedHeadersHidden: '*Les en-têtes enregistrés sont masqués pour des raisons de sécurité',
      headers: 'En-têtes',
      headerNamePlaceholder: "Nom de l'en-tête",
      headerValuePlaceholder: "Valeur de l'en-tête",
      addHeader: 'Ajouter un en-tête',
      hiddenHeaderValue: '(masqué)',
      bodyTemplate: {
        title: 'Modèle du corps de la requête',
        headingVariable: '<variable/> : titre de la notification',
        messageVariable: '<variable/> : texte de la notification',
        invalidJsonFormat: 'Format JSON non valide',
        invalidJson: 'JSON non valide',
      },
      exampleRequest: {
        title: 'Exemple de requête',
        headers: 'En-têtes :',
      },
    },
    slack: {
      howToConnect: "Comment connecter Slack (obtenir le jeton du bot et l'ID du chat) ?",
      testFailedHint:
        "Vérifiez que le canal est public, ou que le bot a été ajouté au canal privé (via @invite) ou au groupe. Pour les messages directs, utilisez l'ID utilisateur indiqué dans le profil Slack.",
    },
    discord: {
      channelWebhookUrl: 'URL du webhook du salon',
      howTo: {
        title: "Comment obtenir l'URL du webhook Discord :",
        createChannel: '1. Créez ou choisissez un salon Discord',
        openChannelSettings: '2. Ouvrez les paramètres du salon (icône en forme de roue dentée)',
        openIntegrations: '3. Allez dans Intégrations',
        createWebhook: '4. Créez un nouveau webhook',
        copyWebhookUrl: "5. Copiez l'URL du webhook",
        privacyNote: 'Remarque : rendez le salon privé si nécessaire',
      },
    },
    teams: {
      howToConnect: 'Comment connecter Microsoft Teams ?',
      powerAutomateUrl: 'URL Power Automate',
      powerAutomateUrlLabel: 'URL Power Automate :',
      powerAutomateUrlTooltip:
        'URL HTTP de votre flux Power Automate (déclencheur When an HTTP request is received)',
      showFullUrl: 'Afficher',
      hideFullUrl: 'Masquer',
    },
    mattermost: {
      howToConnect: 'Comment connecter Mattermost ?',
      connectVia: 'Connexion via',
      incomingWebhookUrl: 'URL du webhook entrant',
      serverUrl: 'URL du serveur',
      channelIdTooltip:
        'ID du canal (26 caractères), pas son nom. Ouvrez le canal, cliquez sur son nom et choisissez View Info.',
      channel: 'Canal',
      showOptionalSettings: 'Afficher les paramètres facultatifs',
      hideOptionalSettings: 'Masquer les paramètres facultatifs',
      channelOverride: 'Canal de remplacement',
      channelOverrideTooltip:
        "Publier dans ce canal plutôt que dans celui pour lequel le webhook a été créé. Mattermost l'ignore si le webhook est verrouillé sur un canal.",
      postAsUsername: "Nom de l'expéditeur",
      postAsUsernameTooltip:
        "Nécessite l'option Enable integrations to override usernames dans la console système de Mattermost, sinon ce champ est ignoré.",
      postAsIconUrl: "URL de l'icône de l'expéditeur",
      postAsIconUrlTooltip:
        "Nécessite l'option Enable integrations to override profile picture icons dans la console système de Mattermost, sinon ce champ est ignoré.",
      skipTlsTooltip:
        'Ne pas vérifier le certificat TLS. Activez cette option si votre serveur Mattermost utilise un certificat auto-signé. Attention : cela réduit la sécurité.',
      webhookHowTo: {
        title: "Comment obtenir l'URL d'un webhook entrant :",
        openIncomingWebhooks: '1. Menu principal → Integrations → Incoming Webhooks',
        addWebhook: '2. Cliquez sur Add Incoming Webhook et choisissez le canal',
        copyUrl: "3. Copiez l'URL générée",
      },
      botHowTo: {
        title: "Comment obtenir le jeton d'un bot :",
        addBotAccount: '1. Integrations → Bot Accounts → Add Bot Account',
        copyToken: '2. Copiez le jeton, affiché une seule fois après la création',
        addBotToChannel: "3. Ajoutez le bot à l'équipe et au canal",
      },
    },
  },
  backups: {
    encryption: {
      none: 'Aucun',
      encrypted: 'Activé',
    },
    encryptionOptions: {
      none: 'Sans chiffrement',
      encrypted: 'Chiffrer les fichiers de sauvegarde',
    },
    list: {
      title: 'Sauvegardes',
      scheduledBackupsDisabled:
        'Les sauvegardes planifiées sont désactivées (vous pouvez les réactiver dans la configuration des sauvegardes)',
      empty: 'Aucune sauvegarde pour le moment',
      allLoaded: 'Toutes les sauvegardes sont chargées ({{count}} au total)',
      clickToSeeErrorDetails: "Cliquez pour voir le détail de l'erreur",
      errorDetailsTitle: "Détail de l'erreur de sauvegarde",
      restoreTitle: 'Restauration depuis une sauvegarde',
      columns: {
        createdAt: 'Créée le',
        created: 'Créée le',
        size: 'Taille',
        duration: 'Durée',
        actions: 'Actions',
      },
      actions: {
        cancelBackup: 'Annuler la sauvegarde',
        deleteBackup: 'Supprimer la sauvegarde',
      },
    },
    filters: {
      allTypes: 'Tous les types',
      allStatuses: 'Tous les statuts',
      before: 'Avant le',
    },
    config: {
      backupsEnabled: 'Sauvegardes activées',
      storage: 'Stockage',
      selectStorage: 'Choisissez un stockage',
      createNewStorage: 'Créer un stockage',
      addStorageTitle: 'Ajouter un stockage',
      storageDescription:
        "Un stockage est l'endroit où les sauvegardes sont conservées (disque local, S3, Google Drive, etc.)",
      storageChangeWarning:
        'Si vous changez de stockage, toutes les sauvegardes de ce stockage seront supprimées.',
      storageChangeAcknowledge: "J'ai compris",
      encryption: 'Chiffrement',
      notifications: 'Notifications',
      noNotifications: 'Aucune',
      schedule: {
        weekday: 'Jour de la semaine',
        dayOfMonth: 'Jour du mois',
        cronExpression: 'Expression cron (UTC)',
        timeOfDay: 'Heure',
        nextRun: 'Prochaine exécution : {{time}}',
        invalidCron: 'Expression cron non valide',
        cronHelp: {
          format: 'Format cron : minute, heure, jour, mois, jour de la semaine (UTC)',
          examples: 'Exemples :',
          dailyAt2am: 'Tous les jours à 2 h 00 UTC',
          every6Hours: 'Toutes les 6 heures',
          mondaysAt3am: 'Tous les lundis à 3 h 00 UTC',
          twiceAMonth: 'Le 1er et le 15 du mois à 4 h 30 UTC',
        },
      },
      gfs: {
        hourly: 'Horaires : {{count}}',
        daily: 'Quotidiennes : {{count}}',
        weekly: 'Hebdomadaires : {{count}}',
        monthly: 'Mensuelles : {{count}}',
        yearly: 'Annuelles : {{count}}',
        notConfigured: 'Non configuré',
      },
    },
    logical: {
      retentionPolicyTypes: {
        gfs: 'GFS (conserver les N dernières sauvegardes horaires, quotidiennes, hebdomadaires, mensuelles et annuelles)',
        timePeriod: 'Période (N derniers jours)',
        count: 'Nombre (N dernières sauvegardes)',
      },
      notificationTypes: {
        backupSuccess: 'Sauvegarde réussie',
        backupFailed: 'Sauvegarde échouée',
      },
      list: {
        backupNow: 'Sauvegarder maintenant',
        makeBackupNow: 'Lancer une sauvegarde maintenant',
        encrypted: 'Chiffrée',
        sizeTooltip:
          "En haut : la taille du fichier de sauvegarde compressé réellement conservé dans le stockage (local, S3, Google Drive, etc.). En bas (en gris) : la taille d'origine de la base de données non compressée au moment de la sauvegarde. Une sauvegarde est en général environ 5 fois plus petite que la base.",
        databaseSize: '{{size}} (taille de la base)',
        deleteConfirmation: 'Voulez-vous vraiment supprimer cette sauvegarde ?',
        actions: {
          restore: 'Restaurer depuis la sauvegarde',
          verifyRestore:
            'Vérifier la restauration : un agent restaurera cette sauvegarde dans une base de données temporaire et indiquera le nombre de lignes',
          download: {
            postgresql:
              'Télécharger le fichier de sauvegarde. Il peut être restauré manuellement avec pg_restore (format custom)',
            mysql:
              'Télécharger le fichier de sauvegarde. Il peut être restauré manuellement avec le client mysql (dump SQL)',
            mariadb:
              'Télécharger le fichier de sauvegarde. Il peut être restauré manuellement avec le client mariadb (dump SQL)',
            mongodb:
              'Télécharger le fichier de sauvegarde. Il peut être restauré manuellement avec mongorestore (archive)',
            other: 'Télécharger le fichier de sauvegarde',
          },
        },
        verifyRestore: {
          title: 'Vérifier la restauration ?',
          description:
            'Un agent restaurera cette sauvegarde dans une base de données temporaire et indiquera le nombre de lignes. Selon la taille de la sauvegarde, cela peut prendre un moment. <docsLink>Comment ça marche ?</docsLink>',
          queue: "Mettre en file d'attente",
          queued: "Vérification de restauration mise en file d'attente",
        },
      },
      config: {
        interval: 'Intervalle de sauvegarde',
        weekday: 'Jour de la semaine',
        dayOfMonth: 'Jour du mois',
        timeOfDay: 'Heure de la sauvegarde',
        encryptionTooltip:
          'Si la sauvegarde est chiffrée, les fichiers de sauvegarde dans votre stockage (S3, local, etc.) ne sont pas utilisables directement. Vous pouvez restaurer les sauvegardes via Databasus ou les télécharger déchiffrées avec le bouton « Télécharger ».',
        retentionPolicy: 'Politique de rétention',
        retention: {
          timePeriodTooltip:
            'Durée de conservation des sauvegardes. Les sauvegardes plus anciennes sont supprimées automatiquement.',
          count: 'Sauvegardes les plus récentes',
          countTooltip:
            'Ne conserve que le nombre indiqué de sauvegardes, les plus récentes. Les plus anciennes sont supprimées automatiquement.',
          countValue: 'Sauvegardes les plus récentes : {{count}}',
          showGfsHint: "Qu'est-ce que GFS (Grandfather-Father-Son) ?",
          hideGfsHint: 'Masquer',
          gfsHint:
            "Rotation GFS (Grandfather-Father-Son) : conserve les N dernières sauvegardes horaires, quotidiennes, hebdomadaires, mensuelles et annuelles. Vous gardez ainsi des sauvegardes sur de longues périodes sans occuper trop d'espace de stockage.",
          gfsTooltip:
            'Rotation Grandfather-Father-Son : conserve les N dernières sauvegardes horaires, quotidiennes, hebdomadaires, mensuelles et annuelles.',
          gfsFields: {
            hourly: 'Sauvegardes horaires',
            daily: 'Sauvegardes quotidiennes',
            weekly: 'Sauvegardes hebdomadaires',
            monthly: 'Sauvegardes mensuelles',
            yearly: 'Sauvegardes annuelles',
          },
        },
        advancedSettings: 'Paramètres avancés',
        retryIfFailed: "Relancer la sauvegarde en cas d'échec",
        retryIfFailedShort: "Relance en cas d'échec",
        retryIfFailedTooltip:
          "Relancer automatiquement les sauvegardes échouées. Une sauvegarde peut échouer à cause d'une panne réseau, d'un problème de stockage ou d'une indisponibilité temporaire de la base de données.",
        maxFailedTries: 'Nombre maximal de tentatives',
        maxFailedTriesTooltip:
          'Nombre maximal de nouvelles tentatives après un échec. Vous recevrez une notification quand toutes les tentatives auront échoué.',
      },
    },
    physical: {
      types: {
        full: 'Complète',
        incremental: 'Incrémentale',
        wal: 'WAL',
      },
      retentions: {
        chains: 'Chaînes',
        fullBackups: 'Sauvegardes complètes',
        chainsAndFullBackups: 'Chaînes et sauvegardes complètes',
      },
      fullBackupsPolicies: {
        lastN: 'Nombre',
        gfs: 'GFS',
      },
      notificationTypes: {
        backupSuccess: 'Sauvegarde réussie',
        backupFailed: 'Sauvegarde échouée',
        chainBroken: 'Chaîne rompue',
        walGap: 'Trou dans les WAL',
      },
      list: {
        totalUsage: 'Espace utilisé : {{size}}',
        backUpNow: 'Sauvegarder maintenant',
        fullBackup: 'Sauvegarde complète',
        incrementalBackup: 'Sauvegarde incrémentale',
        restore: 'Restaurer',
        restoreFromThisBackup: 'Restaurer depuis cette sauvegarde',
        pointInTimeRestoreTitle: 'Restauration à un instant donné',
        deleteFullConfirmation:
          'Si vous supprimez cette sauvegarde complète, toute sa chaîne disparaît avec elle (toutes les sauvegardes incrémentales qui en dépendent). Cette action est irréversible. Continuer ?',
        deleteIncrementalConfirmation:
          'Si vous supprimez cette sauvegarde incrémentale, toutes les incrémentales suivantes qui en dépendent disparaissent avec elle. Cette action est irréversible. Continuer ?',
      },
      config: {
        fullBackupCadence: 'Fréquence des sauvegardes complètes',
        incrementalBackupCadence: 'Fréquence des sauvegardes incrémentales',
        incrementalMoreFrequent:
          'Les sauvegardes incrémentales doivent être plus fréquentes que les sauvegardes complètes.',
        encryptionTooltip:
          'Si les sauvegardes sont chiffrées, les fichiers dans votre stockage ne sont pas utilisables directement. Vous pouvez les restaurer via Databasus ou les télécharger déchiffrés.',
        retention: 'Rétention',
        fullBackupsOnlyRetention:
          "Cette base de données n'a que des sauvegardes complètes : la rétention s'applique donc à elles.",
        retentionOptions: {
          chains: 'Conserver les N dernières chaînes',
          chainsAndFullBackups: 'Conserver les N dernières chaînes et des sauvegardes complètes',
        },
        retentionHelp: {
          chain:
            "Une chaîne, c'est une sauvegarde complète plus les incrémentales (et les WAL reçus en streaming) qui en dépendent. Une restauration a besoin de la sauvegarde complète et de chaque incrémentale de sa chaîne.",
          chains:
            "Ne conserve que les N dernières chaînes. Quand une chaîne sort de la fenêtre, sa sauvegarde complète et toutes ses incrémentales sont supprimées : vous ne pouvez restaurer qu'à l'intérieur des chaînes conservées.",
          chainsAndFullBackups:
            "Conserve les N dernières chaînes pour restaurer à un instant récent, et garde en plus d'anciennes sauvegardes complètes autonomes selon la politique ci-dessous (GFS ou N dernières) : elles restent des points de restauration à long terme une fois leurs incrémentales supprimées.",
        },
        chainsCount: 'Chaînes',
        chainsCountTooltip:
          'Nombre de chaînes de sauvegarde les plus récentes à conserver. Une chaîne est une sauvegarde complète plus ses incrémentales.',
        fullBackups: 'Sauvegardes complètes',
        fullBackupsCount: 'Sauvegardes complètes les plus récentes',
        gfsFields: {
          hourly: 'Horaires',
          daily: 'Quotidiennes',
          weekly: 'Hebdomadaires',
          monthly: 'Mensuelles',
          yearly: 'Annuelles',
        },
        chainsKept: 'Chaînes conservées',
        fullBackupsKept: 'Sauvegardes complètes conservées',
        fullBackupsCountValue: 'Les plus récentes : {{count}}',
        gfsTooltip:
          'Rotation Grandfather-Father-Son : conserve les N dernières sauvegardes complètes horaires, quotidiennes, hebdomadaires, mensuelles et annuelles.',
      },
      restore: {
        errorHints: {
          downloadInProgress:
            "Un téléchargement pour restauration est déjà en cours pour cette base de données. Attendez qu'il se termine, puis réessayez.",
          targetTimeUnreachable:
            "Impossible d'atteindre l'instant demandé : il y a un trou dans les WAL ou l'instant est hors de la plage disponible. Choisissez un autre instant ou restaurez le dernier point disponible.",
        },
        backupIntro:
          "Commande de restauration de cette sauvegarde. Une sauvegarde complète n'a besoin que de son propre fichier ; une incrémentale a aussi besoin de sa sauvegarde complète et de toutes les incrémentales qui la précèdent.",
        pointInTimeIntro:
          "Restauration à un instant donné. Choisissez l'instant cible, ou laissez le champ vide pour restaurer le dernier point disponible.",
        targetTime: 'Instant cible',
        latestAvailable: 'Dernier point disponible',
        preparing: 'Préparation de la commande de restauration...',
        linkExpiry:
          'Le lien de téléchargement est à usage unique et expire au bout de 15 minutes. Si une commande échoue avec une erreur 401, rouvrez cette fenêtre pour générer un nouveau lien.',
        copied: 'Copié',
        methods: {
          script: 'Par script',
          manual: 'Manuelle',
        },
        environments: {
          host: "PostgreSQL sur l'hôte",
        },
        fields: {
          restoreDirectory: 'Répertoire de restauration',
          pgBinPath: 'Chemin des binaires PostgreSQL',
          pgBinPathPlaceholder: '{{path}} (facultatif)',
          pgImage: 'Image PostgreSQL',
        },
        scriptTitles: {
          host: "À exécuter sur l'hôte de restauration",
          docker: 'À exécuter là où Docker est disponible',
        },
        manualSteps: {
          download: "Téléchargez l'archive",
          extract: 'Extrayez-la',
          verify: 'Vérifiez le transfert',
          reconstruct: 'Reconstruisez le répertoire de données',
          checkConfig: 'Vérifiez les fichiers de configuration du cluster',
          wireUpRecovery: 'Décompressez les WAL et configurez la récupération',
          recoveryParameters: "Si la récupération s'arrête à cause de paramètres",
        },
        beforeYouRun: {
          title: 'Avant de lancer',
          host: {
            clientTools:
              "Installez les outils clients PostgreSQL {{version}}. S'ils ne sont pas dans le PATH, indiquez le chemin des binaires ci-dessus.",
            zstd: 'zstd est nécessaire pour décompresser la sauvegarde.',
            zstdWithWal: 'zstd est nécessaire pour décompresser la sauvegarde et ses WAL.',
            emptyDirectory:
              "Le répertoire de restauration doit être vide et ne pas appartenir à un cluster en cours d'utilisation.",
          },
          docker: {
            image:
              'Utilisez une image postgres:{{version}} : la version majeure doit correspondre à celle de la source.',
            hostTools:
              "L'hôte a besoin de zstd, tar et curl : le téléchargement et la décompression de la sauvegarde de base se font sur l'hôte, et seul pg_combinebackup tourne dans le conteneur. L'image postgres n'a donc besoin d'aucun outil supplémentaire.",
            hostToolsWithWal:
              "L'hôte a besoin de zstd, tar et curl : le téléchargement et la décompression de la sauvegarde de base et des WAL se font sur l'hôte, et seul pg_combinebackup tourne dans le conteneur. L'image postgres n'a donc besoin d'aucun outil supplémentaire.",
          },
        },
        afterItFinishes: {
          title: 'Une fois terminé',
          host: {
            clusterLocation: 'Le cluster se trouve dans <dataDir/>.',
            changeOwner: 'En tant que root, changez son propriétaire : <command/>.',
            start:
              'Démarrez-le en tant que postgres : <command/>, ou faites pointer data_directory vers lui.',
            walReplay:
              'PostgreSQL rejoue les WAL puis passe en mode primaire : surveillez le log pour voir la fin de la récupération.',
          },
          docker: {
            clusterLocation: "<dataDir/> sur l'hôte contient le cluster restauré.",
            volumeMount:
              "PostgreSQL {{version}} garde son répertoire de données dans <dataDir/>, sous le volume <volumeDir/>. Montez le répertoire de sortie comme racine de ce volume, comme l'attend un docker-compose avec <code>./pgdata:/var/lib/postgresql</code> :",
            dataDirectoryMount:
              "Montez le cluster (bind mount) sur le répertoire de données de l'image <dataDir/> :",
            ownership:
              "Le propriétaire doit correspondre à l'uid de postgres dans l'image (999 dans l'image officielle).",
          },
          missingConfigFiles:
            "Une sauvegarde de base ne copie que le répertoire de données. Si votre PostgreSQL source garde sa configuration en dehors (disposition Debian/Ubuntu, <configDir/>), alors <code>postgresql.conf</code>, <code>pg_hba.conf</code> et <code>pg_ident.conf</code> ne sont pas dans la sauvegarde et le serveur ne démarrera pas. Demandez leurs chemins réels à la source avec <code>psql -Atc \"SHOW config_file\"</code>, copiez les trois fichiers, puis commentez <code>data_directory</code>, <code>hba_file</code>, <code>ident_file</code>, <code>external_pid_file</code> et les lignes <code>ssl</code> / <code>ssl_*</code>, créez <code>conf.d</code> et, pour un conteneur, définissez <code>listen_addresses = '*'</code>. Si ces fichiers sont présents et que vous obtenez toujours la même erreur, c'est le répertoire monté qui est en cause, pas la configuration.",
        },
      },
    },
  },
  storages: {
    types: {
      local: 'Stockage local',
      s3: 'S3',
      googleDrive: 'Google Drive',
      nas: 'NAS',
      azureBlob: 'Azure Blob Storage',
      ftp: 'FTP',
      sftp: 'SFTP',
      rclone: 'Rclone',
    },
    s3StorageClasses: {
      default: 'Par défaut (Standard)',
      standard: 'Standard',
      standardIa: 'Standard - Infrequent Access',
      onezoneIa: 'One Zone - Infrequent Access',
      intelligentTiering: 'Intelligent Tiering',
      reducedRedundancy: 'Reduced Redundancy',
      glacierIr: 'Glacier Instant Retrieval',
    },
    azureBlobAuthMethods: {
      accountKey: 'Clé du compte',
      connectionString: 'Chaîne de connexion',
    },
    sftpAuthMethods: {
      password: 'Mot de passe',
      privateKey: 'Clé privée',
    },
    list: {
      addStorage: 'Ajouter un stockage',
      description:
        "Un stockage est l'endroit où les sauvegardes sont conservées (disque local, S3, etc.)",
      backToList: '← Retour aux stockages',
    },
    card: {
      type: {
        local: 'Type : stockage local',
        s3: 'Type : S3',
        googleDrive: 'Type : Google Drive',
        nas: 'Type : NAS',
        azureBlob: 'Type : Azure Blob Storage',
        ftp: 'Type : FTP',
        sftp: 'Type : SFTP',
        rclone: 'Type : Rclone',
      },
      hasSaveError: "Erreur d'enregistrement",
    },
    config: {
      namePlaceholder: 'Saisissez un nom...',
      settingsTitle: 'Paramètres du stockage',
      lastSaveError: {
        title: "Erreur d'enregistrement",
        errorLabel: 'Erreur :',
        clearHint: 'Pour effacer cette erreur (au choix) :',
        clearByTestingConnection:
          '- testez la connexion avec le bouton ci-dessous (même si vous avez déjà modifié les paramètres) ;',
        clearByNextSave: '- attendez que la prochaine sauvegarde soit enregistrée sans erreur ;',
      },
      removeConfirmation:
        'Voulez-vous vraiment supprimer ce stockage ? Cette action est irréversible. Les sauvegardes qui se trouvent dans ce stockage seront aussi supprimées.',
      removeBlockedByDatabases:
        "Ce stockage est utilisé par des bases de données. Retirez-le d'abord de ces bases de données.",
    },
    actions: {
      testConnection: 'Tester la connexion',
    },
    connectionTestSucceeded: {
      title: 'Connexion réussie !',
      description: 'La connexion au stockage fonctionne',
    },
    transfer: {
      title: 'Transférer le stockage vers un autre espace de travail',
      blockedByDatabases:
        "Ce stockage est utilisé par des bases de données. Transférez ou supprimez d'abord ces bases de données.",
      description: "Choisissez l'espace de travail vers lequel transférer ce stockage.",
      targetWorkspace: 'Espace de travail cible',
      selectWorkspace: 'Choisissez un espace de travail',
      submit: 'Transférer',
    },
    fields: {
      path: 'Chemin',
      remotePath: 'Chemin distant',
      share: 'Partage',
      domain: 'Domaine',
      useSsl: 'Utiliser SSL',
      useSslTls: 'Utiliser SSL/TLS',
      skipTls: 'Sans vérification TLS',
      skipTlsVerify: 'Ignorer la vérification TLS',
      skipHostKey: "Sans vérification de la clé d'hôte",
      authMethod: "Méthode d'authentification",
      privateKey: 'Clé privée',
      credentials: 'Identifiants',
      s3Bucket: 'Bucket S3',
      region: 'Région',
      accessKey: "Clé d'accès",
      secretKey: 'Clé secrète',
      endpoint: 'Endpoint',
      folderPrefix: 'Préfixe de dossier',
      prefix: 'Préfixe',
      blobPrefix: 'Préfixe des blobs',
      virtualHost: 'Hôte virtuel',
      storageClass: 'Classe de stockage',
      connection: 'Chaîne de connexion',
      connectionString: 'Chaîne de connexion',
      accountName: 'Nom du compte',
      accountKey: 'Clé du compte',
      containerName: 'Nom du conteneur',
      clientId: 'ID client',
      clientSecret: 'Secret client',
      userToken: 'Jeton utilisateur',
      config: 'Configuration',
    },
    edit: {
      namePlaceholder: 'Mon stockage',
      advancedSettings: 'Paramètres avancés',
      optionalPlaceholder: '{{example}} (facultatif)',
      usernamePlaceholder: "nom d'utilisateur",
      passwordPlaceholder: 'mot de passe',
      remotePathTooltip: 'Chemin du répertoire distant où stocker les sauvegardes (facultatif)',
      local: {
        diskSpaceWarning:
          "Attention : avec un stockage local, vous risquez de manquer d'espace disque. Il est recommandé d'utiliser S3 ou un stockage sans limite de capacité",
      },
      s3: {
        cloudflareR2Guide: 'Comment utiliser Cloudflare R2 ?',
        endpointTooltip:
          "URL d'un endpoint compatible S3 personnalisé (facultatif, laissez vide pour AWS S3)",
        prefixTooltip:
          "Préfixe facultatif pour toutes les clés d'objet (par ex. 'backups/' ou 'my_team/'). Peut ne pas fonctionner avec certains stockages compatibles S3. Non modifiable après la création (sinon les sauvegardes seront perdues).",
        useVirtualHostedStyle: 'Utiliser le style virtual-hosted',
        virtualHostTooltip:
          "Utiliser des URL de type virtual-hosted (bucket.s3.region.amazonaws.com) au lieu du type path-style (s3.region.amazonaws.com/bucket). Peut être nécessaire en cas d'erreurs COS.",
        skipTls: 'Sans vérification TLS',
        skipTlsVerifyTooltip:
          'Ne pas vérifier le certificat TLS. Activez cette option si votre stockage compatible S3 utilise un certificat auto-signé. Attention : cela réduit la sécurité.',
        storageClassTooltip:
          "Classe de stockage S3 des objets envoyés. Laissez la valeur par défaut pour Standard. Certains fournisseurs proposent des classes moins chères comme One Zone IA. N'utilisez pas Glacier/Deep Archive : les fichiers doivent être accessibles immédiatement pour les restaurations.",
      },
      googleDrive: {
        guideLink: 'Comment connecter Google Drive ?',
        authorize: 'Autoriser',
      },
      nas: {
        shareWithSlashWarning:
          'Le partage doit être un nom de partage unique. Utilisez le champ « Chemin » pour les sous-répertoires (par ex. Partage : Databasus, Chemin : DB1)',
        useSslTooltip: 'Activer le chiffrement SSL/TLS pour sécuriser la connexion',
        domainTooltip:
          "Nom de domaine Windows (facultatif, laissez vide si vous n'utilisez pas l'authentification de domaine)",
        pathPlaceholder: '{{example}} (facultatif, sans barre oblique au début)',
        pathTooltip: 'Chemin du sous-répertoire dans le partage (facultatif)',
      },
      azureBlob: {
        connectionStringTooltip:
          'Chaîne de connexion Azure Storage, disponible dans le portail Azure',
        endpointTooltip:
          "URL d'endpoint personnalisée (facultatif, laissez vide pour Azure standard)",
        prefixTooltip:
          "Préfixe facultatif pour tous les noms de blobs (par ex. 'backups/' ou 'my_team/')",
      },
      ftp: {
        enableFtps: 'Activer FTPS',
        useSslTooltip:
          'Utiliser le chiffrement TLS explicite (FTPS) pour sécuriser le transfert de fichiers',
        skipCertificateVerification: 'Ignorer la vérification du certificat',
        skipTlsVerifyTooltip:
          'Ne pas vérifier le certificat TLS. Activez cette option si votre serveur FTP utilise un certificat auto-signé. Attention : cela réduit la sécurité.',
      },
      sftp: {
        privateKeyTooltip:
          'Collez votre clé privée SSH (format PEM). Les clés RSA, DSA, ECDSA et Ed25519 sont prises en charge.',
        skipHostKeyVerification: "Ignorer la vérification de la clé d'hôte",
        skipHostKeyTooltip:
          "Ne pas vérifier la clé d'hôte SSH. Activez cette option si vous faites confiance au serveur. Attention : cela réduit la sécurité.",
      },
      rclone: {
        configTooltip:
          "Collez ici le contenu de votre rclone.conf. Pour l'obtenir, exécutez 'rclone config file' et copiez le contenu du fichier. Cette configuration prend en charge plus de 70 fournisseurs de stockage cloud.",
        configHiddenNote:
          '*le contenu est masqué pour ne pas exposer de données sensibles. Pour mettre à jour la configuration existante, collez-en une nouvelle ici',
        docsLink: 'Documentation de Rclone',
        remotePathTooltip:
          "Préfixe de chemin facultatif sur le stockage distant, où les sauvegardes seront conservées (par ex. '/backups' ou 'my-folder/backups')",
      },
    },
  },
};

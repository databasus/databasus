import type { en } from './en';

export const es: typeof en = {
  language: {
    nativeName: 'Español',
  },
  common: {
    actions: {
      add: 'Añadir',
      back: 'Atrás',
      cancel: 'Cancelar',
      close: 'Cerrar',
      confirm: 'Confirmar',
      continue: 'Continuar',
      copy: 'Copiar',
      create: 'Crear',
      delete: 'Eliminar',
      download: 'Descargar',
      edit: 'Editar',
      ok: 'Aceptar',
      refresh: 'Actualizar',
      remove: 'Eliminar',
      retry: 'Reintentar',
      save: 'Guardar',
    },
    answers: {
      no: 'No',
      yes: 'Sí',
    },
    fields: {
      host: 'Host',
      name: 'Nombre',
      password: 'Contraseña',
      port: 'Puerto',
      status: 'Estado',
      type: 'Tipo',
      username: 'Usuario',
    },
    states: {
      disabled: 'Desactivado',
      enabled: 'Activado',
      loading: 'Cargando...',
    },
    messages: {
      copiedToClipboard: 'Copiado al portapapeles',
    },
    // Joins translated words into one list (Chinese uses 、)
    listSeparator: ', ',
    confirmation: {
      title: 'Confirmación',
    },
  },
  app: {
    documentTitle: 'Databasus - copias de seguridad de bases de datos (PostgreSQL, MySQL, MongoDB)',
    languageControl: {
      title: 'Idioma: {{language}}',
    },
    theme: {
      title: 'Tema: {{theme}}',
      light: 'Claro',
      dark: 'Oscuro',
      system: 'Sistema',
    },
    navigation: {
      dashboard: 'Panel',
      databases: 'Bases de datos',
      storages: 'Almacenamientos',
      notifiers: 'Canales de notificación',
      workspaceSettings: 'Ajustes',
      profile: 'Perfil',
      databasusSettings: 'Ajustes de Databasus',
      users: 'Usuarios',
      docs: 'Documentación',
      documentation: 'Documentación',
      community: 'Comunidad',
      newVersionAvailable: 'Nueva versión disponible',
    },
    diskUsage: {
      title: 'Uso del disco',
      hint: 'Para crear copias de seguridad en local y restaurarlas, necesita suficiente espacio en el disco. Para restaurar una copia hace falta tanto espacio libre como el que ocupa la copia.',
      compactSummary: 'En uso: {{used}} de {{total}} GB<lineBreak/>({{percent}}%)',
      summary: 'En uso: {{used}} de {{total}} GB ({{percent}}%)',
    },
    workspaceSelection: {
      createShort: 'Crear',
      create: 'Crear espacio de trabajo',
      selected: 'Espacio de trabajo actual',
      placeholder: 'Seleccione un espacio de trabajo',
      search: 'Buscar espacios de trabajo...',
      notFound: 'No se encontraron espacios de trabajo',
      createNew: '+ Nuevo espacio de trabajo',
    },
    sponsorship: {
      link: 'Patrocinar',
      hint: '¿Databasus le ha salvado los datos? Apoye el proyecto para que siga siendo gratuito, mantenido e independiente.',
    },
    starButton: {
      label: 'Dar estrella en GitHub',
      ariaLabel: 'Dar una estrella a databasus/databasus en GitHub',
    },
    clipboardPaste: {
      title: 'Pegar desde el portapapeles',
      submit: 'Aplicar',
      description:
        'El acceso automático al portapapeles no está disponible. Pegue el contenido a continuación.',
      placeholder: 'Pegue aquí la cadena de conexión...',
    },
    auth: {
      accountsCheckFailed: 'No se pudo comprobar si la instancia tiene alguna cuenta: {{error}}',
    },
    oauthCallback: {
      failedTitle: 'Error de autenticación',
      codeNotFound: 'No se encontró el código de autorización',
      stateMissing: 'Falta el parámetro state de OAuth',
      invalidProvider: 'Proveedor de OAuth no válido',
      returnToSignIn: 'Volver al inicio de sesión',
      completing: 'Completando la autenticación...',
    },
    oauthStorage: {
      configNotFound: 'No se encontró la configuración del almacenamiento de Google Drive',
      exchangeFailed: 'No se pudo canjear el código de OAuth',
      unsupportedType: 'Este tipo de almacenamiento no admite OAuth',
      invalidState: 'El parámetro state de OAuth no es válido',
      paramNotFound:
        'No se encontró el parámetro de OAuth. Compruebe que la URL de redirección esté bien configurada.',
      addStorageTitle: 'Añadir almacenamiento',
      storageDescription:
        'Un almacenamiento es el lugar donde se guardan las copias de seguridad (disco local, S3, etc.)',
    },
  },
  errors: {
    unknown: 'Algo salió mal. Inténtelo de nuevo.',
    networkUnreachable:
      'No se puede conectar con el servidor de Databasus. Compruebe su conexión e inténtelo de nuevo.',
    requestFailed:
      'El servidor no pudo completar la solicitud (HTTP {{status}}). Inténtelo de nuevo más tarde.',
    signInCodeIncorrect: 'El código no es correcto.',
    pendingSignInNotUsable:
      'Este inicio de sesión ya no puede completarse. Vuelva a iniciar sesión.',
    signInCodeNotSent:
      'La instancia no ha podido enviar el código de acceso. Inténtelo más tarde o avise a un administrador.',
    tooManySignInCodes:
      'Se han pedido demasiados códigos de acceso para esta cuenta en la última hora. Inténtelo más tarde.',
    signInCodeResentTooSoon:
      'Se envió un código hace menos de un minuto. Espere un minuto antes de pedir otro.',
    rateLimitExceeded: 'Demasiados intentos. Espere un poco e inténtelo de nuevo.',
    emailNotConfigured: 'La instancia no tiene un servidor de correo configurado.',
    adminEmailMissing:
      'Su cuenta no tiene una dirección de correo válida a la que enviar un correo de prueba.',
  },
  status: {
    logicalBackup: {
      inProgress: 'En curso',
      completed: 'Correcta',
      failed: 'Fallida',
      deleted: 'Eliminada',
      canceled: 'Cancelada',
    },
    physicalBackup: {
      inProgress: 'En curso',
      completed: 'Correcta',
      error: 'Error',
      chainBroken: 'Cadena rota',
      canceled: 'Cancelada',
    },
    restoreVerification: {
      verifiedSuccessful: 'Verificada',
      verificationFailed: 'Verificación fallida',
    },
    restore: {
      inProgress: 'En curso',
      completed: 'Correcta',
      failed: 'Fallida',
      canceled: 'Cancelada',
    },
    verification: {
      pending: 'Pendiente',
      running: 'En ejecución',
      completed: 'Correcta',
      failed: 'Fallida',
      canceled: 'Cancelada',
    },
    health: {
      available: 'Disponible',
      unavailable: 'No disponible',
    },
    agent: {
      neverSeen: 'Nunca conectado',
      online: 'En línea',
      stale: 'Con retraso',
      offline: 'Desconectado',
    },
  },
  intervals: {
    types: {
      hourly: 'Cada hora',
      daily: 'Diario',
      weekly: 'Semanal',
      monthly: 'Mensual',
      cron: 'Cron',
    },
    weekdays: {
      mon: 'Lun',
      tue: 'Mar',
      wed: 'Mié',
      thu: 'Jue',
      fri: 'Vie',
      sat: 'Sáb',
      sun: 'Dom',
    },
    periods: {
      day: '1 día',
      week: '1 semana',
      month: '1 mes',
      threeMonths: '3 meses',
      sixMonths: '6 meses',
      year: '1 año',
      twoYears: '2 años',
      threeYears: '3 años',
      fourYears: '4 años',
      fiveYears: '5 años',
      forever: 'Sin límite',
    },
  },
  auditLogs: {
    title: 'Registros de auditoría',
    columns: {
      user: 'Usuario',
      message: 'Mensaje',
      workspace: 'Espacio de trabajo',
      created: 'Fecha',
    },
    systemUser: 'Sistema',
    loadedOfTotal: 'Cargados: {{loaded}} de {{total}}',
    empty: 'No hay registros de auditoría.',
    loadingMore: 'Cargando más registros...',
    allLoaded: 'Todos los registros cargados ({{count}} en total)',
  },
  settings: {
    title: 'Ajustes de Databasus',
    externalRegistrations: {
      title: 'Permitir registros externos',
      description:
        'Si está activado, cualquier persona puede crear una cuenta en Databasus. Si está desactivado, solo es posible registrarse por invitación.',
    },
    memberInvitations: {
      title: 'Permitir invitaciones de miembros',
      description:
        'Si está activado, los miembros pueden invitar a nuevos usuarios a Databasus. Si no, solo pueden invitar los administradores.',
    },
    memberWorkspaceCreation: {
      title: 'Los miembros pueden crear espacios de trabajo',
      description:
        'Si está activado, los miembros (usuarios que no son administradores) pueden crear espacios de trabajo. Si no, solo pueden crearlos los administradores.',
    },
    twoFactorAuth: {
      title: 'Activar el inicio de sesión en dos pasos',
      description:
        'Si se activa, además de la contraseña hay que introducir un código de seis dígitos enviado al correo de la cuenta. No se aplica al inicio de sesión con Google ni con GitHub.',
      mailServerRequired:
        'Hace falta un servidor de correo. <docsLink>Configure SMTP</docsLink> primero.',
    },
    saving: 'Guardando...',
    saveChanges: 'Guardar cambios',
    reset: 'Restablecer',
    updated: 'Ajustes guardados',
    readMore: 'Más información sobre los ajustes <docsLink>en la documentación</docsLink>',
    mailServer: {
      title: 'Servidor de correo',
      configured: 'El servidor de correo <status>está configurado</status>.',
      notConfigured: 'El servidor de correo <status>no está configurado</status>.',
      description:
        'Envía las invitaciones, los códigos para restablecer la contraseña y los códigos de acceso. Los canales de notificación por correo tienen su propia configuración SMTP.',
      docs: 'La <docsLink>documentación de SMTP</docsLink> explica cómo conectar un servidor de correo.',
      sendTestEmail: 'Enviar correo de prueba',
      testEmailSent: 'Correo de prueba enviado a {{email}}',
    },
    healthcheck: {
      title: 'Comprobación de disponibilidad',
      openInNewTab: 'Abrir en una pestaña nueva',
      copied: 'Endpoint de comprobación de disponibilidad copiado al portapapeles',
      description:
        'Use este endpoint para supervisar la disponibilidad de su instancia de Databasus',
    },
  },
  dashboard: {
    tiles: {
      databases: 'Bases de datos',
      backups: 'Copias de seguridad',
      backupsSize: 'Tamaño total de las copias',
      backupsSizeHint:
        'Solo cuenta las copias correctas. En las bases de datos físicas también se incluyen los segmentos WAL.',
      installation: 'Todos los espacios de trabajo',
      installationHint:
        'Totales de todos los espacios de trabajo de esta instancia de Databasus. Solo los ven los administradores.',
      installationDetails: 'Bases de datos: {{databasesCount}}, copias: {{backupsCount}}',
    },
    list: {
      title: 'Bases de datos',
      empty: 'Este espacio de trabajo todavía no tiene bases de datos',
      columns: {
        database: 'Base de datos',
        healthcheck: 'Disponibilidad',
        backups: 'Copias',
        meanSize: 'Tamaño medio',
        totalSize: 'Tamaño total',
        lastBackup: 'Última copia',
        storage: 'Almacenamiento',
      },
      successfulBackups: 'Correctas: {{successfulBackupsCount}}',
      failedBackups: 'Fallidas: {{failedBackupsCount}}',
      noBackups: 'Todavía no hay copias',
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
      disable: 'Desactivado',
      require: 'Obligatorio',
      verifyCa: 'Verificación de CA',
      verifyFull: 'Verificación completa',
    },
    physicalBackupTypes: {
      full: 'Solo copias completas',
      fullIncremental: 'Completas + incrementales',
      fullIncrementalWalStream: 'Completas + incrementales + streaming de WAL',
    },
    list: {
      addDatabase: 'Añadir base de datos',
      searchPlaceholder: 'Buscar base de datos',
      noSearchResults: 'No se encontraron bases de datos que coincidan con "{{query}}"',
      description: 'Una base de datos es lo que protegemos con copias de seguridad',
      backToList: '← Volver a las bases de datos',
    },
    card: {
      storage: 'Almacenamiento: <storageName/>',
      lastBackup: 'Última copia de seguridad {{relativeTime}}',
      hasBackupError: 'Error en la copia de seguridad',
    },
    tabs: {
      config: 'Configuración',
      backups: 'Copias de seguridad',
      verifications: 'Verificaciones',
    },
    create: {
      title: 'Añadir una base de datos para copias de seguridad',
      namePlaceholder: 'Mi base de datos favorita',
      creating: 'Creando la base de datos...',
      complete: 'Finalizar',
      backupType: {
        title: 'Elija el tipo de copia de seguridad',
        logical: {
          title: 'Lógica',
          description:
            'Recomendada para bases de datos de menos de 50 GB. Más fácil de configurar.',
        },
        physical: {
          title: 'Física',
          description:
            'Para bases de datos de más de 50 GB. Permite la recuperación a un punto en el tiempo (PITR) y mejora el RPO/RTO, pero requiere configuración adicional.',
        },
      },
    },
    config: {
      namePlaceholder: 'Introduzca un nombre...',
      removing: 'Eliminando la base de datos...',
      lastBackupError: {
        title: 'Error en la última copia de seguridad',
        errorLabel: 'Error:',
        clearHint: 'Para quitar este error, haga una de estas dos cosas:',
        clearByTestingConnection:
          '- pruebe la conexión con el botón de abajo (aunque ya haya cambiado los ajustes);',
        clearByNextBackup: '- espere a que la siguiente copia de seguridad termine sin errores;',
      },
      sections: {
        database: 'Ajustes de la base de datos',
        backupConfig: 'Configuración de copias de seguridad',
        healthcheck: 'Comprobación de disponibilidad',
        restoreVerification: 'Verificación de restauración',
        notifiers: 'Canales de notificación',
      },
      copyConfirmation:
        '¿Seguro que desea copiar esta base de datos? Se creará una base de datos nueva con los mismos ajustes.',
      removeConfirmation:
        '¿Seguro que desea eliminar esta base de datos? Esta acción no se puede deshacer.',
      copied: {
        title: 'Base de datos copiada',
        description: 'Base de datos "{{name}}" creada',
      },
      connectionTestSucceeded: {
        title: 'La conexión funciona',
        description: 'Conexión con la base de datos comprobada',
      },
    },
    actions: {
      testConnection: 'Probar conexión',
      replace: 'Reemplazar',
    },
    fields: {
      databaseType: 'Tipo de base de datos',
      version: 'Versión de {{engine}}',
      databaseName: 'Nombre de la BD',
      sslMode: 'Modo SSL',
      useHttps: 'Usar HTTPS',
      cpuCount: 'Número de CPU',
      backupType: 'Tipo de copia de seguridad',
      clientCertificate: 'Certificado de cliente',
      clientKey: 'Clave de cliente',
      serverCaCertificate: 'Certificado CA del servidor',
      includeSchemas: 'Incluir esquemas',
      excludeTables: 'Excluir tablas',
      skipUserMappings: 'Omitir user mappings',
      excludeExtensions: 'Excluir extensiones',
      restoreOwnership: 'Restaurar propietarios',
      restorePrivileges: 'Restaurar privilegios',
      excludeEvents: 'Excluir eventos',
      galeraReplication: 'Replicación de Galera',
      useSrvConnection: 'Usar conexión SRV',
      directConnection: 'Conexión directa',
      authDatabase: 'Base de datos de autenticación',
      excludeCollections: 'Excluir colecciones',
    },
    edit: {
      parseFromClipboard: 'Rellenar desde el portapapeles',
      clipboardEmpty: 'El portapapeles está vacío',
      clipboardReadFailed: 'No se pudo leer el portapapeles. Revise los permisos del navegador.',
      connectionStringParsed: 'Campos rellenados a partir de la cadena de conexión',
      connectionStringParsedWithoutPassword:
        'Campos rellenados a partir de la cadena de conexión. Introduzca la contraseña manualmente.',
      connectionTestPassed: {
        title: 'Conexión comprobada',
        description: 'Puede pasar al siguiente paso',
      },
      ipWhitelistHint:
        'Si su base de datos solo acepta conexiones desde una lista de IP permitidas, añada a esa lista la IP del servidor de Databasus.',
      localhostHint:
        '<docsLink>Lea este artículo</docsLink> para saber cómo hacer copias de seguridad de una base de datos local',
      advancedSettings: 'Ajustes avanzados',
      placeholders: {
        host: 'Introduzca el host de {{engine}}',
        port: 'Introduzca el puerto de {{engine}}',
        username: 'Introduzca el usuario de {{engine}}',
        password: 'Introduzca la contraseña de {{engine}}',
        databaseName: 'Introduzca el nombre de la base de datos de {{engine}}',
      },
      cpuCountTooltip:
        'Número de núcleos de CPU que se usan para crear y restaurar copias de seguridad. Un valor más alto puede acelerar el proceso, pero consume más recursos.',
      excludeTablesPlaceholder: 'Ninguna tabla excluida',
      excludeTablesTooltip:
        'Tablas que no se incluirán en la copia de seguridad. Puede pegar una lista separada por comas o por saltos de línea.',
      serverCaCertificateTooltip:
        'Opcional. Si se indica, el certificado del servidor se verifica con esta CA (verify-ca / verify-full).',
      copyFailed: 'No se pudo copiar',
      postgresql: {
        supabaseHint:
          '<docsLink>Lea este artículo</docsLink> para saber cómo hacer copias de seguridad de una base de datos de Supabase',
        includeSchemasPlaceholder: 'Todos los esquemas (predeterminado)',
        excludeTablesTooltip:
          "Tablas que no se incluirán en la copia de seguridad. Use 'tablename' o 'schema.tablename'. Se admiten patrones glob (p. ej., 'logs_*'). Puede pegar una lista separada por comas o por saltos de línea.",
        skipUserMappingsTooltip:
          'No restaura los user mappings (sentencias CREATE USER MAPPING). Active esta opción si el rol de la copia de seguridad no puede leer sus credenciales: si no, se vuelcan sin opciones y rompen la restauración con FDW como oracle_fdw.',
        excludeExtensionsTooltip:
          'No restaura las definiciones de extensiones (sentencias CREATE EXTENSION). Active esta opción si restaura en un servicio de PostgreSQL gestionado en el que el proveedor administra las extensiones.',
        restoreOwnershipTooltip:
          'Aplica las sentencias ALTER OWNER del volcado para que los objetos restaurados conserven su propietario original. El usuario de la conexión debe poder asignar esos roles; normalmente, un superusuario.',
        restorePrivilegesTooltip:
          'Aplica las sentencias GRANT y REVOKE del volcado para que los objetos restaurados conserven sus ACL originales. El usuario de la conexión debe poder conceder permisos a los roles referenciados; normalmente, un superusuario.',
        backupTypeTooltips: {
          full: 'Copias completas periódicas e independientes. Cada copia se basta por sí misma.',
          fullIncremental:
            'Copias completas más incrementales, que guardan solo los cambios desde la copia anterior. Ocupan menos y son más rápidas.',
          fullIncrementalWalStream:
            'Añade streaming continuo de WAL a las copias completas e incrementales. Es la única opción que permite la recuperación a un punto en el tiempo (PITR), pero ocupa más espacio y la restauración es más lenta.',
        },
      },
      mariadb: {
        skipEvents: 'Omitir eventos',
        skipEventsTooltip:
          'No incluye los eventos de la base de datos en la copia de seguridad. Active esta opción si el programador de eventos está desactivado en su servidor MariaDB.',
        skipGaleraDisable: 'No desactivar al restaurar',
        skipGaleraDisableTooltip:
          'Por defecto, Databasus ejecuta SET SESSION wsrep_on=OFF durante la restauración para evitar errores de tamaño de writeset en Galera. Esto requiere el privilegio SUPER. Active esta opción para saltarse ese paso si su proveedor gestionado no concede SUPER; en ese caso, las restauraciones grandes pueden toparse con los límites de writeset de Galera.',
      },
      mongodb: {
        srvTooltip:
          'Active esta opción para las conexiones SRV de MongoDB Atlas (mongodb+srv://). Las conexiones SRV no necesitan puerto.',
        srvOverTunnelTooltip:
          'No disponible a través de un túnel SSH: SRV resuelve su propia lista de hosts y no pasa por el puerto redirigido.',
        directConnectionTooltip:
          'Se conecta directamente a un único servidor, sin descubrir el replica set. Útil si el servidor está detrás de un balanceador de carga, un proxy o un túnel.',
        directConnectionOverTunnelTooltip:
          'Siempre activado mientras el túnel SSH esté activo: un túnel expone una sola dirección, así que no se puede descubrir el replica set.',
        excludeCollectionsPlaceholder: 'Ninguna colección excluida',
        excludeCollectionsTooltip:
          'Colecciones que no se incluirán en la copia de seguridad. Puede pegar una lista separada por comas o por saltos de línea.',
      },
    },
    show: {
      galeraDisableSkipped: 'No desactivar al restaurar',
    },
    sshTunnel: {
      authTypes: {
        password: 'Contraseña',
        privateKey: 'Clave privada',
      },
      title: 'Túnel SSH',
      tooltip:
        'Para una base de datos dentro de una red cerrada. Databasus se conecta al host SSH de abajo, y ese host llega a la base de datos con el host y el puerto indicados arriba.',
      host: 'Host SSH',
      hostTooltip:
        'El host de la base de datos indicado arriba lo resuelve el host SSH, no Databasus. Use 127.0.0.1 si la base de datos se ejecuta en el propio host SSH.',
      port: 'Puerto SSH',
      username: 'Usuario SSH',
      usernamePlaceholder: 'Introduzca el usuario SSH',
      authType: 'Autenticación SSH',
      credentials: 'Credenciales SSH',
      password: 'Contraseña SSH',
      passwordPlaceholder: 'Introduzca la contraseña SSH',
      privateKey: 'Clave privada SSH',
      keyPassphrase: 'Frase de contraseña de la clave',
      keyPassphrasePlaceholder: 'Solo para claves cifradas',
    },
    notifiers: {
      title: 'Canales de notificación',
      description:
        'Un canal de notificación es el lugar adonde se envían las notificaciones (correo electrónico, Slack, Telegram, etc.)',
      multipleHint:
        'Puede elegir varios canales de notificación: las notificaciones se enviarán a todos ellos.',
      selectPlaceholder: 'Seleccione canales de notificación',
      createNew: 'Crear canal de notificación',
      addTitle: 'Añadir canal de notificación',
      notifyTo: 'Notificar a',
      empty: 'No hay canales de notificación configurados',
    },
    readOnlyUser: {
      dialogTitle: 'Crear usuario de solo lectura',
      readOnly: {
        checking: 'Comprobando el usuario de solo lectura...',
        title: '¿Crear un usuario de solo lectura para Databasus?',
        description:
          'Un usuario de solo lectura es un usuario de {{databaseType}} con permisos limitados: puede leer los datos de su base de datos, pero no modificarlos. Es lo recomendado para las copias de seguridad porque:',
        noWriteCredentials:
          '<bold>Con un usuario de solo lectura no se guardan credenciales con permisos de escritura</bold>. Aunque ocurra lo peor y alguien hackee el sistema, nadie podrá dañar sus datos.',
        create: 'Sí, crear usuario de solo lectura',
        skipTitle: '¿Omitir la creación del usuario de solo lectura?',
        skipQuestion: '¿Seguro que desea no crear un usuario de solo lectura?',
      },
      replicationOnly: {
        checking: 'Comprobando el usuario de solo replicación...',
        title: '¿Crear un usuario de solo replicación para Databasus?',
        description:
          'Un usuario de solo replicación es un usuario de {{databaseType}} con permisos limitados: puede leer los datos de su base de datos, pero no modificarlos. Es lo recomendado para las copias de seguridad porque:',
        noWriteCredentials:
          '<bold>Con un usuario de solo replicación no se guardan credenciales con permisos de escritura</bold>. Aunque ocurra lo peor y alguien hackee el sistema, nadie podrá dañar sus datos.',
        create: 'Sí, crear usuario de solo replicación',
        skipTitle: '¿Omitir la creación del usuario de solo replicación?',
        skipQuestion: '¿Seguro que desea no crear un usuario de solo replicación?',
      },
      reasons: {
        preventsModifications:
          'evita modificaciones accidentales de los datos durante la copia de seguridad',
        leastPrivilege: 'sigue el principio de mínimo privilegio',
        bestPractice: 'es una buena práctica de seguridad',
      },
      securityNote:
        'Databasus aplica seguridad de nivel empresarial (<docsLink>más detalles aquí</docsLink>). Aun así, no es posible cubrir todos los riesgos.',
      noWritePrivileges: 'El usuario actual <bold>no tiene privilegios de escritura</bold>.',
      noWriteRoles: 'El usuario actual <bold>no tiene roles de escritura</bold>.',
      writePrivileges: 'El usuario actual tiene estos privilegios de escritura:',
      writeRoles: 'El usuario actual tiene estos roles de escritura:',
      expand: '(mostrar)',
      collapse: '(ocultar)',
      skip: 'Omitir',
      skipRisk:
        'No se recomienda usar un usuario con todos los permisos para las copias de seguridad: supone riesgos de seguridad. Databasus le recomienda encarecidamente no omitir este paso.',
      skipAdvice:
        'La protección al 100% no existe. Más vale prevenir, aunque el riesgo de un hackeo completo sea del 0,01%. Por eso es mejor seguir el camino seguro con un usuario de solo lectura.',
      acceptRisks: 'Sí, acepto los riesgos',
      continueSecurely: 'Continuar por el camino seguro',
      forcedWalRotationUnavailable:
        'El servidor de origen no concedería al nuevo usuario EXECUTE sobre pg_switch_wal(), que el streaming continuo de WAL necesita para mantener el punto de recuperación cerca del presente. Las copias completas e incrementales funcionan con normalidad con estas credenciales.',
    },
    transfer: {
      title: 'Transferir la base de datos a otro espacio de trabajo',
      targetWorkspace: 'Espacio de trabajo de destino',
      selectWorkspace: 'Seleccione un espacio de trabajo',
      storage: {
        title: 'Almacenamiento',
        transferExisting: 'Transferir con el almacenamiento actual',
        selectFromTarget: 'Elegir un almacenamiento del espacio de trabajo de destino',
        blocked:
          'Bases de datos que usan este almacenamiento: {{count}}. La transferencia está bloqueada porque otras bases de datos dependen de él.',
        canTransfer: 'El almacenamiento se puede transferir',
        selectPlaceholder: 'Seleccione un almacenamiento',
        createNew: 'Crear almacenamiento',
        addTitle: 'Añadir almacenamiento',
        description:
          'Un almacenamiento es el lugar donde se guardan las copias de seguridad (disco local, S3, Google Drive, etc.)',
      },
      notifiers: {
        title: 'Canales de notificación (opcional)',
        transferWithDatabase: 'Transferir los canales de notificación con la base de datos',
        selectFromTarget: 'Elegir canales de notificación del espacio de trabajo de destino',
        willTransfer: 'Se transferirán:',
        willNotTransfer: 'NO se transferirán (los usan otras bases de datos):',
        usedByDatabases: '{{name}} (bases de datos que lo usan: {{count}})',
        noneTransferred:
          'No se transferirá ningún canal de notificación. Después de la transferencia podrá elegir canales del espacio de trabajo de destino.',
        selectPlaceholder: 'Seleccione canales de notificación (opcional)',
      },
      submit: 'Transferir',
      transferred: {
        title: 'Base de datos transferida',
        description: 'Base de datos "{{name}}" transferida al nuevo espacio de trabajo',
      },
    },
    connectionString: {
      errors: {
        empty: 'La cadena de conexión está vacía',
        unrecognizedFormat: 'Formato de cadena de conexión no reconocido',
        hostMissing: 'Falta el host en la cadena de conexión',
        usernameMissing: 'Falta el usuario en la cadena de conexión',
        passwordMissing: 'Falta la contraseña en la cadena de conexión',
        databaseMissing: 'Falta el nombre de la base de datos en la cadena de conexión',
        parseFailed: 'No se pudo analizar la cadena de conexión',
        jdbc: {
          invalidFormat:
            'Formato de cadena de conexión JDBC no válido. Formato esperado: {{expectedFormat}}',
          queryParametersMissing:
            'Faltan los parámetros de consulta (user y password) en la cadena de conexión JDBC',
          usernameMissing: 'Falta el usuario (parámetro user) en la cadena de conexión JDBC',
          passwordMissing: 'Falta el parámetro password en la cadena de conexión JDBC',
          parseFailed: 'No se pudo analizar la cadena de conexión JDBC',
        },
        keyValue: {
          hostMissing: 'Falta el host en la cadena de conexión. Use host=hostname',
          usernameMissing: 'Falta el usuario en la cadena de conexión. Use user=username',
          passwordMissing:
            'Falta la contraseña en la cadena de conexión. Use password=yourpassword',
          databaseMissing:
            'Falta el nombre de la base de datos en la cadena de conexión. Use database=database',
          parseFailed: 'No se pudo analizar la cadena de conexión en formato clave=valor',
        },
        libpq: {
          databaseMissing:
            'Falta el nombre de la base de datos en la cadena de conexión. Use dbname=database',
          parseFailed: 'No se pudo analizar la cadena de conexión de libpq',
        },
      },
    },
    physicalConnectionErrors: {
      grantAsSuperuserNote: 'Concédalo con el comando de abajo (ejecútelo como superusuario).',
      applyAndRestartNote: 'Aplique el cambio de abajo y reinicie PostgreSQL.',
      pgHbaNoEntry: {
        title: 'No se permite la replicación desde este host',
        summary:
          'pg_hba.conf no tiene ninguna entrada que permita a Databasus una conexión de replicación. Las copias físicas se transmiten por replicación, y eso requiere una regla "host replication": una regla "host all" normal no la cubre.',
        addLineNote:
          'Añada la línea de abajo a pg_hba.conf. El primer "all" admite cualquier usuario y el segundo, cualquier host. Si quiere restringir el acceso, limite el usuario al rol de replicación concreto y/o el host a un rango CIDR.',
        reloadNote:
          'Después de modificar pg_hba.conf, <bold>recargue la configuración de PG</bold>.',
        managedNote:
          'En PostgreSQL gestionado (RDS / Azure / GCP) no se puede editar pg_hba.conf: habilite el acceso externo o de replicación en la consola del proveedor.',
      },
      badCredentials: {
        title: 'Usuario o contraseña incorrectos',
        summary:
          'PostgreSQL rechazó las credenciales. Compruebe el usuario y la contraseña de esta base de datos.',
      },
      noReplicationPrivilege: {
        title: 'El usuario no puede usar la replicación',
        summary:
          'El usuario se conectó, pero no tiene el privilegio REPLICATION que requieren las copias físicas.',
        managedNote:
          'En AWS RDS use GRANT rds_replication TO {{username}}; en Azure / GCP habilite la replicación para el rol en la consola del proveedor.',
      },
      noWalSwitchPrivilege: {
        title: 'El usuario no puede forzar el cambio de segmento WAL',
        summary:
          'El streaming continuo de WAL solo sube un segmento cuando el origen lo cierra, así que en una base de datos con pocas escrituras el WAL más reciente se queda en el origen hasta que se llena el segmento de 16 MB. Forzar el cambio con un temporizador es lo que mantiene el punto de recuperación entre copias cerca del presente, y para eso hace falta EXECUTE sobre pg_switch_wal(). Las copias completas e incrementales no lo necesitan: se restauran al momento en que terminó cada copia y no reproducen WAL archivado.',
        switchBackupTypeNote:
          'O cambie el tipo de copia de seguridad a <bold>Completas + incrementales</bold>, que no reproduce WAL archivado.',
        managedNote:
          'En PostgreSQL gestionado (RDS / Azure / GCP) la función pertenece al superusuario de la plataforma y no se puede conceder EXECUTE sobre ella a un rol de cliente, así que allí no está disponible el streaming continuo de WAL: use copias completas e incrementales.',
      },
      walLevelInvalid: {
        title: 'wal_level es demasiado bajo',
        summary:
          'Para las copias físicas, wal_level debe ser "replica" o "logical". Ahora tiene un valor inferior.',
        applyNote:
          'Aplique el cambio de abajo y reinicie PostgreSQL (wal_level solo surte efecto tras un reinicio).',
        managedNote:
          'En PostgreSQL gestionado, configure wal_level en el grupo de parámetros del proveedor.',
      },
      noWalSenders: {
        title: 'No hay procesos WAL sender disponibles',
        summary:
          'max_wal_senders es 0, así que PostgreSQL no puede transmitir WAL para las copias de seguridad.',
        managedNote:
          'En PostgreSQL gestionado, configure max_wal_senders en el grupo de parámetros del proveedor.',
      },
      noReplicationSlots: {
        title: 'No hay slots de replicación disponibles',
        summary:
          'max_replication_slots es 0, así que PostgreSQL no puede reservar un slot para las copias de seguridad.',
        managedNote:
          'En PostgreSQL gestionado, configure max_replication_slots en el grupo de parámetros del proveedor.',
      },
      walSummaryDisabled: {
        title: 'summarize_wal está desactivado',
        summary:
          'Las copias incrementales necesitan summarize_wal activado. Ahora está desactivado en este servidor.',
        applyNote: 'Aplique el cambio de abajo (no hace falta reiniciar).',
        thenNote: 'y después',
        managedNote:
          'En PostgreSQL gestionado, configure summarize_wal en el grupo de parámetros del proveedor.',
      },
      customTablespaces: {
        title: 'No se admiten tablespaces personalizados',
        summary:
          'Este clúster tiene tablespaces fuera de pg_default / pg_global, y las copias físicas no pueden transmitirlos. Elimine los tablespaces personalizados o cambie esta base de datos a copias lógicas.',
      },
      systemIdentifierMismatch: {
        title: 'Es un clúster distinto',
        summary:
          'El clúster de esta dirección tiene un identificador de sistema distinto del clúster para el que se configuró esta copia de seguridad. Apunte Databasus al clúster original o cree una nueva entrada de base de datos.',
      },
      connectionFailed: {
        title: 'No se pudo conectar',
        summary:
          'Databasus no pudo abrir una conexión de replicación. Compruebe el host, el puerto y el modo SSL, y que ningún cortafuegos bloquee la conexión.',
      },
    },
  },
  restores: {
    list: {
      empty: 'Todavía no hay restauraciones',
    },
    targetDatabase: {
      select: 'Elegir la base de datos de destino',
      description:
        'Introduzca los datos de la base de datos en la que se restaurará la copia de seguridad. <underline>La base de datos vacía debe crearse antes de la restauración</underline>. Durante la restauración se borrarán todos sus datos actuales',
      notInUseHint:
        'Asegúrese de que nadie esté usando la base de datos ahora mismo (lo más probable es que no quiera restaurar los datos en la misma BD de la que se hizo la copia)',
      submit: 'Restaurar en esta BD',
    },
    card: {
      startedAt: 'Inicio',
      duration: 'Duración',
      errorDetailsTooltip: 'Haga clic para ver los detalles del error',
      cancelTooltip: 'Cancelar restauración',
      expectedDurationHint:
        'La restauración suele tardar de 3 a 5 veces más que la copia de seguridad (a veces menos, a veces más, según el tipo de datos)',
      expectedDurationLimit: 'Así que puede tardar hasta {{duration}} (normalmente bastante menos)',
    },
    errorDetails: {
      title: 'Detalles del error de restauración',
      copied: 'Mensaje de error copiado al portapapeles',
      excludeExtensionsTip:
        '<bold>💡 Consejo:</bold> este error suele aparecer al restaurar en servicios de PostgreSQL gestionados (como Yandex Cloud, AWS RDS o similares). Pruebe a activar <bold>"Excluir extensiones"</bold> en los ajustes avanzados antes de restaurar.',
    },
    cancelConfirmation: {
      warning:
        '<bold>⚠️ Advertencia:</bold> si cancela esta restauración, lo más probable es que la base de datos quede dañada o incompleta. Tendrá que volver a crearla antes de intentar otra restauración.',
      question: '¿Seguro que desea cancelar?',
      confirm: 'Sí, cancelar la restauración',
    },
  },
  healthcheck: {
    attempts: {
      title: 'Comprobaciones de disponibilidad',
      period: 'Periodo',
      periods: {
        today: 'Hoy',
        sevenDays: '7 días',
        thirtyDays: '30 días',
        allTime: 'Todo',
      },
      empty: 'Todavía no hay datos',
    },
    config: {
      enable: 'Activar comprobación de disponibilidad',
      isEnabled: 'Comprobación de disponibilidad activada',
      notifyWhenUnavailable: 'Notificar si no está disponible',
      checkIntervalMinutes: 'Intervalo de comprobación (minutos)',
      attemptsBeforeDown: 'Intentos antes de marcar caída',
      storeAttemptsDays: 'Guardar historial (días)',
      tooltips: {
        enable: 'Activa o desactiva la comprobación de disponibilidad de esta base de datos',
        notifyWhenUnavailable:
          'Envía notificaciones cuando la base de datos deja de estar disponible',
        checkIntervalMinutes:
          'Con qué frecuencia se comprueba la disponibilidad de la base de datos (en minutos)',
        attemptsBeforeDown:
          'Número de intentos fallidos antes de marcar la base de datos como no disponible',
        storeAttemptsDays:
          'Cuántos días se guarda el historial de comprobaciones de disponibilidad',
      },
    },
  },
  workspaces: {
    roles: {
      owner: 'Propietario',
      admin: 'Administrador',
      member: 'Miembro',
      viewer: 'Lector',
    },
    fields: {
      name: 'Nombre del espacio de trabajo',
      namePlaceholder: 'Introduzca el nombre del espacio de trabajo',
    },
    create: {
      title: 'Crear espacio de trabajo',
      submit: 'Crear espacio de trabajo',
      defaultName: 'Mi espacio de trabajo',
      description:
        'Un espacio de trabajo agrupa:<lineBreak/>- sus bases de datos;<lineBreak/>- almacenamientos (disco local, S3, Google Drive, etc.);<lineBreak/>- canales de notificación (correo electrónico, Slack, Telegram, etc.);<lineBreak/>- control de acceso (si trabaja en equipo).',
      nameRequired: 'Introduzca un nombre para el espacio de trabajo',
      created: 'Espacio de trabajo creado',
      permissionDenied: {
        title: 'Permiso denegado',
        description:
          'No tiene permiso para crear espacios de trabajo. Pida a un administrador que lo cree por usted.',
      },
    },
    settings: {
      title: 'Ajustes del espacio de trabajo',
      readOnly: 'No tiene permiso para modificar estos ajustes',
      nameRequired: 'El nombre del espacio de trabajo es obligatorio',
      saving: 'Guardando...',
      saveChanges: 'Guardar cambios',
      reset: 'Restablecer',
      updated: 'Información básica guardada',
    },
    delete: {
      dangerZone: 'Zona de peligro',
      title: 'Eliminar este espacio de trabajo',
      description:
        'Un espacio de trabajo eliminado no se puede recuperar. Todos sus datos y recursos asociados se eliminarán de forma permanente.',
      submit: 'Eliminar espacio de trabajo',
      deleting: 'Eliminando...',
      notFound: 'No se encontró el espacio de trabajo',
      noPermission: 'No tiene permiso para eliminar este espacio de trabajo',
      hasDatabases:
        'No se puede eliminar el espacio de trabajo. Elimine primero todas sus bases de datos. Bases de datos encontradas: {{count}}.',
      deleted: 'Espacio de trabajo eliminado',
      confirmation: {
        title: 'Eliminar espacio de trabajo',
        question: '¿Seguro que desea eliminar el espacio de trabajo <workspaceName/>?',
        warning:
          '<bold>Esta acción no se puede deshacer.</bold> Todos los datos y recursos asociados se eliminarán de forma permanente.',
        submit: 'Eliminar espacio de trabajo',
      },
    },
    members: {
      title: 'Usuarios',
      count: 'Miembros: {{count}}',
      empty: 'No hay miembros',
      emptyHint: 'Haga clic en "Añadir miembro" para empezar',
      fields: {
        member: 'Miembro',
        role: 'Rol',
        joined: 'Miembro desde',
        actions: 'Acciones',
        email: 'Correo electrónico',
        emailPlaceholder: 'Introduzca el correo electrónico',
      },
      add: {
        button: 'Añadir miembro',
        title: 'Añadir miembro',
        submit: 'Añadir miembro',
        emailRequired: 'El correo electrónico es obligatorio',
        hint: 'Si el usuario ya existe, se añadirá directamente. Si no, se le enviará una invitación.',
        added: 'Miembro añadido',
      },
      invited: {
        title: 'Usuario invitado',
        sentTo: 'Invitación enviada a {{email}}',
        description:
          'El usuario todavía no existe en el sistema, pero ya está invitado al espacio de trabajo. Cuando se registre con ese correo electrónico, pasará a ser miembro del espacio de trabajo automáticamente.',
      },
      roleChanged: 'Rol del miembro actualizado',
      remove: {
        tooltip: 'Eliminar miembro',
        title: 'Eliminar miembro',
        question: '¿Seguro que desea eliminar a "{{email}}" de este espacio de trabajo?',
        removed: 'Miembro "{{email}}" eliminado',
      },
    },
    transferOwnership: {
      button: 'Transferir la propiedad',
      title: 'Transferir la propiedad del espacio de trabajo',
      submit: 'Transferir la propiedad',
      warning:
        '<bold>Advertencia:</bold> esta acción no se puede deshacer. Dejará de ser el propietario de este espacio de trabajo y el nuevo propietario tendrá el control total.',
      noEligibleMembers:
        'No hay miembros a los que transferir la propiedad. Para hacerlo, el espacio de trabajo necesita al menos otro miembro.',
      newOwner: 'Nuevo propietario',
      newOwnerPlaceholder: 'Seleccione el miembro que recibirá la propiedad',
      newOwnerHint: 'El miembro seleccionado pasará a ser el propietario del espacio de trabajo',
      memberRequired: 'Seleccione el miembro al que transferir la propiedad',
      memberNotFound: 'No se encontró el miembro seleccionado',
      transferred: 'Propiedad transferida',
    },
    auditLogs: {
      empty: 'No hay registros de auditoría de este espacio de trabajo.',
    },
  },
  users: {
    roles: {
      admin: 'Administrador',
      member: 'Miembro',
    },
    fields: {
      name: 'Nombre',
      email: 'Correo electrónico',
      yourName: 'Su nombre',
      yourEmail: 'Su correo electrónico',
      confirmPassword: 'Confirmar contraseña',
      newPassword: 'Nueva contraseña',
    },
    validation: {
      nameRequired: 'El nombre es obligatorio',
      emailRequired: 'El correo electrónico es obligatorio',
      passwordTooShort: 'La contraseña debe tener al menos {{minLength}} caracteres',
      passwordsDoNotMatch: 'Las contraseñas no coinciden',
    },
    oauth: {
      divider: 'o con correo electrónico',
      continueWithGithub: 'Continuar con GitHub',
      continueWithGoogle: 'Continuar con Google',
      invalidConfiguration: 'Configuración de OAuth no válida',
    },
    signIn: {
      title: 'Iniciar sesión',
      submit: 'Iniciar sesión',
      noAccount: '¿No tiene cuenta? <signUpLink>Regístrese</signUpLink>',
      forgotPassword: '¿Olvidó su contraseña?',
    },
    signInCode: {
      title: 'Introduzca el código',
      description:
        'Hemos enviado un código de seis dígitos a {{email}}. Deja de funcionar 10 minutos después de enviarse.',
      code: 'Código de acceso',
      submit: 'Iniciar sesión',
      resend: 'Enviar otro código',
      backToSignIn: 'Volver al inicio de sesión',
      invalidCode: 'El código tiene 6 dígitos',
      codeResent: 'Hemos enviado un código nuevo. El anterior ya no funciona.',
    },
    signUp: {
      title: 'Registro',
      namePlaceholder: 'Juan Pérez',
      submit: 'Registrarse',
      administersInstance: 'Esta es la primera cuenta de esta instancia, así que la administrará.',
      hasAccount: '¿Ya tiene cuenta? <signInLink>Inicie sesión</signInLink>',
    },
    requestPasswordReset: {
      title: 'Restablecer contraseña',
      description:
        'Introduzca su correo electrónico y le enviaremos un código para restablecer la contraseña.',
      submit: 'Enviar código',
      codeSent: 'Si el correo electrónico existe, se ha enviado un código de restablecimiento',
      rememberPassword: '¿Recuerda su contraseña? <signInLink>Inicie sesión</signInLink>',
    },
    resetPassword: {
      title: 'Restablecer contraseña',
      description: 'Introduzca el código que recibió por correo electrónico y su nueva contraseña.',
      code: 'Código de restablecimiento',
      confirmPassword: 'Confirmar contraseña',
      submit: 'Restablecer contraseña',
      invalidCode: 'El código debe tener 6 dígitos',
      succeeded: 'Contraseña restablecida. Redirigiendo al inicio de sesión...',
      noCode: '¿No recibió el código? <requestCodeLink>Solicite uno nuevo</requestCodeLink>',
      backToSignIn: 'Volver al inicio de sesión',
    },
    profile: {
      title: 'Perfil',
      information: 'Información del perfil',
      userId: 'ID de usuario',
      namePlaceholder: 'Introduzca su nombre',
      emailPlaceholder: 'Introduzca su correo electrónico',
      role: 'Rol',
      saveChanges: 'Guardar cambios',
      noChanges: 'No hay cambios que guardar',
      updated: 'Perfil actualizado',
      logout: 'Cerrar sesión',
      changePassword: {
        title: 'Cambiar contraseña',
        newPasswordPlaceholder: 'Introduzca la nueva contraseña',
        confirmPassword: 'Confirmar nueva contraseña',
        confirmPasswordPlaceholder: 'Repita la nueva contraseña',
        passwordsDoNotMatch: 'Las contraseñas nuevas no coinciden',
        changing: 'Cambiando la contraseña...',
        submit: 'Cambiar contraseña',
        signedInWithNewPassword: 'Sesión iniciada con la nueva contraseña',
      },
    },
    list: {
      title: 'Usuarios de Databasus',
      loadedOfTotal: 'Usuarios: {{loaded}} de {{total}}',
      searchPlaceholder: 'Buscar por correo electrónico o nombre...',
      empty: 'No se encontraron usuarios.',
      allLoaded: 'Todos los usuarios cargados ({{count}} en total)',
      columns: {
        user: 'Usuario',
        systemRole: 'Rol en el sistema',
        isActive: '¿Activo?',
        created: 'Fecha de creación',
      },
      roleLabel: 'Rol:',
      activeLabel: 'Activo:',
      viewAuditLogs: 'Ver registros de auditoría',
      activated: 'Usuario activado',
      deactivated: 'Usuario desactivado',
      roleChanged: 'Rol del usuario cambiado',
    },
    auditLogs: {
      title: 'Registros de auditoría del usuario',
      empty: 'No hay registros de auditoría de este usuario.',
    },
  },
  verification: {
    triggers: {
      manual: 'Manual',
      scheduled: 'Programada',
    },
    notificationTypes: {
      verificationSuccess: 'Verificación superada',
      verificationFailed: 'Verificación fallida',
    },
    config: {
      scheduledVerification: 'Verificación programada',
      scheduledVerificationTooltip:
        'Restaura periódicamente la última copia de seguridad en un contenedor de Postgres efímero para comprobar que se puede restaurar. <docsLink>Cómo funciona</docsLink>',
      interval: 'Intervalo de verificación',
      afterBackup: 'Tras cada copia de seguridad',
      afterBackupHint:
        'Se ejecuta automáticamente después de cada copia de seguridad correcta. Si ya hay verificaciones pendientes en la cola, se cancelan para que la cola no crezca sin fin cuando las copias se hacen más rápido que las verificaciones.',
      weekday: 'Día de la semana de la verificación',
      dayOfMonth: 'Día del mes de la verificación',
      timeOfDay: 'Hora de la verificación',
      cron: {
        expression: 'Expresión cron (UTC)',
        format: 'Formato cron: minuto, hora, día, mes, día de la semana (UTC)',
        examples: 'Ejemplos:',
        exampleEverySunday: '- {{expression}} — todos los domingos a las 4:00 UTC',
        exampleEverySixHours: '- {{expression}} — cada 6 horas',
        nextRun: 'Próxima ejecución: {{dateTime}}<lineBreak/>({{relativeTime}})',
        invalid: 'Expresión cron no válida',
      },
      notifications: 'Notificaciones',
      noNotifications: 'Ninguna',
    },
    runs: {
      title: 'Verificaciones de restauración',
      description:
        'Cada fila es un intento de restaurar una copia de seguridad de esta base de datos en una base de datos temporal',
      noAgentsWarning:
        'No hay agentes de verificación registrados. Añada uno en los ajustes de Databasus, en la sección "Agentes de verificación". <docsLink>Más información</docsLink>',
      empty: 'Todavía no hay verificaciones de restauración',
      allLoaded: 'Todas las verificaciones cargadas ({{total}} en total)',
      attempt: 'intento {{attempt}}',
      restoreDuration: 'restauración: {{duration}}',
      durationTooltip: 'Tiempo de restauración + verificación informado por el agente.',
      fields: {
        createdAt: 'Fecha de creación',
        trigger: 'Ejecución',
        duration: 'Duración',
        actions: 'Acciones',
      },
      actions: {
        cancel: 'Cancelar verificación',
        viewDetails: 'Ver detalles',
      },
      details: {
        title: 'Detalles de la verificación de restauración',
        loadFailed: 'No se pudieron cargar los detalles de la verificación de restauración.',
        cancellationReason: 'Motivo de la cancelación',
        failure: 'Error',
        sections: {
          status: 'Estado',
          timeline: 'Cronología',
          results: 'Resultados y diagnóstico',
        },
        fields: {
          attempt: 'Intento',
          startedAt: 'Inicio',
          finishedAt: 'Fin',
          restoreDuration: 'Duración de la restauración',
          verifyDuration: 'Duración de la verificación',
          restoredDbSize: 'Tamaño de la BD restaurada',
          schemas: 'Esquemas',
          tables: 'Tablas',
          pgRestoreExitCode: 'Código de salida de pg_restore',
        },
        tableStats: {
          title: 'Filas por tabla',
          empty: 'No hay estadísticas por tabla.',
          schema: 'Esquema',
          table: 'Tabla',
          rows: 'Filas',
        },
      },
    },
    agents: {
      title: 'Agentes de verificación',
      description:
        'Agentes que ejecutan verificaciones de restauración para confirmar que una copia de seguridad se puede restaurar (<docsLink>más información</docsLink>)',
      empty: 'Todavía no hay agentes registrados.',
      columns: {
        capacity: 'Capacidad',
        created: 'Fecha de creación',
      },
      capacity: {
        cpu: '{{count}} CPU',
        ram: '{{count}} GB de RAM',
        disk: '{{count}} GB de disco',
        jobs: 'trabajos: {{count}}',
        notReported: 'sin datos todavía',
      },
      actions: {
        viewInstallCommands: 'Ver comandos de instalación',
        rotateToken: 'Rotar token',
      },
      deleteConfirmation: '¿Eliminar este agente?',
      deleted: 'Agente "{{name}}" eliminado',
      copyFailed: 'No se pudo copiar',
      agentId: 'ID del agente:',
      create: {
        title: 'Crear agente de verificación',
        tokenNotice: 'Se generará un token que se mostrará una sola vez en la siguiente pantalla.',
        namePlaceholder: 'Nombre del agente',
        nameRequired: 'El nombre es obligatorio',
      },
      rotate: {
        title: 'Rotar token',
        description:
          'Al rotar el token de <agentName/>, el token actual deja de ser válido de inmediato. Cualquier agente que siga usando el token antiguo será rechazado en su siguiente heartbeat.',
      },
      token: {
        title: 'Token del agente',
        tokenFor: 'Token de <agentName/>:',
        shownOnce:
          'Se muestra una sola vez. Guárdelo en un lugar seguro: no podrá recuperarlo después.',
        confirmSaved: 'He guardado el token',
      },
      install: {
        title: 'Comandos de instalación',
        description:
          'Comandos de instalación de <agentName/>. Sustituya <tokenPlaceholder/> por el token que guardó al crear este agente o al rotar su token por última vez.',
        architecture: 'Arquitectura',
        stepInstall: 'Paso 1: instalar',
        stepLaunch: 'Paso 2: iniciar',
        capacityDefaults:
          'Los valores de capacidad de abajo son solo un punto de partida: ajuste <code>--max-cpu</code>, <code>--max-ram-mb</code>, <code>--max-disk-gb</code> y <code>--max-concurrent-jobs</code> a la máquina que ejecuta el agente.',
        afterInstallation: 'Después de la instalación',
        runsInBackground: 'Tras <code>start</code>, el agente se ejecuta en segundo plano',
        checkStatus: 'Ver el estado: <code>./verification-agent status</code>',
        viewLogs:
          'Ver los logs: <code>databasus-verification.log</code> en el directorio de trabajo',
        stop: 'Detener el agente: <code>./verification-agent stop</code>',
      },
    },
  },
  notifiers: {
    types: {
      email: 'Correo electrónico',
      telegram: 'Telegram',
      webhook: 'Webhook',
      slack: 'Slack',
      discord: 'Discord',
      teams: 'Teams',
      mattermost: 'Mattermost',
    },
    notificationTypes: {
      all: 'Todas',
      backupSuccess: 'Copia de seguridad correcta',
      backupFailed: 'Copia de seguridad fallida',
      healthcheckSuccess: 'Comprobación de disponibilidad superada',
      healthcheckFailed: 'Comprobación de disponibilidad fallida',
      verificationSuccess: 'Verificación de restauración superada',
      verificationFailed: 'Verificación de restauración fallida',
    },
    mattermostDeliveryModes: {
      webhook: 'Webhook entrante',
      bot: 'Cuenta de bot',
    },
    list: {
      addNotifier: 'Añadir canal de notificación',
      searchPlaceholder: 'Buscar canal de notificación',
      noSearchResults: 'No se encontraron canales de notificación que coincidan con "{{query}}"',
      description:
        'Un canal de notificación es el lugar adonde se envían las notificaciones (correo electrónico, Slack, Telegram, etc.)',
      backToList: '← Volver a los canales de notificación',
    },
    card: {
      notifyTo: {
        email: 'Notificar por correo electrónico',
        telegram: 'Notificar por Telegram',
        webhook: 'Notificar por webhook',
        slack: 'Notificar por Slack',
        discord: 'Notificar por Discord',
        teams: 'Notificar por Teams',
        mattermost: 'Notificar por Mattermost',
      },
      hasSendError: 'Error de envío',
    },
    create: {
      title: 'Añadir canal de notificación',
      namePlaceholder: 'Mi chat',
    },
    config: {
      namePlaceholder: 'Introduzca un nombre...',
      settingsTitle: 'Ajustes del canal de notificación',
      sendError: {
        title: 'Error de envío',
        errorLabel: 'Error:',
        clearHint: 'Para quitar este error, haga una de estas dos cosas:',
        clearBySendingTest:
          '- envíe una notificación de prueba con el botón de abajo (aunque ya haya cambiado los ajustes);',
        clearByNextNotification: '- espere a que la siguiente notificación se envíe sin errores;',
      },
      removeConfirmation:
        '¿Seguro que desea eliminar este canal de notificación? Esta acción no se puede deshacer.',
      removeBlockedByDatabases:
        'Algunas bases de datos usan este canal de notificación. Quítelo primero de esas bases de datos.',
    },
    testNotification: {
      send: 'Enviar notificación de prueba',
      sent: {
        title: 'Notificación de prueba enviada',
        description: 'La notificación de prueba se envió correctamente',
      },
    },
    transfer: {
      title: 'Transferir el canal de notificación a otro espacio de trabajo',
      blockedByDatabases:
        'Algunas bases de datos usan este canal de notificación. Transfiera o elimine primero esas bases de datos.',
      description: 'Seleccione el espacio de trabajo al que transferir este canal de notificación.',
      targetWorkspace: 'Espacio de trabajo de destino',
      selectWorkspace: 'Seleccione un espacio de trabajo',
      submit: 'Transferir',
    },
    fields: {
      botToken: 'Token del bot',
      targetChatId: 'ID del chat de destino',
      webhookUrl: 'URL del webhook',
      channelId: 'ID del canal',
      skipTlsVerify: 'Omitir verificación TLS',
      skipTls: 'No verificar TLS',
    },
    email: {
      targetEmail: 'Correo electrónico de destino',
      targetEmailTooltip:
        'La dirección de correo electrónico en la que quiere recibir los mensajes',
      smtpHost: 'Host SMTP',
      smtpPort: 'Puerto SMTP',
      smtpUser: 'Usuario SMTP',
      smtpPassword: 'Contraseña SMTP',
      smtpPasswordPlaceholder: 'contraseña',
      from: 'Remitente',
      fromTooltip:
        'Opcional. Dirección del remitente, con o sin nombre: Acme Backups <noreply@example.com>. Si se deja vacía, se usa el usuario SMTP cuando es una dirección de correo y, si no, noreply@ con el host SMTP',
      fromAuto: '(automático)',
      advancedSettings: 'Ajustes avanzados',
      skipTlsTooltip:
        'No verifica el certificado TLS. Active esta opción si su servidor SMTP usa un certificado autofirmado. Advertencia: esto reduce la seguridad.',
      security: 'Seguridad de la conexión',
      securityTooltip:
        'TLS cifra la conexión desde el principio y suele usarse en el puerto 465. STARTTLS cifra la conexión después de abrirla y falla si el servidor no lo admite. «Sin cifrado» envía la contraseña y los mensajes sin cifrar: úselo solo con un relé de confianza que no admita STARTTLS.',
      securityModes: {
        tls: 'TLS',
        starttls: 'STARTTLS',
        none: 'Sin cifrado',
      },
      heloName: 'Nombre de saludo',
      heloNamePlaceholder: 'Automático',
      heloNameTooltip:
        'El nombre con el que Databasus se presenta al servidor de correo (EHLO). Si se deja vacío, se usa el host de DATABASUS_URL o el nombre del equipo.',
    },
    telegram: {
      botTokenHelpLink: '¿Cómo obtener el token de API del bot de Telegram?',
      targetChatIdTooltip:
        'El chat en el que quiere recibir los mensajes (puede ser su chat privado o un grupo)',
      chatIdHelp: {
        toggle: '¿Cómo obtener el ID del chat de Telegram?',
        privateChat:
          'Para obtener el ID de su chat, escriba a <botLink>@getmyid_bot</botLink> en Telegram. <underline>Asegúrese de haber iniciado el chat con el bot</underline>',
        group:
          'Para obtener el ID de un chat de grupo, añada al grupo su bot y <botLink>@getmyid_bot</botLink> y escriba /start (verá el ID del chat)',
      },
      useProxy: 'Usar proxy',
      useProxyTooltip: 'Usa un proxy para las solicitudes a la API de Telegram',
      proxy: 'Proxy',
      proxyUrl: 'URL del proxy',
      proxyUrlTooltip: 'Admite http, https, socks5 y socks5h. Puede incluir usuario y contraseña',
      sendToGroupTopic: 'Enviar a un tema del grupo',
      sendToGroupTopicTooltip:
        'Active esta opción para enviar los mensajes a un tema concreto de un chat de grupo',
      threadId: 'ID del tema',
      threadIdTooltip: 'El ID del tema al que se enviarán los mensajes',
      topicId: 'ID del tema',
      threadIdHelp: {
        steps:
          'Para obtener el ID del tema, abra el tema en su grupo de Telegram, toque su nombre en la parte superior y abra la información del tema. Copie el enlace del tema y tome el último número de la URL.',
        example:
          '<bold>Ejemplo:</bold> si el enlace del tema es <code>https://t.me/c/2831948048/3</code>, el ID del tema es <code>3</code>',
        note: '<bold>Nota:</bold> los temas solo funcionan en chats de grupo, no en chats privados.',
      },
    },
    webhook: {
      method: 'Método',
      sendOn: 'Cuándo enviar',
      sendOnPlaceholder: 'Seleccione los tipos de notificación',
      customHeaders: 'Cabeceras personalizadas',
      customHeadersTooltip:
        'Añada cabeceras HTTP personalizadas a la solicitud del webhook (p. ej., Authorization, X-API-Key)',
      savedHeadersHidden: '*Las cabeceras guardadas se ocultan por seguridad',
      headers: 'Cabeceras',
      headerNamePlaceholder: 'Nombre de la cabecera',
      headerValuePlaceholder: 'Valor de la cabecera',
      addHeader: 'Añadir cabecera',
      hiddenHeaderValue: '(oculto)',
      bodyTemplate: {
        title: 'Plantilla del cuerpo',
        headingVariable: '<variable/> — título de la notificación',
        messageVariable: '<variable/> — texto de la notificación',
        invalidJsonFormat: 'Formato JSON no válido',
        invalidJson: 'JSON no válido',
      },
      exampleRequest: {
        title: 'Ejemplo de solicitud',
        headers: 'Cabeceras:',
      },
    },
    slack: {
      howToConnect: '¿Cómo conectar Slack (cómo obtener el token del bot y el ID del chat)?',
      testFailedHint:
        'Asegúrese de que el canal sea público o de que el bot se haya añadido al canal privado (con @invite) o al grupo. Para mensajes directos, use el User ID del perfil de Slack.',
    },
    discord: {
      channelWebhookUrl: 'URL del webhook del canal',
      howTo: {
        title: 'Cómo obtener la URL del webhook de Discord:',
        createChannel: '1. Cree o elija un canal de Discord',
        openChannelSettings: '2. Abra los ajustes del canal (icono del engranaje)',
        openIntegrations: '3. Vaya a Integraciones',
        createWebhook: '4. Cree un webhook nuevo',
        copyWebhookUrl: '5. Copie la URL del webhook',
        privacyNote: 'Nota: si hace falta, haga privado el canal',
      },
    },
    teams: {
      howToConnect: '¿Cómo conectar Microsoft Teams?',
      powerAutomateUrl: 'URL de Power Automate',
      powerAutomateUrlLabel: 'URL de Power Automate:',
      powerAutomateUrlTooltip:
        'Endpoint HTTP de su flujo de Power Automate (When an HTTP request is received)',
      showFullUrl: 'Mostrar',
      hideFullUrl: 'Ocultar',
    },
    mattermost: {
      howToConnect: '¿Cómo conectar Mattermost?',
      connectVia: 'Conectar mediante',
      incomingWebhookUrl: 'URL del webhook entrante',
      serverUrl: 'URL del servidor',
      channelIdTooltip:
        'ID del canal de 26 caracteres, no su nombre. Abra el canal, haga clic en su nombre y elija View Info.',
      channel: 'Canal',
      showOptionalSettings: 'Mostrar ajustes opcionales',
      hideOptionalSettings: 'Ocultar ajustes opcionales',
      channelOverride: 'Otro canal',
      channelOverrideTooltip:
        'Publica en este canal en lugar de en el canal para el que se creó el webhook. Mattermost lo ignora si el webhook está restringido a un canal.',
      postAsUsername: 'Publicar con el nombre de usuario',
      postAsUsernameTooltip:
        'Requiere activar Enable integrations to override usernames en la consola del sistema de Mattermost; si no, se ignora.',
      postAsIconUrl: 'Publicar con la URL del icono',
      postAsIconUrlTooltip:
        'Requiere activar Enable integrations to override profile picture icons en la consola del sistema de Mattermost; si no, se ignora.',
      skipTlsTooltip:
        'No verifica el certificado TLS. Active esta opción si su servidor de Mattermost usa un certificado autofirmado. Advertencia: esto reduce la seguridad.',
      webhookHowTo: {
        title: 'Cómo obtener la URL de un webhook entrante:',
        openIncomingWebhooks: '1. Menú principal → Integrations → Incoming Webhooks',
        addWebhook: '2. Pulse Add Incoming Webhook y elija el canal',
        copyUrl: '3. Copie la URL generada',
      },
      botHowTo: {
        title: 'Cómo obtener un token de bot:',
        addBotAccount: '1. Integrations → Bot Accounts → Add Bot Account',
        copyToken: '2. Copie el token, que solo se muestra una vez, justo después de crearlo',
        addBotToChannel: '3. Añada el bot al equipo y al canal',
      },
    },
  },
  backups: {
    encryption: {
      none: 'Ninguno',
      encrypted: 'Activado',
    },
    encryptionOptions: {
      none: 'Sin cifrado',
      encrypted: 'Cifrar los archivos de las copias de seguridad',
    },
    list: {
      title: 'Copias de seguridad',
      scheduledBackupsDisabled:
        'Las copias de seguridad programadas están desactivadas (puede volver a activarlas en la configuración de copias de seguridad)',
      empty: 'Todavía no hay copias de seguridad',
      allLoaded: 'Todas las copias de seguridad cargadas ({{count}} en total)',
      clickToSeeErrorDetails: 'Haga clic para ver los detalles del error',
      errorDetailsTitle: 'Detalles del error de la copia de seguridad',
      restoreTitle: 'Restaurar desde la copia de seguridad',
      columns: {
        createdAt: 'Fecha de creación',
        created: 'Fecha de creación',
        size: 'Tamaño',
        duration: 'Duración',
        actions: 'Acciones',
      },
      actions: {
        cancelBackup: 'Cancelar copia de seguridad',
        deleteBackup: 'Eliminar copia de seguridad',
      },
    },
    filters: {
      allTypes: 'Todos los tipos',
      allStatuses: 'Todos los estados',
      before: 'Antes de',
    },
    config: {
      backupsEnabled: 'Copias de seguridad activadas',
      storage: 'Almacenamiento',
      selectStorage: 'Seleccione un almacenamiento',
      createNewStorage: 'Crear almacenamiento',
      addStorageTitle: 'Añadir almacenamiento',
      storageDescription:
        'Un almacenamiento es el lugar donde se guardan las copias de seguridad (disco local, S3, Google Drive, etc.)',
      storageChangeWarning:
        'Si cambia el almacenamiento, se eliminarán todas las copias de seguridad guardadas en él.',
      storageChangeAcknowledge: 'Entendido',
      encryption: 'Cifrado',
      notifications: 'Notificaciones',
      noNotifications: 'Ninguna',
      schedule: {
        weekday: 'Día de la semana',
        dayOfMonth: 'Día del mes',
        cronExpression: 'Expresión cron (UTC)',
        timeOfDay: 'Hora del día',
        nextRun: 'Próxima ejecución: {{time}}',
        invalidCron: 'Expresión cron no válida',
        cronHelp: {
          format: 'Formato cron: minuto, hora, día, mes, día de la semana (UTC)',
          examples: 'Ejemplos:',
          dailyAt2am: 'Todos los días a las 2:00 UTC',
          every6Hours: 'Cada 6 horas',
          mondaysAt3am: 'Todos los lunes a las 3:00 UTC',
          twiceAMonth: 'Los días 1 y 15 a las 4:30 UTC',
        },
      },
      gfs: {
        hourly: 'Horarias: {{count}}',
        daily: 'Diarias: {{count}}',
        weekly: 'Semanales: {{count}}',
        monthly: 'Mensuales: {{count}}',
        yearly: 'Anuales: {{count}}',
        notConfigured: 'Sin configurar',
      },
    },
    logical: {
      retentionPolicyTypes: {
        gfs: 'GFS (conservar las últimas N copias horarias, diarias, semanales, mensuales y anuales)',
        timePeriod: 'Periodo de tiempo (últimos N días)',
        count: 'Cantidad (últimas N copias)',
      },
      notificationTypes: {
        backupSuccess: 'Copia de seguridad correcta',
        backupFailed: 'Copia de seguridad fallida',
      },
      list: {
        backupNow: 'Hacer copia',
        makeBackupNow: 'Hacer copia de seguridad ahora',
        encrypted: 'Cifrada',
        sizeTooltip:
          'Arriba: el tamaño del archivo comprimido de la copia de seguridad que realmente se guarda en el almacenamiento (local, S3, Google Drive, etc.). Abajo (en gris): el tamaño original sin comprimir de la base de datos en el momento de la copia. Las copias suelen comprimirse unas 5 veces.',
        databaseSize: '{{size}} (tamaño de la BD)',
        deleteConfirmation: '¿Seguro que desea eliminar esta copia de seguridad?',
        actions: {
          restore: 'Restaurar desde la copia de seguridad',
          verifyRestore:
            'Verificar la restauración: un agente restaurará esta copia de seguridad en una base de datos temporal e informará del número de filas',
          download: {
            postgresql:
              'Descargar el archivo de la copia de seguridad. Se puede restaurar manualmente con pg_restore (formato custom)',
            mysql:
              'Descargar el archivo de la copia de seguridad. Se puede restaurar manualmente con el cliente mysql (volcado SQL)',
            mariadb:
              'Descargar el archivo de la copia de seguridad. Se puede restaurar manualmente con el cliente mariadb (volcado SQL)',
            mongodb:
              'Descargar el archivo de la copia de seguridad. Se puede restaurar manualmente con mongorestore (formato archive)',
            other: 'Descargar el archivo de la copia de seguridad',
          },
        },
        verifyRestore: {
          title: '¿Verificar la restauración?',
          description:
            'Un agente restaurará esta copia de seguridad en una base de datos temporal e informará del número de filas. Puede tardar un rato, según el tamaño de la copia. <docsLink>¿Cómo funciona?</docsLink>',
          queue: 'Poner en cola',
          queued: 'Verificación de restauración en cola',
        },
      },
      config: {
        interval: 'Intervalo de copia de seguridad',
        weekday: 'Día de la semana de la copia',
        dayOfMonth: 'Día del mes de la copia',
        timeOfDay: 'Hora de la copia',
        encryptionTooltip:
          'Si la copia de seguridad está cifrada, sus archivos en el almacenamiento (S3, local, etc.) no se pueden usar directamente. Puede restaurar las copias a través de Databasus o descargarlas sin cifrar con el botón "Descargar".',
        retentionPolicy: 'Política de retención',
        retention: {
          timePeriodTooltip:
            'Cuánto tiempo se conservan las copias de seguridad. Las copias más antiguas que este periodo se eliminan automáticamente.',
          count: 'Copias más recientes',
          countTooltip:
            'Conserva solo el número indicado de copias más recientes. Las copias más antiguas se eliminan automáticamente.',
          countValue: 'Copias más recientes: {{count}}',
          showGfsHint: '¿Qué es GFS (Grandfather-Father-Son)?',
          hideGfsHint: 'Ocultar',
          gfsHint:
            'Rotación GFS (Grandfather-Father-Son): conserva las últimas N copias horarias, diarias, semanales, mensuales y anuales. Así se pueden guardar copias durante largos periodos sin ocupar demasiado espacio de almacenamiento.',
          gfsTooltip:
            'Rotación Grandfather-Father-Son: conserva las últimas N copias horarias, diarias, semanales, mensuales y anuales.',
          gfsFields: {
            hourly: 'Copias horarias',
            daily: 'Copias diarias',
            weekly: 'Copias semanales',
            monthly: 'Copias mensuales',
            yearly: 'Copias anuales',
          },
        },
        advancedSettings: 'Ajustes avanzados',
        retryIfFailed: 'Reintentar la copia si falla',
        retryIfFailedShort: 'Reintentar si falla',
        retryIfFailedTooltip:
          'Reintenta automáticamente las copias de seguridad fallidas. Una copia puede fallar por problemas de red, del almacenamiento o porque la base de datos no esté disponible temporalmente.',
        maxFailedTries: 'Número máximo de intentos',
        maxFailedTriesTooltip:
          'Número máximo de reintentos de una copia fallida. Recibirá una notificación cuando fallen todos los intentos.',
      },
    },
    physical: {
      types: {
        full: 'Completa',
        incremental: 'Incremental',
        wal: 'WAL',
      },
      retentions: {
        chains: 'Cadenas',
        fullBackups: 'Copias completas',
        chainsAndFullBackups: 'Cadenas y copias completas',
      },
      fullBackupsPolicies: {
        lastN: 'Cantidad',
        gfs: 'GFS',
      },
      notificationTypes: {
        backupSuccess: 'Copia de seguridad correcta',
        backupFailed: 'Copia de seguridad fallida',
        chainBroken: 'Cadena rota',
        walGap: 'Hueco en el WAL',
      },
      list: {
        totalUsage: 'Uso total: {{size}}',
        backUpNow: 'Hacer copia ahora',
        fullBackup: 'Copia completa',
        incrementalBackup: 'Copia incremental',
        restore: 'Restaurar',
        restoreFromThisBackup: 'Restaurar desde esta copia',
        pointInTimeRestoreTitle: 'Restauración a un punto en el tiempo',
        deleteFullConfirmation:
          'Al eliminar esta copia completa se elimina toda su cadena (todas las incrementales que dependen de ella). Esta acción no se puede deshacer. ¿Continuar?',
        deleteIncrementalConfirmation:
          'Al eliminar esta copia incremental se eliminan todas las incrementales posteriores que dependen de ella. Esta acción no se puede deshacer. ¿Continuar?',
      },
      config: {
        fullBackupCadence: 'Frecuencia de las copias completas',
        incrementalBackupCadence: 'Frecuencia de las copias incrementales',
        incrementalMoreFrequent:
          'Las copias incrementales deben ejecutarse con más frecuencia que las completas.',
        encryptionTooltip:
          'Si las copias están cifradas, sus archivos en el almacenamiento no se pueden usar directamente. Puede restaurarlas a través de Databasus o descargarlas sin cifrar.',
        retention: 'Retención',
        fullBackupsOnlyRetention:
          'Esta base de datos solo guarda copias completas, así que la retención se aplica a ellas.',
        retentionOptions: {
          chains: 'Conservar las últimas N cadenas',
          chainsAndFullBackups: 'Conservar las últimas N cadenas y copias completas',
        },
        retentionHelp: {
          chain:
            'Una cadena es una copia completa más las incrementales (y el WAL transmitido) que dependen de ella. Para restaurar hacen falta la copia completa y todas las incrementales de su cadena.',
          chains:
            'Conserva solo las últimas N cadenas. Cuando una cadena queda fuera de ese límite, se eliminan su copia completa y todas sus incrementales: solo se puede restaurar dentro de las cadenas conservadas.',
          chainsAndFullBackups:
            'Conserva las últimas N cadenas para la recuperación a un punto en el tiempo reciente y, además, guarda copias completas independientes más antiguas según la política de abajo (GFS o últimas N) como puntos de restauración a largo plazo una vez eliminadas sus incrementales.',
        },
        chainsCount: 'Cadenas',
        chainsCountTooltip:
          'Número de cadenas de copias más recientes que se conservan. Una cadena es una copia completa más sus incrementales.',
        fullBackups: 'Copias completas',
        fullBackupsCount: 'Copias completas más recientes',
        gfsFields: {
          hourly: 'Horarias',
          daily: 'Diarias',
          weekly: 'Semanales',
          monthly: 'Mensuales',
          yearly: 'Anuales',
        },
        chainsKept: 'Cadenas conservadas',
        fullBackupsKept: 'Copias completas conservadas',
        fullBackupsCountValue: 'Más recientes: {{count}}',
        gfsTooltip:
          'Rotación Grandfather-Father-Son: conserva las últimas N copias completas horarias, diarias, semanales, mensuales y anuales.',
      },
      restore: {
        errorHints: {
          downloadInProgress:
            'Ya hay una descarga de restauración en curso para esta base de datos. Espere a que termine y vuelva a intentarlo.',
          targetTimeUnreachable:
            'No se puede llegar al momento solicitado: hay un hueco en el WAL o el momento está fuera del rango disponible. Elija otro momento o restaure el último punto disponible.',
        },
        backupIntro:
          'Comando de restauración de esta copia. Una copia completa solo necesita su propio archivo; una incremental necesita además su copia completa y todas las incrementales anteriores.',
        pointInTimeIntro:
          'Restauración a un punto en el tiempo. Elija el momento objetivo o déjelo vacío para restaurar el último punto disponible.',
        targetTime: 'Momento objetivo',
        latestAvailable: 'Último punto disponible',
        preparing: 'Preparando el comando de restauración...',
        linkExpiry:
          'El enlace de descarga es de un solo uso y caduca en 15 minutos. Si un comando falla con un error 401, vuelva a abrir este diálogo para generar uno nuevo.',
        copied: 'Copiado',
        methods: {
          script: 'Con script',
          manual: 'Manual',
        },
        environments: {
          host: 'PostgreSQL en el host',
        },
        fields: {
          restoreDirectory: 'Directorio de restauración',
          pgBinPath: 'Ruta de bin de PostgreSQL',
          pgBinPathPlaceholder: '{{path}} (opcional)',
          pgImage: 'Imagen de PostgreSQL',
        },
        scriptTitles: {
          host: 'Ejecútelo en el host de restauración',
          docker: 'Ejecútelo donde haya Docker disponible',
        },
        manualSteps: {
          download: 'Descargue el paquete',
          extract: 'Extráigalo',
          verify: 'Verifique la transferencia',
          reconstruct: 'Reconstruya el directorio de datos',
          checkConfig: 'Revise los archivos de configuración del clúster',
          wireUpRecovery: 'Descomprima el WAL y configure la recuperación',
          recoveryParameters: 'Si la recuperación se detiene por la configuración de parámetros',
        },
        beforeYouRun: {
          title: 'Antes de ejecutar',
          host: {
            clientTools:
              'Instale las herramientas cliente de PostgreSQL {{version}}; si no están en el PATH, indique arriba la ruta de bin.',
            zstd: 'Se necesita zstd para descomprimir la copia.',
            zstdWithWal: 'Se necesita zstd para descomprimir la copia y su WAL.',
            emptyDirectory:
              'El directorio de restauración debe estar vacío y no puede ser el de un clúster en uso.',
          },
          docker: {
            image:
              'Use una imagen postgres:{{version}}: la versión mayor debe coincidir con la de origen.',
            hostTools:
              'El host necesita zstd, tar y curl: la descarga y toda la descompresión (la copia base) se hacen en el host, y en el contenedor solo se ejecuta pg_combinebackup, así que la imagen de postgres no necesita herramientas adicionales.',
            hostToolsWithWal:
              'El host necesita zstd, tar y curl: la descarga y toda la descompresión (la copia base y el WAL) se hacen en el host, y en el contenedor solo se ejecuta pg_combinebackup, así que la imagen de postgres no necesita herramientas adicionales.',
          },
        },
        afterItFinishes: {
          title: 'Cuando termine',
          host: {
            clusterLocation: 'El clúster está en <dataDir/>.',
            changeOwner: 'Como root, cambie el propietario: <command/>.',
            start: 'Inícielo como postgres: <command/>, o apunte data_directory a él.',
            walReplay:
              'PostgreSQL reproduce el WAL y se promociona: consulte el log para ver cuándo termina la recuperación.',
          },
          docker: {
            clusterLocation: '<dataDir/> en el host es el clúster restaurado.',
            volumeMount:
              'PostgreSQL {{version}} guarda su directorio de datos en <dataDir/>, dentro del volumen <volumeDir/>. Monte el directorio de salida como raíz de ese volumen; es lo que espera un docker-compose con <code>./pgdata:/var/lib/postgresql</code>:',
            dataDirectoryMount:
              'Monte el clúster con un bind mount en el directorio de datos de la imagen <dataDir/>:',
            ownership:
              'El propietario debe coincidir con el uid de postgres dentro de la imagen (999 en la oficial).',
          },
          missingConfigFiles:
            'Una copia base solo copia el directorio de datos. Si su PostgreSQL de origen guarda la configuración fuera de él (la estructura de Debian/Ubuntu, <configDir/>), <code>postgresql.conf</code>, <code>pg_hba.conf</code> y <code>pg_ident.conf</code> no están en la copia y el servidor no arrancará. Consulte sus rutas reales en el origen con <code>psql -Atc "SHOW config_file"</code>, copie los tres archivos, comente <code>data_directory</code>, <code>hba_file</code>, <code>ident_file</code>, <code>external_pid_file</code> y las líneas <code>ssl</code> / <code>ssl_*</code>, cree <code>conf.d</code> y, para un contenedor, establezca <code>listen_addresses = \'*\'</code>. Si esos archivos están y el error persiste, lo que está mal es el directorio montado, no la configuración.',
        },
      },
    },
  },
  storages: {
    types: {
      local: 'Almacenamiento local',
      s3: 'S3',
      googleDrive: 'Google Drive',
      nas: 'NAS',
      azureBlob: 'Azure Blob Storage',
      ftp: 'FTP',
      sftp: 'SFTP',
      rclone: 'Rclone',
    },
    s3StorageClasses: {
      default: 'Predeterminada (Standard)',
      standard: 'Standard',
      standardIa: 'Standard - Infrequent Access',
      onezoneIa: 'One Zone - Infrequent Access',
      intelligentTiering: 'Intelligent Tiering',
      reducedRedundancy: 'Reduced Redundancy',
      glacierIr: 'Glacier Instant Retrieval',
    },
    azureBlobAuthMethods: {
      accountKey: 'Clave de la cuenta',
      connectionString: 'Cadena de conexión',
    },
    sftpAuthMethods: {
      password: 'Contraseña',
      privateKey: 'Clave privada',
    },
    list: {
      addStorage: 'Añadir almacenamiento',
      description:
        'Un almacenamiento es el lugar donde se guardan las copias de seguridad (disco local, S3, etc.)',
      backToList: '← Volver a los almacenamientos',
    },
    card: {
      type: {
        local: 'Tipo: almacenamiento local',
        s3: 'Tipo: S3',
        googleDrive: 'Tipo: Google Drive',
        nas: 'Tipo: NAS',
        azureBlob: 'Tipo: Azure Blob Storage',
        ftp: 'Tipo: FTP',
        sftp: 'Tipo: SFTP',
        rclone: 'Tipo: Rclone',
      },
      hasSaveError: 'Error al guardar',
    },
    config: {
      namePlaceholder: 'Introduzca un nombre...',
      settingsTitle: 'Ajustes del almacenamiento',
      lastSaveError: {
        title: 'Error al guardar',
        errorLabel: 'Error:',
        clearHint: 'Para quitar este error, haga una de estas dos cosas:',
        clearByTestingConnection:
          '- pruebe la conexión con el botón de abajo (aunque ya haya cambiado los ajustes);',
        clearByNextSave: '- espere a que la siguiente copia de seguridad se guarde sin errores;',
      },
      removeConfirmation:
        '¿Seguro que desea eliminar este almacenamiento? Esta acción no se puede deshacer. Las copias de seguridad guardadas en este almacenamiento también se eliminarán.',
      removeBlockedByDatabases:
        'Algunas bases de datos usan este almacenamiento. Quítelo primero de esas bases de datos.',
    },
    actions: {
      testConnection: 'Probar conexión',
    },
    connectionTestSucceeded: {
      title: 'La conexión funciona',
      description: 'Conexión con el almacenamiento comprobada',
    },
    transfer: {
      title: 'Transferir el almacenamiento a otro espacio de trabajo',
      blockedByDatabases:
        'Algunas bases de datos usan este almacenamiento. Transfiera o elimine primero esas bases de datos.',
      description: 'Seleccione el espacio de trabajo al que transferir este almacenamiento.',
      targetWorkspace: 'Espacio de trabajo de destino',
      selectWorkspace: 'Seleccione un espacio de trabajo',
      submit: 'Transferir',
    },
    fields: {
      path: 'Ruta',
      remotePath: 'Ruta remota',
      share: 'Recurso compartido',
      domain: 'Dominio',
      useSsl: 'Usar SSL',
      useSslTls: 'Usar SSL/TLS',
      skipTls: 'No verificar TLS',
      skipTlsVerify: 'Omitir verificación TLS',
      skipHostKey: 'No verificar clave de host',
      authMethod: 'Método de autenticación',
      privateKey: 'Clave privada',
      credentials: 'Credenciales',
      s3Bucket: 'Bucket de S3',
      region: 'Región',
      accessKey: 'Clave de acceso',
      secretKey: 'Clave secreta',
      endpoint: 'Endpoint',
      folderPrefix: 'Prefijo de carpeta',
      prefix: 'Prefijo',
      blobPrefix: 'Prefijo de blobs',
      virtualHost: 'Host virtual',
      storageClass: 'Clase de almacenamiento',
      connection: 'Conexión',
      connectionString: 'Cadena de conexión',
      accountName: 'Nombre de la cuenta',
      accountKey: 'Clave de la cuenta',
      containerName: 'Nombre del contenedor',
      clientId: 'ID de cliente',
      clientSecret: 'Secreto de cliente',
      userToken: 'Token de usuario',
      config: 'Configuración',
    },
    edit: {
      namePlaceholder: 'Mi almacenamiento',
      advancedSettings: 'Ajustes avanzados',
      optionalPlaceholder: '{{example}} (opcional)',
      usernamePlaceholder: 'usuario',
      passwordPlaceholder: 'contraseña',
      remotePathTooltip:
        'Ruta del directorio remoto en el que se guardan las copias de seguridad (opcional)',
      local: {
        diskSpaceWarning:
          'Cuidado: con el almacenamiento local puede quedarse sin espacio en disco. Se recomienda usar S3 u otros almacenamientos sin límite de espacio',
      },
      s3: {
        cloudflareR2Guide: '¿Cómo usarlo con Cloudflare R2?',
        endpointTooltip:
          'URL de un endpoint compatible con S3 (opcional; déjelo vacío para AWS S3)',
        prefixTooltip:
          "Prefijo opcional para las claves de todos los objetos (p. ej., 'backups/' o 'my_team/'). Puede no funcionar con algunos almacenamientos compatibles con S3. No se puede cambiar después de crearlo (si no, se perderán las copias de seguridad).",
        useVirtualHostedStyle: 'Usar URL virtual-hosted',
        virtualHostTooltip:
          'Usa URL de estilo virtual-hosted (bucket.s3.region.amazonaws.com) en lugar de path-style (s3.region.amazonaws.com/bucket). Puede ser necesario si aparecen errores de COS.',
        skipTls: 'No verificar TLS',
        skipTlsVerifyTooltip:
          'No verifica el certificado TLS. Active esta opción si su almacenamiento compatible con S3 usa un certificado autofirmado. Advertencia: esto reduce la seguridad.',
        storageClassTooltip:
          'Clase de almacenamiento de S3 para los objetos subidos. Déjela en el valor predeterminado para Standard. Algunos proveedores ofrecen clases más baratas, como One Zone IA. No use Glacier/Deep Archive: los archivos deben estar accesibles de inmediato para las restauraciones.',
      },
      googleDrive: {
        guideLink: '¿Cómo conectar Google Drive?',
        authorize: 'Autorizar',
      },
      nas: {
        shareWithSlashWarning:
          'El recurso compartido debe ser un único nombre. Use el campo Ruta para los subdirectorios (p. ej., Recurso compartido: Databasus, Ruta: DB1)',
        useSslTooltip: 'Activa el cifrado SSL/TLS para una conexión segura',
        domainTooltip:
          'Nombre de dominio de Windows (opcional; déjelo vacío si no usa autenticación de dominio)',
        pathPlaceholder: '{{example}} (opcional, sin barra inicial)',
        pathTooltip: 'Ruta del subdirectorio dentro del recurso compartido (opcional)',
      },
      azureBlob: {
        connectionStringTooltip: 'Cadena de conexión de Azure Storage desde Azure Portal',
        endpointTooltip:
          'URL de un endpoint personalizado (opcional; déjelo vacío para Azure estándar)',
        prefixTooltip:
          "Prefijo opcional para los nombres de todos los blobs (p. ej., 'backups/' o 'my_team/')",
      },
      ftp: {
        enableFtps: 'Activar FTPS',
        useSslTooltip:
          'Usa cifrado TLS explícito (FTPS) para transferir los archivos de forma segura',
        skipCertificateVerification: 'No verificar el certificado',
        skipTlsVerifyTooltip:
          'No verifica el certificado TLS. Active esta opción si su servidor FTP usa un certificado autofirmado. Advertencia: esto reduce la seguridad.',
      },
      sftp: {
        privateKeyTooltip:
          'Pegue su clave privada SSH (formato PEM). Se admiten claves RSA, DSA, ECDSA y Ed25519.',
        skipHostKeyVerification: 'No verificar la clave de host',
        skipHostKeyTooltip:
          'No verifica la clave de host SSH. Active esta opción si confía en el servidor. Advertencia: esto reduce la seguridad.',
      },
      rclone: {
        configTooltip:
          "Pegue aquí el contenido de rclone.conf. Puede obtenerlo ejecutando 'rclone config file' y copiando su contenido. Esta configuración admite más de 70 proveedores de almacenamiento en la nube.",
        configHiddenNote:
          '*el contenido se oculta para no exponer datos sensibles. Si quiere actualizar la configuración, pegue aquí una nueva',
        docsLink: 'Documentación de Rclone',
        remotePathTooltip:
          "Prefijo de ruta opcional en el remote de rclone en el que se guardarán las copias de seguridad (p. ej., '/backups' o 'my-folder/backups')",
      },
    },
  },
};

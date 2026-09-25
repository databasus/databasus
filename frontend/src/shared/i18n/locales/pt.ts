import type { en } from './en';

export const pt: typeof en = {
  language: {
    nativeName: 'Português',
  },
  common: {
    actions: {
      add: 'Adicionar',
      back: 'Voltar',
      cancel: 'Cancelar',
      close: 'Fechar',
      confirm: 'Confirmar',
      continue: 'Continuar',
      copy: 'Copiar',
      create: 'Criar',
      delete: 'Excluir',
      download: 'Baixar',
      edit: 'Editar',
      ok: 'OK',
      refresh: 'Atualizar',
      remove: 'Remover',
      retry: 'Tentar novamente',
      save: 'Salvar',
    },
    answers: {
      no: 'Não',
      yes: 'Sim',
    },
    fields: {
      host: 'Host',
      name: 'Nome',
      password: 'Senha',
      port: 'Porta',
      status: 'Status',
      type: 'Tipo',
      username: 'Nome de usuário',
    },
    states: {
      disabled: 'Desativado',
      enabled: 'Ativado',
      loading: 'Carregando...',
    },
    messages: {
      copiedToClipboard: 'Copiado para a área de transferência',
    },
    // Joins translated words into one list (Chinese uses 、)
    listSeparator: ', ',
    confirmation: {
      title: 'Confirmação',
    },
  },
  app: {
    documentTitle: 'Databasus - backup de bancos de dados (PostgreSQL, MySQL, MongoDB)',
    languageControl: {
      title: 'Idioma: {{language}}',
    },
    theme: {
      title: 'Tema: {{theme}}',
      light: 'Claro',
      dark: 'Escuro',
      system: 'Sistema',
    },
    navigation: {
      dashboard: 'Painel',
      databases: 'Bancos de dados',
      storages: 'Armazenamentos',
      notifiers: 'Canais de notificação',
      workspaceSettings: 'Configurações',
      profile: 'Perfil',
      databasusSettings: 'Configurações do Databasus',
      users: 'Usuários',
      docs: 'Documentação',
      documentation: 'Documentação',
      community: 'Comunidade',
      newVersionAvailable: 'Nova versão disponível',
    },
    diskUsage: {
      title: 'Uso do disco',
      hint: 'Para fazer backups localmente e restaurá-los, o disco precisa ter espaço suficiente. Para restaurar, é preciso ter livre o mesmo espaço que o backup ocupa.',
      compactSummary: '{{used}} de {{total}} GB<lineBreak/>em uso ({{percent}}%)',
      summary: '{{used}} de {{total}} GB em uso ({{percent}}%)',
    },
    workspaceSelection: {
      createShort: 'Criar',
      create: 'Criar espaço de trabalho',
      selected: 'Espaço de trabalho',
      placeholder: 'Selecione um espaço de trabalho',
      search: 'Buscar espaços de trabalho...',
      notFound: 'Nenhum espaço de trabalho encontrado',
      createNew: '+ Novo espaço de trabalho',
    },
    sponsorship: {
      link: 'Apoiar',
      hint: 'O Databasus salvou seus dados? Apoie o projeto para que ele continue gratuito, mantido e independente',
    },
    starButton: {
      label: 'Dar estrela no GitHub',
      ariaLabel: 'Dar estrela a databasus/databasus no GitHub',
    },
    clipboardPaste: {
      title: 'Colar da área de transferência',
      submit: 'Confirmar',
      description:
        'Não foi possível acessar a área de transferência automaticamente. Cole o conteúdo no campo abaixo.',
      placeholder: 'Cole aqui a string de conexão...',
    },
    auth: {
      accountsCheckFailed:
        'Não foi possível verificar se a instância possui alguma conta: {{error}}',
    },
    oauthCallback: {
      failedTitle: 'Falha na autenticação',
      codeNotFound: 'Código de autorização não encontrado',
      stateMissing: 'Parâmetro state do OAuth ausente',
      invalidProvider: 'Provedor OAuth inválido',
      returnToSignIn: 'Voltar para o login',
      completing: 'Concluindo a autenticação...',
    },
    oauthStorage: {
      configNotFound: 'Configuração do armazenamento Google Drive não encontrada',
      exchangeFailed: 'Não foi possível trocar o código OAuth',
      unsupportedType: 'Este tipo de armazenamento não aceita OAuth',
      invalidState: 'O parâmetro state do OAuth é inválido',
      paramNotFound:
        'Parâmetro OAuth não encontrado. Verifique se a URL de redirecionamento está configurada corretamente.',
      addStorageTitle: 'Adicionar armazenamento',
      storageDescription:
        'Armazenamento é o lugar onde os backups ficam guardados (disco local, S3, etc.)',
    },
  },
  errors: {
    unknown: 'Algo deu errado. Tente novamente.',
    networkUnreachable:
      'Não foi possível acessar o servidor do Databasus. Verifique sua conexão e tente novamente.',
    requestFailed:
      'O servidor não conseguiu concluir a solicitação (HTTP {{status}}). Tente novamente mais tarde.',
    signInCodeIncorrect: 'Código incorreto.',
    pendingSignInNotUsable: 'Este login não pode mais ser concluído. Entre novamente.',
    signInCodeNotSent:
      'A instância não conseguiu enviar o código de acesso. Tente mais tarde ou fale com um administrador.',
    tooManySignInCodes:
      'Foram solicitados códigos de acesso demais para esta conta na última hora. Tente mais tarde.',
    signInCodeResentTooSoon:
      'Um código foi enviado há menos de um minuto. Espere um minuto antes de pedir outro.',
    rateLimitExceeded: 'Tentativas demais. Espere um pouco e tente novamente.',
    emailNotConfigured: 'A instância não tem um servidor de e-mail configurado.',
    adminEmailMissing:
      'Sua conta não tem um endereço de e-mail válido para receber o e-mail de teste.',
  },
  status: {
    logicalBackup: {
      inProgress: 'Em andamento',
      completed: 'Bem-sucedido',
      failed: 'Falhou',
      deleted: 'Excluído',
      canceled: 'Cancelado',
    },
    physicalBackup: {
      inProgress: 'Em andamento',
      completed: 'Bem-sucedido',
      error: 'Erro',
      chainBroken: 'Cadeia quebrada',
      canceled: 'Cancelado',
    },
    restoreVerification: {
      verifiedSuccessful: 'Verificado',
      verificationFailed: 'Falha na verificação',
    },
    restore: {
      inProgress: 'Em andamento',
      completed: 'Bem-sucedida',
      failed: 'Falhou',
      canceled: 'Cancelada',
    },
    verification: {
      pending: 'Pendente',
      running: 'Em execução',
      completed: 'Bem-sucedida',
      failed: 'Falhou',
      canceled: 'Cancelada',
    },
    health: {
      available: 'Disponível',
      unavailable: 'Indisponível',
    },
    agent: {
      neverSeen: 'Nunca conectado',
      online: 'Online',
      stale: 'Sem sinal recente',
      offline: 'Offline',
    },
  },
  intervals: {
    types: {
      hourly: 'A cada hora',
      daily: 'Diário',
      weekly: 'Semanal',
      monthly: 'Mensal',
      cron: 'Cron',
    },
    weekdays: {
      mon: 'Seg',
      tue: 'Ter',
      wed: 'Qua',
      thu: 'Qui',
      fri: 'Sex',
      sat: 'Sáb',
      sun: 'Dom',
    },
    periods: {
      day: '1 dia',
      week: '1 semana',
      month: '1 mês',
      threeMonths: '3 meses',
      sixMonths: '6 meses',
      year: '1 ano',
      twoYears: '2 anos',
      threeYears: '3 anos',
      fourYears: '4 anos',
      fiveYears: '5 anos',
      forever: 'Para sempre',
    },
  },
  auditLogs: {
    title: 'Logs de auditoria',
    columns: {
      user: 'Usuário',
      message: 'Mensagem',
      workspace: 'Espaço de trabalho',
      created: 'Data',
    },
    systemUser: 'Sistema',
    loadedOfTotal: 'Carregados: {{loaded}} de {{total}}',
    empty: 'Nenhum log de auditoria encontrado.',
    loadingMore: 'Carregando mais logs...',
    allLoaded: 'Todos os logs foram carregados (total: {{count}})',
  },
  settings: {
    title: 'Configurações do Databasus',
    externalRegistrations: {
      title: 'Permitir registros externos',
      description:
        'Quando ativado, novos usuários podem criar uma conta no Databasus por conta própria. Quando desativado, só é possível se registrar por convite',
    },
    memberInvitations: {
      title: 'Permitir convites de membros',
      description:
        'Quando ativado, os membros podem convidar novos usuários para o Databasus. Quando desativado, só os administradores podem convidar.',
    },
    memberWorkspaceCreation: {
      title: 'Membros podem criar espaços de trabalho',
      description:
        'Quando ativado, os membros (usuários que não são administradores) podem criar novos espaços de trabalho. Quando desativado, só os administradores podem criá-los.',
    },
    twoFactorAuth: {
      title: 'Ativar o login em duas etapas',
      description:
        'Quando ativado, além da senha é preciso informar um código de seis dígitos enviado para o e-mail da conta. Não vale para o login pelo Google ou pelo GitHub.',
      mailServerRequired:
        'É preciso ter um servidor de e-mail. <docsLink>Configure o SMTP</docsLink> primeiro.',
    },
    saving: 'Salvando...',
    saveChanges: 'Salvar alterações',
    reset: 'Redefinir',
    updated: 'Configurações salvas',
    readMore: 'Saiba mais sobre as configurações <docsLink>na documentação</docsLink>',
    mailServer: {
      title: 'Servidor de e-mail',
      configured: 'O servidor de e-mail <status>está configurado</status>.',
      notConfigured: 'O servidor de e-mail <status>não está configurado</status>.',
      description:
        'Ele envia os convites, os códigos de redefinição de senha e os códigos de acesso. Os canais de notificação por e-mail têm configurações SMTP próprias.',
      docs: 'Veja como conectar um servidor de e-mail na <docsLink>documentação de SMTP</docsLink>.',
      sendTestEmail: 'Enviar e-mail de teste',
      testEmailSent: 'E-mail de teste enviado para {{email}}',
    },
    healthcheck: {
      title: 'Verificação de disponibilidade',
      openInNewTab: 'Abrir em uma nova aba',
      copied: 'Endpoint de verificação de disponibilidade copiado',
      description: 'Use este endpoint para monitorar a disponibilidade do seu Databasus',
    },
  },
  dashboard: {
    tiles: {
      databases: 'Bancos de dados',
      backups: 'Backups',
      backupsSize: 'Tamanho total dos backups',
      backupsSizeHint:
        'Conta apenas os backups bem-sucedidos. Nos bancos de dados físicos, os segmentos WAL também entram no total.',
      installation: 'Todos os espaços de trabalho',
      installationHint:
        'Totais de todos os espaços de trabalho desta instância do Databasus. Só os administradores veem esses números.',
      installationDetails: 'Bancos de dados: {{databasesCount}}, backups: {{backupsCount}}',
    },
    list: {
      title: 'Bancos de dados',
      empty: 'Este espaço de trabalho ainda não tem bancos de dados',
      columns: {
        database: 'Banco de dados',
        healthcheck: 'Disponibilidade',
        backups: 'Backups',
        meanSize: 'Tamanho médio',
        totalSize: 'Tamanho total',
        lastBackup: 'Último backup',
        storage: 'Armazenamento',
      },
      successfulBackups: 'Bem-sucedidos: {{successfulBackupsCount}}',
      failedBackups: 'Com falha: {{failedBackupsCount}}',
      noBackups: 'Ainda não há backups',
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
      disable: 'Desativado',
      require: 'Obrigatório',
      verifyCa: 'Verificação da CA',
      verifyFull: 'Verificação completa',
    },
    physicalBackupTypes: {
      full: 'Somente backups completos',
      fullIncremental: 'Completo + incremental',
      fullIncrementalWalStream: 'Completo + incremental + streaming de WAL',
    },
    list: {
      addDatabase: 'Adicionar banco de dados',
      searchPlaceholder: 'Buscar banco de dados',
      noSearchResults: 'Nenhum banco de dados encontrado para "{{query}}"',
      description: 'Banco de dados é a origem dos backups',
      backToList: '← Voltar para os bancos de dados',
    },
    card: {
      storage: 'Armazenamento: <storageName/>',
      lastBackup: 'Último backup {{relativeTime}}',
      hasBackupError: 'Com erro de backup',
    },
    tabs: {
      config: 'Configuração',
      backups: 'Backups',
      verifications: 'Verificações',
    },
    create: {
      title: 'Adicionar banco de dados para backup',
      namePlaceholder: 'Meu banco favorito',
      creating: 'Criando o banco de dados...',
      complete: 'Concluir',
      backupType: {
        title: 'Escolha o tipo de backup',
        logical: {
          title: 'Lógico',
          description:
            'Recomendado para bancos de dados com menos de 50 GB. Mais simples de configurar.',
        },
        physical: {
          title: 'Físico',
          description:
            'Para bancos de dados com mais de 50 GB. Permite a recuperação point-in-time e melhora o RPO/RTO, mas exige configuração adicional.',
        },
      },
    },
    config: {
      namePlaceholder: 'Digite o nome...',
      removing: 'Removendo o banco de dados...',
      lastBackupError: {
        title: 'Erro no último backup',
        errorLabel: 'Erro:',
        clearHint: 'Para limpar este erro, faça uma das opções:',
        clearByTestingConnection:
          '- teste a conexão pelo botão abaixo (mesmo que você já tenha alterado as configurações);',
        clearByNextBackup: '- aguarde o próximo backup terminar sem erros;',
      },
      sections: {
        database: 'Configurações do banco de dados',
        backupConfig: 'Configuração de backup',
        healthcheck: 'Verificação de disponibilidade',
        restoreVerification: 'Verificação de restauração',
        notifiers: 'Canais de notificação',
      },
      copyConfirmation:
        'Copiar este banco de dados? Será criado um novo banco de dados com as mesmas configurações.',
      removeConfirmation: 'Remover este banco de dados? Esta ação não pode ser desfeita.',
      copied: {
        title: 'Banco de dados copiado!',
        description: 'O banco de dados "{{name}}" foi criado',
      },
      connectionTestSucceeded: {
        title: 'Conexão bem-sucedida!',
        description: 'A conexão com o banco de dados foi testada com sucesso',
      },
    },
    actions: {
      testConnection: 'Testar conexão',
      replace: 'Substituir',
    },
    fields: {
      databaseType: 'Tipo de banco de dados',
      version: 'Versão do {{engine}}',
      databaseName: 'Nome do BD',
      sslMode: 'Modo SSL',
      useHttps: 'Usar HTTPS',
      cpuCount: 'Número de CPUs',
      backupType: 'Tipo de backup',
      clientCertificate: 'Certificado do cliente',
      clientKey: 'Chave do cliente',
      serverCaCertificate: 'Certificado CA do servidor',
      includeSchemas: 'Incluir schemas',
      excludeTables: 'Ignorar tabelas',
      skipUserMappings: 'Pular mapeamentos de usuário',
      excludeExtensions: 'Ignorar extensões',
      restoreOwnership: 'Restaurar proprietários',
      restorePrivileges: 'Restaurar privilégios',
      excludeEvents: 'Ignorar eventos',
      galeraReplication: 'Replicação Galera',
      useSrvConnection: 'Usar conexão SRV',
      directConnection: 'Conexão direta',
      authDatabase: 'Banco de autenticação',
      excludeCollections: 'Ignorar coleções',
    },
    edit: {
      parseFromClipboard: 'Preencher da área de transferência',
      clipboardEmpty: 'A área de transferência está vazia',
      clipboardReadFailed:
        'Não foi possível ler a área de transferência. Verifique as permissões do navegador.',
      connectionStringParsed: 'Campos preenchidos a partir da string de conexão',
      connectionStringParsedWithoutPassword:
        'Campos preenchidos a partir da string de conexão. Digite a senha manualmente.',
      connectionTestPassed: {
        title: 'Conexão testada',
        description: 'Você já pode seguir para a próxima etapa',
      },
      ipWhitelistHint:
        'Se o seu banco de dados aceita conexões só de IPs liberados, adicione à lista o IP do servidor do Databasus.',
      localhostHint:
        '<docsLink>Leia este guia</docsLink> para saber como fazer backup de um banco de dados local',
      advancedSettings: 'Configurações avançadas',
      placeholders: {
        host: 'Digite o host do {{engine}}',
        port: 'Digite a porta do {{engine}}',
        username: 'Digite o usuário do {{engine}}',
        password: 'Digite a senha do {{engine}}',
        databaseName: 'Digite o nome do banco de dados {{engine}}',
      },
      cpuCountTooltip:
        'Quantos núcleos de CPU usar no backup e na restauração. Valores maiores podem acelerar essas operações, mas consomem mais recursos.',
      excludeTablesPlaceholder: 'Nenhuma tabela ignorada',
      excludeTablesTooltip:
        'Nomes das tabelas que ficam fora do backup. Você pode colar uma lista separada por vírgulas ou quebras de linha.',
      serverCaCertificateTooltip:
        'Opcional. Se informado, o certificado do servidor é verificado com esta CA (verify-ca / verify-full).',
      copyFailed: 'Não foi possível copiar',
      postgresql: {
        supabaseHint:
          '<docsLink>Leia este guia</docsLink> para saber como fazer backup de um banco de dados Supabase',
        includeSchemasPlaceholder: 'Todos os schemas (padrão)',
        excludeTablesTooltip:
          "Tabelas que ficam fora do backup. Use 'tablename' ou 'schema.tablename'. Padrões glob são aceitos (por exemplo, 'logs_*'). Você pode colar uma lista separada por vírgulas ou quebras de linha.",
        skipUserMappingsTooltip:
          'Não restaura os mapeamentos de usuário (user mappings, instruções CREATE USER MAPPING). Ative se o papel usado no backup não consegue ler as credenciais dos mapeamentos: sem isso, eles vão para o dump sem opções e quebram a restauração de FDWs como o oracle_fdw.',
        excludeExtensionsTooltip:
          'Não restaura as definições de extensões (instruções CREATE EXTENSION). Ative se você está restaurando em um serviço PostgreSQL gerenciado, onde as extensões são administradas pelo provedor.',
        restoreOwnershipTooltip:
          'Aplica as instruções ALTER OWNER do dump para que os objetos restaurados mantenham o proprietário original. O usuário da conexão precisa poder atribuir esses papéis, o que normalmente exige um superusuário.',
        restorePrivilegesTooltip:
          'Aplica as instruções GRANT e REVOKE do dump para que os objetos restaurados mantenham as ACLs originais. O usuário da conexão precisa poder conceder permissões aos papéis citados, o que normalmente exige um superusuário.',
        backupTypeTooltips: {
          full: 'Backups completos periódicos. Cada backup é independente dos outros.',
          fullIncremental:
            'Backups completos e incrementais, que guardam só as alterações desde o backup anterior. Ocupam menos espaço e são mais rápidos.',
          fullIncrementalWalStream:
            'Acrescenta streaming contínuo de WAL aos backups completos e incrementais. Só esta opção permite a recuperação point-in-time (PITR), mas ocupa mais espaço e a restauração é mais lenta.',
        },
      },
      mariadb: {
        skipEvents: 'Pular eventos',
        skipEventsTooltip:
          'Não inclui os eventos do banco de dados no backup. Ative se o agendador de eventos estiver desativado no seu servidor MariaDB.',
        skipGaleraDisable: 'Não desativar na restauração',
        skipGaleraDisableTooltip:
          'Por padrão, o Databasus executa SET SESSION wsrep_on=OFF durante a restauração para evitar erros de tamanho de writeset do Galera. Isso exige o privilégio SUPER. Ative esta opção para pular esse passo se o seu provedor gerenciado não concede SUPER. Nesse caso, restaurações grandes podem esbarrar nos limites de writeset do Galera.',
      },
      mongodb: {
        srvTooltip:
          'Ative para conexões SRV do MongoDB Atlas (mongodb+srv://). Conexões SRV não precisam de porta.',
        srvOverTunnelTooltip:
          'Indisponível com túnel SSH: o SRV resolve a própria lista de hosts e ignora a porta encaminhada.',
        directConnectionTooltip:
          'Conecta diretamente a um único servidor, sem descobrir o replica set. Útil quando o servidor está atrás de um balanceador de carga, proxy ou túnel.',
        directConnectionOverTunnelTooltip:
          'Sempre ativada enquanto o túnel SSH estiver ligado: o túnel expõe um único endereço, então não é possível descobrir o replica set.',
        excludeCollectionsPlaceholder: 'Nenhuma coleção ignorada',
        excludeCollectionsTooltip:
          'Nomes das coleções que ficam fora do backup. Você pode colar uma lista separada por vírgulas ou quebras de linha.',
      },
    },
    show: {
      galeraDisableSkipped: 'Não desativar na restauração',
    },
    sshTunnel: {
      authTypes: {
        password: 'Senha',
        privateKey: 'Chave privada',
      },
      title: 'Túnel SSH',
      tooltip:
        'Para um banco de dados em uma rede fechada. O Databasus se conecta ao host SSH abaixo, e esse host acessa o banco de dados pelo host e pela porta informados acima.',
      host: 'Host SSH',
      hostTooltip:
        'Quem resolve o host do banco de dados informado acima é o host SSH, não o Databasus. Use 127.0.0.1 quando o banco de dados roda no próprio host SSH.',
      port: 'Porta SSH',
      username: 'Usuário SSH',
      usernamePlaceholder: 'Digite o usuário SSH',
      authType: 'Autenticação SSH',
      credentials: 'Credenciais SSH',
      password: 'Senha SSH',
      passwordPlaceholder: 'Digite a senha SSH',
      privateKey: 'Chave privada SSH',
      keyPassphrase: 'Senha da chave',
      keyPassphrasePlaceholder: 'Só para chave criptografada',
    },
    notifiers: {
      title: 'Canais de notificação',
      description:
        'Canal de notificação é o lugar para onde as notificações são enviadas (e-mail, Slack, Telegram, etc.)',
      multipleHint: 'Você pode selecionar vários canais: as notificações vão para todos eles.',
      selectPlaceholder: 'Selecione os canais de notificação',
      createNew: 'Criar canal de notificação',
      addTitle: 'Adicionar canal de notificação',
      notifyTo: 'Notificar em',
      empty: 'Nenhum canal de notificação configurado',
    },
    readOnlyUser: {
      dialogTitle: 'Criar usuário somente leitura',
      readOnly: {
        checking: 'Verificando o usuário somente leitura...',
        title: 'Criar um usuário somente leitura para o Databasus?',
        description:
          'Um usuário somente leitura é um usuário do {{databaseType}} com permissões limitadas: ele só consegue ler os dados do banco, não alterá-los. Isso é recomendado para backups porque:',
        noWriteCredentials:
          '<bold>Com um usuário somente leitura, nenhuma credencial com permissão de escrita fica guardada</bold>. Mesmo no pior cenário, uma invasão, ninguém conseguirá corromper seus dados.',
        create: 'Sim, criar usuário somente leitura',
        skipTitle: 'Pular a criação do usuário somente leitura?',
        skipQuestion: 'Tem certeza de que deseja pular a criação de um usuário somente leitura?',
      },
      replicationOnly: {
        checking: 'Verificando o usuário somente para replicação...',
        title: 'Criar um usuário somente para replicação para o Databasus?',
        description:
          'Um usuário somente para replicação é um usuário do {{databaseType}} com permissões limitadas: ele só consegue ler os dados do banco, não alterá-los. Isso é recomendado para backups porque:',
        noWriteCredentials:
          '<bold>Com um usuário somente para replicação, nenhuma credencial com permissão de escrita fica guardada</bold>. Mesmo no pior cenário, uma invasão, ninguém conseguirá corromper seus dados.',
        create: 'Sim, criar usuário somente para replicação',
        skipTitle: 'Pular a criação do usuário somente para replicação?',
        skipQuestion:
          'Tem certeza de que deseja pular a criação de um usuário somente para replicação?',
      },
      reasons: {
        preventsModifications: 'evita alterações acidentais nos dados durante o backup',
        leastPrivilege: 'segue o princípio do menor privilégio',
        bestPractice: 'é uma boa prática de segurança',
      },
      securityNote:
        'O Databasus segue padrões de segurança de nível corporativo (<docsLink>veja os detalhes aqui</docsLink>). Mesmo assim, não é possível se proteger de todos os riscos.',
      noWritePrivileges: 'O usuário atual <bold>não tem privilégios de escrita</bold>.',
      noWriteRoles: 'O usuário atual <bold>não tem papéis com permissão de escrita</bold>.',
      writePrivileges: 'O usuário atual tem os seguintes privilégios de escrita:',
      writeRoles: 'O usuário atual tem os seguintes papéis com permissão de escrita:',
      expand: '(expandir)',
      collapse: '(recolher)',
      skip: 'Pular',
      skipRisk:
        'Usar um usuário com permissões totais para backups não é recomendado e pode trazer riscos de segurança. O Databasus recomenda fortemente não pular esta etapa.',
      skipAdvice:
        'Proteção de 100% não existe. É melhor se precaver mesmo contra um risco de 0,01% de invasão total. Por isso, o caminho mais seguro é usar um usuário somente leitura.',
      acceptRisks: 'Sim, aceito os riscos',
      continueSecurely: 'Continuar pelo caminho seguro',
      forcedWalRotationUnavailable:
        'Esta origem não permite conceder ao novo usuário EXECUTE em pg_switch_wal(), e o streaming contínuo de WAL precisa disso para manter o ponto de recuperação próximo do momento atual. Backups completos e incrementais funcionam normalmente com estas credenciais.',
    },
    transfer: {
      title: 'Transferir banco de dados para outro espaço de trabalho',
      targetWorkspace: 'Espaço de trabalho de destino',
      selectWorkspace: 'Selecione o espaço de trabalho',
      storage: {
        title: 'Armazenamento',
        transferExisting: 'Transferir com o armazenamento atual',
        selectFromTarget: 'Selecionar um armazenamento do espaço de trabalho de destino',
        blocked:
          'Bancos de dados que usam este armazenamento: {{count}}. A transferência está bloqueada porque outros bancos de dados dependem dele.',
        canTransfer: 'O armazenamento pode ser transferido',
        selectPlaceholder: 'Selecione o armazenamento',
        createNew: 'Criar armazenamento',
        addTitle: 'Adicionar armazenamento',
        description:
          'Armazenamento é o lugar onde os backups ficam guardados (disco local, S3, Google Drive, etc.)',
      },
      notifiers: {
        title: 'Canais de notificação (opcional)',
        transferWithDatabase: 'Transferir os canais de notificação com o banco de dados',
        selectFromTarget: 'Selecionar canais de notificação do espaço de trabalho de destino',
        willTransfer: 'Serão transferidos:',
        willNotTransfer: 'NÃO serão transferidos (usados por outros bancos de dados):',
        usedByDatabases: '{{name}} (usado por bancos de dados: {{count}})',
        noneTransferred:
          'Nenhum canal de notificação será transferido. Depois da transferência, você pode selecionar canais do espaço de trabalho de destino.',
        selectPlaceholder: 'Selecione os canais de notificação (opcional)',
      },
      submit: 'Transferir',
      transferred: {
        title: 'Banco de dados transferido!',
        description: 'O banco de dados "{{name}}" foi transferido para o novo espaço de trabalho',
      },
    },
    connectionString: {
      errors: {
        empty: 'A string de conexão está vazia',
        unrecognizedFormat: 'Formato de string de conexão não reconhecido',
        hostMissing: 'Falta o host na string de conexão',
        usernameMissing: 'Falta o nome de usuário na string de conexão',
        passwordMissing: 'Falta a senha na string de conexão',
        databaseMissing: 'Falta o nome do banco de dados na string de conexão',
        parseFailed: 'Não foi possível interpretar a string de conexão',
        jdbc: {
          invalidFormat: 'Formato de string de conexão JDBC inválido. Esperado: {{expectedFormat}}',
          queryParametersMissing:
            'Faltam os parâmetros de consulta (user e password) na string de conexão JDBC',
          usernameMissing: 'Falta o nome de usuário (parâmetro user) na string de conexão JDBC',
          passwordMissing: 'Falta o parâmetro password na string de conexão JDBC',
          parseFailed: 'Não foi possível interpretar a string de conexão JDBC',
        },
        keyValue: {
          hostMissing: 'Falta o host na string de conexão. Use host=hostname',
          usernameMissing: 'Falta o nome de usuário na string de conexão. Use user=username',
          passwordMissing: 'Falta a senha na string de conexão. Use password=yourpassword',
          databaseMissing:
            'Falta o nome do banco de dados na string de conexão. Use database=database',
          parseFailed: 'Não foi possível interpretar a string de conexão no formato chave=valor',
        },
        libpq: {
          databaseMissing:
            'Falta o nome do banco de dados na string de conexão. Use dbname=database',
          parseFailed: 'Não foi possível interpretar a string de conexão libpq',
        },
      },
    },
    physicalConnectionErrors: {
      grantAsSuperuserNote:
        'Conceda o privilégio com o comando abaixo (execute como superusuário).',
      applyAndRestartNote: 'Aplique a alteração abaixo e reinicie o PostgreSQL.',
      pgHbaNoEntry: {
        title: 'A replicação não é permitida a partir deste host',
        summary:
          'O pg_hba.conf do PostgreSQL não tem uma entrada que permita ao Databasus uma conexão de replicação. Os backups físicos são transmitidos por replicação, que exige uma regra "host replication": uma regra comum "host all" não cobre esse caso.',
        addLineNote:
          'Adicione a linha abaixo ao pg_hba.conf. O primeiro "all" aceita qualquer usuário, e o segundo, qualquer host. Para restringir o acesso, troque o usuário pelo papel de replicação específico e/ou o host por uma faixa CIDR.',
        reloadNote:
          'Depois de alterar o pg_hba.conf, <bold>recarregue a configuração do PG</bold>.',
        managedNote:
          'No PostgreSQL gerenciado (RDS / Azure / GCP) não é possível editar o pg_hba.conf: ative o acesso externo ou de replicação no console do provedor.',
      },
      badCredentials: {
        title: 'Usuário ou senha incorretos',
        summary:
          'O PostgreSQL recusou as credenciais. Confira o usuário e a senha deste banco de dados.',
      },
      noReplicationPrivilege: {
        title: 'O usuário não pode fazer replicação',
        summary:
          'O usuário se conectou, mas não tem o privilégio REPLICATION, que os backups físicos exigem.',
        managedNote:
          'No AWS RDS, use GRANT rds_replication TO {{username}}; no Azure / GCP, ative a replicação para o papel no console do provedor.',
      },
      noWalSwitchPrivilege: {
        title: 'O usuário não pode forçar a troca de segmento WAL',
        summary:
          'O streaming contínuo de WAL só envia um segmento depois que a origem o fecha. Por isso, em um banco de dados com poucas gravações, o WAL mais recente fica na origem até o segmento de 16 MB encher. Forçar a troca periodicamente é o que mantém o ponto de recuperação entre backups próximo do momento atual, e isso exige EXECUTE em pg_switch_wal(). Backups completos e incrementais não precisam disso: eles restauram até o ponto em que cada backup terminou e não reproduzem WAL arquivado.',
        switchBackupTypeNote:
          'Ou mude o tipo de backup para <bold>Completo + incremental</bold>, que não reproduz WAL arquivado.',
        managedNote:
          'No PostgreSQL gerenciado (RDS / Azure / GCP), a função pertence ao superusuário da plataforma e não é possível conceder EXECUTE nela a um papel do cliente. Por isso, o streaming contínuo de WAL não está disponível nesses serviços: use backups completos + incrementais.',
      },
      walLevelInvalid: {
        title: 'O wal_level está baixo demais',
        summary:
          'Para backups físicos, o wal_level precisa ser "replica" ou "logical". No momento, o valor é inferior.',
        applyNote:
          'Aplique a alteração abaixo e reinicie o PostgreSQL (o wal_level só entra em vigor depois de reiniciar).',
        managedNote:
          'No PostgreSQL gerenciado, defina wal_level no grupo de parâmetros do provedor.',
      },
      noWalSenders: {
        title: 'Nenhum WAL sender disponível',
        summary:
          'O max_wal_senders é 0, então o PostgreSQL não consegue transmitir WAL para os backups.',
        managedNote:
          'No PostgreSQL gerenciado, defina max_wal_senders no grupo de parâmetros do provedor.',
      },
      noReplicationSlots: {
        title: 'Nenhum slot de replicação disponível',
        summary:
          'O max_replication_slots é 0, então o PostgreSQL não consegue alocar um slot para os backups.',
        managedNote:
          'No PostgreSQL gerenciado, defina max_replication_slots no grupo de parâmetros do provedor.',
      },
      walSummaryDisabled: {
        title: 'O summarize_wal está desativado',
        summary:
          'O summarize_wal precisa estar ativado para backups incrementais. No momento, está desativado neste servidor.',
        applyNote: 'Aplique a alteração abaixo (não é preciso reiniciar).',
        thenNote: 'e depois',
        managedNote:
          'No PostgreSQL gerenciado, defina summarize_wal no grupo de parâmetros do provedor.',
      },
      customTablespaces: {
        title: 'Tablespaces personalizados não são suportados',
        summary:
          'Este cluster tem tablespaces fora de pg_default / pg_global, e os backups físicos não conseguem transmiti-los. Remova os tablespaces personalizados ou mude este banco de dados para backups lógicos.',
      },
      systemIdentifierMismatch: {
        title: 'Este é outro cluster',
        summary:
          'O cluster neste endereço tem um identificador de sistema diferente daquele para o qual este backup foi configurado. Aponte o Databasus para o cluster original ou adicione um novo banco de dados.',
      },
      connectionFailed: {
        title: 'Não foi possível conectar',
        summary:
          'O Databasus não conseguiu abrir uma conexão de replicação. Verifique o host, a porta e o modo SSL, e confirme que nenhum firewall está bloqueando a conexão.',
      },
    },
  },
  restores: {
    list: {
      empty: 'Nenhuma restauração ainda',
    },
    targetDatabase: {
      select: 'Selecione o banco de dados de destino da restauração',
      description:
        'Informe os dados do banco de dados em que o backup será restaurado. <underline>O banco de dados vazio para a restauração precisa ser criado antes</underline>. Durante a restauração, todos os dados atuais dele serão apagados',
      notInUseHint:
        'Confira se ninguém está usando o banco de dados agora (provavelmente você não quer restaurar os dados no mesmo BD de onde o backup foi feito)',
      submit: 'Restaurar neste BD',
    },
    card: {
      startedAt: 'Início',
      duration: 'Duração',
      errorDetailsTooltip: 'Clique para ver os detalhes do erro',
      cancelTooltip: 'Cancelar restauração',
      expectedDurationHint:
        'A restauração costuma levar de 3 a 5 vezes mais tempo que o backup (às vezes menos, às vezes mais, dependendo do tipo de dados)',
      expectedDurationLimit: 'Ou seja, pode levar até {{duration}} (geralmente bem menos)',
    },
    errorDetails: {
      title: 'Detalhes do erro de restauração',
      copied: 'Mensagem de erro copiada para a área de transferência',
      excludeExtensionsTip:
        '<bold>💡 Dica:</bold> este erro costuma aparecer ao restaurar em serviços PostgreSQL gerenciados (como Yandex Cloud, AWS RDS e similares). Antes de restaurar, tente ativar <bold>"Ignorar extensões"</bold> nas Configurações avançadas.',
    },
    cancelConfirmation: {
      warning:
        '<bold>⚠️ Atenção:</bold> se você cancelar esta restauração, o banco de dados provavelmente ficará corrompido ou incompleto. Será preciso recriá-lo antes de tentar outra restauração.',
      question: 'Tem certeza de que deseja cancelar?',
      confirm: 'Sim, cancelar restauração',
    },
  },
  healthcheck: {
    attempts: {
      title: 'Verificações de disponibilidade',
      period: 'Período',
      periods: {
        today: 'Hoje',
        sevenDays: '7 dias',
        thirtyDays: '30 dias',
        allTime: 'Todo o período',
      },
      empty: 'Ainda não há dados',
    },
    config: {
      enable: 'Verificar disponibilidade',
      isEnabled: 'Verificação ativada',
      notifyWhenUnavailable: 'Notificar quando indisponível',
      checkIntervalMinutes: 'Intervalo de verificação (minutos)',
      attemptsBeforeDown: 'Falhas até marcar indisponível',
      storeAttemptsDays: 'Guardar histórico (dias)',
      tooltips: {
        enable: 'Ativa ou desativa a verificação de disponibilidade deste banco de dados',
        notifyWhenUnavailable: 'Envia notificações quando o banco de dados fica indisponível',
        checkIntervalMinutes:
          'Com que frequência verificar a disponibilidade do banco de dados (em minutos)',
        attemptsBeforeDown:
          'Quantas tentativas com falha são necessárias para o banco de dados ser considerado indisponível',
        storeAttemptsDays:
          'Por quantos dias guardar o histórico de verificações de disponibilidade',
      },
    },
  },
  workspaces: {
    roles: {
      owner: 'Proprietário',
      admin: 'Administrador',
      member: 'Membro',
      viewer: 'Visualizador',
    },
    fields: {
      name: 'Nome do espaço de trabalho',
      namePlaceholder: 'Digite o nome do espaço de trabalho',
    },
    create: {
      title: 'Criar espaço de trabalho',
      submit: 'Criar espaço de trabalho',
      defaultName: 'Meu espaço de trabalho',
      description:
        'O espaço de trabalho é onde você agrupa:<lineBreak/>- seus bancos de dados;<lineBreak/>- armazenamentos (disco local, S3, Google Drive, etc.);<lineBreak/>- canais de notificação (e-mail, Slack, Telegram, etc.);<lineBreak/>- controle de acesso (se você trabalha em equipe).',
      nameRequired: 'Digite o nome do espaço de trabalho',
      created: 'Espaço de trabalho criado',
      permissionDenied: {
        title: 'Permissão negada',
        description:
          'Você não tem permissão para criar espaços de trabalho. Peça ao administrador para criar um para você.',
      },
    },
    settings: {
      title: 'Configurações do espaço de trabalho',
      readOnly: 'Você não tem permissão para alterar estas configurações',
      nameRequired: 'O nome do espaço de trabalho é obrigatório',
      saving: 'Salvando...',
      saveChanges: 'Salvar alterações',
      reset: 'Redefinir',
      updated: 'Informações básicas salvas',
    },
    delete: {
      dangerZone: 'Zona de perigo',
      title: 'Excluir este espaço de trabalho',
      description:
        'Não há como desfazer a exclusão de um espaço de trabalho. Todos os dados e recursos ligados a ele serão removidos permanentemente.',
      submit: 'Excluir espaço de trabalho',
      deleting: 'Excluindo...',
      notFound: 'Espaço de trabalho não encontrado',
      noPermission: 'Você não tem permissão para excluir este espaço de trabalho',
      hasDatabases:
        'Não é possível excluir o espaço de trabalho enquanto houver bancos de dados nele. Remova-os primeiro. Bancos de dados encontrados: {{count}}.',
      deleted: 'Espaço de trabalho excluído',
      confirmation: {
        title: 'Excluir espaço de trabalho',
        question: 'Tem certeza de que deseja excluir o espaço de trabalho <workspaceName/>?',
        warning:
          '<bold>Esta ação não pode ser desfeita.</bold> Todos os dados e recursos associados serão removidos permanentemente.',
        submit: 'Excluir espaço de trabalho',
      },
    },
    members: {
      title: 'Usuários',
      count: 'Membros: {{count}}',
      empty: 'Nenhum membro encontrado',
      emptyHint: 'Clique em "Adicionar membro" para começar',
      fields: {
        member: 'Membro',
        role: 'Papel',
        joined: 'Entrou em',
        actions: 'Ações',
        email: 'Endereço de e-mail',
        emailPlaceholder: 'Digite o endereço de e-mail',
      },
      add: {
        button: 'Adicionar membro',
        title: 'Adicionar membro',
        submit: 'Adicionar membro',
        emailRequired: 'O e-mail é obrigatório',
        hint: 'Se o usuário já tiver conta, ele será adicionado diretamente. Caso contrário, receberá um convite.',
        added: 'Membro adicionado',
      },
      invited: {
        title: 'Usuário convidado',
        sentTo: 'Convite enviado para {{email}}',
        description:
          'O usuário ainda não está no sistema, mas já foi convidado para o espaço de trabalho. Quando ele se registrar com esse e-mail, vai se tornar membro do espaço de trabalho automaticamente.',
      },
      roleChanged: 'Papel do membro atualizado',
      remove: {
        tooltip: 'Remover membro',
        title: 'Remover membro',
        question: 'Tem certeza de que deseja remover "{{email}}" deste espaço de trabalho?',
        removed: 'Membro "{{email}}" removido',
      },
    },
    transferOwnership: {
      button: 'Transferir propriedade',
      title: 'Transferir a propriedade do espaço de trabalho',
      submit: 'Transferir propriedade',
      warning:
        '<bold>Atenção:</bold> esta ação não pode ser desfeita. Você deixará de ser o proprietário deste espaço de trabalho, e o novo proprietário terá controle total sobre ele.',
      noEligibleMembers:
        'Não há membros para quem transferir a propriedade. O espaço de trabalho precisa ter pelo menos mais um membro.',
      newOwner: 'Selecione o novo proprietário',
      newOwnerPlaceholder: 'Selecione um membro',
      newOwnerHint: 'O membro selecionado se tornará o proprietário do espaço de trabalho',
      memberRequired: 'Selecione o membro que receberá a propriedade',
      memberNotFound: 'Membro selecionado não encontrado',
      transferred: 'Propriedade transferida',
    },
    auditLogs: {
      empty: 'Nenhum log de auditoria encontrado para este espaço de trabalho.',
    },
  },
  users: {
    roles: {
      admin: 'Administrador',
      member: 'Membro',
    },
    fields: {
      name: 'Nome',
      email: 'E-mail',
      yourName: 'Seu nome',
      yourEmail: 'Seu e-mail',
      confirmPassword: 'Confirme a senha',
      newPassword: 'Nova senha',
    },
    validation: {
      nameRequired: 'O nome é obrigatório',
      emailRequired: 'O e-mail é obrigatório',
      passwordTooShort: 'A senha precisa ter pelo menos {{minLength}} caracteres',
      passwordsDoNotMatch: 'As senhas não coincidem',
    },
    oauth: {
      divider: 'ou use o e-mail',
      continueWithGithub: 'Continuar com GitHub',
      continueWithGoogle: 'Continuar com Google',
      invalidConfiguration: 'Configuração OAuth inválida',
    },
    signIn: {
      title: 'Entrar',
      submit: 'Entrar',
      noAccount: 'Não tem uma conta? <signUpLink>Crie uma</signUpLink>',
      forgotPassword: 'Esqueceu a senha?',
    },
    signInCode: {
      title: 'Digite o código',
      description:
        'Enviamos um código de seis dígitos para {{email}}. Ele para de funcionar 10 minutos depois do envio.',
      code: 'Código de acesso',
      submit: 'Entrar',
      resend: 'Enviar outro código',
      backToSignIn: 'Voltar para o login',
      invalidCode: 'O código tem 6 dígitos',
      codeResent: 'Enviamos um novo código. O anterior não funciona mais.',
    },
    signUp: {
      title: 'Criar conta',
      namePlaceholder: 'João Silva',
      submit: 'Criar conta',
      administersInstance: 'Esta é a primeira conta desta instância, portanto ela a administrará.',
      hasAccount: 'Já tem uma conta? <signInLink>Entre</signInLink>',
    },
    requestPasswordReset: {
      title: 'Redefinir senha',
      description: 'Digite seu endereço de e-mail e enviaremos um código de redefinição.',
      submit: 'Enviar código',
      codeSent: 'Se esse e-mail estiver cadastrado, enviamos um código de redefinição',
      rememberPassword: 'Lembrou a senha? <signInLink>Entre</signInLink>',
    },
    resetPassword: {
      title: 'Redefinir senha',
      description: 'Digite o código enviado para o seu e-mail e a nova senha.',
      code: 'Código de redefinição',
      confirmPassword: 'Confirme a senha',
      submit: 'Redefinir senha',
      invalidCode: 'O código deve ter 6 dígitos',
      succeeded: 'Senha redefinida! Redirecionando para o login...',
      noCode: 'Não recebeu o código? <requestCodeLink>Solicite um novo</requestCodeLink>',
      backToSignIn: 'Voltar para o login',
    },
    profile: {
      title: 'Perfil',
      information: 'Informações do perfil',
      userId: 'ID do usuário',
      namePlaceholder: 'Digite seu nome',
      emailPlaceholder: 'Digite seu e-mail',
      role: 'Papel',
      saveChanges: 'Salvar alterações',
      noChanges: 'Não há alterações para salvar',
      updated: 'Perfil atualizado',
      logout: 'Sair',
      changePassword: {
        title: 'Alterar senha',
        newPasswordPlaceholder: 'Digite a nova senha',
        confirmPassword: 'Confirme a nova senha',
        confirmPasswordPlaceholder: 'Digite a nova senha novamente',
        passwordsDoNotMatch: 'As novas senhas não coincidem',
        changing: 'Alterando a senha...',
        submit: 'Alterar senha',
        signedInWithNewPassword: 'Você entrou com a nova senha',
      },
    },
    list: {
      title: 'Usuários do Databasus',
      loadedOfTotal: 'Usuários: {{loaded}} de {{total}}',
      searchPlaceholder: 'Buscar por e-mail ou nome...',
      empty: 'Nenhum usuário encontrado.',
      allLoaded: 'Todos os usuários foram carregados (total: {{count}})',
      columns: {
        user: 'Usuário',
        systemRole: 'Papel no sistema',
        isActive: 'Ativo?',
        created: 'Criado em',
      },
      roleLabel: 'Papel:',
      activeLabel: 'Ativo:',
      viewAuditLogs: 'Ver logs de auditoria',
      activated: 'Usuário ativado',
      deactivated: 'Usuário desativado',
      roleChanged: 'Papel do usuário alterado',
    },
    auditLogs: {
      title: 'Logs de auditoria do usuário',
      empty: 'Nenhum log de auditoria encontrado para este usuário.',
    },
  },
  verification: {
    triggers: {
      manual: 'Manual',
      scheduled: 'Agendada',
    },
    notificationTypes: {
      verificationSuccess: 'Verificação bem-sucedida',
      verificationFailed: 'Falha na verificação',
    },
    config: {
      scheduledVerification: 'Verificação agendada',
      scheduledVerificationTooltip:
        'Restaura periodicamente o backup mais recente em um container Postgres temporário para confirmar que ele pode ser restaurado. <docsLink>Veja como funciona</docsLink>',
      interval: 'Intervalo de verificação',
      afterBackup: 'Após o backup',
      afterBackupHint:
        'Roda automaticamente após cada backup bem-sucedido. Se houver verificações pendentes na fila, elas são canceladas para que a fila não cresça sem parar quando os backups forem mais rápidos que as verificações.',
      weekday: 'Dia da semana da verificação',
      dayOfMonth: 'Dia do mês da verificação',
      timeOfDay: 'Horário da verificação',
      cron: {
        expression: 'Expressão cron (UTC)',
        format: 'Formato cron: minuto hora dia mês dia-da-semana (UTC)',
        examples: 'Exemplos:',
        exampleEverySunday: '- {{expression}} — todo domingo às 04:00 UTC',
        exampleEverySixHours: '- {{expression}} — a cada 6 horas',
        nextRun: 'Próxima execução: {{dateTime}}<lineBreak/>({{relativeTime}})',
        invalid: 'Expressão cron inválida',
      },
      notifications: 'Notificações',
      noNotifications: 'Nenhuma',
    },
    runs: {
      title: 'Verificações de restauração',
      description:
        'Cada linha é uma tentativa de restaurar um backup deste banco de dados em uma cópia temporária',
      noAgentsWarning:
        'Nenhum agente de verificação registrado. Adicione um nas configurações do Databasus, na seção "Agentes de verificação". <docsLink>Saiba mais</docsLink>',
      empty: 'Nenhuma verificação de restauração ainda',
      allLoaded: 'Todas as verificações de restauração foram carregadas (total: {{total}})',
      attempt: 'tentativa {{attempt}}',
      restoreDuration: 'restauração: {{duration}}',
      durationTooltip: 'Tempo de restauração + verificação informado pelo agente.',
      fields: {
        createdAt: 'Criada em',
        trigger: 'Execução',
        duration: 'Duração',
        actions: 'Ações',
      },
      actions: {
        cancel: 'Cancelar verificação',
        viewDetails: 'Ver detalhes',
      },
      details: {
        title: 'Detalhes da verificação de restauração',
        loadFailed: 'Não foi possível carregar os detalhes da verificação de restauração.',
        cancellationReason: 'Motivo do cancelamento',
        failure: 'Falha',
        sections: {
          status: 'Status',
          timeline: 'Linha do tempo',
          results: 'Resultados e diagnóstico',
        },
        fields: {
          attempt: 'Tentativa',
          startedAt: 'Início',
          finishedAt: 'Fim',
          restoreDuration: 'Duração da restauração',
          verifyDuration: 'Duração da verificação',
          restoredDbSize: 'Tamanho do BD restaurado',
          schemas: 'Schemas',
          tables: 'Tabelas',
          pgRestoreExitCode: 'Código de saída do pg_restore',
        },
        tableStats: {
          title: 'Linhas por tabela',
          empty: 'Nenhuma estatística por tabela foi informada.',
          schema: 'Schema',
          table: 'Tabela',
          rows: 'Linhas',
        },
      },
    },
    agents: {
      title: 'Agentes de verificação',
      description:
        'Agentes que executam as verificações de restauração para confirmar que um backup pode ser restaurado (<docsLink>saiba mais</docsLink>)',
      empty: 'Nenhum agente registrado ainda.',
      columns: {
        capacity: 'Capacidade',
        created: 'Criado em',
      },
      capacity: {
        cpu: '{{count}} CPU',
        ram: '{{count}} GB de RAM',
        disk: '{{count}} GB de disco',
        jobs: 'trabalhos: {{count}}',
        notReported: 'ainda não informada',
      },
      actions: {
        viewInstallCommands: 'Ver comandos de instalação',
        rotateToken: 'Trocar token',
      },
      deleteConfirmation: 'Excluir este agente?',
      deleted: 'Agente "{{name}}" excluído',
      copyFailed: 'Não foi possível copiar',
      agentId: 'ID do agente:',
      create: {
        title: 'Criar agente de verificação',
        tokenNotice: 'Um token será gerado e mostrado uma única vez na próxima tela.',
        namePlaceholder: 'Nome do agente',
        nameRequired: 'O nome é obrigatório',
      },
      rotate: {
        title: 'Trocar token',
        description:
          'Trocar o token de <agentName/> invalida o token atual na hora. Um agente que ainda use o token antigo será recusado no próximo heartbeat.',
      },
      token: {
        title: 'Token do agente',
        tokenFor: 'Token de <agentName/>:',
        shownOnce:
          'Ele aparece só uma vez. Guarde-o em um lugar seguro: não será possível vê-lo de novo.',
        confirmSaved: 'Já salvei o token',
      },
      install: {
        title: 'Comandos de instalação',
        description:
          'Comandos de instalação para <agentName/>. Substitua <tokenPlaceholder/> pelo token que você salvou ao criar este agente ou na última troca de token.',
        architecture: 'Arquitetura',
        stepInstall: 'Etapa 1 — Instalação',
        stepLaunch: 'Etapa 2 — Inicialização',
        capacityDefaults:
          'Os valores de capacidade abaixo são apenas um ponto de partida: ajuste <code>--max-cpu</code>, <code>--max-ram-mb</code>, <code>--max-disk-gb</code> e <code>--max-concurrent-jobs</code> à máquina que executa o agente.',
        afterInstallation: 'Depois da instalação',
        runsInBackground: 'Após o <code>start</code>, o agente roda em segundo plano',
        checkStatus: 'Ver o status: <code>./verification-agent status</code>',
        viewLogs: 'Ver os logs: <code>databasus-verification.log</code> no diretório de trabalho',
        stop: 'Parar o agente: <code>./verification-agent stop</code>',
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
      all: 'Todos',
      backupSuccess: 'Backup bem-sucedido',
      backupFailed: 'Falha no backup',
      healthcheckSuccess: 'Verificação de disponibilidade bem-sucedida',
      healthcheckFailed: 'Falha na verificação de disponibilidade',
      verificationSuccess: 'Verificação de restauração bem-sucedida',
      verificationFailed: 'Falha na verificação de restauração',
    },
    mattermostDeliveryModes: {
      webhook: 'Webhook de entrada',
      bot: 'Conta de bot',
    },
    list: {
      addNotifier: 'Adicionar canal de notificação',
      searchPlaceholder: 'Buscar canal de notificação',
      noSearchResults: 'Nenhum canal de notificação encontrado para "{{query}}"',
      description:
        'Canal de notificação é o lugar para onde as notificações são enviadas (e-mail, Slack, Telegram, etc.)',
      backToList: '← Voltar para os canais de notificação',
    },
    card: {
      notifyTo: {
        email: 'Notificações por e-mail',
        telegram: 'Notificações no Telegram',
        webhook: 'Notificações via webhook',
        slack: 'Notificações no Slack',
        discord: 'Notificações no Discord',
        teams: 'Notificações no Teams',
        mattermost: 'Notificações no Mattermost',
      },
      hasSendError: 'Com erro de envio',
    },
    create: {
      title: 'Adicionar canal de notificação',
      namePlaceholder: 'Meu chat',
    },
    config: {
      namePlaceholder: 'Digite o nome...',
      settingsTitle: 'Configurações do canal de notificação',
      sendError: {
        title: 'Erro de envio',
        errorLabel: 'Erro:',
        clearHint: 'Para limpar este erro, faça uma das opções:',
        clearBySendingTest:
          '- envie uma notificação de teste pelo botão abaixo (mesmo que você já tenha alterado as configurações);',
        clearByNextNotification: '- aguarde a próxima notificação ser enviada sem erros;',
      },
      removeConfirmation: 'Remover este canal de notificação? Esta ação não pode ser desfeita.',
      removeBlockedByDatabases:
        'Este canal de notificação está em uso em alguns bancos de dados. Remova-o desses bancos de dados primeiro.',
    },
    testNotification: {
      send: 'Enviar notificação de teste',
      sent: {
        title: 'Notificação de teste enviada!',
        description: 'A notificação de teste foi enviada com sucesso',
      },
    },
    transfer: {
      title: 'Transferir canal de notificação para outro espaço de trabalho',
      blockedByDatabases:
        'Este canal de notificação está em uso em alguns bancos de dados. Transfira ou remova esses bancos de dados primeiro.',
      description: 'Selecione o espaço de trabalho para onde transferir este canal de notificação.',
      targetWorkspace: 'Espaço de trabalho de destino',
      selectWorkspace: 'Selecione o espaço de trabalho',
      submit: 'Transferir',
    },
    fields: {
      botToken: 'Token do bot',
      targetChatId: 'ID do chat de destino',
      webhookUrl: 'URL do webhook',
      channelId: 'ID do canal',
      skipTlsVerify: 'Não verificar TLS',
      skipTls: 'Não verificar TLS',
    },
    email: {
      targetEmail: 'E-mail de destino',
      targetEmailTooltip: 'O e-mail em que você quer receber as mensagens',
      smtpHost: 'Host SMTP',
      smtpPort: 'Porta SMTP',
      smtpUser: 'Usuário SMTP',
      smtpPassword: 'Senha SMTP',
      smtpPasswordPlaceholder: 'senha',
      from: 'Remetente',
      fromTooltip:
        'Opcional. Endereço do remetente, com ou sem nome: Acme Backups <noreply@example.com>. Se ficar vazio, será usado o usuário SMTP quando ele for um endereço de e-mail; caso contrário, noreply@ com o host SMTP',
      fromAuto: '(automático)',
      advancedSettings: 'Configurações avançadas',
      skipTlsTooltip:
        'Não verifica o certificado TLS. Ative se o seu servidor SMTP usa um certificado autoassinado. Atenção: isso reduz a segurança.',
      security: 'Segurança da conexão',
      securityTooltip:
        'TLS criptografa a conexão desde o início e costuma ser usado na porta 465. STARTTLS criptografa a conexão depois de aberta e falha se o servidor não oferecer esse recurso. “Sem criptografia” envia a senha e as mensagens sem proteção: use apenas com um relay confiável que não tenha STARTTLS.',
      securityModes: {
        tls: 'TLS',
        starttls: 'STARTTLS',
        none: 'Sem criptografia',
      },
      heloName: 'Nome de saudação',
      heloNamePlaceholder: 'Automático',
      heloNameTooltip:
        'O nome com que o Databasus se apresenta ao servidor de e-mail (EHLO). Se ficar vazio, será usado o host de DATABASUS_URL ou o nome da máquina.',
    },
    telegram: {
      botTokenHelpLink: 'Como obter o token da API do bot do Telegram?',
      targetChatIdTooltip:
        'O chat em que você quer receber as mensagens (pode ser o seu chat privado ou um grupo)',
      chatIdHelp: {
        toggle: 'Como obter o ID do chat no Telegram?',
        privateChat:
          'Para obter o ID do seu chat, envie uma mensagem para <botLink>@getmyid_bot</botLink> no Telegram. <underline>Confirme que você iniciou o chat com o bot</underline>',
        group:
          'Para obter o ID do chat de um grupo, adicione ao grupo o seu bot e o <botLink>@getmyid_bot</botLink> e envie /start (o ID do chat vai aparecer)',
      },
      useProxy: 'Usar proxy',
      useProxyTooltip: 'Usa um proxy para as requisições à API do Telegram',
      proxy: 'Proxy',
      proxyUrl: 'URL do proxy',
      proxyUrlTooltip: 'Aceita http, https, socks5 e socks5h. Pode incluir usuário e senha',
      sendToGroupTopic: 'Enviar para um tópico do grupo',
      sendToGroupTopicTooltip:
        'Ative para enviar as mensagens a um tópico específico de um chat em grupo',
      threadId: 'ID do tópico',
      threadIdTooltip: 'O ID do tópico para onde as mensagens devem ser enviadas',
      topicId: 'ID do tópico',
      threadIdHelp: {
        steps:
          'Para obter o ID do tópico, abra o tópico no seu grupo do Telegram, toque no nome dele no topo e abra as informações do tópico. Copie o link do tópico e pegue o último número da URL.',
        example:
          '<bold>Exemplo:</bold> se o link do tópico for <code>https://t.me/c/2831948048/3</code>, o ID do tópico é <code>3</code>',
        note: '<bold>Observação:</bold> tópicos só funcionam em chats de grupo, não em chats privados.',
      },
    },
    webhook: {
      method: 'Método',
      sendOn: 'Quando enviar',
      sendOnPlaceholder: 'Selecione os tipos de notificação',
      customHeaders: 'Cabeçalhos personalizados',
      customHeadersTooltip:
        'Adicione cabeçalhos HTTP personalizados à requisição do webhook (por exemplo, Authorization, X-API-Key)',
      savedHeadersHidden: '*Os cabeçalhos salvos ficam ocultos por segurança',
      headers: 'Cabeçalhos',
      headerNamePlaceholder: 'Nome do cabeçalho',
      headerValuePlaceholder: 'Valor do cabeçalho',
      addHeader: 'Adicionar cabeçalho',
      hiddenHeaderValue: '(oculto)',
      bodyTemplate: {
        title: 'Modelo do corpo',
        headingVariable: '<variable/> — título da notificação',
        messageVariable: '<variable/> — mensagem da notificação',
        invalidJsonFormat: 'Formato JSON inválido',
        invalidJson: 'JSON inválido',
      },
      exampleRequest: {
        title: 'Exemplo de requisição',
        headers: 'Cabeçalhos:',
      },
    },
    slack: {
      howToConnect: 'Como conectar o Slack (como obter o token do bot e o ID do chat)?',
      testFailedHint:
        'Confira se o canal é público ou se o bot foi adicionado ao canal privado (via @invite) ou ao grupo. Para mensagens diretas, use o ID de usuário do perfil no Slack.',
    },
    discord: {
      channelWebhookUrl: 'URL do webhook do canal',
      howTo: {
        title: 'Como obter a URL do webhook do Discord:',
        createChannel: '1. Crie ou selecione um canal do Discord',
        openChannelSettings: '2. Abra as configurações do canal (ícone de engrenagem)',
        openIntegrations: '3. Vá em Integrações',
        createWebhook: '4. Crie um novo webhook',
        copyWebhookUrl: '5. Copie a URL do webhook',
        privacyNote: 'Observação: se necessário, torne o canal privado',
      },
    },
    teams: {
      howToConnect: 'Como conectar o Microsoft Teams?',
      powerAutomateUrl: 'URL do Power Automate',
      powerAutomateUrlLabel: 'URL do Power Automate:',
      powerAutomateUrlTooltip:
        'Endpoint HTTP do seu fluxo do Power Automate (gatilho When an HTTP request is received)',
      showFullUrl: 'Mostrar',
      hideFullUrl: 'Ocultar',
    },
    mattermost: {
      howToConnect: 'Como conectar o Mattermost?',
      connectVia: 'Conectar via',
      incomingWebhookUrl: 'URL do webhook de entrada',
      serverUrl: 'URL do servidor',
      channelIdTooltip:
        'ID do canal com 26 caracteres, não o nome dele. Abra o canal, clique no nome e escolha View Info.',
      channel: 'Canal',
      showOptionalSettings: 'Mostrar configurações opcionais',
      hideOptionalSettings: 'Ocultar configurações opcionais',
      channelOverride: 'Substituir canal',
      channelOverrideTooltip:
        'Publica neste canal em vez daquele para o qual o webhook foi criado. O Mattermost ignora esta opção quando o webhook está preso a um canal.',
      postAsUsername: 'Publicar com o nome de usuário',
      postAsUsernameTooltip:
        'Exige a opção Enable integrations to override usernames ativada no console do sistema do Mattermost. Caso contrário, é ignorado.',
      postAsIconUrl: 'Publicar com o ícone (URL)',
      postAsIconUrlTooltip:
        'Exige a opção Enable integrations to override profile picture icons ativada no console do sistema do Mattermost. Caso contrário, é ignorado.',
      skipTlsTooltip:
        'Não verifica o certificado TLS. Ative se o seu servidor Mattermost usa um certificado autoassinado. Atenção: isso reduz a segurança.',
      webhookHowTo: {
        title: 'Como obter a URL de um webhook de entrada:',
        openIncomingWebhooks: '1. Menu principal → Integrations → Incoming Webhooks',
        addWebhook: '2. Clique em Add Incoming Webhook e escolha o canal',
        copyUrl: '3. Copie a URL gerada',
      },
      botHowTo: {
        title: 'Como obter um token de bot:',
        addBotAccount: '1. Integrations → Bot Accounts → Add Bot Account',
        copyToken: '2. Copie o token, que aparece uma única vez, logo após a criação',
        addBotToChannel: '3. Adicione o bot à equipe e ao canal',
      },
    },
  },
  backups: {
    encryption: {
      none: 'Nenhuma',
      encrypted: 'Ativada',
    },
    encryptionOptions: {
      none: 'Sem criptografia',
      encrypted: 'Criptografar os arquivos de backup',
    },
    list: {
      title: 'Backups',
      scheduledBackupsDisabled:
        'Os backups agendados estão desativados (você pode reativá-los na configuração de backup)',
      empty: 'Nenhum backup ainda',
      allLoaded: 'Todos os backups foram carregados (total: {{count}})',
      clickToSeeErrorDetails: 'Clique para ver os detalhes do erro',
      errorDetailsTitle: 'Detalhes do erro de backup',
      restoreTitle: 'Restaurar a partir do backup',
      columns: {
        createdAt: 'Criado em',
        created: 'Criado em',
        size: 'Tamanho',
        duration: 'Duração',
        actions: 'Ações',
      },
      actions: {
        cancelBackup: 'Cancelar backup',
        deleteBackup: 'Excluir backup',
      },
    },
    filters: {
      allTypes: 'Todos os tipos',
      allStatuses: 'Todos os status',
      before: 'Antes de',
    },
    config: {
      backupsEnabled: 'Backups ativados',
      storage: 'Armazenamento',
      selectStorage: 'Selecione o armazenamento',
      createNewStorage: 'Criar armazenamento',
      addStorageTitle: 'Adicionar armazenamento',
      storageDescription:
        'Armazenamento é o lugar onde os backups ficam guardados (disco local, S3, Google Drive, etc.)',
      storageChangeWarning:
        'Se você trocar o armazenamento, todos os backups guardados nele serão excluídos.',
      storageChangeAcknowledge: 'Entendi',
      encryption: 'Criptografia',
      notifications: 'Notificações',
      noNotifications: 'Nenhuma',
      schedule: {
        weekday: 'Dia da semana',
        dayOfMonth: 'Dia do mês',
        cronExpression: 'Expressão cron (UTC)',
        timeOfDay: 'Horário',
        nextRun: 'Próxima execução: {{time}}',
        invalidCron: 'Expressão cron inválida',
        cronHelp: {
          format: 'Formato cron: minuto hora dia mês dia-da-semana (UTC)',
          examples: 'Exemplos:',
          dailyAt2am: 'Todos os dias às 02:00 UTC',
          every6Hours: 'A cada 6 horas',
          mondaysAt3am: 'Toda segunda-feira às 03:00 UTC',
          twiceAMonth: 'Nos dias 1 e 15 às 04:30 UTC',
        },
      },
      gfs: {
        hourly: 'Por hora: {{count}}',
        daily: 'Diários: {{count}}',
        weekly: 'Semanais: {{count}}',
        monthly: 'Mensais: {{count}}',
        yearly: 'Anuais: {{count}}',
        notConfigured: 'Não configurado',
      },
    },
    logical: {
      retentionPolicyTypes: {
        gfs: 'GFS (manter os últimos N backups por hora, diários, semanais, mensais e anuais)',
        timePeriod: 'Período (últimos N dias)',
        count: 'Quantidade (últimos N backups)',
      },
      notificationTypes: {
        backupSuccess: 'Backup bem-sucedido',
        backupFailed: 'Falha no backup',
      },
      list: {
        backupNow: 'Fazer backup',
        makeBackupNow: 'Fazer backup agora',
        encrypted: 'Criptografado',
        sizeTooltip:
          'Em cima: o tamanho do arquivo de backup compactado que de fato fica no armazenamento (local, S3, Google Drive, etc.). Embaixo, em cinza: o tamanho original do banco de dados, sem compactação, no momento do backup. Os backups costumam ficar cerca de 5 vezes menores.',
        databaseSize: '{{size}} (tamanho do BD)',
        deleteConfirmation: 'Tem certeza de que deseja excluir este backup?',
        actions: {
          restore: 'Restaurar a partir do backup',
          verifyRestore:
            'Verificar a restauração: coloca na fila um agente que restaura este backup em um banco de dados temporário e informa a contagem de linhas',
          download: {
            postgresql:
              'Baixar o arquivo de backup. Ele pode ser restaurado manualmente com o pg_restore (formato custom)',
            mysql:
              'Baixar o arquivo de backup. Ele pode ser restaurado manualmente com o cliente mysql (dump SQL)',
            mariadb:
              'Baixar o arquivo de backup. Ele pode ser restaurado manualmente com o cliente mariadb (dump SQL)',
            mongodb:
              'Baixar o arquivo de backup. Ele pode ser restaurado manualmente com o mongorestore (formato archive)',
            other: 'Baixar o arquivo de backup',
          },
        },
        verifyRestore: {
          title: 'Verificar a restauração?',
          description:
            'Um agente vai restaurar este backup em um banco de dados temporário e informar a contagem de linhas. Isso pode levar algum tempo, dependendo do tamanho do backup. <docsLink>Como funciona?</docsLink>',
          queue: 'Colocar na fila',
          queued: 'Verificação de restauração na fila',
        },
      },
      config: {
        interval: 'Intervalo de backup',
        weekday: 'Dia da semana do backup',
        dayOfMonth: 'Dia do mês do backup',
        timeOfDay: 'Horário do backup',
        encryptionTooltip:
          'Se o backup for criptografado, os arquivos de backup no seu armazenamento (S3, local, etc.) não podem ser usados diretamente. Você pode restaurar os backups pelo Databasus ou baixá-los descriptografados pelo botão "Baixar".',
        retentionPolicy: 'Política de retenção',
        retention: {
          timePeriodTooltip:
            'Por quanto tempo manter os backups. Backups mais antigos que esse período são excluídos automaticamente.',
          count: 'Backups mais recentes',
          countTooltip:
            'Mantém só a quantidade informada de backups mais recentes. Os backups mais antigos além dessa quantidade são excluídos automaticamente.',
          countValue: 'Backups mais recentes: {{count}}',
          showGfsHint: 'O que é GFS (Grandfather-Father-Son)?',
          hideGfsHint: 'Ocultar',
          gfsHint:
            'Rotação GFS (Grandfather-Father-Son): mantém os últimos N backups por hora, diários, semanais, mensais e anuais. Assim é possível guardar backups por longos períodos sem ocupar espaço demais no armazenamento.',
          gfsTooltip:
            'Rotação Grandfather-Father-Son: mantém os últimos N backups por hora, diários, semanais, mensais e anuais.',
          gfsFields: {
            hourly: 'Backups por hora',
            daily: 'Backups diários',
            weekly: 'Backups semanais',
            monthly: 'Backups mensais',
            yearly: 'Backups anuais',
          },
        },
        advancedSettings: 'Configurações avançadas',
        retryIfFailed: 'Repetir o backup em caso de falha',
        retryIfFailedShort: 'Repetir se falhar',
        retryIfFailedTooltip:
          'Repete automaticamente os backups que falharem. Um backup pode falhar por problemas de rede, problemas no armazenamento ou indisponibilidade temporária do banco de dados.',
        maxFailedTries: 'Máximo de tentativas',
        maxFailedTriesTooltip:
          'Número máximo de novas tentativas para backups com falha. Você receberá uma notificação quando todas as tentativas falharem.',
      },
    },
    physical: {
      types: {
        full: 'Completo',
        incremental: 'Incremental',
        wal: 'WAL',
      },
      retentions: {
        chains: 'Cadeias',
        fullBackups: 'Backups completos',
        chainsAndFullBackups: 'Cadeias e backups completos',
      },
      fullBackupsPolicies: {
        lastN: 'Quantidade',
        gfs: 'GFS',
      },
      notificationTypes: {
        backupSuccess: 'Backup bem-sucedido',
        backupFailed: 'Falha no backup',
        chainBroken: 'Cadeia quebrada',
        walGap: 'Lacuna no WAL',
      },
      list: {
        totalUsage: 'Uso total: {{size}}',
        backUpNow: 'Fazer backup agora',
        fullBackup: 'Backup completo',
        incrementalBackup: 'Backup incremental',
        restore: 'Restaurar',
        restoreFromThisBackup: 'Restaurar a partir deste backup',
        pointInTimeRestoreTitle: 'Restauração point-in-time',
        deleteFullConfirmation:
          'Excluir este backup completo remove toda a cadeia dele (todos os incrementais que dependem dele). Esta ação não pode ser desfeita. Continuar?',
        deleteIncrementalConfirmation:
          'Excluir este backup incremental remove todos os incrementais posteriores que dependem dele. Esta ação não pode ser desfeita. Continuar?',
      },
      config: {
        fullBackupCadence: 'Frequência dos backups completos',
        incrementalBackupCadence: 'Frequência dos backups incrementais',
        incrementalMoreFrequent:
          'Os backups incrementais precisam rodar com mais frequência que os completos.',
        encryptionTooltip:
          'Se os backups forem criptografados, os arquivos no seu armazenamento não podem ser usados diretamente. Você pode restaurá-los pelo Databasus ou baixá-los descriptografados.',
        retention: 'Retenção',
        fullBackupsOnlyRetention:
          'Este banco de dados guarda só backups completos, então a retenção se aplica a eles.',
        retentionOptions: {
          chains: 'Manter as últimas N cadeias',
          chainsAndFullBackups: 'Manter as últimas N cadeias e backups completos',
        },
        retentionHelp: {
          chain:
            'Uma cadeia é um backup completo mais os incrementais (e o WAL transmitido) que dependem dele. Para restaurar, é preciso o backup completo e todos os incrementais da cadeia.',
          chains:
            'Mantém só as últimas N cadeias. Quando uma cadeia sai desse limite, o backup completo e todos os incrementais dela são excluídos: só é possível restaurar dentro das cadeias mantidas.',
          chainsAndFullBackups:
            'Mantém as últimas N cadeias para a recuperação point-in-time recente e, além disso, guarda backups completos mais antigos conforme a política abaixo (GFS ou últimos N). Eles servem como pontos de restauração de longo prazo depois que os incrementais deles forem excluídos.',
        },
        chainsCount: 'Cadeias',
        chainsCountTooltip:
          'Quantas cadeias de backup mais recentes manter. Uma cadeia é um backup completo mais os incrementais dele.',
        fullBackups: 'Backups completos',
        fullBackupsCount: 'Backups completos mais recentes',
        gfsFields: {
          hourly: 'Por hora',
          daily: 'Diários',
          weekly: 'Semanais',
          monthly: 'Mensais',
          yearly: 'Anuais',
        },
        chainsKept: 'Cadeias mantidas',
        fullBackupsKept: 'Backups completos mantidos',
        fullBackupsCountValue: 'Mais recentes: {{count}}',
        gfsTooltip:
          'Rotação Grandfather-Father-Son: mantém os últimos N backups completos por hora, diários, semanais, mensais e anuais.',
      },
      restore: {
        errorHints: {
          downloadInProgress:
            'Já há um download de restauração em andamento para este banco de dados. Aguarde a conclusão e tente novamente.',
          targetTimeUnreachable:
            'Não é possível chegar ao momento solicitado: há uma lacuna no WAL ou o horário está fora do intervalo disponível. Escolha outro horário ou restaure o ponto mais recente disponível.',
        },
        backupIntro:
          'Comando de restauração deste backup. Um backup completo precisa apenas do próprio arquivo; um incremental precisa também do backup completo dele e de todos os incrementais anteriores.',
        pointInTimeIntro:
          'Restauração point-in-time. Escolha o momento de destino ou deixe em branco para restaurar o ponto mais recente disponível.',
        targetTime: 'Momento de destino',
        latestAvailable: 'Mais recente disponível',
        preparing: 'Preparando o comando de restauração...',
        linkExpiry:
          'O link de download só pode ser usado uma vez e expira em 15 minutos. Se um comando falhar com erro 401, abra esta janela de novo para gerar um novo link.',
        copied: 'Copiado',
        methods: {
          script: 'Por script',
          manual: 'Manual',
        },
        environments: {
          host: 'PostgreSQL no host',
        },
        fields: {
          restoreDirectory: 'Diretório de restauração',
          pgBinPath: 'Caminho do bin do PostgreSQL',
          pgBinPathPlaceholder: '{{path}} (opcional)',
          pgImage: 'Imagem do PostgreSQL',
        },
        scriptTitles: {
          host: 'Execute no host de restauração',
          docker: 'Execute onde o Docker estiver disponível',
        },
        manualSteps: {
          download: 'Baixe o pacote',
          extract: 'Extraia o pacote',
          verify: 'Verifique a transferência',
          reconstruct: 'Reconstrua o diretório de dados',
          checkConfig: 'Verifique os arquivos de configuração do cluster',
          wireUpRecovery: 'Descompacte o WAL e configure a recuperação',
          recoveryParameters: 'Se a recuperação parar por causa de parâmetros',
        },
        beforeYouRun: {
          title: 'Antes de executar',
          host: {
            clientTools:
              'Instale as ferramentas de cliente do PostgreSQL {{version}} e informe o caminho do bin acima se elas não estiverem no PATH.',
            zstd: 'O zstd é necessário para descompactar o backup.',
            zstdWithWal: 'O zstd é necessário para descompactar o backup e o WAL dele.',
            emptyDirectory:
              'O diretório de restauração precisa estar vazio e não pode pertencer a um cluster em uso.',
          },
          docker: {
            image:
              'Use uma imagem postgres:{{version}}: a versão principal precisa ser a mesma da origem.',
            hostTools:
              'O host precisa de zstd, tar e curl: o download e toda a descompactação (do backup base) rodam no host, e só o pg_combinebackup roda no container, então a imagem do postgres não precisa de ferramentas extras.',
            hostToolsWithWal:
              'O host precisa de zstd, tar e curl: o download e toda a descompactação (do backup base e do WAL) rodam no host, e só o pg_combinebackup roda no container, então a imagem do postgres não precisa de ferramentas extras.',
          },
        },
        afterItFinishes: {
          title: 'Depois que terminar',
          host: {
            clusterLocation: 'O cluster fica em <dataDir/>.',
            changeOwner: 'Como root, altere o dono: <command/>.',
            start:
              'Inicie como postgres: <command/>, ou aponte o data_directory para esse diretório.',
            walReplay:
              'O PostgreSQL reproduz o WAL e é promovido: acompanhe no log a conclusão da recuperação.',
          },
          docker: {
            clusterLocation: '<dataDir/> no host é o cluster restaurado.',
            volumeMount:
              'O PostgreSQL {{version}} mantém o diretório de dados em <dataDir/>, dentro do volume <volumeDir/>. Monte o diretório de saída como a raiz desse volume, que é o que um docker-compose com <code>./pgdata:/var/lib/postgresql</code> espera:',
            dataDirectoryMount:
              'Monte o cluster (bind mount) no diretório de dados da imagem, <dataDir/>:',
            ownership:
              'O dono precisa ser o mesmo uid do postgres dentro da imagem (999 na imagem oficial).',
          },
          missingConfigFiles:
            'Um backup base copia só o diretório de dados. Se o PostgreSQL de origem guarda a configuração fora dele (é o layout do Debian/Ubuntu, <configDir/>), então <code>postgresql.conf</code>, <code>pg_hba.conf</code> e <code>pg_ident.conf</code> não estão no backup e o servidor não vai iniciar. Consulte os caminhos reais na origem com <code>psql -Atc "SHOW config_file"</code>, copie os três arquivos e depois comente <code>data_directory</code>, <code>hba_file</code>, <code>ident_file</code>, <code>external_pid_file</code> e as linhas <code>ssl</code> / <code>ssl_*</code>, crie <code>conf.d</code> e defina <code>listen_addresses = \'*\'</code> para um container. Se esses arquivos estão presentes e o mesmo erro continua, o problema é o diretório montado, não a configuração.',
        },
      },
    },
  },
  storages: {
    types: {
      local: 'Armazenamento local',
      s3: 'S3',
      googleDrive: 'Google Drive',
      nas: 'NAS',
      azureBlob: 'Azure Blob Storage',
      ftp: 'FTP',
      sftp: 'SFTP',
      rclone: 'Rclone',
    },
    s3StorageClasses: {
      default: 'Padrão (Standard)',
      standard: 'Standard',
      standardIa: 'Standard - Infrequent Access',
      onezoneIa: 'One Zone - Infrequent Access',
      intelligentTiering: 'Intelligent Tiering',
      reducedRedundancy: 'Reduced Redundancy',
      glacierIr: 'Glacier Instant Retrieval',
    },
    azureBlobAuthMethods: {
      accountKey: 'Chave da conta',
      connectionString: 'String de conexão',
    },
    sftpAuthMethods: {
      password: 'Senha',
      privateKey: 'Chave privada',
    },
    list: {
      addStorage: 'Adicionar armazenamento',
      description:
        'Armazenamento é o lugar onde os backups ficam guardados (disco local, S3, etc.)',
      backToList: '← Voltar para os armazenamentos',
    },
    card: {
      type: {
        local: 'Tipo: armazenamento local',
        s3: 'Tipo: S3',
        googleDrive: 'Tipo: Google Drive',
        nas: 'Tipo: NAS',
        azureBlob: 'Tipo: Azure Blob Storage',
        ftp: 'Tipo: FTP',
        sftp: 'Tipo: SFTP',
        rclone: 'Tipo: Rclone',
      },
      hasSaveError: 'Com erro ao salvar',
    },
    config: {
      namePlaceholder: 'Digite o nome...',
      settingsTitle: 'Configurações do armazenamento',
      lastSaveError: {
        title: 'Erro ao salvar',
        errorLabel: 'Erro:',
        clearHint: 'Para limpar este erro, faça uma das opções:',
        clearByTestingConnection:
          '- teste a conexão pelo botão abaixo (mesmo que você já tenha alterado as configurações);',
        clearByNextSave: '- aguarde o próximo backup ser salvo sem erros;',
      },
      removeConfirmation:
        'Remover este armazenamento? Esta ação não pode ser desfeita. Os backups guardados neste armazenamento também serão removidos.',
      removeBlockedByDatabases:
        'Este armazenamento está em uso em alguns bancos de dados. Remova-o desses bancos de dados primeiro.',
    },
    actions: {
      testConnection: 'Testar conexão',
    },
    connectionTestSucceeded: {
      title: 'Conexão bem-sucedida!',
      description: 'A conexão com o armazenamento foi testada com sucesso',
    },
    transfer: {
      title: 'Transferir armazenamento para outro espaço de trabalho',
      blockedByDatabases:
        'Este armazenamento está em uso em alguns bancos de dados. Transfira ou remova esses bancos de dados primeiro.',
      description: 'Selecione o espaço de trabalho para onde transferir este armazenamento.',
      targetWorkspace: 'Espaço de trabalho de destino',
      selectWorkspace: 'Selecione o espaço de trabalho',
      submit: 'Transferir',
    },
    fields: {
      path: 'Caminho',
      remotePath: 'Caminho remoto',
      share: 'Compartilhamento',
      domain: 'Domínio',
      useSsl: 'Usar SSL',
      useSslTls: 'Usar SSL/TLS',
      skipTls: 'Não verificar TLS',
      skipTlsVerify: 'Não verificar TLS',
      skipHostKey: 'Não verificar a chave do host',
      authMethod: 'Método de autenticação',
      privateKey: 'Chave privada',
      credentials: 'Credenciais',
      s3Bucket: 'Bucket S3',
      region: 'Região',
      accessKey: 'Chave de acesso',
      secretKey: 'Chave secreta',
      endpoint: 'Endpoint',
      folderPrefix: 'Prefixo da pasta',
      prefix: 'Prefixo',
      blobPrefix: 'Prefixo dos blobs',
      virtualHost: 'Host virtual',
      storageClass: 'Classe de armazenamento',
      connection: 'String de conexão',
      connectionString: 'String de conexão',
      accountName: 'Nome da conta',
      accountKey: 'Chave da conta',
      containerName: 'Nome do container',
      clientId: 'ID do cliente',
      clientSecret: 'Chave secreta do cliente',
      userToken: 'Token do usuário',
      config: 'Configuração',
    },
    edit: {
      namePlaceholder: 'Meu armazenamento',
      advancedSettings: 'Configurações avançadas',
      optionalPlaceholder: '{{example}} (opcional)',
      usernamePlaceholder: 'usuário',
      passwordPlaceholder: 'senha',
      remotePathTooltip: 'Caminho do diretório remoto onde os backups serão guardados (opcional)',
      local: {
        diskSpaceWarning:
          'Cuidado: com o armazenamento local, o espaço em disco pode acabar. Recomendamos usar o S3 ou armazenamentos sem limite de espaço',
      },
      s3: {
        cloudflareR2Guide: 'Como usar com o Cloudflare R2?',
        endpointTooltip:
          'URL de um endpoint compatível com S3 (opcional; deixe em branco para o AWS S3)',
        prefixTooltip:
          "Prefixo opcional para todas as chaves de objeto (por exemplo, 'backups/' ou 'my_team/'). Pode não funcionar com alguns armazenamentos compatíveis com S3. Não pode ser alterado depois da criação (caso contrário, os backups serão perdidos).",
        useVirtualHostedStyle: 'Usar estilo virtual-hosted',
        virtualHostTooltip:
          'Usa URLs no estilo virtual-hosted (bucket.s3.region.amazonaws.com) em vez do estilo path (s3.region.amazonaws.com/bucket). Pode ser necessário se aparecerem erros de COS.',
        skipTls: 'Não verificar TLS',
        skipTlsVerifyTooltip:
          'Não verifica o certificado TLS. Ative se o seu armazenamento compatível com S3 usa um certificado autoassinado. Atenção: isso reduz a segurança.',
        storageClassTooltip:
          'Classe de armazenamento S3 para os objetos enviados. Deixe o padrão para usar Standard. Alguns provedores oferecem classes mais baratas, como One Zone IA. Não use Glacier/Deep Archive: os arquivos precisam estar disponíveis na hora para as restaurações.',
      },
      googleDrive: {
        guideLink: 'Como conectar o Google Drive?',
        authorize: 'Autorizar',
      },
      nas: {
        shareWithSlashWarning:
          'Informe só o nome de um compartilhamento. Use o campo Caminho para subdiretórios (por exemplo, Compartilhamento: Databasus, Caminho: DB1)',
        useSslTooltip: 'Ativa a criptografia SSL/TLS para proteger a conexão',
        domainTooltip:
          'Nome do domínio Windows (opcional; deixe em branco se não usar autenticação de domínio)',
        pathPlaceholder: '{{example}} (opcional, sem barra no início)',
        pathTooltip: 'Caminho do subdiretório dentro do compartilhamento (opcional)',
      },
      azureBlob: {
        connectionStringTooltip: 'String de conexão do Azure Storage, disponível no Azure Portal',
        endpointTooltip:
          'URL de um endpoint personalizado (opcional; deixe em branco para o Azure padrão)',
        prefixTooltip:
          "Prefixo opcional para todos os nomes de blob (por exemplo, 'backups/' ou 'my_team/')",
      },
      ftp: {
        enableFtps: 'Ativar FTPS',
        useSslTooltip:
          'Usa criptografia TLS explícita (FTPS) para transferir os arquivos com segurança',
        skipCertificateVerification: 'Não verificar o certificado',
        skipTlsVerifyTooltip:
          'Não verifica o certificado TLS. Ative se o seu servidor FTP usa um certificado autoassinado. Atenção: isso reduz a segurança.',
      },
      sftp: {
        privateKeyTooltip:
          'Cole sua chave privada SSH (formato PEM). Aceita chaves RSA, DSA, ECDSA e Ed25519.',
        skipHostKeyVerification: 'Não verificar a chave do host',
        skipHostKeyTooltip:
          'Não verifica a chave SSH do host. Ative se você confia no servidor. Atenção: isso reduz a segurança.',
      },
      rclone: {
        configTooltip:
          "Cole aqui o conteúdo do rclone.conf. Você pode obtê-lo executando 'rclone config file' e copiando o resultado. Essa configuração aceita mais de 70 provedores de armazenamento na nuvem.",
        configHiddenNote:
          '*o conteúdo fica oculto para não expor dados sensíveis. Se quiser atualizar a configuração existente, cole uma nova aqui',
        docsLink: 'Documentação do Rclone',
        remotePathTooltip:
          "Prefixo de caminho opcional no remote do rclone onde os backups serão guardados (por exemplo, '/backups' ou 'my-folder/backups')",
      },
    },
  },
};

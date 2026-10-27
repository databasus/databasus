import type { en } from './en';

export const id: typeof en = {
  language: { nativeName: 'Bahasa Indonesia' },
  common: {
    actions: {
      add: 'Tambah',
      back: 'Kembali',
      cancel: 'Batal',
      close: 'Tutup',
      confirm: 'Konfirmasi',
      continue: 'Lanjut',
      copy: 'Salin',
      create: 'Buat',
      delete: 'Hapus',
      download: 'Unduh',
      edit: 'Edit',
      ok: 'OK',
      refresh: 'Muat ulang',
      remove: 'Hapus',
      retry: 'Coba lagi',
      save: 'Simpan',
    },
    answers: { no: 'Tidak', yes: 'Ya' },
    fields: {
      host: 'Host',
      name: 'Nama',
      password: 'Kata sandi',
      port: 'Port',
      status: 'Status',
      type: 'Jenis',
      username: 'Nama pengguna',
    },
    states: { disabled: 'Nonaktif', enabled: 'Aktif', loading: 'Memuat...' },
    messages: { copiedToClipboard: 'Disalin ke clipboard' },
    listSeparator: ', ',
    confirmation: { title: 'Konfirmasi' },
  },
  app: {
    documentTitle: 'Databasus - backup database (PostgreSQL, MySQL, MongoDB)',
    languageControl: { title: 'Bahasa: {{language}}' },
    theme: { title: 'Tema: {{theme}}', light: 'Terang', dark: 'Gelap', system: 'Sistem' },
    navigation: {
      databases: 'Database',
      storages: 'Penyimpanan',
      notifiers: 'Kanal notifikasi',
      workspaceSettings: 'Pengaturan',
      profile: 'Profil',
      databasusSettings: 'Pengaturan Databasus',
      users: 'Pengguna',
      docs: 'Dokumentasi',
      documentation: 'Dokumentasi',
      community: 'Komunitas',
      newVersionAvailable: 'Versi baru tersedia',
    },
    diskUsage: {
      title: 'Penggunaan disk',
      hint: 'Backup lokal dan restore membutuhkan ruang disk yang cukup. Untuk restore, sediakan ruang kosong sebesar ukuran backup.',
      compactSummary: '{{used}} dari {{total}} GB<lineBreak/>ROM terpakai ({{percent}}%)',
      summary: '{{used}} dari {{total}} GB terpakai ({{percent}}%)',
    },
    workspaceSelection: {
      createShort: 'Buat',
      create: 'Buat ruang kerja',
      selected: 'Ruang kerja terpilih',
      placeholder: 'Pilih ruang kerja',
      search: 'Cari ruang kerja...',
      notFound: 'Ruang kerja tidak ditemukan',
      createNew: '+ Buat ruang kerja baru',
    },
    starButton: {
      label: 'Beri bintang di GitHub',
      ariaLabel: 'Beri bintang untuk databasus/databasus di GitHub',
    },
    clipboardPaste: {
      title: 'Tempel dari clipboard',
      submit: 'Kirim',
      description: 'Akses clipboard otomatis tidak tersedia. Tempel konten di bawah ini.',
      placeholder: 'Tempel connection string di sini...',
    },
    auth: { accountsCheckFailed: 'Gagal memeriksa apakah instance ini memiliki akun: {{error}}' },
    oauthCallback: {
      failedTitle: 'Autentikasi gagal',
      codeNotFound: 'Kode otorisasi tidak ditemukan',
      stateMissing: 'Parameter state OAuth tidak ada',
      invalidProvider: 'Penyedia OAuth tidak valid',
      returnToSignIn: 'Kembali ke halaman masuk',
      completing: 'Menyelesaikan autentikasi...',
    },
    oauthStorage: {
      configNotFound: 'Konfigurasi penyimpanan Google Drive tidak ditemukan',
      exchangeFailed: 'Gagal menukar kode OAuth',
      unsupportedType: 'Jenis penyimpanan ini tidak mendukung OAuth',
      invalidState: 'Parameter state OAuth tidak valid',
      paramNotFound:
        'Parameter OAuth tidak ditemukan. Pastikan URL redirect dikonfigurasi dengan benar.',
      addStorageTitle: 'Tambah penyimpanan',
      storageDescription: 'Penyimpanan adalah tempat menyimpan backup (disk lokal, S3, dll.)',
    },
  },
  errors: {
    unknown: 'Terjadi kesalahan. Coba lagi.',
    networkUnreachable: 'Server Databasus tidak dapat dijangkau. Periksa koneksi dan coba lagi.',
    requestFailed:
      'Server tidak dapat menyelesaikan permintaan (HTTP {{status}}). Coba lagi nanti.',
    signInCodeIncorrect: 'Kode salah.',
    pendingSignInNotUsable: 'Proses masuk ini tidak bisa dilanjutkan. Silakan masuk lagi.',
    signInCodeNotSent:
      'Instance tidak dapat mengirim kode masuk. Coba lagi nanti atau hubungi admin.',
    tooManySignInCodes:
      'Terlalu banyak permintaan kode masuk untuk akun ini dalam satu jam terakhir. Coba lagi nanti.',
    signInCodeResentTooSoon:
      'Kode baru dikirim kurang dari semenit lalu. Tunggu satu menit sebelum meminta kode lagi.',
    rateLimitExceeded: 'Terlalu banyak percobaan. Tunggu sebentar dan coba lagi.',
    emailNotConfigured: 'Server email belum dikonfigurasi pada instance ini.',
    adminEmailMissing: 'Akun kamu tidak memiliki alamat email yang valid untuk menerima email uji.',
  },
  status: {
    logicalBackup: {
      inProgress: 'Sedang berjalan',
      completed: 'Berhasil',
      failed: 'Gagal',
      deleted: 'Dihapus',
      canceled: 'Dibatalkan',
    },
    physicalBackup: {
      inProgress: 'Sedang berjalan',
      completed: 'Berhasil',
      error: 'Error',
      chainBroken: 'Rantai terputus',
      canceled: 'Dibatalkan',
    },
    restoreVerification: {
      verifiedSuccessful: 'Terverifikasi',
      verificationFailed: 'Verifikasi gagal',
    },
    restore: {
      inProgress: 'Sedang berjalan',
      completed: 'Berhasil',
      failed: 'Gagal',
      canceled: 'Dibatalkan',
    },
    verification: {
      pending: 'Menunggu',
      running: 'Sedang berjalan',
      completed: 'Berhasil',
      failed: 'Gagal',
      canceled: 'Dibatalkan',
    },
    health: { available: 'Tersedia', unavailable: 'Tidak tersedia' },
    agent: {
      neverSeen: 'Belum pernah terhubung',
      online: 'Online',
      stale: 'Respons terlambat',
      offline: 'Offline',
    },
  },
  intervals: {
    types: {
      hourly: 'Setiap jam',
      daily: 'Setiap hari',
      weekly: 'Setiap minggu',
      monthly: 'Setiap bulan',
      cron: 'Cron',
    },
    weekdays: {
      mon: 'Sen',
      tue: 'Sel',
      wed: 'Rab',
      thu: 'Kam',
      fri: 'Jum',
      sat: 'Sab',
      sun: 'Min',
    },
    periods: {
      day: '1 hari',
      week: '1 minggu',
      month: '1 bulan',
      threeMonths: '3 bulan',
      sixMonths: '6 bulan',
      year: '1 tahun',
      twoYears: '2 tahun',
      threeYears: '3 tahun',
      fourYears: '4 tahun',
      fiveYears: '5 tahun',
      forever: 'Selamanya',
    },
  },
  auditLogs: {
    title: 'Log audit',
    columns: { user: 'Pengguna', message: 'Pesan', workspace: 'Ruang kerja', created: 'Dibuat' },
    systemUser: 'Sistem',
    loadedOfTotal: 'Dimuat: {{loaded}} dari {{total}}',
    empty: 'Log audit tidak ditemukan.',
    loadingMore: 'Memuat log lainnya...',
    allLoaded: 'Semua log dimuat (total {{count}})',
  },
  settings: {
    title: 'Pengaturan Databasus',
    externalRegistrations: {
      title: 'Izinkan pendaftaran eksternal',
      description:
        'Jika aktif, pengguna baru bisa mendaftar akun di Databasus. Jika nonaktif, pengguna baru hanya bisa mendaftar melalui undangan.',
    },
    memberInvitations: {
      title: 'Izinkan undangan dari anggota',
      description:
        'Jika aktif, anggota bisa mengundang pengguna baru ke Databasus. Jika nonaktif, hanya admin yang bisa mengundang pengguna.',
    },
    memberWorkspaceCreation: {
      title: 'Anggota bisa membuat ruang kerja',
      description:
        'Jika aktif, anggota (pengguna non-admin) bisa membuat ruang kerja baru. Jika nonaktif, hanya admin yang bisa membuat ruang kerja.',
    },
    twoFactorAuth: {
      title: 'Aktifkan autentikasi dua faktor saat masuk',
      description:
        'Jika aktif, masuk dengan kata sandi juga membutuhkan kode enam digit yang dikirim ke alamat email akun. Ini tidak berlaku untuk masuk melalui Google atau GitHub.',
      mailServerRequired:
        'Fitur ini membutuhkan server email. <docsLink>Konfigurasikan SMTP</docsLink> terlebih dahulu.',
    },
    saving: 'Menyimpan...',
    saveChanges: 'Simpan perubahan',
    reset: 'Reset',
    updated: 'Pengaturan berhasil diperbarui',
    readMore: 'Baca selengkapnya tentang pengaturan <docsLink>di sini</docsLink>',
    mailServer: {
      title: 'Server email',
      configured: 'Server email <status>sudah dikonfigurasi</status>.',
      notConfigured: 'Server email <status>belum dikonfigurasi</status>.',
      description:
        'Server ini mengirim undangan, kode reset kata sandi, dan kode masuk. Kanal notifikasi email memiliki pengaturan SMTP sendiri.',
      docs: 'Lihat <docsLink>dokumentasi SMTP</docsLink> untuk menghubungkan server email.',
      sendTestEmail: 'Kirim email uji',
      testEmailSent: 'Email uji dikirim ke {{email}}',
    },
    healthcheck: {
      title: 'Pemeriksaan ketersediaan',
      openInNewTab: 'Klik untuk membuka di tab baru',
      copied: 'Endpoint pemeriksaan ketersediaan disalin ke clipboard',
      description: 'Gunakan endpoint ini untuk memantau ketersediaan sistem Databasus',
    },
  },
  databases: {
    types: { postgresql: 'PostgreSQL', mysql: 'MySQL', mariadb: 'MariaDB', mongodb: 'MongoDB' },
    sslModes: {
      disable: 'Nonaktif',
      require: 'Wajib',
      verifyCa: 'Verifikasi CA',
      verifyFull: 'Verifikasi penuh',
    },
    physicalBackupTypes: {
      full: 'Hanya backup penuh',
      fullIncremental: 'Penuh + inkremental',
      fullIncrementalWalStream: 'Penuh + inkremental + WAL streaming',
    },
    list: {
      addDatabase: 'Tambah database',
      searchPlaceholder: 'Cari database',
      noSearchResults: 'Tidak ada database yang cocok dengan "{{query}}"',
      description: 'Database adalah data yang akan di-backup',
      backToList: '← Kembali ke database',
    },
    card: {
      storage: 'Penyimpanan: <storageName/>',
      lastBackup: 'Backup terakhir {{relativeTime}}',
      hasBackupError: 'Ada error backup',
    },
    tabs: {
      config: 'Konfigurasi',
      backups: 'Backup',
      verifications: 'Verifikasi',
      monitoring: 'Pemantauan',
    },
    create: {
      title: 'Tambah database untuk backup',
      namePlaceholder: 'Database favorit saya',
      creating: 'Membuat database...',
      complete: 'Selesai',
      backupType: {
        title: 'Pilih jenis backup',
        logical: {
          title: 'Logis',
          description: 'Disarankan untuk database di bawah 50 GB. Lebih mudah disiapkan.',
        },
        physical: {
          title: 'Fisik',
          description:
            'Untuk database di atas 50 GB. Mendukung pemulihan ke titik waktu tertentu dan RPO/RTO yang lebih baik, tetapi membutuhkan pengaturan tambahan.',
        },
      },
    },
    config: {
      namePlaceholder: 'Masukkan nama...',
      removing: 'Menghapus database...',
      lastBackupError: {
        title: 'Error backup terakhir',
        errorLabel: 'Error:',
        clearHint: 'Untuk menghapus error ini, pilih salah satu:',
        clearByTestingConnection:
          '- uji koneksi melalui tombol di bawah (meskipun pengaturan sudah diperbarui);',
        clearByNextBackup: '- tunggu sampai backup berikutnya selesai tanpa error;',
      },
      sections: {
        database: 'Pengaturan database',
        backupConfig: 'Konfigurasi backup',
        healthcheck: 'Pengaturan pemeriksaan ketersediaan',
        restoreVerification: 'Verifikasi restore',
        notifiers: 'Pengaturan kanal notifikasi',
      },
      copyConfirmation:
        'Yakin ingin menyalin database ini? Database baru dengan pengaturan yang sama akan dibuat.',
      removeConfirmation: 'Yakin ingin menghapus database ini? Tindakan ini tidak bisa dibatalkan.',
      copied: { title: 'Database berhasil disalin!', description: '"{{name}}" berhasil dibuat' },
      connectionTestSucceeded: {
        title: 'Uji koneksi berhasil!',
        description: 'Koneksi database berhasil diuji',
      },
    },
    actions: { testConnection: 'Uji koneksi', replace: 'Ganti' },
    fields: {
      databaseType: 'Jenis database',
      version: 'Versi {{engine}}',
      databaseName: 'Nama database',
      sslMode: 'Mode SSL',
      useHttps: 'Gunakan HTTPS',
      cpuCount: 'Jumlah CPU',
      backupType: 'Jenis backup',
      clientCertificate: 'Sertifikat klien',
      clientKey: 'Kunci klien',
      serverCaCertificate: 'Sertifikat CA server',
      includeSchemas: 'Sertakan schema',
      excludeTables: 'Kecualikan tabel',
      excludeWholeTables: 'Kecualikan seluruh tabel',
      excludeTableData: 'Kecualikan data tabel',
      skipUserMappings: 'Lewati user mapping',
      excludeExtensions: 'Kecualikan extension',
      restoreOwnership: 'Restore kepemilikan',
      restorePrivileges: 'Restore hak akses',
      excludeEvents: 'Kecualikan event',
      galeraReplication: 'Replikasi Galera',
      useSrvConnection: 'Gunakan koneksi SRV',
      directConnection: 'Koneksi langsung',
      authDatabase: 'Database autentikasi',
      excludeCollections: 'Kecualikan collection',
    },
    edit: {
      parseFromClipboard: 'Baca dari clipboard',
      clipboardEmpty: 'Clipboard kosong',
      clipboardReadFailed: 'Gagal membaca clipboard. Periksa izin browser.',
      connectionStringParsed: 'Connection string berhasil dibaca',
      connectionStringParsedWithoutPassword:
        'Connection string berhasil dibaca. Masukkan kata sandi secara manual.',
      connectionTestPassed: {
        title: 'Uji koneksi berhasil',
        description: 'Kamu bisa melanjutkan ke langkah berikutnya',
      },
      ipWhitelistHint:
        'Jika database menggunakan daftar IP yang diizinkan, pastikan IP server Databasus sudah ditambahkan ke daftar tersebut.',
      localhostHint:
        '<docsLink>Baca panduan ini</docsLink> untuk mempelajari cara melakukan backup database lokal',
      advancedSettings: 'Pengaturan lanjutan',
      placeholders: {
        host: 'Masukkan host {{engine}}',
        port: 'Masukkan port {{engine}}',
        username: 'Masukkan nama pengguna {{engine}}',
        password: 'Masukkan kata sandi {{engine}}',
        databaseName: 'Masukkan nama database {{engine}}',
      },
      cpuCountTooltip:
        'Jumlah core CPU untuk backup dan restore. Nilai lebih tinggi bisa mempercepat proses, tetapi memakai lebih banyak sumber daya.',
      excludeTablesPlaceholder: 'Tidak ada tabel yang dikecualikan',
      excludeTablesTooltip:
        'Nama tabel yang dikecualikan dari backup. Tempel daftar yang dipisahkan dengan koma atau baris baru.',
      serverCaCertificateTooltip:
        'Opsional. Jika diisi, sertifikat server diverifikasi menggunakan CA ini (verify-ca / verify-full).',
      copyFailed: 'Gagal menyalin',
      postgresql: {
        supabaseHint:
          '<docsLink>Baca panduan ini</docsLink> untuk mempelajari cara melakukan backup database Supabase',
        includeSchemasPlaceholder: 'Semua schema (default)',
        excludeTablesTooltip:
          "Tabel yang sama sekali tidak disertakan dalam backup: tabel ini tidak akan ada setelah restore. Gunakan 'tablename' atau 'schema.tablename'. Nama tanpa schema hanya cocok dengan tabel pada search path default, biasanya 'public'; gunakan '*.tablename' untuk mencocokkan semua schema. Mendukung pola glob (misalnya 'logs_*'). Tempel daftar yang dipisahkan dengan koma atau baris baru.",
        excludeTableDataTooltip:
          "Tabel yang di-backup tanpa baris datanya: setelah restore, struktur, index, dan constraint tetap ada, tetapi tabel kosong. Gunakan 'tablename' atau 'schema.tablename'. Nama tanpa schema hanya cocok dengan tabel pada search path default, biasanya 'public'; gunakan '*.tablename' untuk mencocokkan semua schema. Mendukung pola glob (misalnya 'logs_*'). Tabel yang tercantum di kedua kolom akan dikecualikan sepenuhnya. Partisi dari tabel berpartisi harus dicantumkan dengan namanya sendiri atau pola seperti 'events_*'. Jika tabel lain dalam backup memiliki foreign key yang merujuk ke baris tabel yang dikosongkan, restore gagal membuat ulang foreign key tersebut. Tempel daftar yang dipisahkan dengan koma atau baris baru.",
        excludeTableDataPlaceholder: 'Tidak ada data tabel yang dikecualikan',
        skipUserMappingsTooltip:
          'Lewati restore user mapping (perintah CREATE USER MAPPING). Aktifkan jika role backup tidak bisa membaca kredensial mapping. Jika tidak, mapping diekspor tanpa opsi dan membuat restore gagal untuk FDW seperti oracle_fdw.',
        excludeExtensionsTooltip:
          'Lewati restore definisi extension (perintah CREATE EXTENSION). Aktifkan jika tujuan restore adalah layanan PostgreSQL terkelola yang extension-nya dikelola penyedia.',
        restoreOwnershipTooltip:
          'Jalankan perintah ALTER OWNER dari dump agar objek hasil restore tetap memiliki pemilik aslinya. Pengguna koneksi harus bisa menetapkan role tersebut, biasanya membutuhkan superuser.',
        restorePrivilegesTooltip:
          'Jalankan perintah GRANT dan REVOKE dari dump agar objek hasil restore tetap memiliki ACL aslinya. Pengguna koneksi harus bisa memberikan hak akses kepada role yang dirujuk, biasanya membutuhkan superuser.',
        backupTypeTooltips: {
          full: 'Backup penuh berkala yang berdiri sendiri. Setiap backup lengkap tanpa bergantung pada backup lain.',
          fullIncremental:
            'Backup penuh ditambah backup inkremental yang hanya menyimpan perubahan sejak backup sebelumnya. Lebih kecil dan cepat.',
          fullIncrementalWalStream:
            'Menambahkan WAL streaming berkelanjutan pada backup penuh dan inkremental. Hanya opsi ini yang mendukung pemulihan ke titik waktu tertentu (PITR), tetapi membutuhkan lebih banyak ruang dan waktu restore lebih lama.',
        },
      },
      mariadb: {
        skipEvents: 'Lewati event',
        skipEventsTooltip:
          'Lewati backup event database. Aktifkan jika event scheduler dinonaktifkan di server MariaDB.',
        skipGaleraDisable: 'Lewati penonaktifan saat restore',
        skipGaleraDisableTooltip:
          'Secara default, Databasus menjalankan SET SESSION wsrep_on=OFF saat restore untuk menghindari error ukuran writeset Galera. Ini membutuhkan hak akses SUPER. Aktifkan untuk melewatinya jika penyedia terkelola tidak mengizinkan SUPER. Restore berukuran besar mungkin kemudian mencapai batas writeset Galera.',
      },
      mongodb: {
        srvTooltip:
          'Aktifkan untuk koneksi SRV MongoDB Atlas (mongodb+srv://). Koneksi SRV tidak membutuhkan port.',
        srvOverTunnelTooltip:
          'Tidak tersedia melalui SSH tunnel. SRV menentukan daftar host sendiri dan melewati port yang diteruskan.',
        directConnectionTooltip:
          'Hubungkan langsung ke satu server tanpa penemuan replica set. Berguna jika server berada di balik load balancer, proxy, atau tunnel.',
        directConnectionOverTunnelTooltip:
          'Selalu aktif selama SSH tunnel aktif. Tunnel hanya menyediakan satu alamat sehingga penemuan replica set tidak bisa digunakan.',
        excludeCollectionsPlaceholder: 'Tidak ada collection yang dikecualikan',
        excludeCollectionsTooltip:
          'Nama collection yang dikecualikan dari backup. Tempel daftar yang dipisahkan dengan koma atau baris baru.',
      },
    },
    show: { galeraDisableSkipped: 'Lewati penonaktifan saat restore' },
    sshTunnel: {
      authTypes: { password: 'Kata sandi', privateKey: 'Private key' },
      title: 'SSH tunnel',
      tooltip:
        'Untuk database di jaringan tertutup. Databasus terhubung ke host SSH di bawah, lalu host tersebut menjangkau database melalui host dan port di atas.',
      host: 'Host SSH',
      hostTooltip:
        'Host database di atas di-resolve oleh host SSH, bukan Databasus. Gunakan 127.0.0.1 jika database berjalan di host SSH itu sendiri.',
      port: 'Port SSH',
      username: 'Nama pengguna SSH',
      usernamePlaceholder: 'Masukkan nama pengguna SSH',
      authType: 'Autentikasi SSH',
      credentials: 'Kredensial SSH',
      password: 'Kata sandi SSH',
      passwordPlaceholder: 'Masukkan kata sandi SSH',
      privateKey: 'Private key SSH',
      keyPassphrase: 'Passphrase kunci',
      keyPassphrasePlaceholder: 'Hanya untuk kunci terenkripsi',
    },
    notifiers: {
      title: 'Kanal notifikasi',
      description:
        'Kanal notifikasi adalah tujuan pengiriman notifikasi (email, Slack, Telegram, dll.)',
      multipleHint:
        'Kamu bisa memilih beberapa kanal notifikasi. Notifikasi akan dikirim ke semuanya.',
      selectPlaceholder: 'Pilih kanal notifikasi',
      createNew: 'Buat kanal notifikasi baru',
      addTitle: 'Tambah kanal notifikasi',
      notifyTo: 'Kirim notifikasi ke',
      empty: 'Belum ada kanal notifikasi yang dikonfigurasi',
    },
    readOnlyUser: {
      dialogTitle: 'Buat pengguna read-only',
      readOnly: {
        checking: 'Memeriksa pengguna read-only...',
        title: 'Buat pengguna read-only untuk Databasus?',
        description:
          'Pengguna read-only adalah pengguna {{databaseType}} dengan izin terbatas yang hanya bisa membaca data database, bukan mengubahnya. Ini disarankan untuk backup karena:',
        noWriteCredentials:
          '<bold>Pengguna read-only membuat kita tidak perlu menyimpan kredensial dengan akses tulis sama sekali</bold>. Bahkan dalam skenario peretasan terburuk, tidak ada yang bisa merusak data kamu.',
        create: 'Ya, buat pengguna read-only',
        skipTitle: 'Lewati pembuatan pengguna read-only?',
        skipQuestion: 'Yakin ingin melewati pembuatan pengguna read-only?',
      },
      replicationOnly: {
        checking: 'Memeriksa pengguna khusus replikasi...',
        title: 'Buat pengguna khusus replikasi untuk Databasus?',
        description:
          'Pengguna khusus replikasi adalah pengguna {{databaseType}} dengan izin terbatas yang hanya bisa membaca data database, bukan mengubahnya. Ini disarankan untuk backup karena:',
        noWriteCredentials:
          '<bold>Pengguna khusus replikasi membuat kita tidak perlu menyimpan kredensial dengan akses tulis sama sekali</bold>. Bahkan dalam skenario peretasan terburuk, tidak ada yang bisa merusak data kamu.',
        create: 'Ya, buat pengguna khusus replikasi',
        skipTitle: 'Lewati pembuatan pengguna khusus replikasi?',
        skipQuestion: 'Yakin ingin melewati pembuatan pengguna khusus replikasi?',
      },
      reasons: {
        preventsModifications: 'mencegah perubahan data yang tidak disengaja saat backup',
        leastPrivilege: 'mengikuti prinsip hak akses minimum',
        bestPractice: 'merupakan praktik keamanan yang disarankan',
      },
      securityNote:
        'Databasus menerapkan keamanan tingkat enterprise (<docsLink>baca selengkapnya di sini</docsLink>). Namun, tidak ada perlindungan yang mencakup semua risiko.',
      noWritePrivileges: 'Pengguna saat ini <bold>tidak memiliki hak akses tulis</bold>.',
      noWriteRoles: 'Pengguna saat ini <bold>tidak memiliki role dengan akses tulis</bold>.',
      writePrivileges: 'Pengguna saat ini memiliki hak akses tulis berikut:',
      writeRoles: 'Pengguna saat ini memiliki role dengan akses tulis berikut:',
      expand: '(perluas)',
      collapse: '(ciutkan)',
      skip: 'Lewati',
      skipRisk:
        'Menggunakan pengguna dengan izin penuh untuk backup tidak disarankan dan bisa menimbulkan risiko keamanan. Databasus sangat menyarankan agar langkah ini tidak dilewati.',
      skipAdvice:
        'Perlindungan 100% tidak pernah mungkin. Lebih baik tetap aman meskipun risiko peretasan penuh hanya 0,01%. Karena itu, pilih cara aman dengan pengguna read-only.',
      acceptRisks: 'Ya, saya menerima risikonya',
      continueSecurely: 'Lanjutkan dengan cara aman',
      forcedWalRotationUnavailable:
        'Sumber ini tidak akan memberikan EXECUTE pada pg_switch_wal() kepada pengguna baru. WAL streaming berkelanjutan membutuhkannya agar titik pemulihan tetap dekat dengan waktu sekarang. Backup penuh dan inkremental tetap berfungsi normal dengan kredensial ini.',
    },
    transfer: {
      title: 'Pindahkan database ke ruang kerja lain',
      targetWorkspace: 'Ruang kerja tujuan',
      selectWorkspace: 'Pilih ruang kerja',
      storage: {
        title: 'Penyimpanan',
        transferExisting: 'Pindahkan bersama penyimpanan saat ini',
        selectFromTarget: 'Pilih penyimpanan dari ruang kerja tujuan',
        blocked:
          'Database yang memakai penyimpanan ini: {{count}}. Pemindahan diblokir karena database lain bergantung padanya.',
        canTransfer: 'Penyimpanan bisa dipindahkan',
        selectPlaceholder: 'Pilih penyimpanan',
        createNew: 'Buat penyimpanan baru',
        addTitle: 'Tambah penyimpanan',
        description:
          'Penyimpanan adalah tempat menyimpan backup (disk lokal, S3, Google Drive, dll.)',
      },
      notifiers: {
        title: 'Kanal notifikasi (opsional)',
        transferWithDatabase: 'Pindahkan kanal notifikasi bersama database',
        selectFromTarget: 'Pilih kanal notifikasi dari ruang kerja tujuan',
        willTransfer: 'Akan dipindahkan:',
        willNotTransfer: 'TIDAK akan dipindahkan (dipakai database lain):',
        usedByDatabases: '{{name}} (dipakai oleh database: {{count}})',
        noneTransferred:
          'Tidak ada kanal notifikasi yang akan dipindahkan. Kamu bisa memilih kanal notifikasi dari ruang kerja tujuan setelah pemindahan.',
        selectPlaceholder: 'Pilih kanal notifikasi (opsional)',
      },
      submit: 'Pindahkan',
      transferred: {
        title: 'Database berhasil dipindahkan!',
        description: '"{{name}}" sudah dipindahkan ke ruang kerja baru',
      },
    },
    connectionString: {
      errors: {
        empty: 'Connection string kosong',
        unrecognizedFormat: 'Format connection string tidak dikenali',
        hostMissing: 'Host tidak ada dalam connection string',
        usernameMissing: 'Nama pengguna tidak ada dalam connection string',
        passwordMissing: 'Kata sandi tidak ada dalam connection string',
        databaseMissing: 'Nama database tidak ada dalam connection string',
        parseFailed: 'Gagal membaca connection string',
        jdbc: {
          invalidFormat:
            'Format connection string JDBC tidak valid. Format yang diharapkan: {{expectedFormat}}',
          queryParametersMissing:
            'Connection string JDBC tidak memiliki parameter query (user dan password)',
          usernameMissing: 'Nama pengguna (parameter user) tidak ada dalam connection string JDBC',
          passwordMissing: 'Parameter password tidak ada dalam connection string JDBC',
          parseFailed: 'Gagal membaca connection string JDBC',
        },
        keyValue: {
          hostMissing: 'Host tidak ada dalam connection string. Gunakan host=hostname',
          usernameMissing: 'Nama pengguna tidak ada dalam connection string. Gunakan user=username',
          passwordMissing:
            'Kata sandi tidak ada dalam connection string. Gunakan password=yourpassword',
          databaseMissing:
            'Nama database tidak ada dalam connection string. Gunakan database=database',
          parseFailed: 'Gagal membaca connection string key-value',
        },
        libpq: {
          databaseMissing:
            'Nama database tidak ada dalam connection string. Gunakan dbname=database',
          parseFailed: 'Gagal membaca connection string libpq',
        },
      },
    },
    physicalConnectionErrors: {
      grantAsSuperuserNote:
        'Berikan hak akses dengan perintah di bawah (jalankan sebagai superuser).',
      applyAndRestartNote: 'Terapkan perubahan di bawah, lalu restart PostgreSQL.',
      pgHbaNoEntry: {
        title: 'Replikasi dari host ini tidak diizinkan',
        summary:
          'PostgreSQL tidak memiliki entri pg_hba.conf yang mengizinkan koneksi replikasi dari Databasus. Backup fisik memakai streaming melalui replikasi yang membutuhkan aturan "host replication". Aturan "host all" biasa tidak mencakupnya.',
        addLineNote:
          'Tambahkan baris di bawah ke pg_hba.conf. "all" pertama mencocokkan semua pengguna dan yang kedua semua host. Untuk membatasi akses, gunakan role replikasi tertentu dan/atau rentang CIDR host.',
        reloadNote: 'Setelah memperbarui pg_hba.conf, <bold>reload PG</bold>.',
        managedNote:
          'Pada PostgreSQL terkelola (RDS / Azure / GCP), pg_hba.conf tidak bisa diedit. Aktifkan akses eksternal atau replikasi di konsol penyedia.',
      },
      badCredentials: {
        title: 'Nama pengguna atau kata sandi salah',
        summary:
          'PostgreSQL menolak kredensial. Periksa kembali nama pengguna dan kata sandi database ini.',
      },
      noReplicationPrivilege: {
        title: 'Pengguna tidak bisa menjalankan replikasi',
        summary:
          'Pengguna berhasil terhubung, tetapi tidak memiliki hak akses REPLICATION yang dibutuhkan backup fisik.',
        managedNote:
          'Pada AWS RDS, gunakan GRANT rds_replication TO {{username}}; pada Azure / GCP, aktifkan replikasi untuk role di konsol penyedia.',
      },
      noWalSwitchPrivilege: {
        title: 'Pengguna tidak bisa memaksa pergantian segmen WAL',
        summary:
          'WAL streaming berkelanjutan hanya mengunggah segmen setelah sumber menutupnya. Pada database yang jarang menerima penulisan, WAL terbaru tetap di sumber sampai segmen 16 MB penuh. Pergantian paksa secara berkala menjaga titik pemulihan di antara backup tetap dekat dengan waktu sekarang dan membutuhkan EXECUTE pada pg_switch_wal(). Backup penuh dan inkremental tidak membutuhkannya: keduanya memulihkan keadaan saat setiap backup selesai dan tidak memutar ulang WAL arsip.',
        switchBackupTypeNote:
          'Atau ubah jenis backup menjadi <bold>Penuh + inkremental</bold>, yang tidak memutar ulang WAL arsip.',
        managedNote:
          'Pada PostgreSQL terkelola (RDS / Azure / GCP), fungsi ini dimiliki superuser platform dan role pelanggan tidak bisa diberi EXECUTE untuk fungsi tersebut. Jadi, WAL streaming berkelanjutan tidak tersedia di sana. Gunakan backup penuh + inkremental.',
      },
      walLevelInvalid: {
        title: 'wal_level terlalu rendah',
        summary:
          'wal_level harus "replica" atau "logical" untuk backup fisik. Nilainya saat ini lebih rendah.',
        applyNote:
          'Terapkan perubahan di bawah, lalu restart PostgreSQL (wal_level baru berlaku setelah restart).',
        managedNote: 'Pada PostgreSQL terkelola, atur wal_level di parameter group penyedia.',
      },
      noWalSenders: {
        title: 'Tidak ada proses pengirim WAL yang tersedia',
        summary:
          'max_wal_senders bernilai 0 sehingga PostgreSQL tidak bisa melakukan streaming WAL untuk backup.',
        managedNote: 'Pada PostgreSQL terkelola, atur max_wal_senders di parameter group penyedia.',
      },
      noReplicationSlots: {
        title: 'Tidak ada slot replikasi yang tersedia',
        summary:
          'max_replication_slots bernilai 0 sehingga PostgreSQL tidak bisa mengalokasikan slot untuk backup.',
        managedNote:
          'Pada PostgreSQL terkelola, atur max_replication_slots di parameter group penyedia.',
      },
      walSummaryDisabled: {
        title: 'Peringkasan WAL nonaktif',
        summary:
          'summarize_wal harus aktif untuk backup inkremental. Saat ini nonaktif di server ini.',
        applyNote: 'Terapkan perubahan di bawah (tidak perlu restart).',
        thenNote: 'lalu',
        managedNote: 'Pada PostgreSQL terkelola, atur summarize_wal di parameter group penyedia.',
      },
      customTablespaces: {
        title: 'Tablespace khusus tidak didukung',
        summary:
          'Cluster ini memiliki tablespace di luar pg_default / pg_global yang tidak bisa di-stream oleh backup fisik. Hapus tablespace khusus atau ubah database ini ke backup logis.',
      },
      systemIdentifierMismatch: {
        title: 'Ini adalah cluster yang berbeda',
        summary:
          'Cluster di alamat ini memiliki system identifier yang berbeda dari cluster yang digunakan saat backup ini disiapkan. Arahkan Databasus ke cluster asli atau buat entri database baru.',
      },
      connectionFailed: {
        title: 'Tidak dapat terhubung',
        summary:
          'Databasus tidak dapat membuka koneksi replikasi. Periksa host, port, mode SSL, dan pastikan firewall tidak memblokir koneksi.',
      },
    },
  },
  restores: {
    list: { empty: 'Belum ada restore' },
    targetDatabase: {
      select: 'Pilih database tujuan restore',
      description:
        'Masukkan informasi database tujuan restore backup. <underline>Database kosong untuk restore harus dibuat terlebih dahulu</underline>. Saat restore, semua data yang ada akan dihapus',
      notInUseHint:
        'Pastikan database sedang tidak digunakan (biasanya kamu tidak ingin melakukan restore ke database yang sama dengan sumber backup)',
      submit: 'Restore ke database ini',
    },
    card: {
      startedAt: 'Dimulai pada',
      duration: 'Durasi',
      errorDetailsTooltip: 'Klik untuk melihat detail error',
      cancelTooltip: 'Batalkan restore',
      expectedDurationHint:
        'Restore biasanya membutuhkan waktu 3x-5x lebih lama daripada backup (bisa lebih singkat atau lebih lama tergantung jenis data)',
      expectedDurationLimit:
        'Jadi, proses ini diperkirakan membutuhkan waktu hingga {{duration}} (biasanya jauh lebih cepat)',
    },
    errorDetails: {
      title: 'Detail error restore',
      copied: 'Pesan error disalin ke clipboard',
      excludeExtensionsTip:
        '<bold>💡 Tip:</bold> Error ini biasanya terjadi saat restore ke layanan PostgreSQL terkelola (seperti Yandex Cloud, AWS RDS, atau sejenisnya). Coba aktifkan <bold>"Kecualikan extension"</bold> di Pengaturan lanjutan sebelum restore.',
    },
    cancelConfirmation: {
      warning:
        '<bold>⚠️ Peringatan:</bold> Membatalkan restore ini kemungkinan akan membuat database rusak atau tidak lengkap. Kamu perlu membuat ulang database sebelum mencoba restore lagi.',
      question: 'Yakin ingin membatalkan?',
      confirm: 'Ya, batalkan restore',
    },
  },
  healthcheck: {
    attempts: {
      title: 'Riwayat pemeriksaan ketersediaan',
      period: 'Periode',
      periods: {
        today: 'Hari ini',
        sevenDays: '7 hari',
        thirtyDays: '30 hari',
        allTime: 'Sepanjang waktu',
      },
      empty: 'Belum ada data',
    },
    config: {
      enable: 'Aktifkan pemeriksaan ketersediaan',
      isEnabled: 'Pemeriksaan ketersediaan aktif',
      notifyWhenUnavailable: 'Kirim notifikasi saat tidak tersedia',
      checkIntervalMinutes: 'Interval pemeriksaan (menit)',
      attemptsBeforeDown: 'Percobaan sebelum dianggap tidak tersedia',
      storeAttemptsDays: 'Simpan riwayat (hari)',
      tooltips: {
        enable: 'Aktifkan atau nonaktifkan pemantauan ketersediaan database ini',
        notifyWhenUnavailable: 'Kirim notifikasi saat database menjadi tidak tersedia',
        checkIntervalMinutes: 'Seberapa sering ketersediaan database diperiksa (dalam menit)',
        attemptsBeforeDown: 'Jumlah pemeriksaan gagal sebelum database ditandai tidak tersedia',
        storeAttemptsDays: 'Jumlah hari penyimpanan riwayat pemeriksaan ketersediaan',
      },
    },
  },
  workspaces: {
    roles: { owner: 'Pemilik', admin: 'Admin', member: 'Anggota', viewer: 'Pengamat' },
    fields: { name: 'Nama ruang kerja', namePlaceholder: 'Masukkan nama ruang kerja' },
    create: {
      title: 'Buat ruang kerja',
      submit: 'Buat ruang kerja',
      defaultName: 'Ruang kerja saya',
      description:
        'Ruang kerja adalah tempat mengelompokkan:<lineBreak/>- database;<lineBreak/>- penyimpanan (disk lokal, S3, Google Drive, dll.);<lineBreak/>- kanal notifikasi (email, Slack, Telegram, dll.);<lineBreak/>- kontrol akses (jika bekerja dalam tim);',
      nameRequired: 'Masukkan nama ruang kerja',
      created: 'Ruang kerja berhasil dibuat',
      permissionDenied: {
        title: 'Akses ditolak',
        description:
          'Kamu tidak memiliki izin untuk membuat ruang kerja. Minta admin untuk membuatkannya.',
      },
    },
    settings: {
      title: 'Pengaturan ruang kerja',
      readOnly: 'Kamu tidak memiliki izin untuk mengubah pengaturan ini',
      nameRequired: 'Nama ruang kerja wajib diisi',
      saving: 'Menyimpan...',
      saveChanges: 'Simpan perubahan',
      reset: 'Reset',
      updated: 'Informasi dasar berhasil diperbarui',
    },
    delete: {
      dangerZone: 'Zona berbahaya',
      title: 'Hapus ruang kerja ini',
      description:
        'Ruang kerja yang dihapus tidak bisa dikembalikan. Semua data dan sumber daya yang terkait dengan ruang kerja ini akan dihapus permanen.',
      submit: 'Hapus ruang kerja',
      deleting: 'Menghapus...',
      notFound: 'Ruang kerja tidak ditemukan',
      noPermission: 'Kamu tidak memiliki izin untuk menghapus ruang kerja ini',
      hasDatabases:
        'Ruang kerja tidak bisa dihapus. Hapus semua database terlebih dahulu. Database ditemukan: {{count}}.',
      deleted: 'Ruang kerja berhasil dihapus',
      confirmation: {
        title: 'Hapus ruang kerja',
        question: 'Yakin ingin menghapus ruang kerja <workspaceName/>?',
        warning:
          '<bold>Tindakan ini tidak bisa dibatalkan.</bold> Semua data dan sumber daya terkait akan dihapus permanen.',
        submit: 'Hapus ruang kerja',
      },
    },
    members: {
      title: 'Pengguna',
      count: 'Anggota: {{count}}',
      empty: 'Tidak ada anggota',
      emptyHint: 'Klik "Tambah anggota" untuk memulai',
      fields: {
        member: 'Anggota',
        role: 'Role',
        joined: 'Bergabung',
        actions: 'Tindakan',
        email: 'Alamat email',
        emailPlaceholder: 'Masukkan alamat email',
      },
      add: {
        button: 'Tambah anggota',
        title: 'Tambah anggota',
        submit: 'Tambah anggota',
        emailRequired: 'Email wajib diisi',
        hint: 'Jika pengguna sudah terdaftar, pengguna akan langsung ditambahkan. Jika belum, undangan akan dikirim.',
        added: 'Anggota berhasil ditambahkan',
      },
      invited: {
        title: 'Pengguna diundang',
        sentTo: 'Undangan dikirim ke {{email}}',
        description:
          'Pengguna belum terdaftar di sistem, tetapi sudah diundang ke ruang kerja. Setelah mendaftar dengan email tersebut, pengguna otomatis menjadi anggota ruang kerja.',
      },
      roleChanged: 'Role anggota berhasil diperbarui',
      remove: {
        tooltip: 'Hapus anggota',
        title: 'Hapus anggota',
        question: 'Yakin ingin menghapus "{{email}}" dari ruang kerja ini?',
        removed: 'Anggota "{{email}}" berhasil dihapus',
      },
    },
    transferOwnership: {
      button: 'Alihkan kepemilikan',
      title: 'Alihkan kepemilikan ruang kerja',
      submit: 'Alihkan kepemilikan',
      warning:
        '<bold>Peringatan:</bold> Tindakan ini tidak bisa dibatalkan. Kamu akan kehilangan kepemilikan ruang kerja ini dan pemilik baru akan memiliki kendali penuh.',
      noEligibleMembers:
        'Tidak ada anggota yang bisa menerima kepemilikan. Ruang kerja harus memiliki setidaknya satu anggota lain untuk mengalihkan kepemilikan.',
      newOwner: 'Pilih pemilik baru',
      newOwnerPlaceholder: 'Pilih anggota penerima kepemilikan',
      newOwnerHint: 'Anggota terpilih akan menjadi pemilik ruang kerja',
      memberRequired: 'Pilih anggota penerima kepemilikan',
      memberNotFound: 'Anggota terpilih tidak ditemukan',
      transferred: 'Kepemilikan berhasil dialihkan',
    },
    auditLogs: { empty: 'Tidak ada log audit untuk ruang kerja ini.' },
  },
  users: {
    roles: { admin: 'Admin', member: 'Anggota' },
    fields: {
      name: 'Nama',
      email: 'Email',
      yourName: 'Nama kamu',
      yourEmail: 'Email kamu',
      confirmPassword: 'Konfirmasi kata sandi',
      newPassword: 'Kata sandi baru',
    },
    validation: {
      nameRequired: 'Nama wajib diisi',
      emailRequired: 'Email wajib diisi',
      passwordTooShort: 'Kata sandi harus terdiri dari setidaknya {{minLength}} karakter',
      passwordsDoNotMatch: 'Kata sandi tidak cocok',
    },
    oauth: {
      divider: 'atau lanjutkan',
      continueWithGithub: 'Lanjutkan dengan GitHub',
      continueWithGoogle: 'Lanjutkan dengan Google',
      invalidConfiguration: 'Konfigurasi OAuth tidak valid',
    },
    signIn: {
      title: 'Masuk',
      submit: 'Masuk',
      noAccount: 'Belum punya akun? <signUpLink>Daftar</signUpLink>',
      forgotPassword: 'Lupa kata sandi?',
    },
    signInCode: {
      title: 'Masukkan kode',
      description:
        'Kami mengirim kode enam digit ke {{email}}. Kode berlaku selama 10 menit sejak dikirim.',
      code: 'Kode masuk',
      submit: 'Masuk',
      resend: 'Kirim kode lagi',
      backToSignIn: 'Kembali ke halaman masuk',
      invalidCode: 'Kode harus terdiri dari 6 digit',
      codeResent: 'Kode baru sedang dikirim. Kode sebelumnya sudah tidak berlaku.',
    },
    signUp: {
      title: 'Daftar',
      namePlaceholder: 'Nama lengkap',
      submit: 'Daftar',
      administersInstance:
        'Ini adalah akun pertama di instance ini dan akan menjadi admin instance.',
      hasAccount: 'Sudah punya akun? <signInLink>Masuk</signInLink>',
    },
    requestPasswordReset: {
      title: 'Reset kata sandi',
      description: 'Masukkan alamat email untuk menerima kode reset.',
      submit: 'Kirim kode reset',
      codeSent: 'Jika email terdaftar, kode reset sudah dikirim',
      rememberPassword: 'Ingat kata sandi? <signInLink>Masuk</signInLink>',
    },
    resetPassword: {
      title: 'Reset kata sandi',
      description: 'Masukkan kode yang dikirim ke email dan kata sandi baru.',
      code: 'Kode reset',
      confirmPassword: 'Konfirmasi kata sandi',
      submit: 'Reset kata sandi',
      invalidCode: 'Kode harus terdiri dari 6 digit',
      succeeded: 'Kata sandi berhasil direset! Mengarahkan ke halaman masuk...',
      noCode: 'Belum menerima kode? <requestCodeLink>Minta kode baru</requestCodeLink>',
      backToSignIn: 'Kembali ke halaman masuk',
    },
    profile: {
      title: 'Profil',
      information: 'Informasi profil',
      userId: 'ID pengguna',
      namePlaceholder: 'Masukkan nama',
      emailPlaceholder: 'Masukkan email',
      role: 'Role',
      saveChanges: 'Simpan perubahan',
      noChanges: 'Tidak ada perubahan untuk disimpan',
      updated: 'Profil berhasil diperbarui',
      logout: 'Keluar',
      changePassword: {
        title: 'Ubah kata sandi',
        newPasswordPlaceholder: 'Masukkan kata sandi baru',
        confirmPassword: 'Konfirmasi kata sandi baru',
        confirmPasswordPlaceholder: 'Konfirmasi kata sandi baru',
        passwordsDoNotMatch: 'Kata sandi baru tidak cocok',
        changing: 'Mengubah kata sandi...',
        submit: 'Ubah kata sandi',
        signedInWithNewPassword: 'Berhasil masuk dengan kata sandi baru',
      },
    },
    list: {
      title: 'Pengguna Databasus',
      loadedOfTotal: 'Pengguna: {{loaded}} dari {{total}}',
      searchPlaceholder: 'Cari berdasarkan email atau nama...',
      empty: 'Pengguna tidak ditemukan.',
      allLoaded: 'Semua pengguna dimuat (total {{count}})',
      columns: {
        user: 'Pengguna',
        systemRole: 'Role sistem',
        isActive: 'Aktif?',
        created: 'Dibuat',
      },
      roleLabel: 'Role:',
      activeLabel: 'Aktif:',
      viewAuditLogs: 'Lihat log audit',
      activated: 'Pengguna berhasil diaktifkan',
      deactivated: 'Pengguna berhasil dinonaktifkan',
      roleChanged: 'Role pengguna berhasil diubah',
    },
    auditLogs: { title: 'Log audit pengguna', empty: 'Tidak ada log audit untuk pengguna ini.' },
  },
  verification: {
    triggers: { manual: 'Manual', scheduled: 'Terjadwal' },
    notificationTypes: {
      verificationSuccess: 'Verifikasi berhasil',
      verificationFailed: 'Verifikasi gagal',
    },
    config: {
      scheduledVerification: 'Verifikasi terjadwal',
      scheduledVerificationTooltip:
        'Lakukan restore backup terbaru secara berkala ke container Postgres sementara untuk memastikan backup bisa di-restore. <docsLink>Baca cara kerjanya di sini</docsLink>',
      interval: 'Interval verifikasi',
      afterBackup: 'Setelah backup',
      afterBackupHint:
        'Berjalan otomatis setelah setiap backup berhasil. Verifikasi yang masih menunggu di antrean dibatalkan agar antrean tidak terus bertambah saat backup lebih cepat daripada verifikasi.',
      weekday: 'Hari verifikasi',
      dayOfMonth: 'Tanggal verifikasi',
      timeOfDay: 'Waktu verifikasi',
      cron: {
        expression: 'Ekspresi cron (UTC)',
        format: 'Format cron: menit jam tanggal bulan hari (UTC)',
        examples: 'Contoh:',
        exampleEverySunday: '- {{expression}} - Setiap Minggu pukul 04.00 UTC',
        exampleEverySixHours: '- {{expression}} - Setiap 6 jam',
        nextRun: 'Jadwal berikutnya {{dateTime}}<lineBreak/>({{relativeTime}})',
        invalid: 'Ekspresi cron tidak valid',
      },
      notifications: 'Notifikasi',
      noNotifications: 'Tidak ada',
    },
    runs: {
      title: 'Verifikasi restore',
      description:
        'Setiap baris adalah satu percobaan restore backup database ini ke salinan sementara',
      noAgentsWarning:
        'Belum ada agen verifikasi yang terdaftar. Tambahkan di tab Pengaturan Databasus, bagian "Agen verifikasi". <docsLink>Baca selengkapnya</docsLink>',
      empty: 'Belum ada verifikasi restore',
      allLoaded: 'Semua verifikasi restore dimuat (total {{total}})',
      attempt: 'percobaan {{attempt}}',
      restoreDuration: 'restore {{duration}}',
      durationTooltip: 'Waktu restore + verifikasi yang dilaporkan agen.',
      fields: {
        createdAt: 'Dibuat pada',
        trigger: 'Pemicu',
        duration: 'Durasi',
        actions: 'Tindakan',
      },
      actions: { cancel: 'Batalkan verifikasi', viewDetails: 'Lihat detail' },
      details: {
        title: 'Detail verifikasi restore',
        loadFailed: 'Tidak dapat memuat detail verifikasi restore.',
        cancellationReason: 'Alasan pembatalan',
        failure: 'Kegagalan',
        sections: { status: 'Status', timeline: 'Linimasa', results: 'Hasil dan diagnostik' },
        fields: {
          attempt: 'Percobaan',
          startedAt: 'Dimulai pada',
          finishedAt: 'Selesai pada',
          restoreDuration: 'Durasi restore',
          verifyDuration: 'Durasi verifikasi',
          restoredDbSize: 'Ukuran database hasil restore',
          schemas: 'Schema',
          tables: 'Tabel',
          pgRestoreExitCode: 'Exit code pg_restore',
        },
        tableStats: {
          title: 'Jumlah baris per tabel',
          empty: 'Tidak ada statistik per tabel yang dilaporkan.',
          schema: 'Schema',
          table: 'Tabel',
          rows: 'Baris',
        },
      },
    },
    agents: {
      title: 'Agen verifikasi',
      description:
        'Agen yang menjalankan verifikasi restore untuk memastikan backup bisa di-restore (<docsLink>baca selengkapnya</docsLink>)',
      empty: 'Belum ada agen yang terdaftar.',
      columns: { capacity: 'Kapasitas', created: 'Dibuat' },
      capacity: {
        cpu: '{{count}} CPU',
        ram: '{{count}} GB RAM',
        disk: '{{count}} GB disk',
        jobs: 'tugas: {{count}}',
        notReported: 'belum dilaporkan',
      },
      actions: { viewInstallCommands: 'Lihat perintah instalasi', rotateToken: 'Rotasi token' },
      deleteConfirmation: 'Hapus agen ini?',
      deleted: 'Agen "{{name}}" dihapus',
      copyFailed: 'Gagal menyalin',
      agentId: 'ID agen:',
      create: {
        title: 'Buat agen verifikasi',
        tokenNotice: 'Token akan dibuat dan hanya ditampilkan satu kali di layar berikutnya.',
        namePlaceholder: 'Nama agen',
        nameRequired: 'Nama wajib diisi',
      },
      rotate: {
        title: 'Rotasi token',
        description:
          'Rotasi token untuk <agentName/> langsung menonaktifkan token saat ini. Worker yang masih memakai token lama akan ditolak pada heartbeat berikutnya.',
      },
      token: {
        title: 'Token agen',
        tokenFor: 'Token untuk <agentName/>:',
        shownOnce:
          'Hanya ditampilkan satu kali. Simpan dengan aman karena token tidak bisa dilihat lagi.',
        confirmSaved: 'Saya sudah menyimpan token',
      },
      install: {
        title: 'Perintah instalasi',
        description:
          'Perintah instalasi untuk <agentName/>. Ganti <tokenPlaceholder/> dengan token yang disimpan saat membuat agen atau melakukan rotasi token terakhir.',
        architecture: 'Arsitektur',
        stepInstall: 'Langkah 1 - Instal',
        stepLaunch: 'Langkah 2 - Jalankan',
        capacityDefaults:
          'Nilai kapasitas di bawah adalah nilai awal. Sesuaikan <code>--max-cpu</code>, <code>--max-ram-mb</code>, <code>--max-disk-gb</code>, dan <code>--max-concurrent-jobs</code> dengan mesin tempat agen berjalan.',
        afterInstallation: 'Setelah instalasi',
        runsInBackground: 'Agen berjalan di latar belakang setelah <code>start</code>',
        checkStatus: 'Periksa status: <code>./verification-agent status</code>',
        viewLogs: 'Lihat log: <code>databasus-verification.log</code> di direktori kerja',
        stop: 'Hentikan agen: <code>./verification-agent stop</code>',
      },
    },
  },
  notifiers: {
    advancedSettings: 'Pengaturan lanjutan',
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
      all: 'Semua',
      backupSuccess: 'Backup berhasil',
      backupFailed: 'Backup gagal',
      healthcheckSuccess: 'Pemeriksaan ketersediaan berhasil',
      healthcheckFailed: 'Pemeriksaan ketersediaan gagal',
      verificationSuccess: 'Verifikasi berhasil',
      verificationFailed: 'Verifikasi gagal',
    },
    mattermostDeliveryModes: { webhook: 'Incoming webhook', bot: 'Akun bot' },
    list: {
      addNotifier: 'Tambah kanal notifikasi',
      searchPlaceholder: 'Cari kanal notifikasi',
      noSearchResults: 'Tidak ada kanal notifikasi yang cocok dengan "{{query}}"',
      description:
        'Kanal notifikasi adalah tujuan pengiriman notifikasi (email, Slack, Telegram, dll.)',
      backToList: '← Kembali ke kanal notifikasi',
    },
    card: {
      notifyTo: {
        email: 'Kirim notifikasi ke Email',
        telegram: 'Kirim notifikasi ke Telegram',
        webhook: 'Kirim notifikasi ke Webhook',
        slack: 'Kirim notifikasi ke Slack',
        discord: 'Kirim notifikasi ke Discord',
        teams: 'Kirim notifikasi ke Teams',
        mattermost: 'Kirim notifikasi ke Mattermost',
      },
      hasSendError: 'Ada error pengiriman',
    },
    create: { title: 'Tambah kanal notifikasi', namePlaceholder: 'Chat dengan saya' },
    config: {
      namePlaceholder: 'Masukkan nama...',
      settingsTitle: 'Pengaturan kanal notifikasi',
      sendError: {
        title: 'Error pengiriman',
        errorLabel: 'Error:',
        clearHint: 'Untuk menghapus error ini, pilih salah satu:',
        clearBySendingTest:
          '- kirim notifikasi uji melalui tombol di bawah (meskipun pengaturan sudah diperbarui);',
        clearByNextNotification: '- tunggu sampai notifikasi berikutnya terkirim tanpa error;',
      },
      removeConfirmation:
        'Yakin ingin menghapus kanal notifikasi ini? Tindakan ini tidak bisa dibatalkan.',
      removeBlockedByDatabases:
        'Kanal notifikasi dipakai beberapa database. Hapus kanal notifikasi dari database tersebut terlebih dahulu.',
    },
    testNotification: {
      send: 'Kirim notifikasi uji',
      sent: { title: 'Notifikasi uji terkirim!', description: 'Notifikasi uji berhasil dikirim' },
    },
    transfer: {
      title: 'Pindahkan kanal notifikasi ke ruang kerja lain',
      blockedByDatabases:
        'Kanal notifikasi ini dipakai beberapa database. Pindahkan atau hapus database terkait terlebih dahulu.',
      description: 'Pilih ruang kerja tujuan untuk kanal notifikasi ini.',
      targetWorkspace: 'Ruang kerja tujuan',
      selectWorkspace: 'Pilih ruang kerja',
      submit: 'Pindahkan',
    },
    fields: {
      botToken: 'Token bot',
      targetChatId: 'ID chat tujuan',
      webhookUrl: 'URL webhook',
      channelId: 'ID kanal',
      skipTlsVerify: 'Lewati verifikasi TLS',
      skipTls: 'Lewati TLS',
    },
    email: {
      targetEmail: 'Email tujuan',
      targetEmailTooltip: 'Alamat email untuk menerima pesan',
      smtpHost: 'Host SMTP',
      smtpPort: 'Port SMTP',
      smtpUser: 'Pengguna SMTP',
      smtpPassword: 'Kata sandi SMTP',
      smtpPasswordPlaceholder: 'kata sandi',
      from: 'Pengirim',
      fromTooltip:
        'Opsional. Alamat pengirim, dengan atau tanpa nama: Acme Backups <noreply@example.com>. Jika kosong, pengguna SMTP dipakai jika berupa alamat email. Jika bukan, noreply@ diikuti host SMTP akan dipakai',
      fromAuto: '(otomatis)',
      skipTlsTooltip:
        'Lewati verifikasi sertifikat TLS. Aktifkan jika server SMTP menggunakan sertifikat self-signed. Peringatan: ini mengurangi keamanan.',
      security: 'Keamanan koneksi',
      securityTooltip:
        'TLS mengenkripsi koneksi sejak awal dan biasanya memakai port 465. STARTTLS mengenkripsi koneksi setelah dibuka dan gagal jika server tidak menyediakannya. Tanpa enkripsi, kata sandi dan pesan dikirim tanpa perlindungan. Gunakan hanya untuk relay tepercaya tanpa STARTTLS.',
      securityModes: { tls: 'TLS', starttls: 'STARTTLS', none: 'Tidak ada (tanpa enkripsi)' },
      heloName: 'Nama sapaan',
      heloNamePlaceholder: 'Otomatis',
      heloNameTooltip:
        'Nama yang dipakai Databasus untuk memperkenalkan diri ke server email (EHLO). Jika kosong, host dari DATABASUS_URL atau nama host mesin akan dipakai.',
    },
    telegram: {
      botTokenHelpLink: 'Bagaimana cara mendapatkan token API bot Telegram?',
      targetChatIdTooltip: 'Chat tujuan untuk menerima pesan (bisa chat pribadi atau grup)',
      chatIdHelp: {
        toggle: 'Bagaimana cara mendapatkan ID chat Telegram?',
        privateChat:
          'Untuk mendapatkan ID chat, kirim pesan ke <botLink>@getmyid_bot</botLink> di Telegram. <underline>Pastikan kamu sudah memulai chat dengan bot</underline>',
        group:
          'Untuk mendapatkan ID chat grup, tambahkan bot kamu bersama <botLink>@getmyid_bot</botLink> ke grup dan kirim /start (ID chat akan ditampilkan)',
      },
      botApiUrl: 'URL Bot API',
      botApiUrlTooltip:
        'Opsional. Server Telegram Bot API self-hosted atau reverse proxy di depan API Telegram. Jika kosong, API resmi Telegram dipakai. Server ini menerima token bot. Bot harus keluar dari API resmi sebelum bisa dipakai dengan server self-hosted',
      useProxy: 'Gunakan proxy',
      useProxyTooltip: 'Gunakan proxy untuk permintaan API Telegram',
      proxy: 'Proxy',
      proxyUrl: 'URL proxy',
      proxyUrlTooltip:
        'Mendukung http, https, socks5, socks5h. Bisa menyertakan nama pengguna dan kata sandi',
      sendToGroupTopic: 'Kirim ke topik grup',
      sendToGroupTopicTooltip: 'Aktifkan untuk mengirim pesan ke thread tertentu dalam chat grup',
      threadId: 'ID thread',
      threadIdTooltip: 'ID thread tujuan pengiriman pesan',
      topicId: 'ID topik',
      threadIdHelp: {
        steps:
          'Untuk mendapatkan ID thread, buka thread di grup Telegram, ketuk nama thread di bagian atas, lalu ketuk “Thread Info”. Salin tautan thread dan ambil angka terakhir dari URL.',
        example:
          '<bold>Contoh:</bold> Jika tautan thread adalah <code>https://t.me/c/2831948048/3</code>, ID thread adalah <code>3</code>',
        note: '<bold>Catatan:</bold> Thread hanya berfungsi di chat grup, bukan chat pribadi.',
      },
    },
    webhook: {
      method: 'Metode',
      sendOn: 'Kirim saat',
      sendOnPlaceholder: 'Pilih jenis notifikasi',
      customHeaders: 'Header khusus',
      customHeadersTooltip:
        'Tambahkan header HTTP khusus ke permintaan webhook (misalnya Authorization, X-API-Key)',
      savedHeadersHidden: '*Header tersimpan disembunyikan demi keamanan',
      headers: 'Header',
      headerNamePlaceholder: 'Nama header',
      headerValuePlaceholder: 'Nilai header',
      addHeader: 'Tambah header',
      hiddenHeaderValue: '(tersembunyi)',
      bodyTemplate: {
        title: 'Template body',
        headingVariable: '<variable/> - judul notifikasi',
        messageVariable: '<variable/> - pesan notifikasi',
        invalidJsonFormat: 'Format JSON tidak valid',
        invalidJson: 'JSON tidak valid',
      },
      exampleRequest: { title: 'Contoh permintaan', headers: 'Header:' },
    },
    slack: {
      howToConnect: 'Bagaimana cara menghubungkan Slack (mendapatkan token bot dan ID chat)?',
      testFailedHint:
        'Pastikan kanal bersifat publik atau bot sudah ditambahkan ke kanal privat (melalui @invite) atau grup. Untuk pesan langsung, gunakan User ID dari profil Slack.',
    },
    discord: {
      channelWebhookUrl: 'URL webhook kanal',
      howTo: {
        title: 'Cara mendapatkan URL webhook Discord:',
        createChannel: '1. Buat atau pilih kanal Discord',
        openChannelSettings: '2. Buka pengaturan kanal (ikon roda gigi)',
        openIntegrations: '3. Buka Integrations',
        createWebhook: '4. Buat webhook baru',
        copyWebhookUrl: '5. Salin URL webhook',
        privacyNote: 'Catatan: jadikan kanal privat jika diperlukan',
      },
    },
    teams: {
      howToConnect: 'Bagaimana cara menghubungkan Microsoft Teams?',
      powerAutomateUrl: 'URL Power Automate',
      powerAutomateUrlLabel: 'URL Power Automate:',
      powerAutomateUrlTooltip:
        'Endpoint HTTP dari flow Power Automate (When an HTTP request is received)',
      showFullUrl: 'Tampilkan',
      hideFullUrl: 'Sembunyikan',
    },
    mattermost: {
      howToConnect: 'Bagaimana cara menghubungkan Mattermost?',
      connectVia: 'Hubungkan melalui',
      incomingWebhookUrl: 'URL incoming webhook',
      serverUrl: 'URL server',
      channelIdTooltip:
        'ID kanal 26 karakter, bukan nama kanal. Buka kanal, klik namanya, lalu pilih View Info.',
      channel: 'Kanal',
      showOptionalSettings: 'Tampilkan pengaturan opsional',
      hideOptionalSettings: 'Sembunyikan pengaturan opsional',
      channelOverride: 'Kanal pengganti',
      channelOverrideTooltip:
        'Kirim ke kanal ini sebagai pengganti kanal tujuan awal webhook. Mattermost mengabaikannya jika webhook dikunci ke satu kanal.',
      postAsUsername: 'Kirim dengan nama pengguna',
      postAsUsernameTooltip:
        'Membutuhkan Enable integrations to override usernames di konsol sistem Mattermost. Jika tidak aktif, pengaturan ini diabaikan.',
      postAsIconUrl: 'Kirim dengan URL ikon',
      postAsIconUrlTooltip:
        'Membutuhkan Enable integrations to override profile picture icons di konsol sistem Mattermost. Jika tidak aktif, pengaturan ini diabaikan.',
      skipTlsTooltip:
        'Lewati verifikasi sertifikat TLS. Aktifkan jika server Mattermost menggunakan sertifikat self-signed. Peringatan: ini mengurangi keamanan.',
      webhookHowTo: {
        title: 'Cara mendapatkan URL incoming webhook:',
        openIncomingWebhooks: '1. Menu utama - Integrations - Incoming Webhooks',
        addWebhook: '2. Add Incoming Webhook, pilih kanal',
        copyUrl: '3. Salin URL yang dihasilkan',
      },
      botHowTo: {
        title: 'Cara mendapatkan token bot:',
        addBotAccount: '1. Integrations - Bot Accounts - Add Bot Account',
        copyToken: '2. Salin token yang hanya ditampilkan sekali setelah dibuat',
        addBotToChannel: '3. Tambahkan bot ke tim dan kanal',
      },
    },
  },
  backups: {
    encryption: { none: 'Tidak ada', encrypted: 'Aktif' },
    encryptionOptions: { none: 'Tidak ada', encrypted: 'Enkripsi file backup' },
    list: {
      title: 'Backup',
      scheduledBackupsDisabled:
        'Backup terjadwal nonaktif (bisa diaktifkan kembali di konfigurasi backup)',
      empty: 'Belum ada backup',
      allLoaded: 'Semua backup dimuat (total {{count}})',
      clickToSeeErrorDetails: 'Klik untuk melihat detail error',
      errorDetailsTitle: 'Detail error backup',
      restoreTitle: 'Restore dari backup',
      columns: {
        createdAt: 'Dibuat pada',
        created: 'Dibuat',
        size: 'Ukuran',
        duration: 'Durasi',
        actions: 'Tindakan',
      },
      actions: { cancelBackup: 'Batalkan backup', deleteBackup: 'Hapus backup' },
    },
    filters: { allTypes: 'Semua jenis', allStatuses: 'Semua status', before: 'Sebelum' },
    config: {
      backupsEnabled: 'Backup aktif',
      storage: 'Penyimpanan',
      selectStorage: 'Pilih penyimpanan',
      createNewStorage: 'Buat penyimpanan baru',
      addStorageTitle: 'Tambah penyimpanan',
      storageDescription:
        'Penyimpanan adalah tempat menyimpan backup (disk lokal, S3, Google Drive, dll.)',
      storageChangeWarning:
        'Jika penyimpanan diubah, semua backup di penyimpanan ini akan dihapus.',
      storageChangeAcknowledge: 'Saya mengerti',
      encryption: 'Enkripsi',
      notifications: 'Notifikasi',
      noNotifications: 'Tidak ada',
      schedule: {
        weekday: 'Hari',
        dayOfMonth: 'Tanggal',
        cronExpression: 'Ekspresi cron (UTC)',
        timeOfDay: 'Waktu',
        nextRun: 'Jadwal berikutnya {{time}}',
        invalidCron: 'Ekspresi cron tidak valid',
        cronHelp: {
          format: 'Format cron: menit jam tanggal bulan hari (UTC)',
          examples: 'Contoh:',
          dailyAt2am: 'Setiap hari pukul 02.00 UTC',
          every6Hours: 'Setiap 6 jam',
          mondaysAt3am: 'Setiap Senin pukul 03.00 UTC',
          twiceAMonth: 'Tanggal 1 dan 15 pukul 04.30 UTC',
        },
      },
      gfs: {
        hourly: 'Per jam: {{count}}',
        daily: 'Harian: {{count}}',
        weekly: 'Mingguan: {{count}}',
        monthly: 'Bulanan: {{count}}',
        yearly: 'Tahunan: {{count}}',
        notConfigured: 'Belum dikonfigurasi',
      },
    },
    logical: {
      retentionPolicyTypes: {
        gfs: 'GFS (simpan N backup terakhir per jam, harian, mingguan, bulanan, dan tahunan)',
        timePeriod: 'Periode waktu (N hari terakhir)',
        count: 'Jumlah (N backup terakhir)',
      },
      notificationTypes: { backupSuccess: 'Backup berhasil', backupFailed: 'Backup gagal' },
      list: {
        backupNow: 'Backup sekarang',
        makeBackupNow: 'Jalankan backup sekarang',
        encrypted: 'Terenkripsi',
        sizeTooltip:
          'Atas: ukuran file backup terkompresi yang benar-benar disimpan di penyimpanan (lokal, S3, Google Drive, dll.). Bawah (abu-abu): ukuran asli database tanpa kompresi saat backup dibuat. Backup biasanya dikompresi sekitar 5x.',
        databaseSize: '{{size}} (ukuran database)',
        deleteConfirmation: 'Yakin ingin menghapus backup ini?',
        actions: {
          restore: 'Restore dari backup',
          verifyRestore:
            'Verifikasi restore - antrekan agen untuk melakukan restore backup ini ke database sementara dan melaporkan jumlah baris',
          download: {
            postgresql:
              'Unduh file backup. File bisa di-restore secara manual melalui pg_restore (dari format custom)',
            mysql:
              'Unduh file backup. File bisa di-restore secara manual melalui klien mysql (dari dump SQL)',
            mariadb:
              'Unduh file backup. File bisa di-restore secara manual melalui klien mariadb (dari dump SQL)',
            mongodb:
              'Unduh file backup. File bisa di-restore secara manual melalui mongorestore (dari arsip)',
            other: 'Unduh file backup',
          },
        },
        verifyRestore: {
          title: 'Verifikasi restore?',
          description:
            'Agen akan melakukan restore backup ini ke database sementara dan melaporkan jumlah baris. Proses ini bisa memakan waktu tergantung ukuran backup. <docsLink>Bagaimana cara kerjanya?</docsLink>',
          queue: 'Antrekan verifikasi',
          queued: 'Verifikasi restore masuk antrean',
        },
      },
      config: {
        interval: 'Interval backup',
        weekday: 'Hari backup',
        dayOfMonth: 'Tanggal backup',
        timeOfDay: 'Waktu backup',
        encryptionTooltip:
          'Jika backup dienkripsi, file backup di penyimpanan (S3, lokal, dll.) tidak bisa langsung digunakan. Kamu bisa melakukan restore melalui Databasus atau mengunduhnya tanpa enkripsi melalui tombol Unduh.',
        retentionPolicy: 'Kebijakan retensi',
        retention: {
          timePeriodTooltip:
            'Berapa lama backup disimpan. Backup yang lebih lama dari periode ini otomatis dihapus.',
          count: 'Backup terbaru',
          countTooltip:
            'Simpan hanya sejumlah backup terbaru yang ditentukan. Backup lebih lama yang melebihi jumlah ini otomatis dihapus.',
          countValue: 'Backup terbaru: {{count}}',
          showGfsHint: 'Apa itu GFS (Grandfather-Father-Son)?',
          hideGfsHint: 'Sembunyikan',
          gfsHint:
            'Rotasi GFS (Grandfather-Father-Son): simpan N backup terakhir per jam, harian, mingguan, bulanan, dan tahunan. Ini memungkinkan penyimpanan backup jangka panjang tanpa memakai terlalu banyak ruang.',
          gfsTooltip:
            'Rotasi Grandfather-Father-Son: simpan N backup terakhir per jam, harian, mingguan, bulanan, dan tahunan.',
          gfsFields: {
            hourly: 'Backup per jam',
            daily: 'Backup harian',
            weekly: 'Backup mingguan',
            monthly: 'Backup bulanan',
            yearly: 'Backup tahunan',
          },
        },
        advancedSettings: 'Pengaturan lanjutan',
        retryIfFailed: 'Ulangi backup jika gagal',
        retryIfFailedShort: 'Ulangi jika gagal',
        retryIfFailedTooltip:
          'Ulangi backup yang gagal secara otomatis. Backup bisa gagal karena gangguan jaringan, masalah penyimpanan, atau database yang sementara tidak tersedia.',
        maxFailedTries: 'Jumlah maksimum percobaan ulang',
        maxFailedTriesTooltip:
          'Jumlah maksimum percobaan ulang untuk backup yang gagal. Kamu akan menerima notifikasi jika semua percobaan gagal.',
      },
    },
    physical: {
      types: { full: 'Penuh', incremental: 'Inkremental', wal: 'WAL' },
      retentions: {
        chains: 'Rantai',
        fullBackups: 'Backup penuh',
        chainsAndFullBackups: 'Rantai dan backup penuh',
      },
      fullBackupsPolicies: { lastN: 'Jumlah', gfs: 'GFS' },
      notificationTypes: {
        backupSuccess: 'Backup berhasil',
        backupFailed: 'Backup gagal',
        chainBroken: 'Rantai terputus',
        walGap: 'Celah WAL',
      },
      list: {
        totalUsage: 'Total penggunaan: {{size}}',
        backUpNow: 'Backup sekarang',
        fullBackup: 'Backup penuh',
        incrementalBackup: 'Backup inkremental',
        restore: 'Restore',
        restoreFromThisBackup: 'Restore dari backup ini',
        pointInTimeRestoreTitle: 'Restore ke titik waktu tertentu',
        walRange: {
          from: 'Dari',
          to: 'Sampai',
          segments: 'Segmen',
          gap: 'Celah',
          gapBeforeTooltip:
            'Ada segmen WAL yang hilang antara rentang ini dan sebelumnya. Restore ke titik waktu tertentu tidak bisa memutar ulang perubahan melewati celah ini.',
        },
        deleteFullConfirmation:
          'Menghapus backup penuh ini akan menghapus seluruh rantainya (semua backup inkremental yang bergantung padanya). Tindakan ini tidak bisa dibatalkan. Lanjutkan?',
        deleteIncrementalConfirmation:
          'Menghapus backup inkremental ini akan menghapus semua backup inkremental sesudahnya yang bergantung padanya. Tindakan ini tidak bisa dibatalkan. Lanjutkan?',
      },
      config: {
        fullBackupCadence: 'Frekuensi backup penuh',
        incrementalBackupCadence: 'Frekuensi backup inkremental',
        incrementalMoreFrequent:
          'Backup inkremental harus berjalan lebih sering daripada backup penuh.',
        encryptionTooltip:
          'Jika backup dienkripsi, file di penyimpanan tidak bisa langsung digunakan. Kamu bisa melakukan restore melalui Databasus atau mengunduhnya tanpa enkripsi.',
        retention: 'Retensi',
        fullBackupsOnlyRetention:
          'Database ini hanya menyimpan backup penuh, jadi retensi berlaku untuk backup penuh.',
        retentionOptions: {
          chains: 'Simpan N rantai terakhir',
          chainsAndFullBackups: 'Simpan N rantai terakhir dan backup penuh',
        },
        retentionHelp: {
          chain:
            'Rantai terdiri dari satu backup penuh beserta backup inkremental (dan WAL hasil streaming) yang bergantung padanya. Restore membutuhkan backup penuh dan setiap backup inkremental dalam rantainya.',
          chains:
            'Hanya menyimpan N rantai terakhir. Saat rantai tidak lagi disimpan, backup penuh dan semua backup inkrementalnya dihapus. Restore hanya bisa dilakukan dalam rantai yang masih disimpan.',
          chainsAndFullBackups:
            'Menyimpan N rantai terakhir untuk restore ke titik waktu tertentu dalam periode terbaru, serta backup penuh lama yang berdiri sendiri sesuai kebijakan di bawah (GFS atau N terakhir). Backup penuh tersebut menjadi titik restore jangka panjang setelah backup inkrementalnya dihapus.',
        },
        chainsCount: 'Rantai',
        chainsCountTooltip:
          'Jumlah rantai backup terbaru yang disimpan. Rantai terdiri dari backup penuh beserta backup inkrementalnya.',
        fullBackups: 'Backup penuh',
        fullBackupsCount: 'Backup penuh terbaru',
        gfsFields: {
          hourly: 'Per jam',
          daily: 'Harian',
          weekly: 'Mingguan',
          monthly: 'Bulanan',
          yearly: 'Tahunan',
        },
        chainsKept: 'Rantai yang disimpan',
        fullBackupsKept: 'Backup penuh yang disimpan',
        fullBackupsCountValue: 'Terbaru: {{count}}',
        gfsTooltip:
          'Rotasi Grandfather-Father-Son: simpan N backup penuh terakhir per jam, harian, mingguan, bulanan, dan tahunan.',
      },
      restore: {
        errorHints: {
          downloadInProgress:
            'Unduhan untuk restore database ini sedang berlangsung. Tunggu sampai selesai, lalu coba lagi.',
          targetTimeUnreachable:
            'Waktu tujuan tidak dapat dicapai karena ada celah WAL atau waktunya di luar rentang yang tersedia. Pilih waktu lain atau restore titik terbaru yang tersedia.',
        },
        backupIntro:
          'Perintah restore untuk backup ini. Backup penuh hanya membutuhkan filenya sendiri. Backup inkremental juga membutuhkan backup penuh dan semua backup inkremental sebelumnya.',
        pointInTimeIntro:
          'Restore ke titik waktu tertentu. Pilih waktu tujuan atau biarkan kosong untuk melakukan restore titik terbaru yang tersedia.',
        targetTime: 'Waktu tujuan',
        latestAvailable: 'Terbaru yang tersedia',
        preparing: 'Menyiapkan perintah restore...',
        linkExpiry:
          'Tautan unduhan hanya bisa digunakan sekali dan kedaluwarsa dalam 15 menit. Jika perintah gagal dengan 401, buka kembali dialog ini untuk membuat tautan baru.',
        copied: 'Disalin',
        methods: { script: 'Melalui script', manual: 'Manual' },
        environments: { host: 'PostgreSQL di host' },
        fields: {
          restoreDirectory: 'Direktori restore',
          pgBinPath: 'Path bin PostgreSQL',
          pgBinPathPlaceholder: '{{path}} (opsional)',
          pgImage: 'Image PostgreSQL',
        },
        scriptTitles: {
          host: 'Jalankan di host tujuan restore',
          docker: 'Jalankan di mesin yang menyediakan Docker',
        },
        manualSteps: {
          download: 'Unduh bundle',
          extract: 'Ekstrak',
          verify: 'Verifikasi transfer',
          reconstruct: 'Rekonstruksi direktori data',
          checkConfig: 'Periksa file konfigurasi cluster',
          wireUpRecovery: 'Dekompresi WAL dan siapkan pemulihan',
          recoveryParameters: 'Jika pemulihan berhenti karena pengaturan parameter',
        },
        beforeYouRun: {
          title: 'Sebelum menjalankan',
          host: {
            clientTools:
              'Instal tool klien PostgreSQL {{version}}. Atur path bin di atas jika tool tidak ada di PATH.',
            zstd: 'zstd dibutuhkan untuk mendekompresi backup.',
            zstdWithWal: 'zstd dibutuhkan untuk mendekompresi backup dan WAL-nya.',
            emptyDirectory:
              'Direktori restore harus kosong dan bukan cluster yang sedang digunakan.',
          },
          docker: {
            image: 'Gunakan image postgres:{{version}}. Versi mayor harus sama dengan sumber.',
            hostTools:
              'Host membutuhkan zstd, tar, dan curl. Unduhan dan seluruh dekompresi backup dasar berjalan di host, sedangkan hanya pg_combinebackup yang berjalan dalam container. Jadi, image postgres tidak membutuhkan tool tambahan.',
            hostToolsWithWal:
              'Host membutuhkan zstd, tar, dan curl. Unduhan dan seluruh dekompresi backup dasar serta WAL berjalan di host, sedangkan hanya pg_combinebackup yang berjalan dalam container. Jadi, image postgres tidak membutuhkan tool tambahan.',
          },
        },
        afterItFinishes: {
          title: 'Setelah selesai',
          host: {
            clusterLocation: 'Cluster berada di <dataDir/>.',
            changeOwner: 'Ubah kepemilikannya sebagai root: <command/>.',
            start:
              'Jalankan sebagai postgres: <command/>, atau arahkan data_directory ke direktori tersebut.',
            walReplay:
              'PostgreSQL memutar ulang WAL lalu beralih menjadi primary. Pantau log untuk mengetahui kapan pemulihan selesai.',
          },
          docker: {
            clusterLocation: '<dataDir/> di host berisi cluster hasil restore.',
            volumeMount:
              'PostgreSQL {{version}} menyimpan direktori data di <dataDir/>, di bawah volume <volumeDir/>. Mount direktori keluaran sebagai root volume tersebut, sesuai yang diharapkan docker-compose <code>./pgdata:/var/lib/postgresql</code>:',
            dataDirectoryMount: 'Bind-mount cluster ke direktori data image <dataDir/>:',
            ownership:
              'Kepemilikan harus sesuai dengan uid postgres di dalam image (999 pada image resmi).',
          },
          missingConfigFiles:
            'Backup dasar hanya menyalin direktori data. Jika PostgreSQL sumber menyimpan konfigurasi di luar direktori tersebut, seperti tata letak Debian/Ubuntu di <configDir/>, maka <code>postgresql.conf</code>, <code>pg_hba.conf</code>, dan <code>pg_ident.conf</code> tidak ada dalam backup dan server tidak akan berjalan. Cari path aslinya di sumber dengan <code>psql -Atc "SHOW config_file"</code>, salin ketiga file tersebut, lalu jadikan <code>data_directory</code>, <code>hba_file</code>, <code>ident_file</code>, <code>external_pid_file</code>, serta baris <code>ssl</code> / <code>ssl_*</code> sebagai komentar. Buat <code>conf.d</code> dan atur <code>listen_addresses = \'*\'</code> untuk container. Jika file tersebut sudah ada tetapi error yang sama tetap muncul, direktori yang di-mount keliru, bukan konfigurasinya.',
        },
      },
      monitoring: {
        periods: {
          hour: 'Satu jam terakhir',
          day: 'Satu hari terakhir',
          week: 'Satu minggu terakhir',
        },
        levels: {
          notStreaming: 'Penyimpanan perubahan berkelanjutan nonaktif',
          collectingData: 'Mengumpulkan data',
          broken: 'Ada masalah',
          needsAttention: 'Perlu perhatian',
          healthy: 'Semua berjalan baik',
        },
        reasonLevels: {
          broken: 'Bermasalah',
          needsAttention: 'Perlu perhatian',
          info: 'Informasi',
        },
        keepingUpStates: {
          unknown: 'Tidak diketahui',
          fallingBehind: 'Tertinggal',
          catchingUp: 'Mengejar ketertinggalan',
          idle: 'Sepi',
          keepingUp: 'Mengikuti laju penulisan',
        },
        banner: {
          collectingDataDescription: 'Streaming dimulai. Angka pertama muncul dalam satu menit.',
          notStreamingDescription:
            'Backup nonaktif atau belum ada penyimpanan yang dipilih. <configLink>Buka konfigurasi</configLink> untuk mengaktifkannya.',
          cannotConfirm: 'Status saat ini belum bisa dipastikan',
          cannotConfirmDescription: 'Status terakhir yang diketahui: {{level}}, {{duration}} lalu.',
          measuredAgo: 'Diukur {{duration}} lalu',
          healthyHeadline: 'Semua perubahan sedang disimpan',
          restoreUpTo:
            'Database bisa di-restore ke waktu mana pun hingga {{moment}} ({{age}} lalu).',
          restoreOnlyUpTo:
            'Sampai backup penuh baru selesai, database hanya bisa di-restore hingga {{moment}} ({{age}} lalu).',
          restoreFromUpTo:
            'Database bisa di-restore ke waktu mana pun dari {{from}} hingga {{moment}} ({{age}} lalu).',
          restoreFromUpToWithoutAge:
            'Database bisa di-restore ke waktu mana pun dari {{from}} hingga {{moment}}.',
          restoreUpToWithoutAge: 'Database bisa di-restore ke waktu mana pun hingga {{moment}}.',
          restoreOnlyUpToWithoutAge:
            'Sampai backup penuh baru selesai, database hanya bisa di-restore hingga {{moment}}.',
          restoreNothingPending:
            'Restore belum bisa dilakukan sampai backup penuh pertama selesai. Backup sedang diproses.',
          restoreNothing:
            'Restore belum bisa dilakukan karena belum ada backup penuh yang selesai.',
          startFullBackup: 'Mulai backup penuh',
          fullBackupRequested: 'Backup penuh diminta.',
          notMeasuredYet: 'Belum diukur',
          refreshFailed:
            'Gagal memperbarui data. Angka terakhir tetap ditampilkan dan percobaan berikutnya berjalan dalam 30 detik.',
        },
        keepingUp: {
          keepingUp:
            'Databasus menyimpan perubahan secepat database menulisnya: {{writeRate}} ditulis, {{archiveRate}} disimpan.',
          fallingBehind:
            'Database menulis perubahan lebih cepat daripada Databasus menyimpannya, dengan selisih sekitar {{differenceRate}}: {{writeRate}} dibanding {{archiveRate}}.',
          fallingBehindBacklog:
            '{{unsavedAmount}} belum disimpan dan bertambah sekitar {{growthPerHour}} per jam.',
          catchingUp:
            'Databasus sedang menyimpan antrean perubahan: tersisa {{unsavedAmount}}, perkiraan {{duration}} lagi.',
          catchingUpWithoutEstimate:
            'Databasus sedang menyimpan antrean perubahan: tersisa {{unsavedAmount}}.',
          idle: 'Database hampir tidak menulis data. Perubahan baru mencapai penyimpanan dalam beberapa menit.',
          rebuildConsequence:
            'Dengan laju ini, Databasus akan memulai rantai backup baru dari backup penuh dalam sekitar {{duration}}.',
          limitConsequence:
            'Dengan laju ini, database akan menghapus perubahan yang belum diterima Databasus dalam sekitar {{duration}}.',
        },
        reasons: {
          streamingStopped: {
            title: 'Penyimpanan berhenti',
            whatHappened: 'Databasus berhenti menerima perubahan {{duration}} lalu.',
            whatToDo:
              'Databasus mencoba lagi secara otomatis. Jika terus berlanjut, periksa log Databasus.',
          },
          sourceUnreachable: {
            title: 'Database tidak dapat dijangkau',
            whatHappened: 'Databasus tidak dapat menjangkau database selama {{duration}}.',
            whatToDo: 'Pastikan server database berjalan dan bisa dijangkau dari Databasus.',
          },
          walChainBroken: {
            title: 'Perubahan di antara backup hilang',
            whatHappened:
              'Sebagian riwayat perubahan tidak pernah mencapai penyimpanan sehingga perubahan setelahnya tidak bisa di-restore.',
            whatToDo: 'Mulai backup penuh di tab Backup.',
            whatToDoPending: 'Backup penuh baru sedang diproses. Pantau di tab Backup.',
          },
          slotMissing: {
            title: 'Database berhenti menyimpan perubahan',
            whatHappened:
              'Database tidak lagi menyimpan perubahan untuk Databasus karena reservasi yang dibuat Databasus sudah hilang.',
            whatToDo:
              'Databasus membuat ulang reservasi saat streaming dimulai kembali, tetapi perubahan di antaranya hilang. Mulai backup penuh di tab Backup dan pastikan tidak ada yang menghapus reservasi di database.',
            whatToDoPending:
              'Databasus membuat ulang reservasi saat streaming dimulai kembali dan backup penuh baru sedang diproses. Pastikan tidak ada yang menghapus reservasi di database.',
          },
          slotLost: {
            title: 'Database menghapus perubahan yang belum disimpan',
            whatHappened: 'Database menghapus perubahan yang belum diterima Databasus.',
            whatToDo:
              'Databasus otomatis memulai ulang dengan backup penuh baru. Agar tidak terulang, percepat penyimpanan atau izinkan database menyimpan lebih banyak perubahan.',
          },
          archiveNotAdvancing: {
            title: 'Perubahan tidak mencapai penyimpanan',
            whatHappened:
              'Tidak ada perubahan baru yang mencapai penyimpanan selama {{duration}} meskipun database terus menulis.',
            whatToDo: 'Pastikan penyimpanan bisa dijangkau dan memiliki ruang kosong.',
          },
          archiveSlowerThanSource: {
            title: 'Penyimpanan terlalu lambat',
            whatHappened:
              'Database menulis lebih cepat daripada Databasus menyimpan. Antrean bertambah {{amount}} dalam 10 menit.',
            whatToDo:
              'Periksa kecepatan jaringan ke penyimpanan atau pindahkan penyimpanan lebih dekat ke Databasus.',
          },
          slotKeepsExtraWal: {
            title: 'Database menyimpan perubahan tambahan',
            whatHappened:
              'Selama {{duration}}, database menyimpan log perubahan untuk Databasus lebih banyak dari biasanya dan jumlahnya terus bertambah.',
            whatToDo: 'Pastikan server database memiliki ruang disk kosong.',
          },
          slotLimitApproaching: {
            title: 'Perubahan yang belum disimpan akan segera dihapus',
            whatHappened:
              'Dengan laju ini, database akan menghapus perubahan yang belum diterima Databasus dalam sekitar {{duration}}.',
            whatToDo:
              'Percepat penyimpanan atau naikkan <code>max_slot_wal_keep_size</code> di database.',
          },
          nearSlotRebuild: {
            title: 'Hampir harus memulai ulang',
            whatHappened:
              'Perubahan yang belum diterima Databasus mencapai {{amount}} dari batas {{limit}} yang membuat Databasus memulai ulang dari backup penuh.',
            whatToDo: 'Percepat penyimpanan atau naikkan batas di pengaturan backup.',
          },
          localQueueFilling: {
            title: 'Buffer Databasus mulai penuh',
            whatHappened: 'Buffer Databasus berisi {{amount}} dari {{limit}}.',
            whatToDo:
              'Databasus otomatis menjeda penerimaan saat buffer penuh. Periksa kecepatan penyimpanan.',
          },
          frequentRestarts: {
            title: 'Penerimaan sering dimulai ulang',
            whatHappened: 'Penerimaan dimulai ulang dalam satu jam terakhir: {{count}}.',
            whatToDo: 'Periksa log Databasus untuk mencari penyebabnya.',
          },
          forcedRotationRefused: {
            title: 'Perubahan saat sepi terlambat mencapai penyimpanan',
            whatHappened:
              'Database menolak menutup file perubahan tepat waktu sehingga perubahan saat sepi lebih lambat mencapai penyimpanan.',
            whatToDo:
              'Berikan izin kepada pengguna backup untuk menjalankan <code>pg_switch_wal()</code>.',
          },
        },
        notices: {
          noSlotWalLimit: {
            title: 'Perubahan yang disimpan tidak dibatasi',
            whatHappened: 'Database menyimpan perubahan untuk Databasus tanpa batas.',
            whatToDo:
              'Batas melindungi disk server database jika Databasus lama tidak berjalan. Konsekuensinya, backup penuh baru diperlukan saat batas tercapai. Atur dengan <code>max_slot_wal_keep_size</code>.',
          },
          slotKeepsExtraWalSteadily: {
            title: 'Database menyimpan perubahan tambahan',
            whatHappened:
              'Database menyimpan log perubahan untuk Databasus lebih banyak dari biasanya.',
            whatToDo:
              'Ini normal saat beban tinggi. Tidak perlu tindakan selama jumlahnya tidak bertambah.',
          },
          noFullBackupYet: {
            title: 'Belum ada backup penuh',
            whatHappened:
              'Backup penuh pertama belum selesai sehingga perubahan yang tersimpan belum bisa di-restore.',
            whatToDo: 'Tunggu sampai selesai atau mulai di tab Backup.',
          },
        },
        figures: {
          lastSaved: 'Perubahan terakhir disimpan',
          waitingToBeSaved: 'Menunggu disimpan',
          notMeasured: 'Belum diukur',
          averageHint: 'rata-rata 10 menit',
        },
        restoreTimeline: {
          title: 'Yang bisa di-restore',
          restorable: 'Bisa di-restore',
          lost: 'Tidak bisa di-restore',
          notSavedYet: 'Belum disimpan ke penyimpanan',
          lostSentence:
            'Database tidak bisa di-restore ke waktu antara {{from}} dan {{to}} karena perubahan ini tidak pernah mencapai penyimpanan.',
          moreLost: 'Rentang lain yang tidak bisa di-restore: {{count}}.',
          range: '{{from}} – {{to}}',
        },
        technicalDetails: { title: 'Detail teknis' },
        path: {
          database: 'Database',
          databasus: 'Databasus',
          storage: 'Penyimpanan',
          notReceivedYet: 'Belum diterima Databasus',
          rebuildLimit: 'Pada {{amount}}, Databasus memulai rantai baru dari backup penuh',
          allowanceLeft: 'Sisa ruang sebelum database menghapus perubahan: {{amount}}',
          nothingWaiting: 'Tidak ada yang menunggu',
          ratesCaption:
            'Kecepatan adalah rata-rata 10 menit terakhir, sedangkan jumlah adalah nilai saat ini. Database yang menulis dalam lonjakan bisa tetap menunjukkan kecepatan meskipun tidak ada antrean.',
          waitingInBuffer: 'Menunggu di buffer',
          pauseLimit: 'Penerimaan dijeda pada {{amount}}',
          receivingPaused: 'Penerimaan dijeda',
          nothingSavedYet: 'Belum ada yang disimpan',
          overLimit: 'Melebihi batas',
          ofLimit: '{{amount}} dari {{limit}}',
          notSavedYetAmount: 'Belum disimpan ke penyimpanan: {{amount}}',
          unknownAmount: 'Tidak diketahui',
        },
        tooltips: {
          notSavedYet:
            'Perubahan yang sudah ditulis database tetapi belum ada di penyimpanan: perubahan yang belum diterima Databasus ditambah yang menunggu di buffer. Istilah teknis: unarchived WAL.',
          notReceivedYet:
            'Perubahan yang disimpan database untuk Databasus karena belum diterima Databasus. Istilah teknis: replication slot lag.',
          rebuildLimit:
            'Setelah jumlah ini terlampaui, Databasus menghapus slot replikasi dan memulai rantai backup baru agar disk server database tidak penuh. Istilah teknis: WAL lag threshold.',
          allowanceLeft:
            'Seberapa banyak lagi database bisa menulis sebelum menghapus perubahan yang belum diterima Databasus. Istilah teknis: safe_wal_size, dibatasi max_slot_wal_keep_size.',
          waitingInBuffer:
            'Perubahan yang sudah diterima Databasus tetapi belum disimpan ke penyimpanan. Istilah teknis: local WAL queue.',
          pauseLimit:
            'Saat buffer mencapai ukuran ini, Databasus menjeda penerimaan sampai buffer terkuras. Istilah teknis: back pressure high watermark.',
          lastSaved:
            'Waktu file perubahan terbaru mencapai penyimpanan. Istilah teknis: newest archived WAL segment.',
          writeRate:
            'Kecepatan database menulis perubahan selama 10 menit terakhir. Istilah teknis: WAL write rate.',
          saveRate:
            'Kecepatan Databasus menyimpan perubahan ke penyimpanan selama 10 menit terakhir. Istilah teknis: WAL archive rate.',
        },
        charts: {
          keepingUpTitle: 'Apakah Databasus mengikuti laju penulisan?',
          unsavedTitle: 'Berapa banyak yang belum disimpan?',
          unsavedSubtitle: 'Termasuk yang belum diterima Databasus',
          bufferTitle: 'Buffer Databasus',
          databaseWrites: 'Penulisan database',
          savedToStorage: 'Disimpan ke penyimpanan',
          notSavedYet: 'Belum disimpan ke penyimpanan',
          notReceivedYet: 'Belum diterima Databasus',
          waitingInBuffer: 'Menunggu di buffer',
          pauseLimit: 'Penerimaan dijeda di sini',
          restarts: 'Restart',
          pauses: 'Jeda',
          notEnoughData:
            'Belum cukup data untuk periode ini. Grafik mungkin sudah tersedia untuk periode yang lebih singkat.',
          dataSince: 'Data sejak {{time}}',
          noMeasurements: 'Tidak ada pengukuran dari {{from}} hingga {{to}}.',
          perSecond: '{{amount}}/s',
        },
        details: {
          slotStatus: 'Status slot (wal_status)',
          allowance: 'Sisa kapasitas (safe_wal_size)',
          noLimit: 'Tanpa batas',
          averageSegmentWrite: 'Rata-rata waktu penulisan segmen',
          longestSegmentWrite: 'Waktu penulisan segmen terlama',
          uploadConcurrency: 'Unggahan bersamaan',
          walToStoredRatio: 'Rasio ukuran WAL terhadap ukuran tersimpan',
          walSegmentSize: 'Ukuran segmen WAL',
          sessionStart: 'Awal sesi streaming',
          streamerState: 'Status streamer',
          sourceState: 'Status sumber',
          restartsLastHour: 'Restart dalam satu jam terakhir',
          milliseconds: '{{value}} ms',
          notAvailable: 'Tidak tersedia',
          streamerStates: {
            running: 'Berjalan',
            notRunning: 'Tidak berjalan',
            notStarted: 'Belum dimulai',
          },
          sourceStates: {
            reachable: 'Dapat dijangkau',
            unreachable: 'Tidak dapat dijangkau',
            slotMissing: 'Dapat dijangkau, slot replikasi tidak ada',
          },
        },
        durations: {
          seconds: '{{seconds}} dtk',
          minutes: '{{minutes}} mnt',
          hoursMinutes: '{{hours}} jam {{minutes}} mnt',
          daysHours: '{{days}} hari {{hours}} jam',
        },
      },
    },
  },
  storages: {
    types: {
      local: 'Penyimpanan lokal',
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
    azureBlobAuthMethods: { accountKey: 'Kunci akun', connectionString: 'Connection string' },
    sftpAuthMethods: { password: 'Kata sandi', privateKey: 'Private key' },
    list: {
      addStorage: 'Tambah penyimpanan',
      description: 'Penyimpanan adalah tempat menyimpan backup (disk lokal, S3, dll.)',
      backToList: '← Kembali ke penyimpanan',
    },
    card: {
      type: {
        local: 'Jenis: penyimpanan lokal',
        s3: 'Jenis: S3',
        googleDrive: 'Jenis: Google Drive',
        nas: 'Jenis: NAS',
        azureBlob: 'Jenis: Azure Blob Storage',
        ftp: 'Jenis: FTP',
        sftp: 'Jenis: SFTP',
        rclone: 'Jenis: Rclone',
      },
      hasSaveError: 'Ada error penyimpanan',
    },
    config: {
      namePlaceholder: 'Masukkan nama...',
      settingsTitle: 'Pengaturan penyimpanan',
      lastSaveError: {
        title: 'Error penyimpanan',
        errorLabel: 'Error:',
        clearHint: 'Untuk menghapus error ini, pilih salah satu:',
        clearByTestingConnection:
          '- uji koneksi melalui tombol di bawah (meskipun pengaturan sudah diperbarui);',
        clearByNextSave: '- tunggu sampai penyimpanan berikutnya selesai tanpa error;',
      },
      removeConfirmation:
        'Yakin ingin menghapus penyimpanan ini? Tindakan ini tidak bisa dibatalkan. Backup yang memakai penyimpanan ini juga akan dihapus.',
      removeBlockedByDatabases:
        'Penyimpanan dipakai beberapa database. Hapus penyimpanan dari database tersebut terlebih dahulu.',
    },
    actions: { testConnection: 'Uji koneksi' },
    connectionTestSucceeded: {
      title: 'Uji koneksi berhasil!',
      description: 'Koneksi penyimpanan berhasil diuji',
    },
    transfer: {
      title: 'Pindahkan penyimpanan ke ruang kerja lain',
      blockedByDatabases:
        'Penyimpanan ini dipakai beberapa database. Pindahkan atau hapus database terkait terlebih dahulu.',
      description: 'Pilih ruang kerja tujuan untuk penyimpanan ini.',
      targetWorkspace: 'Ruang kerja tujuan',
      selectWorkspace: 'Pilih ruang kerja',
      submit: 'Pindahkan',
    },
    fields: {
      path: 'Path',
      remotePath: 'Path remote',
      share: 'Share',
      domain: 'Domain',
      useSsl: 'Gunakan SSL',
      useSslTls: 'Gunakan SSL/TLS',
      skipTls: 'Lewati TLS',
      skipTlsVerify: 'Lewati verifikasi TLS',
      skipHostKey: 'Lewati host key',
      authMethod: 'Metode autentikasi',
      privateKey: 'Private key',
      credentials: 'Kredensial',
      s3Bucket: 'Bucket S3',
      region: 'Region',
      accessKey: 'Access key',
      secretKey: 'Secret key',
      endpoint: 'Endpoint',
      folderPrefix: 'Prefix folder',
      prefix: 'Prefix',
      blobPrefix: 'Prefix blob',
      virtualHost: 'Virtual host',
      storageClass: 'Kelas penyimpanan',
      connection: 'Koneksi',
      connectionString: 'Connection string',
      accountName: 'Nama akun',
      accountKey: 'Kunci akun',
      containerName: 'Nama container',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      userToken: 'Token pengguna',
      config: 'Konfigurasi',
    },
    edit: {
      namePlaceholder: 'Penyimpanan saya',
      advancedSettings: 'Pengaturan lanjutan',
      optionalPlaceholder: '{{example}} (opsional)',
      usernamePlaceholder: 'nama pengguna',
      passwordPlaceholder: 'kata sandi',
      remotePathTooltip: 'Path direktori remote untuk menyimpan backup (opsional)',
      local: {
        diskSpaceWarning:
          'Hati-hati: penyimpanan lokal bisa kehabisan ruang disk. Sebaiknya gunakan S3 atau penyimpanan tanpa batas kapasitas',
      },
      s3: {
        cloudflareR2Guide: 'Bagaimana cara menggunakan Cloudflare R2?',
        endpointTooltip:
          'URL endpoint khusus yang kompatibel dengan S3 (opsional, kosongkan untuk AWS S3)',
        prefixTooltip:
          "Prefix opsional untuk semua object key (misalnya 'backups/' atau 'my_team/'). Mungkin tidak berfungsi pada beberapa penyimpanan yang kompatibel dengan S3. Tidak bisa diubah setelah dibuat (jika diubah, backup akan hilang).",
        useVirtualHostedStyle: 'Gunakan gaya virtual-hosted',
        virtualHostTooltip:
          'Gunakan URL bergaya virtual-hosted (bucket.s3.region.amazonaws.com), bukan path-style (s3.region.amazonaws.com/bucket). Mungkin diperlukan jika muncul error COS.',
        skipTls: 'Lewati TLS',
        skipTlsVerifyTooltip:
          'Lewati verifikasi sertifikat TLS. Aktifkan jika penyimpanan yang kompatibel dengan S3 menggunakan sertifikat self-signed. Peringatan: ini mengurangi keamanan.',
        storageClassTooltip:
          'Kelas penyimpanan S3 untuk objek yang diunggah. Biarkan default untuk Standard. Beberapa penyedia menawarkan kelas lebih murah seperti One Zone IA. Jangan gunakan Glacier/Deep Archive karena file harus langsung tersedia untuk restore.',
      },
      googleDrive: {
        guideLink: 'Bagaimana cara menghubungkan Google Drive?',
        authorize: 'Otorisasi',
      },
      nas: {
        shareWithSlashWarning:
          'Share harus berupa satu nama share. Gunakan kolom Path untuk subdirektori (misalnya Share: Databasus, Path: DB1)',
        useSslTooltip: 'Aktifkan enkripsi SSL/TLS untuk koneksi aman',
        domainTooltip:
          'Nama domain Windows (opsional, kosongkan jika tidak memakai autentikasi domain)',
        pathPlaceholder: '{{example}} (opsional, tanpa garis miring di awal)',
        pathTooltip: 'Path subdirektori di dalam share (opsional)',
      },
      azureBlob: {
        connectionStringTooltip: 'Connection string Azure Storage dari Azure Portal',
        endpointTooltip: 'URL endpoint khusus (opsional, kosongkan untuk Azure standar)',
        prefixTooltip:
          "Prefix opsional untuk semua nama blob (misalnya 'backups/' atau 'my_team/')",
      },
      ftp: {
        enableFtps: 'Aktifkan FTPS',
        useSslTooltip: 'Gunakan enkripsi TLS eksplisit (FTPS) untuk transfer file yang aman',
        skipCertificateVerification: 'Lewati verifikasi sertifikat',
        skipTlsVerifyTooltip:
          'Lewati verifikasi sertifikat TLS. Aktifkan jika server FTP menggunakan sertifikat self-signed. Peringatan: ini mengurangi keamanan.',
      },
      sftp: {
        privateKeyTooltip:
          'Tempel private key SSH (format PEM). Mendukung kunci RSA, DSA, ECDSA, dan Ed25519.',
        skipHostKeyVerification: 'Lewati verifikasi host key',
        skipHostKeyTooltip:
          'Lewati verifikasi host key SSH. Aktifkan jika kamu memercayai server. Peringatan: ini mengurangi keamanan.',
      },
      rclone: {
        configTooltip:
          "Tempel isi rclone.conf di sini. Jalankan 'rclone config file' dan salin isinya. Konfigurasi ini mendukung lebih dari 70 penyedia penyimpanan cloud.",
        configHiddenNote:
          '*konten disembunyikan agar data sensitif tidak terekspos. Untuk memperbarui konfigurasi yang ada, masukkan konfigurasi baru di sini',
        docsLink: 'Dokumentasi Rclone',
        remotePathTooltip:
          "Prefix path opsional di remote tempat backup disimpan (misalnya '/backups' atau 'my-folder/backups')",
      },
    },
  },
};

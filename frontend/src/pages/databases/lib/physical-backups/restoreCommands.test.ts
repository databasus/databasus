import { describe, expect, it } from 'vitest';

import {
  buildChownCommand,
  buildDockerRunCommand,
  buildPgCtlStartCommand,
  buildDockerScriptCommand,
  buildManualSteps,
  buildScriptCommand,
  clusterDataDir,
  containerDataDir,
  containerVolumeDir,
} from '@/pages/databases/lib/physical-backups/restoreCommands';

const scriptUrl = 'https://app.example.com/api/v1/physical-recovery-scripts';
const bundleUrl = 'https://app.example.com/api/v1/physical-restore-sets?token=abc';

describe('container paths', () => {
  it('uses the flat data path for PostgreSQL 17 and below', () => {
    expect(containerDataDir('17')).toBe('/var/lib/postgresql/data');
    expect(containerVolumeDir('17')).toBe('/var/lib/postgresql/data');
  });

  it('uses the version-specific PGDATA and volume root for PostgreSQL 18', () => {
    expect(containerDataDir('18')).toBe('/var/lib/postgresql/18/docker');
    expect(containerVolumeDir('18')).toBe('/var/lib/postgresql');
  });

  it('builds the cluster under the output dir mirroring the image layout', () => {
    expect(clusterDataDir('./out', '17')).toBe('./out/data');
    expect(clusterDataDir('./out', '18')).toBe('./out/18/docker');
  });
});

describe('buildScriptCommand', () => {
  it('omits --pg-bin and --target-time when neither is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '',
      targetTime: '',
    });

    expect(command).not.toContain('--pg-bin');
    expect(command).not.toContain('--target-time');
    expect(command).toContain(`"${bundleUrl}" "./databasus-restore"`);
  });

  it('keeps shell expansions literal in the selected paths', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './restore $HOME/`date`',
      pgBin: '/opt/pg tools',
      targetTime: '',
    });

    expect(command).toContain('"./restore \\$HOME/\\`date\\`"');
    expect(command).toContain('--pg-bin "/opt/pg tools"');
  });

  it('includes --pg-bin when a bin path is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '/usr/lib/postgresql/18/bin',
      targetTime: '',
    });

    expect(command).toContain('--pg-bin "/usr/lib/postgresql/18/bin"');
  });

  it('includes --target-time when a target time is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '',
      targetTime: '2026-06-12 14:30:00+00:00',
    });

    expect(command).toContain('--target-time "2026-06-12 14:30:00+00:00"');
  });
});

describe('buildDockerScriptCommand', () => {
  it('runs the script on the host, combines via the chosen image, and references the bundle once', () => {
    const command = buildDockerScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      image: 'postgres:18',
      targetTime: '',
    });

    expect(command).toContain(`curl -fsSL "${scriptUrl}" | sh -s --`);
    expect(command).toContain('--combine-image "postgres:18"');
    expect(command).not.toContain('--target-time');
    expect(command.match(/physical-restore-sets/g)).toHaveLength(1);
  });

  it('passes --target-time to the host script when a target time is set', () => {
    const command = buildDockerScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      image: 'postgres:18',
      targetTime: '2026-06-12 14:30:00+00:00',
    });

    expect(command).toContain(
      '--combine-image "postgres:18" --target-time "2026-06-12 14:30:00+00:00"',
    );
  });
});

describe('buildManualSteps', () => {
  const reconstructTitleKey = 'backups.physical.restore.manualSteps.reconstruct';
  const configCheckTitleKey = 'backups.physical.restore.manualSteps.checkConfig';
  const wireUpRecoveryTitleKey = 'backups.physical.restore.manualSteps.wireUpRecovery';
  const recoveryParametersTitleKey = 'backups.physical.restore.manualSteps.recoveryParameters';

  const base = {
    bundleUrl,
    outputDir: './databasus-restore',
    pgVersion: '18',
    pgBin: '',
    image: 'postgres:18',
    targetTime: '',
  };

  it('decompresses each backup blob before pg_combinebackup and skips recovery when there is no WAL', () => {
    const steps = buildManualSteps({ ...base, environment: 'host', hasWal: false });
    const titleKeys = steps.map((step) => step.titleKey);
    const combine = steps.find((step) => step.titleKey === reconstructTitleKey);

    expect(combine?.code).toContain('base.tar');
    expect(combine?.code).toContain('pg_combinebackup bundle/recon/full');
    expect(combine!.code.indexOf('base.tar')).toBeLessThan(
      combine!.code.indexOf('pg_combinebackup'),
    );
    expect(titleKeys).not.toContain(wireUpRecoveryTitleKey);
    expect(titleKeys).not.toContain(recoveryParametersTitleKey);
  });

  it('adds a recovery-parameter note pointing at pg_controldata when there is WAL', () => {
    const host = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const docker = buildManualSteps({ ...base, environment: 'docker', hasWal: true });
    const hostNote = host.find((step) => step.titleKey === recoveryParametersTitleKey);
    const dockerNote = docker.find((step) => step.titleKey === recoveryParametersTitleKey);

    expect(hostNote?.code).toContain('pg_controldata "./databasus-restore/18/docker"');
    expect(hostNote?.code).toContain('max_connections');
    expect(dockerNote?.code).toContain(
      'docker run --rm -v "$PWD:/work" -w /work postgres:18 pg_controldata',
    );
  });

  it('builds the cluster at the version-specific path', () => {
    const pg18 = buildManualSteps({ ...base, pgVersion: '18', environment: 'host', hasWal: false });
    const pg17 = buildManualSteps({ ...base, pgVersion: '17', environment: 'host', hasWal: false });
    const combine18 = pg18.find((step) => step.titleKey === reconstructTitleKey);
    const combine17 = pg17.find((step) => step.titleKey === reconstructTitleKey);

    expect(combine18?.code).toContain('-o "./databasus-restore/18/docker"');
    expect(combine17?.code).toContain('-o "./databasus-restore/data"');
  });

  it('includes the WAL recovery step with a PGDATA-relative restore_command when there is WAL', () => {
    const steps = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const recovery = steps.find((step) => step.titleKey === wireUpRecoveryTitleKey);

    expect(recovery).toBeDefined();
    expect(recovery?.code).toContain('./databasus-restore/18/docker/recovery.signal');
    expect(recovery?.code).toContain('databasus_wal_restore/%f');
    // restore_command is relative to PGDATA (no host-absolute path) so replay is container-portable.
    expect(recovery?.code).not.toContain('wal_abs');
  });

  it('bakes the target time into recovery settings only when one is given', () => {
    const withTarget = buildManualSteps({
      ...base,
      environment: 'host',
      hasWal: true,
      targetTime: '2026-06-12 14:30:00+00:00',
    });
    const recovery = withTarget.find((step) => step.titleKey === wireUpRecoveryTitleKey);

    expect(recovery?.code).toContain("recovery_target_time = '2026-06-12 14:30:00+00:00'");
    expect(recovery?.code).toContain("recovery_target_action = 'promote'");

    const withoutTarget = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const latestRecovery = withoutTarget.find((step) => step.titleKey === wireUpRecoveryTitleKey);

    expect(latestRecovery?.code).not.toContain('recovery_target_time');
  });

  it('reports missing configuration files rather than writing them, for both environments', () => {
    const hostSteps = buildManualSteps({ ...base, environment: 'host', hasWal: false });
    const dockerSteps = buildManualSteps({ ...base, environment: 'docker', hasWal: false });
    const hostCheckStep = hostSteps.find((step) => step.titleKey === configCheckTitleKey);
    const dockerCheckStep = dockerSteps.find((step) => step.titleKey === configCheckTitleKey);

    for (const checkStep of [hostCheckStep, dockerCheckStep]) {
      expect(checkStep?.code).toContain('echo "MISSING: $f"');
      expect(checkStep?.code).toContain('SHOW config_file');
      // Reporting only - the restore never invents an access-control policy for the user.
      expect(checkStep?.code).not.toContain('scram-sha-256');
      expect(checkStep?.code).not.toContain('trust');
    }
  });

  it('names the version-specific cluster and Debian configuration directories', () => {
    const pg17Steps = buildManualSteps({
      ...base,
      pgVersion: '17',
      environment: 'host',
      hasWal: false,
    });
    const pg18Steps = buildManualSteps({
      ...base,
      pgVersion: '18',
      environment: 'host',
      hasWal: false,
    });

    const pg17CheckStep = pg17Steps.find((step) => step.titleKey === configCheckTitleKey);
    const pg18CheckStep = pg18Steps.find((step) => step.titleKey === configCheckTitleKey);

    expect(pg17CheckStep?.code).toContain('"./databasus-restore/data/$f"');
    expect(pg17CheckStep?.code).toContain('/etc/postgresql/17/main');
    expect(pg18CheckStep?.code).toContain('"./databasus-restore/18/docker/$f"');
    expect(pg18CheckStep?.code).toContain('/etc/postgresql/18/main');
  });

  it('checks the configuration after reconstruction and before recovery is wired up', () => {
    const titleKeys = buildManualSteps({ ...base, environment: 'host', hasWal: true }).map(
      (step) => step.titleKey,
    );

    expect(titleKeys.indexOf(configCheckTitleKey)).toBeGreaterThan(
      titleKeys.indexOf(reconstructTitleKey),
    );
    expect(titleKeys.indexOf(configCheckTitleKey)).toBeLessThan(
      titleKeys.indexOf(wireUpRecoveryTitleKey),
    );
  });

  it('decompresses on the host then runs pg_combinebackup through docker in the docker environment', () => {
    const steps = buildManualSteps({ ...base, environment: 'docker', hasWal: false });
    const combine = steps.find((step) => step.titleKey === reconstructTitleKey);

    expect(combine?.code).toContain(
      'docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" -w /work postgres:18',
    );
    expect(combine?.code).toContain('pg_combinebackup bundle/recon/full');
    expect(combine!.code.indexOf('base.tar')).toBeLessThan(combine!.code.indexOf('docker run'));
  });
});

describe('restore startup commands', () => {
  it('preserves spaces and literal shell characters in ownership and Host startup paths', () => {
    const dataDir = './restore $HOME/18/docker';

    expect(buildChownCommand(dataDir)).toBe(
      'chown -R postgres:postgres "./restore \\$HOME/18/docker"',
    );
    expect(buildPgCtlStartCommand({ dataDir, pgBin: '/opt/pg tools' })).toBe(
      '"/opt/pg tools/pg_ctl" -D "./restore \\$HOME/18/docker" start',
    );
  });

  it.each([
    ['/restore work', '"/restore work:/var/lib/postgresql"'],
    ['./restore work', '"$PWD/./restore work:/var/lib/postgresql"'],
  ])('mounts %s and starts the selected digest-pinned image', (hostDir, mount) => {
    const image = 'postgres:18@sha256:abcdef';

    expect(buildDockerRunCommand({ hostDir, containerDir: '/var/lib/postgresql', image })).toBe(
      `docker run -e POSTGRES_PASSWORD=... -v ${mount} "${image}"`,
    );
  });
});

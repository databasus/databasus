#!/usr/bin/env bash
set -Eeuo pipefail

readonly harness_label_key="com.databasus.storage-test"
run_id="${RANDOM}-$$-$(date +%s)"
readonly run_id
readonly harness_label="${harness_label_key}=${run_id}"
readonly selected_case="${CASE:-all}"
readonly databasus_image="${DATABASUS_IMAGE:?DATABASUS_IMAGE must be set}"
readonly samba_fixture_image="dockurr/samba@sha256:dbd8ebd6392cc1e8e98312d96d163f0039b644b84aee741d89efdd0abd3d32bf"
readonly nfs_fixture_image="itsthenetwork/nfs-server-alpine@sha256:7fa99ae65c23c5af87dd4300e543a86b119ed15ba61422444207efc7abd0ba20"
suite_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly suite_directory

case_root=""

cleanup() {
    local container_id
    local container_name
    local volume_name
    local volume_type
    local -a client_container_ids=()
    local -a server_container_ids=()
    local -a volume_names=()
    local -a network_ids=()

    while read -r container_id container_name; do
        [[ -n "${container_id}" ]] || continue
        if [[ "${container_name}" == *-server-* ]]; then
            server_container_ids+=("${container_id}")
        else
            client_container_ids+=("${container_id}")
        fi
    done < <(docker ps -a --filter "label=${harness_label}" --format '{{.ID}} {{.Names}}' 2>/dev/null)

    if (("${#client_container_ids[@]}")); then
        docker rm -fv "${client_container_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t volume_names < <(docker volume ls -q --filter "label=${harness_label}" 2>/dev/null)
    for volume_name in "${volume_names[@]}"; do
        volume_type="$(docker volume inspect --format '{{index .Options "type"}}' "${volume_name}" 2>/dev/null || true)"
        if [[ "${volume_type}" == "cifs" || "${volume_type}" == "nfs" ]]; then
            docker volume rm -f "${volume_name}" >/dev/null 2>&1 || true
        fi
    done

    if (("${#server_container_ids[@]}")); then
        docker rm -fv "${server_container_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t network_ids < <(docker network ls -q --filter "label=${harness_label}" 2>/dev/null)
    if (("${#network_ids[@]}")); then
        docker network rm "${network_ids[@]}" >/dev/null 2>&1 || true
    fi

    mapfile -t volume_names < <(docker volume ls -q --filter "label=${harness_label}" 2>/dev/null)
    if (("${#volume_names[@]}")); then
        docker volume rm -f "${volume_names[@]}" >/dev/null 2>&1 || true
    fi

    if [[ "${case_root}" == /tmp/databasus-storage-test.* ]]; then
        docker run --rm --label "${harness_label}" --entrypoint /bin/chmod \
            --volume "${case_root}:/case-root" "${databasus_image}" \
            -R 0777 /case-root >/dev/null 2>&1 || true
        rm -rf -- "${case_root}" || true
    fi
}

trap cleanup EXIT
case_root="$(mktemp -d /tmp/databasus-storage-test.XXXXXXXX)"

wait_for_health() {
    local container_name="$1"
    local health_status

    for ((readiness_attempt = 1; readiness_attempt <= 120; readiness_attempt++)); do
        health_status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${container_name}")"
        if [[ "${health_status}" == "healthy" ]]; then
            return
        fi
        if [[ "$(docker inspect --format '{{.State.Running}}' "${container_name}")" != "true" ]]; then
            docker logs "${container_name}" >&2
            return 1
        fi
        sleep 1
    done

    docker logs "${container_name}" >&2
    echo "container did not become healthy" >&2
    return 1
}

wait_for_log() {
    local container_name="$1"
    local required_log_line="$2"

    for ((log_attempt = 1; log_attempt <= 60; log_attempt++)); do
        if docker logs "${container_name}" 2>&1 | grep -F "${required_log_line}" >/dev/null; then
            return
        fi
        sleep 1
    done

    docker logs "${container_name}" >&2
    echo "container did not report readiness: ${required_log_line}" >&2
    return 1
}

wait_for_stop() {
    local container_name="$1"

    for ((exit_attempt = 1; exit_attempt <= 100; exit_attempt++)); do
        if [[ "$(docker inspect --format '{{.State.Running}}' "${container_name}")" != "true" ]]; then
            return
        fi
        sleep 0.1
    done

    echo "container did not stop" >&2
    return 1
}

get_exit_code() {
    local container_name="$1"

    docker inspect --format '{{.State.ExitCode}}' "${container_name}"
}

wait_for_exit() {
    local container_name="$1"

    wait_for_stop "${container_name}"
    [[ "$(get_exit_code "${container_name}")" != "0" ]]
}

start_databasus() {
    local container_name="$1"
    shift

    docker run --detach \
        --name "${container_name}" \
        --label "${harness_label}" \
        --health-interval 1s \
        --health-start-period 1s \
        "$@" \
        "${databasus_image}" >/dev/null
    wait_for_health "${container_name}"
}

start_failing_databasus() {
    local container_name="$1"
    shift

    docker run --detach \
        --name "${container_name}" \
        --label "${harness_label}" \
        "$@" \
        "${databasus_image}" >/dev/null
    wait_for_exit "${container_name}"
}

assert_storage_permission_failure() {
    local container_name="$1"
    local operation_description="${2:-Required operation:}"
    local container_logs

    container_logs="$(docker logs "${container_name}" 2>&1)"
    grep -Eq 'ERROR: Databasus cannot write to .+ as UID [0-9]+ and GID [0-9]+\.' <<<"${container_logs}"
    grep -Fq "${operation_description}" <<<"${container_logs}"
    grep -Fq "Set PUID and PGID or fix the mounted directory permissions:" <<<"${container_logs}"
    grep -Fq "https://databasus.com/advanced-config/#docker-storage-permissions" <<<"${container_logs}"
    if grep -Fq "Starting PostgreSQL..." <<<"${container_logs}"; then
        echo "PostgreSQL started after a failed storage check" >&2
        return 1
    fi
    if grep -Fq "Starting Databasus application..." <<<"${container_logs}"; then
        echo "Databasus started after a failed storage check" >&2
        return 1
    fi
    assert_no_storage_probe_remains "${container_name}"
}

assert_no_storage_probe_remains() {
    local container_name="$1"
    local remaining_probe

    remaining_probe="$(docker run --rm \
        --label "${harness_label}" \
        --volumes-from "${container_name}" \
        --entrypoint /bin/sh \
        "${databasus_image}" \
        -c 'find /databasus-data/temp /databasus-data/backups \
            -type f -name ".databasus-storage-test-*" -print -quit 2>/dev/null || true')"
    if [ -n "${remaining_probe}" ]; then
        echo "Storage probe remains after failed startup: ${remaining_probe}" >&2
        return 1
    fi
}

assert_storage_command_saves_and_deletes_probe() {
    local container_name="$1"

    docker exec --user databasus "${container_name}" \
        databasus --test-storage 2>&1 | grep -Fx "storage test passed" >/dev/null
}

assert_single_runtime_identity() {
    local container_name="$1"
    local runtime_gid
    local runtime_uid
    local service_pid

    runtime_uid="$(docker exec "${container_name}" id -u databasus)"
    runtime_gid="$(docker exec "${container_name}" id -g databasus)"
    [[ "${runtime_uid}" != "0" ]]
    [[ "${runtime_gid}" != "0" ]]

    if docker exec "${container_name}" id postgres >/dev/null 2>&1; then
        echo "The image still contains a postgres operating-system account" >&2
        return 1
    fi

    [[ "$(docker exec "${container_name}" stat -c %u /proc/1)" == "${runtime_uid}" ]]
    [[ "$(docker exec "${container_name}" stat -c %g /proc/1)" == "${runtime_gid}" ]]

    service_pid="$(docker exec "${container_name}" /bin/sh -eu -c '
        pidof postgres | cut -d" " -f1
    ')"
    [[ "$(docker exec "${container_name}" stat -c %u "/proc/${service_pid}")" == "${runtime_uid}" ]]
    [[ "$(docker exec "${container_name}" stat -c %g "/proc/${service_pid}")" == "${runtime_gid}" ]]
}

get_generated_database_password() {
    local container_name="$1"

    docker exec "${container_name}" \
        sed -n 's/.* password=\([^ ]*\) .*/\1/p' /dev/shm/databasus-database-dsn
}

can_log_in_over_loopback() {
    local container_name="$1"
    local database_password="$2"

    docker exec -i --user databasus "${container_name}" /bin/sh -c '
        IFS= read -r PGPASSWORD
        export PGPASSWORD
        /usr/lib/postgresql/17/bin/psql -h localhost -p 5437 -U postgres -d postgres -c "SELECT 1"
    ' <<<"${database_password}" >/dev/null 2>&1
}

assert_postgresql_runtime_authentication() {
    local container_name="$1"
    local database_password
    local replication_probe_output

    [[ "$(docker exec "${container_name}" stat -c %a /databasus-data/pgdata/pg_hba.conf)" == "600" ]]
    [[ "$(docker exec "${container_name}" stat -c %a /databasus-data/pgdata/pg_ident.conf)" == "600" ]]

    if docker exec "${container_name}" grep -Fq ".s.PGSQL" /proc/net/unix; then
        echo "PostgreSQL created a Unix socket" >&2
        return 1
    fi

    if docker exec --user databasus "${container_name}" \
        env -u PGPASSWORD /usr/lib/postgresql/17/bin/psql \
        -h localhost -p 5437 -U postgres -d postgres \
        -c 'SELECT 1' >/dev/null 2>&1; then
        echo "PostgreSQL accepted a loopback login without its generated password" >&2
        return 1
    fi

    database_password="$(get_generated_database_password "${container_name}")"
    [[ -n "${database_password}" ]]
    if docker logs "${container_name}" 2>&1 | grep -F "${database_password}" >/dev/null; then
        echo "The container log contains the generated PostgreSQL password" >&2
        return 1
    fi
    if docker exec "${container_name}" /bin/sh -c 'cat /proc/[0-9]*/cmdline 2>/dev/null' |
        tr '\0' ' ' | grep -F "${database_password}" >/dev/null; then
        echo "A process in the container carries the generated PostgreSQL password in its arguments" >&2
        return 1
    fi

    replication_probe_output="$(docker exec -i --user databasus "${container_name}" /bin/sh -c '
        IFS= read -r PGPASSWORD
        export PGPASSWORD
        /usr/lib/postgresql/17/bin/psql \
            "host=localhost port=5437 user=postgres dbname=postgres replication=database" \
            -c IDENTIFY_SYSTEM
    ' <<<"${database_password}" 2>&1 || true)"
    if ! grep -Fq "number of requested standby connections exceeds" \
        <<<"${replication_probe_output}"; then
        echo "PostgreSQL did not reject a replication connection: ${replication_probe_output}" >&2
        return 1
    fi
}

assert_running_container() {
    local container_name="$1"

    assert_storage_command_saves_and_deletes_probe "${container_name}"
    assert_single_runtime_identity "${container_name}"
    assert_postgresql_runtime_authentication "${container_name}"
}

prepare_bind_tree() {
    local data_root="$1"
    local data_uid="$2"
    local data_gid="$3"
    local backup_uid="${4:-${data_uid}}"
    local backup_gid="${5:-${data_gid}}"

    mkdir -p "${data_root}"
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${data_root}:/data" "${databasus_image}" -eu -c '
        mkdir -p /data/pgdata /data/temp /data/backups
        chown "$1:$2" /data /data/pgdata /data/temp
        chown "$3:$4" /data/backups
        chmod 0770 /data /data/backups
        chmod 0700 /data/pgdata /data/temp
    ' sh "${data_uid}" "${data_gid}" "${backup_uid}" "${backup_gid}"
}

run_default_ids() {
    local container_name="databasus-storage-default-ids-${run_id}"
    local volume_name="databasus-storage-default-ids-${run_id}"
    local image_environment

    image_environment="$(docker image inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "${databasus_image}")"
    if grep -Eq '^(PUID|PGID|DATABASUS_PUID|DATABASUS_PGID|POSTGRES_PUID|POSTGRES_PGID)=' <<<"${image_environment}"; then
        echo "The image publishes a runtime identity default" >&2
        return 1
    fi

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" --volume "${volume_name}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "999" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "999" ]]
    assert_running_container "${container_name}"
}

run_custom_ids() {
    local container_name="databasus-storage-custom-ids-${run_id}"
    local volume_name="databasus-storage-custom-ids-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --env PUID=25001 \
        --env PGID=25002 \
        --volume "${volume_name}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "25001" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "25002" ]]
    assert_running_container "${container_name}"
}

run_maximum_ids() {
    local container_name="databasus-storage-maximum-ids-${run_id}"
    local volume_name="databasus-storage-maximum-ids-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --env PUID=4294967294 \
        --env PGID=4294967294 \
        --volume "${volume_name}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "4294967294" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "4294967294" ]]
    assert_running_container "${container_name}"
}

run_automatic_ids() {
    local container_name="databasus-storage-automatic-ids-${run_id}"
    local data_root="${case_root}/automatic-ids"

    prepare_bind_tree "${data_root}" 27501 27502
    start_databasus "${container_name}" --volume "${data_root}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "27501" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "27502" ]]
    assert_running_container "${container_name}"
}

run_empty_pgdata_ids() {
    local container_name="databasus-storage-empty-pgdata-ids-${run_id}"
    local data_volume_name="databasus-storage-empty-pgdata-data-${run_id}"
    local pgdata_volume_name="databasus-storage-empty-pgdata-${run_id}"

    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${pgdata_volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/chown \
        --volume "${pgdata_volume_name}:/pgdata" "${databasus_image}" \
        27501:27502 /pgdata

    start_databasus "${container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --volume "${pgdata_volume_name}:/databasus-data/pgdata"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "27501" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "27502" ]]
    assert_running_container "${container_name}"
}

run_partial_ids() {
    local container_name="databasus-storage-partial-ids-${run_id}"
    local data_root="${case_root}/partial-ids"

    prepare_bind_tree "${data_root}" 27501 27502
    start_databasus "${container_name}" \
        --env PUID=25001 \
        --volume "${data_root}:/databasus-data"

    [[ "$(docker exec "${container_name}" id -u databasus)" == "25001" ]]
    [[ "$(docker exec "${container_name}" id -g databasus)" == "27502" ]]
    assert_running_container "${container_name}"
}

run_invalid_ids() {
    local invalid_value
    local container_name

    for invalid_value in "" 0 abc 4294967295 18446744073709551617; do
        container_name="databasus-storage-invalid-id-${run_id}-${RANDOM}"
        start_failing_databasus "${container_name}" --env "PUID=${invalid_value}"
        docker logs "${container_name}" 2>&1 | grep -F "ERROR: PUID must be a non-zero decimal Linux ID" >/dev/null
    done

    container_name="databasus-storage-colliding-id-${run_id}"
    start_failing_databasus "${container_name}" --env PUID=1
    docker logs "${container_name}" 2>&1 | grep -F "belongs to account" >/dev/null
}

run_bind_root() {
    local container_name="databasus-storage-bind-${run_id}"
    local data_root="${case_root}/bind-root"

    prepare_bind_tree "${data_root}" 999 999
    start_databasus "${container_name}" --volume "${data_root}:/databasus-data"
    assert_running_container "${container_name}"
}

run_named_volume() {
    local container_name="databasus-storage-volume-${run_id}"
    local volume_name="databasus-storage-volume-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" --volume "${volume_name}:/databasus-data"
    assert_running_container "${container_name}"
}

run_split_mounts() {
    local container_name="databasus-storage-split-${run_id}"
    local data_volume_name="databasus-storage-split-data-${run_id}"
    local backup_volume_name="databasus-storage-split-backup-${run_id}"
    local pgdata_volume_name="databasus-storage-split-pgdata-${run_id}"
    local temporary_device_id
    local backup_device_id

    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${backup_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${pgdata_volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --tmpfs /databasus-data/temp:rw,nosuid,size=64m,mode=0700,uid=999,gid=999 \
        --volume "${backup_volume_name}:/databasus-data/backups" \
        --volume "${pgdata_volume_name}:/databasus-data/pgdata"

    temporary_device_id="$(docker exec "${container_name}" stat -c %d /databasus-data/temp)"
    backup_device_id="$(docker exec "${container_name}" stat -c %d /databasus-data/backups)"
    [[ "${temporary_device_id}" != "${backup_device_id}" ]]
    assert_running_container "${container_name}"
}

run_no_unix_sockets() {
    local container_name="databasus-storage-no-unix-sockets-${run_id}"
    local volume_name="databasus-storage-no-unix-sockets-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" \
        --security-opt "seccomp=${suite_directory}/no-unix-sockets.json" \
        --volume "${volume_name}:/databasus-data"
    assert_running_container "${container_name}"
}

run_metadata_changes_denied() {
    local container_name="databasus-storage-metadata-denied-${run_id}"
    local data_root="${case_root}/metadata-denied"

    prepare_bind_tree "${data_root}" 999 999
    start_databasus "${container_name}" \
        --cap-drop CHOWN \
        --cap-drop FOWNER \
        --env PUID=999 \
        --env PGID=999 \
        --volume "${data_root}:/databasus-data"
    assert_running_container "${container_name}"
}

run_read_only() {
    local container_name="databasus-storage-read-only-${run_id}"
    local data_root="${case_root}/read-only"

    prepare_bind_tree "${data_root}" 999 999
    # The Docker engine on Namespace CI runners drops the readonly key of --mount
    # and mounts the directory writable, while it honours the :ro suffix of --volume.
    start_failing_databasus "${container_name}" \
        --env PUID=999 \
        --env PGID=999 \
        --volume "${data_root}:/databasus-data:ro"
    assert_storage_permission_failure "${container_name}"
}

run_wrong_owner() {
    local container_name="databasus-storage-wrong-owner-${run_id}"
    local data_root="${case_root}/wrong-owner"

    prepare_bind_tree "${data_root}" 999 999 27501 27502
    start_failing_databasus "${container_name}" \
        --cap-drop CHOWN \
        --cap-drop FOWNER \
        --env PUID=999 \
        --env PGID=999 \
        --volume "${data_root}:/databasus-data"
    assert_storage_permission_failure "${container_name}" \
        "Failed operation: save a file through local storage:"
}

run_unwritable_data_root() {
    local container_name="databasus-storage-unwritable-root-${run_id}"
    local data_root="${case_root}/unwritable-root"

    prepare_bind_tree "${data_root}" 999 999
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${data_root}:/data" "${databasus_image}" -eu -c '
        chown 27501:27502 /data
        chmod 0555 /data
    '

    start_failing_databasus "${container_name}" \
        --cap-drop CHOWN \
        --cap-drop DAC_OVERRIDE \
        --cap-drop FOWNER \
        --env PUID=999 \
        --env PGID=999 \
        --volume "${data_root}:/databasus-data"
    assert_storage_permission_failure "${container_name}"
    docker logs "${container_name}" 2>&1 | grep -F \
        "Required operation: create, read, and remove a required file." >/dev/null
}

run_network_prerequisites() {
    [[ "$(uname -s)" == "Linux" ]]
    command -v mount.cifs >/dev/null || {
        echo "mount.cifs is required; install cifs-utils on the Docker host" >&2
        return 1
    }
    command -v mount.nfs >/dev/null || {
        echo "mount.nfs is required; install nfs-common on the Docker host" >&2
        return 1
    }

    docker image inspect "${samba_fixture_image}" >/dev/null 2>&1 || docker pull "${samba_fixture_image}" >/dev/null
    docker image inspect "${nfs_fixture_image}" >/dev/null 2>&1 || docker pull "${nfs_fixture_image}" >/dev/null
}

run_cifs() {
    local network_name="databasus-storage-cifs-${run_id}"
    local server_container_name="databasus-storage-cifs-server-${run_id}"
    local client_container_name="databasus-storage-cifs-client-${run_id}"
    local server_volume_name="databasus-storage-cifs-server-${run_id}"
    local data_volume_name="databasus-storage-cifs-data-${run_id}"
    local backup_volume_name="databasus-storage-cifs-backup-${run_id}"
    local server_address

    run_network_prerequisites
    docker network create --label "${harness_label}" "${network_name}" >/dev/null
    docker volume create --label "${harness_label}" "${server_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${data_volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${server_volume_name}:/shared" "${databasus_image}" -c \
        'chown 999:999 /shared && chmod 0770 /shared'
    docker run --detach \
        --name "${server_container_name}" \
        --label "${harness_label}" \
        --network "${network_name}" \
        --health-interval 1s \
        --env NAME=Storage \
        --env USER=storage-test \
        --env PASS=storage-test-password \
        --env UID=999 \
        --env GID=999 \
        --volume "${server_volume_name}:/shared" \
        "${samba_fixture_image}" >/dev/null
    wait_for_health "${server_container_name}"

    server_address="$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${server_container_name}")"
    docker volume create \
        --label "${harness_label}" \
        --driver local \
        --opt type=cifs \
        --opt "device=//${server_address}/Storage" \
        --opt "o=username=storage-test,password=storage-test-password,vers=3.0,uid=0,gid=999,file_mode=0660,dir_mode=0770" \
        "${backup_volume_name}" >/dev/null

    start_databasus "${client_container_name}" \
        --volume "${data_volume_name}:/databasus-data" \
        --volume "${backup_volume_name}:/databasus-data/backups"

    [[ "$(docker exec "${client_container_name}" stat -c %u /databasus-data/backups)" == "0" ]]
    [[ "$(docker exec "${client_container_name}" stat -c %g /databasus-data/backups)" == "999" ]]
    [[ "$(docker exec "${client_container_name}" stat -c %a /databasus-data/backups)" == "770" ]]
    docker exec --user databasus "${client_container_name}" /bin/sh -eu -c '
        printf fixed-mode > /databasus-data/backups/.fixed-mode-check
        test "$(stat -c %a /databasus-data/backups/.fixed-mode-check)" = 660
        rm /databasus-data/backups/.fixed-mode-check
    '
    assert_running_container "${client_container_name}"
}

run_nfs_root_squash() {
    local network_name="databasus-storage-nfs-${run_id}"
    local server_container_name="databasus-storage-nfs-server-${run_id}"
    local client_container_name="databasus-storage-nfs-client-${run_id}"
    local server_volume_name="databasus-storage-nfs-server-${run_id}"
    local pgdata_volume_name="databasus-storage-nfs-pgdata-${run_id}"
    local client_volume_name="databasus-storage-nfs-client-${run_id}"
    local server_address

    run_network_prerequisites
    docker network create --label "${harness_label}" "${network_name}" >/dev/null
    docker volume create --label "${harness_label}" "${server_volume_name}" >/dev/null
    docker volume create --label "${harness_label}" "${pgdata_volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${server_volume_name}:/nfsshare" "${databasus_image}" -c \
        'mkdir -p /nfsshare/pgdata /nfsshare/temp &&
         chown -R 999:999 /nfsshare &&
         chmod 0755 /nfsshare /nfsshare/pgdata /nfsshare/temp'
    docker run --detach \
        --name "${server_container_name}" \
        --label "${harness_label}" \
        --network "${network_name}" \
        --privileged \
        --env SHARED_DIRECTORY=/nfsshare \
        --volume "${server_volume_name}:/nfsshare" \
        --entrypoint /bin/bash \
        "${nfs_fixture_image}" -c \
        "mount -t nfsd nfsd /proc/fs/nfsd && printf 10 > /proc/fs/nfsd/nfsv4gracetime && sed -i 's/no_root_squash/root_squash/' /etc/exports && exec /usr/bin/nfsd.sh" >/dev/null
    wait_for_log "${server_container_name}" "Startup successful."

    server_address="$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${server_container_name}")"
    docker volume create \
        --label "${harness_label}" \
        --driver local \
        --opt type=nfs \
        --opt "o=addr=${server_address},rw,nfsvers=4" \
        --opt device=:/ \
        "${client_volume_name}" >/dev/null

    if docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${client_volume_name}:/export" "${databasus_image}" -c \
        'chown 1:1 /export' >/dev/null 2>&1; then
        echo "root-squashed NFS accepted a client-root ownership change" >&2
        return 1
    fi

    start_databasus "${client_container_name}" \
        --volume "${client_volume_name}:/databasus-data" \
        --volume "${pgdata_volume_name}:/databasus-data/pgdata" \
        --tmpfs /databasus-data/temp:rw,nosuid,size=64m,mode=0700,uid=999,gid=999
    assert_running_container "${client_container_name}"
    docker rm -fv "${client_container_name}" >/dev/null
    start_databasus "${client_container_name}" \
        --volume "${client_volume_name}:/databasus-data" \
        --volume "${pgdata_volume_name}:/databasus-data/pgdata" \
        --tmpfs /databasus-data/temp:rw,nosuid,size=64m,mode=0700,uid=999,gid=999
    assert_running_container "${client_container_name}"
}

get_application_database_dsn() {
    local container_name="$1"

    docker exec --user databasus "${container_name}" /bin/sh -eu -c '
        database_dsn="$(cat /dev/shm/databasus-database-dsn 2>/dev/null || true)"
        if [ -z "${database_dsn}" ]; then
            database_dsn="$(tr "\0" "\n" < /proc/1/environ | sed -n "s/^DATABASE_DSN=//p")"
        fi
        if [ -z "${database_dsn}" ]; then
            set -a
            . /.env
            set +a
            database_dsn="${DATABASE_DSN}"
        fi
        printf "%s" "${database_dsn}"
    '
}

write_upgrade_markers() {
    local container_name="$1"
    local release_tag="$2"
    local database_dsn

    database_dsn="$(get_application_database_dsn "${container_name}")"
    docker exec -i "${container_name}" /usr/lib/postgresql/17/bin/psql \
        "${database_dsn}" -v ON_ERROR_STOP=1 -v release_tag="${release_tag}" <<'SQL'
CREATE TABLE IF NOT EXISTS filesystem_upgrade_markers (
    release_tag text PRIMARY KEY
);
INSERT INTO filesystem_upgrade_markers (release_tag)
VALUES (:'release_tag')
ON CONFLICT (release_tag) DO NOTHING;
SQL

    docker exec "${container_name}" /bin/sh -eu -c '
        printf "%s" "$1" > /databasus-data/backups/filesystem-upgrade-marker
    ' sh "${release_tag}"
    docker exec --user databasus "${container_name}" /bin/sh -eu -c '
        printf "\nfilesystem-upgrade-log-marker:%s\n" "$1" >> /databasus-data/databasus.log
    ' sh "${release_tag}"
    docker exec --user databasus "${container_name}" /bin/sh -eu -c '
        queue_directory="/databasus-data/backups/wal-queue/filesystem-upgrade/$1"
        mkdir -p "${queue_directory}/pending-upload"
        printf "filesystem-upgrade-wal-marker:%s" "$1" > "${queue_directory}/pending-upload/segment"
        chmod 0700 \
            /databasus-data/backups/wal-queue \
            /databasus-data/backups/wal-queue/filesystem-upgrade \
            "${queue_directory}" \
            "${queue_directory}/pending-upload"
        chmod 0600 "${queue_directory}/pending-upload/segment"
    ' sh "${release_tag}"
}

prepare_upgrade_secret() {
    local container_name="$1"
    local release_tag="$2"
    local database_dsn
    local secret_value="filesystem-upgrade-secret-${release_tag}"

    if docker exec "${container_name}" test -f /databasus-data/secret.key; then
        docker exec "${container_name}" sha256sum /databasus-data/secret.key | cut -d' ' -f1
        return
    fi

    database_dsn="$(get_application_database_dsn "${container_name}")"
    docker exec "${container_name}" /usr/lib/postgresql/17/bin/psql \
        "${database_dsn}" -v ON_ERROR_STOP=1 \
        -c "INSERT INTO secret_keys (secret) VALUES ('${secret_value}')" >/dev/null
    printf "%s" "${secret_value}" | sha256sum | cut -d' ' -f1
}

assert_upgrade_markers() {
    local container_name="$1"
    local release_tag="$2"
    local secret_hash="$3"
    local system_identifier="$4"
    local log_size_before_update="$5"
    local database_dsn
    local log_size_after_update
    local runtime_uid

    database_dsn="$(get_application_database_dsn "${container_name}")"
    [[ "$(docker exec "${container_name}" /usr/lib/postgresql/17/bin/psql \
        "${database_dsn}" -tA -v ON_ERROR_STOP=1 \
        -c "SELECT release_tag FROM filesystem_upgrade_markers WHERE release_tag='${release_tag}'")" == "${release_tag}" ]]
    [[ "$(docker exec "${container_name}" cat /databasus-data/backups/filesystem-upgrade-marker)" == "${release_tag}" ]]
    [[ "$(docker exec "${container_name}" sha256sum /databasus-data/secret.key | cut -d' ' -f1)" == "${secret_hash}" ]]
    [[ "$(docker exec "${container_name}" /usr/lib/postgresql/17/bin/pg_controldata \
        /databasus-data/pgdata | sed -n 's/^Database system identifier: *//p')" == "${system_identifier}" ]]
    docker exec "${container_name}" grep -Fq \
        "filesystem-upgrade-log-marker:${release_tag}" /databasus-data/databasus.log
    runtime_uid="$(docker exec "${container_name}" id -u databasus)"
    [[ "$(docker exec "${container_name}" stat -c %u /databasus-data/databasus.log)" == "${runtime_uid}" ]]
    log_size_after_update="$(docker exec "${container_name}" stat -c %s /databasus-data/databasus.log)"
    ((log_size_after_update > log_size_before_update))
    docker exec --user databasus "${container_name}" /bin/sh -eu -c '
        queue_directory="/databasus-data/backups/wal-queue/filesystem-upgrade/$1"
        test "$(cat "${queue_directory}/pending-upload/segment")" = "filesystem-upgrade-wal-marker:$1"
        printf resumed > "${queue_directory}/resume-check"
        rm "${queue_directory}/resume-check"
    ' sh "${release_tag}"
    [[ "$(docker exec "${container_name}" find \
        /databasus-data/backups/wal-queue -xdev ! -user "${runtime_uid}" -print -quit)" == "" ]]
}

run_upgrade() {
    local release_tag="$1"
    local release_image="databasus/databasus:${release_tag}"
    local old_container_name="databasus-storage-upgrade-old-${release_tag}-${run_id}"
    local new_container_name="databasus-storage-upgrade-new-${release_tag}-${run_id}"
    local volume_name="databasus-storage-upgrade-${release_tag}-${run_id}"
    local secret_hash
    local system_identifier
    local log_size_before_update
    local had_socket_directory_before_update=false

    docker image inspect "${release_image}" >/dev/null 2>&1 || docker pull "${release_image}" >/dev/null
    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    docker run --detach \
        --name "${old_container_name}" \
        --label "${harness_label}" \
        --health-interval 1s \
        --health-start-period 1s \
        --volume "${volume_name}:/databasus-data" \
        "${release_image}" >/dev/null
    wait_for_health "${old_container_name}"
    write_upgrade_markers "${old_container_name}" "${release_tag}"
    secret_hash="$(prepare_upgrade_secret "${old_container_name}" "${release_tag}")"
    log_size_before_update="$(docker exec "${old_container_name}" stat -c %s /databasus-data/databasus.log)"
    system_identifier="$(docker exec "${old_container_name}" /usr/lib/postgresql/17/bin/pg_controldata \
        /databasus-data/pgdata | sed -n 's/^Database system identifier: *//p')"
    if docker exec "${old_container_name}" test -d /databasus-data/pgsocket; then
        had_socket_directory_before_update=true
    fi
    docker stop "${old_container_name}" >/dev/null

    start_databasus "${new_container_name}" --volume "${volume_name}:/databasus-data"
    assert_upgrade_markers \
        "${new_container_name}" \
        "${release_tag}" \
        "${secret_hash}" \
        "${system_identifier}" \
        "${log_size_before_update}"
    case "${release_tag}" in
        v3.56.0 | v3.60.0) [[ "${had_socket_directory_before_update}" == true ]] ;;
    esac
    if [[ "${had_socket_directory_before_update}" == true ]]; then
        docker exec "${new_container_name}" test -d /databasus-data/pgsocket
    fi
    assert_running_container "${new_container_name}"
}

run_upgrades() {
    local release_tag

    for release_tag in v3.54.0 v3.55.0 v3.55.1 v3.56.0 v3.60.0; do
        run_upgrade "${release_tag}"
    done
}

edit_stopped_volume() {
    local volume_name="$1"
    local edit_script="$2"

    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${volume_name}:/databasus-data" "${databasus_image}" -eu -c "${edit_script}"
}

create_initialized_volume() {
    local volume_name="$1"
    local container_name="${volume_name}-initializer"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${container_name}" --volume "${volume_name}:/databasus-data"
    docker rm -fv "${container_name}" >/dev/null
}

assert_log_contains() {
    local container_name="$1"
    local required_log_line="$2"

    if ! docker logs "${container_name}" 2>&1 | grep -F "${required_log_line}" >/dev/null; then
        echo "The container log lacks: ${required_log_line}" >&2
        return 1
    fi
}

assert_log_lacks() {
    local container_name="$1"
    local forbidden_log_line="$2"

    if docker logs "${container_name}" 2>&1 | grep -F "${forbidden_log_line}" >/dev/null; then
        echo "The container log contains: ${forbidden_log_line}" >&2
        return 1
    fi
}

run_postgresql_recovery() {
    local volume_name="databasus-storage-recovery-${run_id}"
    local listen_volume_name="databasus-storage-recovery-listen-${run_id}"
    local read_only_volume_name="databasus-storage-recovery-read-only-${run_id}"
    local killed_container_name="databasus-storage-recovery-killed-${run_id}"
    local replayed_container_name="databasus-storage-recovery-replayed-${run_id}"
    local reset_container_name="databasus-storage-recovery-reset-${run_id}"
    local unrecoverable_container_name="databasus-storage-recovery-unrecoverable-${run_id}"
    local unbindable_container_name="databasus-storage-recovery-unbindable-${run_id}"
    local read_only_container_name="databasus-storage-recovery-read-only-${run_id}"
    local previous_password
    local read_only_logs

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    start_databasus "${killed_container_name}" --volume "${volume_name}:/databasus-data"
    previous_password="$(get_generated_database_password "${killed_container_name}")"
    docker exec --user databasus "${killed_container_name}" /bin/sh -eu -c '
        /usr/lib/postgresql/17/bin/psql "$(cat /dev/shm/databasus-database-dsn)" -v ON_ERROR_STOP=1 \
            -c "CREATE TABLE filesystem_recovery_markers (marker text PRIMARY KEY)" \
            -c "INSERT INTO filesystem_recovery_markers VALUES ('\''survives-kill'\'')"
    ' >/dev/null
    docker kill --signal KILL "${killed_container_name}" >/dev/null
    docker rm -fv "${killed_container_name}" >/dev/null

    start_databasus "${replayed_container_name}" --volume "${volume_name}:/databasus-data"
    [[ "$(docker exec --user databasus "${replayed_container_name}" /bin/sh -eu -c '
        /usr/lib/postgresql/17/bin/psql "$(cat /dev/shm/databasus-database-dsn)" -tA \
            -c "SELECT marker FROM filesystem_recovery_markers"
    ')" == "survives-kill" ]]
    assert_log_lacks "${replayed_container_name}" "pg_resetwal"
    if can_log_in_over_loopback "${replayed_container_name}" "${previous_password}"; then
        echo "PostgreSQL accepted the password of the previous run" >&2
        return 1
    fi
    can_log_in_over_loopback "${replayed_container_name}" \
        "$(get_generated_database_password "${replayed_container_name}")"
    docker stop "${replayed_container_name}" >/dev/null
    docker rm -fv "${replayed_container_name}" >/dev/null

    edit_stopped_volume "${volume_name}" '
        find /databasus-data/pgdata/pg_wal -maxdepth 1 -type f -name "000000*" -delete
    '
    start_databasus "${reset_container_name}" --volume "${volume_name}:/databasus-data"
    assert_log_contains "${reset_container_name}" "PostgreSQL recovered successfully after WAL reset!"
    docker stop "${reset_container_name}" >/dev/null
    docker rm -fv "${reset_container_name}" >/dev/null

    edit_stopped_volume "${volume_name}" 'printf "16\n" > /databasus-data/pgdata/PG_VERSION'
    start_failing_databasus "${unrecoverable_container_name}" --volume "${volume_name}:/databasus-data"
    assert_log_contains "${unrecoverable_container_name}" \
        "ERROR: PostgreSQL could not open the database even after a WAL reset attempt."
    assert_log_lacks "${unrecoverable_container_name}" "Starting Databasus application..."

    create_initialized_volume "${listen_volume_name}"
    edit_stopped_volume "${listen_volume_name}" '
        echo "listen_addresses = '\''192.0.2.1'\''" >> /databasus-data/pgdata/postgresql.conf
    '
    start_failing_databasus "${unbindable_container_name}" --volume "${listen_volume_name}:/databasus-data"
    assert_log_contains "${unbindable_container_name}" "Setting the internal PostgreSQL password..."
    assert_log_lacks "${unbindable_container_name}" "ERROR: PostgreSQL rejected the internal password change."
    assert_log_contains "${unbindable_container_name}" "Starting PostgreSQL..."
    assert_log_contains "${unbindable_container_name}" "ERROR: PostgreSQL did not start."
    assert_log_lacks "${unbindable_container_name}" "pg_resetwal"
    assert_log_lacks "${unbindable_container_name}" "Starting Databasus application..."

    create_initialized_volume "${read_only_volume_name}"
    edit_stopped_volume "${read_only_volume_name}" '
        echo "default_transaction_read_only = on" >> /databasus-data/pgdata/postgresql.conf
    '
    start_failing_databasus "${read_only_container_name}" --volume "${read_only_volume_name}:/databasus-data"
    assert_log_contains "${read_only_container_name}" "ERROR: PostgreSQL rejected the internal password change."
    assert_log_lacks "${read_only_container_name}" "Starting PostgreSQL..."
    assert_log_lacks "${read_only_container_name}" "Starting Databasus application..."
    read_only_logs="$(docker logs "${read_only_container_name}" 2>&1)"
    if grep -Fq "ALTER ROLE postgres WITH PASSWORD" <<<"${read_only_logs}" ||
       grep -Fq "PASSWORD '" <<<"${read_only_logs}" ||
       grep -Eq '[0-9a-f]{64}' <<<"${read_only_logs}"; then
        echo "The container log contains the password statement" >&2
        return 1
    fi
}

get_cluster_state() {
    local volume_name="$1"

    docker run --rm --label "${harness_label}" \
        --entrypoint /usr/lib/postgresql/17/bin/pg_controldata \
        --volume "${volume_name}:/databasus-data" "${databasus_image}" \
        /databasus-data/pgdata | sed -n 's/^Database cluster state: *//p'
}

assert_cluster_shut_down() {
    local volume_name="$1"
    local cluster_state

    cluster_state="$(get_cluster_state "${volume_name}")"
    if [[ "${cluster_state}" != "shut down" ]]; then
        echo "The stopped cluster reports the state '${cluster_state}', not 'shut down'" >&2
        return 1
    fi
}

assert_exit_code() {
    local container_name="$1"
    local required_exit_code="$2"
    local exit_code

    exit_code="$(get_exit_code "${container_name}")"
    if [[ "${exit_code}" != "${required_exit_code}" ]]; then
        docker logs "${container_name}" >&2
        echo "The container exited with status ${exit_code}, not ${required_exit_code}" >&2
        return 1
    fi
}

start_databasus_on_new_volume() {
    local container_name="$1"

    docker volume create --label "${harness_label}" "${container_name}" >/dev/null
    start_databasus "${container_name}" --volume "${container_name}:/databasus-data"
}

check_docker_stop_shuts_down_cleanly() {
    local container_name="databasus-storage-clean-stop-${run_id}"
    local stop_started_at_us
    local stop_duration_ms

    start_databasus_on_new_volume "${container_name}"
    # The uutils date on the Namespace runners printed nanoseconds for %3N, so the clock comes
    # from bash instead.
    stop_started_at_us="${EPOCHREALTIME/./}"
    docker stop "${container_name}" >/dev/null
    stop_duration_ms="$(((${EPOCHREALTIME/./} - stop_started_at_us) / 1000))"

    if ((stop_duration_ms >= 10000)); then
        echo "docker stop took ${stop_duration_ms} ms, so the stop timeout ended the container" >&2
        return 1
    fi
    assert_exit_code "${container_name}" 0
    assert_cluster_shut_down "${container_name}"
    assert_application_exit_precedes_postgresql_shutdown "${container_name}"
}

get_first_log_line_number() {
    local container_name="$1"
    local searched_log_line="$2"

    docker logs "${container_name}" 2>&1 | grep -nF "${searched_log_line}" | head -1 | cut -d: -f1
}

assert_application_exit_precedes_postgresql_shutdown() {
    local container_name="$1"
    local application_exit_line_number
    local postgresql_shutdown_line_number

    application_exit_line_number="$(get_first_log_line_number "${container_name}" \
        "Databasus application exited with status 0")"
    postgresql_shutdown_line_number="$(get_first_log_line_number "${container_name}" \
        "received fast shutdown request")"
    if [[ -z "${application_exit_line_number}" || -z "${postgresql_shutdown_line_number}" ]] ||
       ((application_exit_line_number > postgresql_shutdown_line_number)); then
        docker logs "${container_name}" >&2
        echo "The application exit is not logged before the PostgreSQL shutdown request" >&2
        return 1
    fi
}

check_interrupt_shuts_down_cleanly() {
    local container_name="databasus-storage-clean-interrupt-${run_id}"

    start_databasus_on_new_volume "${container_name}"
    docker kill --signal INT "${container_name}" >/dev/null
    wait_for_stop "${container_name}"
    assert_cluster_shut_down "${container_name}"
}

check_application_exit_shuts_down_cleanly() {
    local container_name="databasus-storage-clean-application-exit-${run_id}"

    start_databasus_on_new_volume "${container_name}"
    send_signal_in_container "${container_name}" TERM "$(docker exec "${container_name}" pidof main)"
    wait_for_stop "${container_name}"
    assert_exit_code "${container_name}" 0
    assert_cluster_shut_down "${container_name}"
}

check_postgresql_exit_stops_container() {
    local container_name="databasus-storage-clean-postgresql-exit-${run_id}"

    start_databasus_on_new_volume "${container_name}"
    send_signal_in_container "${container_name}" KILL \
        "$(docker exec "${container_name}" head -1 /databasus-data/pgdata/postmaster.pid)"
    wait_for_exit "${container_name}"
    assert_log_contains "${container_name}" "ERROR: PostgreSQL exited with status"
    assert_log_lacks "${container_name}" "Databasus application exited with status"
}

run_clean_shutdown() {
    check_docker_stop_shuts_down_cleanly
    check_interrupt_shuts_down_cleanly
    check_application_exit_shuts_down_cleanly
    check_postgresql_exit_stops_container
}

# The loop polls without a delay, because the open step of a clean cluster lasts about 0.1 seconds.
# A single-user step started by initdb works on template1, so matching the final postgres argument
# skips it.
# shellcheck disable=SC2016 # the container's shell expands the script
readonly postgresql_freeze_script='
    target_kind="$1"
    while :; do
        for process_directory in /proc/[0-9]*; do
            mapfile -d "" -t process_arguments < "${process_directory}/cmdline" 2>/dev/null || continue
            [[ "${process_arguments[0]:-}" == /usr/lib/postgresql/17/bin/postgres ]] || continue
            process_id="${process_directory#/proc/}"
            case "${target_kind}" in
                single-user)
                    [[ "${process_arguments[1]:-}" == --single &&
                       "${process_arguments[-1]}" == postgres ]] || continue
                    ;;
                regular-server)
                    [[ "${process_arguments[1]:-}" != --single ]] || continue
                    IFS= read -r postmaster_pid < /databasus-data/pgdata/postmaster.pid 2>/dev/null || continue
                    [[ "${postmaster_pid}" == "${process_id}" ]] || continue
                    ;;
            esac
            kill -STOP "${process_id}"
            printf "%s" "${process_id}" > /tmp/frozen-postgresql-pid
            exit 0
        done
    done
'

start_databasus_with_frozen_postgresql() {
    local container_name="$1"
    local target_kind="$2"
    local frozen_pid

    docker volume create --label "${harness_label}" "${container_name}" >/dev/null
    docker run --detach \
        --name "${container_name}" \
        --label "${harness_label}" \
        --volume "${container_name}:/databasus-data" \
        "${databasus_image}" >/dev/null
    docker exec --detach "${container_name}" \
        /bin/bash -c "${postgresql_freeze_script}" freeze-postgresql "${target_kind}"

    for ((freeze_attempt = 1; freeze_attempt <= 300; freeze_attempt++)); do
        if frozen_pid="$(docker exec "${container_name}" cat /tmp/frozen-postgresql-pid 2>/dev/null)"; then
            echo "${frozen_pid}"
            return
        fi
        sleep 0.1
    done

    docker logs "${container_name}" >&2
    echo "No ${target_kind} PostgreSQL process appeared to freeze" >&2
    return 1
}

send_signal_in_container() {
    local container_name="$1"
    local signal_name="$2"
    local process_id="$3"

    docker exec "${container_name}" /bin/bash -c 'kill -"$1" "$2"' send-signal "${signal_name}" "${process_id}"
}

is_process_stopped() {
    local container_name="$1"
    local process_id="$2"

    [[ "$(docker exec "${container_name}" awk '/^State:/ { print $2 }' "/proc/${process_id}/status")" == "T" ]]
}

wait_for_log_within_two_seconds() {
    local container_name="$1"
    local required_log_line="$2"

    for ((log_attempt = 1; log_attempt <= 20; log_attempt++)); do
        if docker logs "${container_name}" 2>&1 | grep -F "${required_log_line}" >/dev/null; then
            return
        fi
        sleep 0.1
    done

    docker logs "${container_name}" >&2
    echo "The container did not log within two seconds: ${required_log_line}" >&2
    return 1
}

wait_for_kill_event() {
    local container_name="$1"
    local container_created_at

    container_created_at="$(docker inspect --format '{{.Created}}' "${container_name}")"
    for ((event_attempt = 1; event_attempt <= 50; event_attempt++)); do
        if docker events \
            --since "${container_created_at}" \
            --until "$(date +%s.%N)" \
            --filter "container=${container_name}" \
            --filter event=kill \
            --format '{{.Action}}' | grep -q .; then
            return
        fi
        sleep 0.1
    done

    echo "Docker did not report sending the stop signal" >&2
    return 1
}

# Status 143 comes from the startup stop handler, while a kill by the stop timeout gives 137.
assert_startup_ended_without_application() {
    local container_name="$1"

    assert_exit_code "${container_name}" 143
    assert_log_lacks "${container_name}" "Starting Databasus application..."
}

check_stop_during_single_user_step() {
    local container_name="databasus-storage-startup-stop-single-user-${run_id}"
    local frozen_pid
    local stop_pid

    frozen_pid="$(start_databasus_with_frozen_postgresql "${container_name}" single-user)"
    docker stop "${container_name}" >/dev/null &
    stop_pid=$!

    wait_for_log_within_two_seconds "${container_name}" "Stop requested during startup"
    if ! is_process_stopped "${container_name}" "${frozen_pid}"; then
        echo "The single-user step ended before the container reported the stop" >&2
        return 1
    fi

    send_signal_in_container "${container_name}" CONT "${frozen_pid}"
    wait "${stop_pid}"
    assert_startup_ended_without_application "${container_name}"
}

# The script may be waiting in pg_isready or psql on the frozen server, and Bash runs a trap only
# after that call returns, so this part cannot see the stop before the server resumes.
check_stop_while_regular_server_starts() {
    local container_name="databasus-storage-startup-stop-regular-server-${run_id}"
    local frozen_pid
    local stop_pid

    frozen_pid="$(start_databasus_with_frozen_postgresql "${container_name}" regular-server)"
    docker stop "${container_name}" >/dev/null &
    stop_pid=$!

    wait_for_kill_event "${container_name}"
    send_signal_in_container "${container_name}" CONT "${frozen_pid}"
    wait "${stop_pid}"
    assert_startup_ended_without_application "${container_name}"
    assert_cluster_shut_down "${container_name}"
}

run_startup_stop() {
    check_stop_during_single_user_step
    check_stop_while_regular_server_starts
}

run_legacy_directory_guard() {
    local container_name="databasus-storage-legacy-directory-${run_id}"
    local volume_name="databasus-storage-legacy-directory-${run_id}"

    docker volume create --label "${harness_label}" "${volume_name}" >/dev/null
    docker run --rm --label "${harness_label}" --entrypoint /bin/sh \
        --volume "${volume_name}:/legacy" "${databasus_image}" \
        -c 'printf legacy > /legacy/marker'
    start_failing_databasus "${container_name}" --volume "${volume_name}:/postgresus-data"
    docker logs "${container_name}" 2>&1 | grep -F "ERROR: Legacy volume detected!" >/dev/null
}

run_case() {
    local case_name="$1"

    echo "Running Docker filesystem case: ${case_name}"
    case "${case_name}" in
        build-only) docker image inspect "${databasus_image}" >/dev/null ;;
        default-ids) run_default_ids ;;
        custom-ids) run_custom_ids ;;
        maximum-ids) run_maximum_ids ;;
        automatic-ids) run_automatic_ids ;;
        empty-pgdata-ids) run_empty_pgdata_ids ;;
        partial-ids) run_partial_ids ;;
        invalid-ids) run_invalid_ids ;;
        bind-root) run_bind_root ;;
        named-volume) run_named_volume ;;
        split-mounts) run_split_mounts ;;
        no-unix-sockets) run_no_unix_sockets ;;
        metadata-changes-denied) run_metadata_changes_denied ;;
        read-only) run_read_only ;;
        wrong-owner) run_wrong_owner ;;
        unwritable-data-root) run_unwritable_data_root ;;
        network-prerequisites) run_network_prerequisites ;;
        cifs) run_cifs ;;
        nfs-root-squash) run_nfs_root_squash ;;
        upgrades) run_upgrades ;;
        postgresql-recovery) run_postgresql_recovery ;;
        clean-shutdown) run_clean_shutdown ;;
        startup-stop) run_startup_stop ;;
        legacy-directory-guard) run_legacy_directory_guard ;;
        *)
            echo "unknown Docker filesystem case: ${case_name}" >&2
            return 2
            ;;
    esac
}

# CI runs each group as a separate job, and CASE=all runs every group in this order.
readonly case_group_names=(permissions network upgrades lifecycle)
declare -rA case_groups=(
    [permissions]="build-only default-ids custom-ids maximum-ids automatic-ids empty-pgdata-ids partial-ids
        invalid-ids bind-root named-volume split-mounts no-unix-sockets metadata-changes-denied read-only
        wrong-owner unwritable-data-root"
    [network]="cifs nfs-root-squash"
    [upgrades]="upgrades"
    [lifecycle]="postgresql-recovery clean-shutdown startup-stop legacy-directory-guard"
)

run_cases() {
    local case_name

    for case_name in "$@"; do
        run_case "${case_name}"
        cleanup
        case_root="$(mktemp -d /tmp/databasus-storage-test.XXXXXXXX)"
    done
}

if [[ -n "${CASE_GROUP:-}" ]]; then
    if [[ -z "${case_groups[${CASE_GROUP}]+set}" ]]; then
        echo "unknown Docker filesystem case group: ${CASE_GROUP} (known: ${case_group_names[*]})" >&2
        exit 2
    fi
    # shellcheck disable=SC2086 # the group is a whitespace-separated list of case names
    run_cases ${case_groups[${CASE_GROUP}]}
elif [[ "${selected_case}" == all ]]; then
    for group_name in "${case_group_names[@]}"; do
        # shellcheck disable=SC2086 # the group is a whitespace-separated list of case names
        run_cases ${case_groups[${group_name}]}
    done
else
    run_case "${selected_case}"
fi

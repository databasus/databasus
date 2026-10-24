#!/bin/bash
set -e

export TMPDIR=/tmp

readonly permissions_documentation_url="https://databasus.com/advanced-config/#docker-storage-permissions"
readonly postgres_binary_directory="/usr/lib/postgresql/17/bin"

# A process started with `docker exec` inherits the container's configured environment, not
# PID 1's, so the generated credential reaches it through this file instead. The location is
# memory-backed, so the value dies with the container rather than outliving the password
# rotation of the next start. backend/internal/lib/config/config.go reads the same path.
readonly published_database_dsn_path="/dev/shm/databasus-database-dsn"

readonly runtime_phase_argument="--as-databasus"

runtime_uid=""
runtime_gid=""
postgres_pid=""
postgresql_step_pid=""
application_pid=""
internal_postgres_password=""

reject_legacy_postgresus_volume() {
    if [ ! -d /postgresus-data ] || [ -z "$(ls -A /postgresus-data 2>/dev/null)" ]; then
        return
    fi

    echo ""
    echo "=========================================="
    echo "ERROR: Legacy volume detected!"
    echo "=========================================="
    echo ""
    echo "You are using the /postgresus-data folder. It seems you changed the image name from Postgresus to Databasus without changing the volume."
    echo ""
    echo "Please either:"
    echo "  1. Switch back to image rostislavdugin/postgresus:latest (supported until ~Dec 2026)"
    echo "  2. Read the migration guide: https://databasus.com/installation/#postgresus-migration"
    echo ""
    echo "=========================================="
    exit 1
}

report_identity_error() {
    local variable_name="$1"
    local variable_value="$2"

    echo "ERROR: ${variable_name} must be a non-zero decimal Linux ID no greater than 4294967294. Received: '${variable_value}'." >&2
    echo "See ${permissions_documentation_url}" >&2
    exit 1
}

validate_linux_id() {
    local variable_name="$1"
    local variable_value="$2"
    local maximum_linux_id=4294967294
    local normalized_id

    if [[ ! "${variable_value}" =~ ^[0-9]+$ ]]; then
        report_identity_error "${variable_name}" "${variable_value}"
    fi

    normalized_id="${variable_value#"${variable_value%%[!0]*}"}"
    normalized_id="${normalized_id:-0}"
    if [ "${normalized_id}" = "0" ] ||
       ((${#normalized_id} > ${#maximum_linux_id})) ||
       { ((${#normalized_id} == ${#maximum_linux_id})) && ((10#${normalized_id} > maximum_linux_id)); }; then
        report_identity_error "${variable_name}" "${variable_value}"
    fi

    echo "${normalized_id}"
}

get_mounted_id() {
    local stat_format="$1"
    local mounted_path
    local mounted_id
    local -a mounted_paths=()

    if [ -d /databasus-data/pgdata ]; then
        mounted_paths+=(/databasus-data/pgdata)
    fi
    mounted_paths+=(/databasus-data/backups /databasus-data)

    for mounted_path in "${mounted_paths[@]}"; do
        if [ ! -e "${mounted_path}" ]; then
            continue
        fi

        mounted_id="$(stat -c "${stat_format}" "${mounted_path}")"
        if [ "${mounted_id}" != "0" ]; then
            echo "${mounted_id}"
            return
        fi
    done

    echo 999
}

resolve_runtime_id() {
    local variable_name="$1"
    local stat_format="$2"
    local variable_value

    if [ "${!variable_name+x}" = "x" ]; then
        variable_value="${!variable_name}"
        validate_linux_id "${variable_name}" "${variable_value}"
        return
    fi

    get_mounted_id "${stat_format}"
}

ensure_id_is_available() {
    local account_database="$1"
    local identity_kind="$2"
    local selected_id="$3"
    local owning_name

    owning_name="$(getent "${account_database}" "${selected_id}" | cut -d: -f1 || true)"
    if [ -z "${owning_name}" ] || [ "${owning_name}" = "databasus" ]; then
        return
    fi

    echo "ERROR: Selected ${identity_kind} ${selected_id} belongs to account '${owning_name}', not 'databasus'." >&2
    echo "Set PUID and PGID to unused IDs or fix the mounted directory ownership:" >&2
    echo "${permissions_documentation_url}" >&2
    exit 1
}

configure_runtime_identity() {
    runtime_uid="$(resolve_runtime_id PUID %u)"
    runtime_gid="$(resolve_runtime_id PGID %g)"

    ensure_id_is_available passwd UID "${runtime_uid}"
    ensure_id_is_available group GID "${runtime_gid}"

    if [ "$(id -g databasus)" != "${runtime_gid}" ]; then
        groupmod -g "${runtime_gid}" databasus
    fi
    if [ "$(id -u databasus)" != "${runtime_uid}" ]; then
        usermod -u "${runtime_uid}" databasus
    fi

    echo "Using databasus runtime identity ${runtime_uid}:${runtime_gid}."
}

generate_frontend_runtime_configuration() {
    local is_email_configured=false

    if [ -n "${SMTP_HOST:-}" ]; then
        is_email_configured=true
    fi

    echo "Generating runtime configuration..."
    cat > /app/ui/build/runtime-config.js <<JSEOF
// Runtime configuration injected at container startup
// This file is generated dynamically and should not be edited manually
window.__RUNTIME_CONFIG__ = {
  GITHUB_CLIENT_ID: '${GITHUB_CLIENT_ID:-}',
  GOOGLE_CLIENT_ID: '${GOOGLE_CLIENT_ID:-}',
  IS_EMAIL_CONFIGURED: '${is_email_configured}',
  CLOUDFLARE_TURNSTILE_SITE_KEY: '${CLOUDFLARE_TURNSTILE_SITE_KEY:-}',
  CONTAINER_ARCH: '${CONTAINER_ARCH:-unknown}'
};
JSEOF

    if [ -n "${ANALYTICS_SCRIPT:-}" ] &&
       ! grep -q "rybbit.databasus.com" /app/ui/build/index.html 2>/dev/null; then
        echo "Injecting analytics script..."
        sed -i "s#</head>#  ${ANALYTICS_SCRIPT}\
  </head>#" /app/ui/build/index.html
    fi
}

report_storage_error() {
    local storage_path="$1"
    local required_operation="$2"

    echo "ERROR: Databasus cannot write to ${storage_path} as UID ${runtime_uid} and GID ${runtime_gid}." >&2
    echo "Required operation: ${required_operation}." >&2
    echo "Set PUID and PGID or fix the mounted directory permissions:" >&2
    echo "${permissions_documentation_url}" >&2
    return 1
}

create_required_data_directories() {
    local required_directory

    for required_directory in \
        /databasus-data \
        /databasus-data/pgdata \
        /databasus-data/temp \
        /databasus-data/backups; do
        if mkdir -p "${required_directory}" 2>/dev/null ||
           gosu databasus mkdir -p "${required_directory}"; then
            continue
        fi

        report_storage_error "${required_directory}" "create the directory"
        return 1
    done
}

normalize_data_permissions() {
    chown "${runtime_uid}:${runtime_gid}" /databasus-data 2>/dev/null || true
    chmod 0770 /databasus-data 2>/dev/null || true
    chown "${runtime_uid}:${runtime_gid}" \
        /databasus-data/pgdata \
        /databasus-data/temp \
        /databasus-data/backups 2>/dev/null || true
    chmod 0700 \
        /databasus-data/pgdata \
        /databasus-data/temp 2>/dev/null || true
    chmod 0770 /databasus-data/backups 2>/dev/null || true
}

can_read_and_open_file_for_writing() {
    local checked_file="$1"

    gosu databasus test -r "${checked_file}" &&
        gosu databasus tee -a "${checked_file}" </dev/null >/dev/null
}

verify_existing_data_files() {
    local existing_data_file

    for existing_data_file in \
        /databasus-data/secret.key \
        /databasus-data/instance.json \
        /databasus-data/databasus.log; do
        if [ ! -e "${existing_data_file}" ]; then
            continue
        fi

        chown "${runtime_uid}:${runtime_gid}" "${existing_data_file}" 2>/dev/null || true
        chmod 0600 "${existing_data_file}" 2>/dev/null || true

        if can_read_and_open_file_for_writing "${existing_data_file}"; then
            continue
        fi

        report_storage_error "${existing_data_file}" "read and open the existing file for writing"
        return 1
    done
}

verify_directory_file_lifecycle() {
    local checked_directory="$1"
    local probe_path="${checked_directory}/.databasus-required-write-check-$$-$RANDOM"
    local probe_content="$2"
    local persisted_probe_content

    if ! printf "%s" "${probe_content}" |
        gosu databasus tee "${probe_path}" >/dev/null; then
        gosu databasus rm -f "${probe_path}" 2>/dev/null || true
        return 1
    fi

    persisted_probe_content="$(gosu databasus cat "${probe_path}" 2>/dev/null || true)"
    if [ "${persisted_probe_content}" != "${probe_content}" ]; then
        gosu databasus rm -f "${probe_path}" 2>/dev/null || true
        return 1
    fi

    gosu databasus rm "${probe_path}"
}

verify_wal_queue_access() {
    local wal_queue_directory=/databasus-data/backups/wal-queue
    local queue_listing
    local queue_directory
    local queue_file
    local inaccessible_queue_directory=""
    local inaccessible_queue_file=""

    if [ ! -d "${wal_queue_directory}" ]; then
        return
    fi

    chown -R "${runtime_uid}:${runtime_gid}" "${wal_queue_directory}" 2>/dev/null || true
    chmod -R u+rwX "${wal_queue_directory}" 2>/dev/null || true

    queue_listing="$(mktemp)"
    if ! gosu databasus find "${wal_queue_directory}" -type d -print0 > "${queue_listing}"; then
        rm -f "${queue_listing}"
        report_storage_error "${wal_queue_directory}" "traverse the existing WAL queue"
        return 1
    fi

    while IFS= read -r -d "" queue_directory; do
        if verify_directory_file_lifecycle "${queue_directory}" queue-check; then
            continue
        fi

        inaccessible_queue_directory="${queue_directory}"
        break
    done < "${queue_listing}"
    if [ -n "${inaccessible_queue_directory}" ]; then
        rm -f "${queue_listing}"
        report_storage_error "${inaccessible_queue_directory}" "create, read, and remove a WAL queue file"
        return 1
    fi

    if ! gosu databasus find "${wal_queue_directory}" -type f -print0 > "${queue_listing}"; then
        rm -f "${queue_listing}"
        report_storage_error "${wal_queue_directory}" "traverse the existing WAL queue"
        return 1
    fi

    while IFS= read -r -d "" queue_file; do
        if can_read_and_open_file_for_writing "${queue_file}"; then
            continue
        fi

        inaccessible_queue_file="${queue_file}"
        break
    done < "${queue_listing}"
    if [ -n "${inaccessible_queue_file}" ]; then
        rm -f "${queue_listing}"
        report_storage_error "${inaccessible_queue_file}" "read and open the existing WAL queue file for writing"
        return 1
    fi

    rm -f "${queue_listing}"
}

verify_service_directories() {
    local required_write_directory

    for required_write_directory in \
        /databasus-data \
        /databasus-data/pgdata; do
        if verify_directory_file_lifecycle "${required_write_directory}" service-check; then
            continue
        fi

        report_storage_error "${required_write_directory}" "create, read, and remove a required file"
        return 1
    done
}

prepare_and_verify_storage() {
    echo "Setting up data directory permissions..."

    create_required_data_directories
    normalize_data_permissions
    verify_existing_data_files
    verify_wal_queue_access
    verify_service_directories
    gosu databasus /app/main --test-storage
}

initialize_postgresql_cluster() {
    if [ -s /databasus-data/pgdata/PG_VERSION ]; then
        return
    fi

    echo "Initializing PostgreSQL database..."
    "${postgres_binary_directory}/initdb" \
        -D /databasus-data/pgdata \
        --encoding=UTF8 \
        --locale=C.UTF-8 \
        --username=postgres

    {
        echo "port = 5437"
        echo "listen_addresses = 'localhost'"
        echo "shared_buffers = 256MB"
        echo "max_connections = 100"
    } >> /databasus-data/pgdata/postgresql.conf
}

install_postgresql_configuration_file() {
    local configuration_file="/databasus-data/pgdata/$1"

    tee "${configuration_file}.new" >/dev/null
    chmod 0600 "${configuration_file}.new"
    mv "${configuration_file}.new" "${configuration_file}"
}

write_postgresql_authentication() {
    install_postgresql_configuration_file pg_hba.conf <<'PG_HBA'
host all all 127.0.0.1/32 scram-sha-256
host all all ::1/128 scram-sha-256
host replication all 127.0.0.1/32 reject
host replication all ::1/128 reject
PG_HBA
    install_postgresql_configuration_file pg_ident.conf </dev/null
}

# Single-user mode listens on nothing, so the cluster can be administered before any client can
# connect. The database name must be the last argument. Docker delivers a stop signal to process 1
# alone, and Bash runs a trap only after a foreground command returns, so the step runs in the
# background where the stop trap can end it.
run_postgresql_single_user() {
    local single_user_input="$1"
    local single_user_output_path="$2"
    local single_user_status=0
    shift 2

    "${postgres_binary_directory}/postgres" \
        --single \
        -D /databasus-data/pgdata \
        "$@" \
        postgres <<<"${single_user_input}" >"${single_user_output_path}" 2>&1 &
    postgresql_step_pid=$!
    wait "${postgresql_step_pid}" || single_user_status=$?
    postgresql_step_pid=""

    return "${single_user_status}"
}

# Opening the cluster replays the write-ahead log, so a failure here is the only sign of a log that
# cannot be recovered without a reset.
open_postgresql_cluster() {
    local single_user_output_path

    single_user_output_path="$(mktemp)"
    if run_postgresql_single_user "" "${single_user_output_path}"; then
        rm -f "${single_user_output_path}"
        return
    fi

    cat "${single_user_output_path}"
    rm -f "${single_user_output_path}"
    return 1
}

report_postgresql_recovery_failure() {
    echo ""
    echo "=========================================="
    echo "ERROR: PostgreSQL could not open the database even after a WAL reset attempt."
    echo "The database may be severely corrupted."
    echo ""
    echo "Options:"
    echo "  1. Delete the volume and start fresh (data loss)"
    echo "  2. Manually inspect /databasus-data/pgdata for issues"
    echo "=========================================="
    return 1
}

open_postgresql_cluster_with_recovery() {
    echo "Opening PostgreSQL database..."
    if open_postgresql_cluster; then
        return
    fi

    echo ""
    echo "=========================================="
    echo "PostgreSQL could not open the database. Attempting WAL reset recovery..."
    echo "=========================================="
    echo ""

    # A single-user backend that panics leaves its lock file behind, and pg_resetwal refuses to run
    # while one exists. No PostgreSQL process runs at this point, so the file is stale.
    rm -f /databasus-data/pgdata/postmaster.pid

    echo "Running pg_resetwal to reset WAL..."
    if ! "${postgres_binary_directory}/pg_resetwal" \
        -f /databasus-data/pgdata ||
       ! open_postgresql_cluster; then
        report_postgresql_recovery_failure
        return 1
    fi

    echo "PostgreSQL recovered successfully after WAL reset!"
}

# The statement reaches PostgreSQL as standard input, so the password never appears in the
# arguments of a process. log_min_error_statement=panic keeps a failing statement, and the password
# inside it, out of the output.
rotate_internal_postgres_password() {
    local single_user_output_path

    internal_postgres_password="$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')"

    echo "Setting the internal PostgreSQL password..."
    single_user_output_path="$(mktemp)"
    if run_postgresql_single_user \
        "ALTER ROLE postgres WITH PASSWORD '${internal_postgres_password}'" \
        "${single_user_output_path}" \
        -c exit_on_error=on \
        -c log_min_error_statement=panic \
        -c password_encryption=scram-sha-256; then
        rm -f "${single_user_output_path}"
        return
    fi

    cat "${single_user_output_path}" >&2
    rm -f "${single_user_output_path}"
    echo "ERROR: PostgreSQL rejected the internal password change." >&2
    return 1
}

start_postgresql() {
    echo "Starting PostgreSQL..."
    "${postgres_binary_directory}/postgres" \
        -D /databasus-data/pgdata \
        -p 5437 \
        -c unix_socket_directories='' \
        -c max_wal_senders=0 \
        -c password_encryption=scram-sha-256 &
    postgres_pid=$!

    echo "Waiting for PostgreSQL to be ready..."
    local postgres_readiness_attempt
    for ((postgres_readiness_attempt = 1; postgres_readiness_attempt <= 30; postgres_readiness_attempt++)); do
        if "${postgres_binary_directory}/pg_isready" \
            -h localhost -p 5437 >/dev/null 2>&1; then
            echo "PostgreSQL is ready!"
            return
        fi
        if ! kill -0 "${postgres_pid}" 2>/dev/null; then
            break
        fi
        sleep 1
    done

    echo "ERROR: PostgreSQL did not start." >&2
    return 1
}

create_metadata_database() {
    echo "Setting up database..."
    PGPASSWORD="${internal_postgres_password}" "${postgres_binary_directory}/psql" \
        -v ON_ERROR_STOP=1 \
        -h localhost \
        -p 5437 \
        -U postgres \
        -d postgres <<'SQL'
SELECT 'CREATE DATABASE databasus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'databasus')
\gexec
SQL
}

verify_postgresql_runtime_authentication() {
    if ! PGPASSWORD="${internal_postgres_password}" "${postgres_binary_directory}/psql" \
        -h localhost \
        -p 5437 \
        -U postgres \
        -d postgres \
        -c 'SELECT 1' >/dev/null; then
        echo "ERROR: Embedded PostgreSQL rejected its generated loopback credential." >&2
        return 1
    fi
}

configure_application_database_dsn() {
    if [ -n "${DATABASE_DSN+x}" ]; then
        return
    fi

    export DATABASE_DSN="host=localhost user=postgres password=${internal_postgres_password} dbname=databasus port=5437 sslmode=disable"

    (umask 077 && printf '%s\n' "${DATABASE_DSN}" > "${published_database_dsn_path}")
}

bootstrap_postgresql() {
    initialize_postgresql_cluster
    write_postgresql_authentication
    open_postgresql_cluster_with_recovery
    rotate_internal_postgres_password
    start_postgresql
    create_metadata_database
    verify_postgresql_runtime_authentication
    configure_application_database_dsn
}

reject_legacy_wal_configuration() {
    local has_backup_type_column
    local has_legacy_wal_database

    echo "Checking for legacy WAL backup configuration..."
    has_backup_type_column="$(PGPASSWORD="${internal_postgres_password}" \
        "${postgres_binary_directory}/psql" \
        -h localhost -p 5437 -U postgres -d databasus -tA \
        -c "SELECT 1 FROM information_schema.columns WHERE table_name='postgresql_databases' AND column_name='backup_type' LIMIT 1" \
        2>/dev/null || true)"

    if [ "${has_backup_type_column}" != "1" ]; then
        echo "No legacy WAL backup data detected."
        return
    fi

    has_legacy_wal_database="$(PGPASSWORD="${internal_postgres_password}" \
        "${postgres_binary_directory}/psql" \
        -h localhost -p 5437 -U postgres -d databasus -tA \
        -c "SELECT 1 FROM postgresql_databases WHERE backup_type='WAL_V1' LIMIT 1" \
        2>/dev/null || true)"

    if [ "${has_legacy_wal_database}" != "1" ]; then
        echo "No legacy WAL backup data detected."
        return
    fi

    echo ""
    echo "=========================================="
    echo "ERROR: Agent (WAL_V1) backup approach is no longer supported."
    echo "=========================================="
    echo ""
    echo "Please downgrade to version 3.42.0, remove all WAL-mode databases"
    echo "manually and then upgrade again. This safeguard exists to avoid"
    echo "corrupting already-set-up agents."
    echo ""
    echo "=========================================="
    return 1
}

wait_for_process_exit() {
    local process_id="$1"

    # A trapped signal interrupts wait before the process has exited.
    while kill -0 "${process_id}" 2>/dev/null; do
        wait "${process_id}" 2>/dev/null || true
    done
}

# A fast shutdown disconnects remaining sessions and writes a shutdown checkpoint, so the next
# start needs no crash recovery.
stop_postgresql_server() {
    kill -INT "${postgres_pid}" 2>/dev/null || true
    wait_for_process_exit "${postgres_pid}"
}

get_signal_exit_status() {
    local signal_name="$1"

    echo "$((128 + $(kill -l "${signal_name}")))"
}

# Exiting before the PostgreSQL process has ended would let the kernel kill it together with
# process 1, possibly during its shutdown checkpoint.
stop_startup() {
    local signal_name="$1"

    trap '' TERM INT
    echo "Stop requested during startup"
    if [ -n "${postgres_pid}" ]; then
        stop_postgresql_server
    elif [ -n "${postgresql_step_pid}" ]; then
        kill -TERM "${postgresql_step_pid}" 2>/dev/null || true
        wait_for_process_exit "${postgresql_step_pid}"
    fi
    exit "$(get_signal_exit_status "${signal_name}")"
}

handle_stop_signal() {
    local signal_name="$1"

    if [ -z "${application_pid}" ]; then
        stop_startup "${signal_name}"
    fi

    kill -s "${signal_name}" "${application_pid}" 2>/dev/null || true
}

install_stop_trap() {
    trap 'handle_stop_signal TERM' TERM
    trap 'handle_stop_signal INT' INT
}

# The application shuts down first, because it records interrupted backups in the metadata
# database while it stops.
supervise_application() {
    local exited_process_id=""
    local exit_status=0

    echo "Starting Databasus application..."
    ./main &
    application_pid=$!

    while [ -z "${exited_process_id}" ]; do
        exit_status=0
        wait -n -p exited_process_id "${application_pid}" "${postgres_pid}" || exit_status=$?
    done

    if [ "${exited_process_id}" = "${postgres_pid}" ]; then
        echo "ERROR: PostgreSQL exited with status ${exit_status} while Databasus was running." >&2
        kill -TERM "${application_pid}" 2>/dev/null || true
        wait_for_process_exit "${application_pid}"
        exit 1
    fi

    echo "Databasus application exited with status ${exit_status}" >&2
    stop_postgresql_server
    exit "${exit_status}"
}

prepare_container_as_root() {
    reject_legacy_postgresus_volume
    configure_runtime_identity
    generate_frontend_runtime_configuration
    prepare_and_verify_storage

    exec gosu databasus "$0" "${runtime_phase_argument}"
}

run_as_databasus() {
    bootstrap_postgresql
    reject_legacy_wal_configuration
    supervise_application
}

main() {
    install_stop_trap

    if [ "${1:-}" = "${runtime_phase_argument}" ]; then
        run_as_databasus
    else
        prepare_container_as_root
    fi
}

main "$@"

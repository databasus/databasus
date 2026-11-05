#!/usr/bin/env bash

# The bundled DB clients in assets/tools are Debian-built and link libedit.so.2, a soname Debian
# patches in downstream; Fedora ships the upstream libedit.so.0. The ABI is identical, so renaming
# via symlink is safe.
apply_libedit_shim() {
    if ldconfig -p | awk '/libedit\.so\.2 / {has_libedit = 1} END {exit !has_libedit}'; then
        return 0
    fi

    local shim_dir="${LIBEDIT_SHIM_DIR:-$HOME/.local/lib/databasus-compat}"
    local libedit=/usr/lib64/libedit.so.0

    if [[ ! -e "$libedit" ]]; then
        libedit=$(ldconfig -p | awk '/libedit\.so\.0 / {print $NF; exit}')
    fi
    if [[ -z "$libedit" ]]; then
        echo "libedit.so.0 not found - install it with: sudo dnf install libedit" >&2
        return 1
    fi

    mkdir -p "$shim_dir"
    ln -sf "$libedit" "$shim_dir/libedit.so.2"
    export LD_LIBRARY_PATH="$shim_dir${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
}

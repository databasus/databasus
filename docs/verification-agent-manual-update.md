# Updating the verification agent by hand

The verification agent normally replaces itself: it creates a temporary executable beside the running binary and renames it over that binary. This fails when the agent's account cannot write to the executable's directory. This guide brings such an agent back to work.

## When you need it

Update logs show permission or read-only filesystem errors, and the server shows no recent accepted heartbeat from the agent.

## Replace the binary

The examples use `/opt/databasus-agent/verification-agent`. Use the actual installed path instead, even when its directory differs from the working directory.

1. Stop the agent.
   - For the built-in daemon, stay in its existing working directory, so it finds its saved configuration and lock file. Run the installed binary by its absolute path as the normal agent account: `/opt/databasus-agent/verification-agent stop`.
   - For systemd or a container, stop that supervisor instead of using the daemon's `stop` command.
2. Install the binary, using an account with write access to the executable's directory. Replace the server address, and select `arch=arm64` instead of `arch=amd64` on ARM64:

   ```sh
   curl --fail --location --output /opt/databasus-agent/verification-agent.new \
     "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" && \
   chmod 755 /opt/databasus-agent/verification-agent.new && \
   mv -f /opt/databasus-agent/verification-agent.new /opt/databasus-agent/verification-agent
   ```

   For a container, replace the deployed image instead.
3. Restart the agent.
   - For the built-in daemon, run `/opt/databasus-agent/verification-agent start` from the existing working directory, as the original agent account.
   - For systemd or a container, restart through the same supervisor.

Keep `databasus-verification.json`, the token and the resource limits.

## Confirm the recovery

Check that the server shows accepted contact from the agent and a completed backup verification. A successful `version` or `status` command alone does not prove recovery.

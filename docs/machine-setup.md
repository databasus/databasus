# Preparing a machine

This page makes `mise run test` about twice as fast on a Fedora machine: 351–392 s instead of 715–766 s on the machine where it was measured. You do it once per machine. It is optional, nothing in the repository depends on it, and CI does not use it. The tools themselves are installed as [`AGENTS.md`](../AGENTS.md#development-environment) describes.

## Why tests are slow

A full run starts and removes about 700 Docker containers. For each one, Docker writes a few small files and asks the disk to confirm they are safely stored. That request is called `fsync`.

A file system is the layer that decides how files are laid out on a disk. Fedora's default file system, btrfs, answers `fsync` slowly. Docker then manages only about 2 containers per second however many tests are waiting, so the tests stand idle while the CPU has nothing to do.

There are two fixes. The first keeps temporary files in memory, and the second stops Docker waiting for those confirmations.

## Keep `/tmp` in memory

Tests write their temporary files to `/tmp`. Fedora normally keeps that folder in RAM, using a file system called tmpfs, and RAM is much faster than a disk. Check that nothing has moved it:

```bash
findmnt /tmp
```

The `FSTYPE` column should say `tmpfs`. If it says anything else, comment out the `/tmp` line in `/etc/fstab` and reboot. A full run used at most 4.4 GB there, and Docker images never go to `/tmp`.

This alone saves about a minute.

## Give Docker its own ext4 file

This is the main gain. We create one big file on the disk and format it with ext4, another file system. Linux can treat such a file as a separate disk, called a loop image. We mount it with two options that make it stop waiting for the disk:

- `nobarrier` means `fsync` no longer waits for the disk to confirm the write.
- `data=writeback` means ext4 does not hold back a file's metadata until the file's data is written.

Docker needs no new settings, because it keeps using the same paths and only what lies behind them changes. It reaches about 10 containers per second.

### The price

Those confirmations exist to protect data from a power loss. Without them, a power cut or a kernel crash can damage the file, and that affects **every project on this machine that uses Docker**, not only Databasus. You would then recreate the file and pull and build your images again, and any containers you had would be gone.

Named volumes stay on the normal disk, because they can hold data you cannot get back. Do not do this if a container on the machine keeps important data outside a named volume.

### Steps

The file is 220 GB but takes only the space it really uses. The commands below stop Docker and copy its data, which takes a while.

```bash
sudo systemctl stop docker docker.socket containerd

sudo mkdir -p /var/lib/docker-fast/mnt
sudo chattr +C /var/lib/docker-fast
sudo truncate -s 220G /var/lib/docker-fast/ext4.img
sudo mkfs.ext4 -q /var/lib/docker-fast/ext4.img
sudo cp -a /etc/fstab /etc/fstab.bak-before-docker-fast
echo '/var/lib/docker-fast/ext4.img /var/lib/docker-fast/mnt ext4 loop,nobarrier,data=writeback,noatime,nofail 0 0' | sudo tee -a /etc/fstab
sudo systemctl daemon-reload
sudo mount /var/lib/docker-fast/mnt

sudo mv /var/lib/docker/volumes /var/lib/docker-volumes
sudo rsync -aHAX --numeric-ids --sparse /var/lib/docker/ /var/lib/docker-fast/mnt/docker/
sudo rsync -aHAX --numeric-ids --sparse /var/lib/containerd/ /var/lib/docker-fast/mnt/containerd/
sudo mkdir /var/lib/docker-fast/mnt/docker/volumes

sudo tee -a /etc/fstab <<'EOF'
/var/lib/docker-fast/mnt/containerd /var/lib/containerd none bind,nofail 0 0
/var/lib/docker-fast/mnt/docker /var/lib/docker none bind,nofail 0 0
/var/lib/docker-volumes /var/lib/docker/volumes none bind,nofail 0 0
EOF

# Docker must not start before these folders are in place, or it would work with the old data.
for service in docker containerd; do
  sudo mkdir -p "/etc/systemd/system/$service.service.d"
  printf '[Unit]\nRequiresMountsFor=/var/lib/containerd /var/lib/docker /var/lib/docker/volumes\n' |
    sudo tee "/etc/systemd/system/$service.service.d/docker-fast.conf"
done

sudo systemctl daemon-reload
sudo mount -a
sudo systemctl start containerd docker
```

Check the result, then reboot and check again:

```bash
findmnt -T /var/lib/docker -o TARGET,FSTYPE,OPTIONS
docker ps -a
```

The first command should show `ext4` with `nobarrier,data=writeback`, and the second should list your containers.

### Going back

The old copies of `/var/lib/docker` and `/var/lib/containerd` stay on the disk, hidden under the new mounts, so they still take up space. Keep them until a reboot has passed.

To undo everything, run the commands below. Docker returns to its old data, and images or containers created since the move are lost with the file.

```bash
sudo systemctl stop docker docker.socket containerd
sudo umount /var/lib/docker/volumes /var/lib/docker /var/lib/containerd /var/lib/docker-fast/mnt
sudo mv /var/lib/docker-volumes /var/lib/docker/volumes

sudo cp -a /etc/fstab.bak-before-docker-fast /etc/fstab
sudo rm /etc/systemd/system/docker.service.d/docker-fast.conf /etc/systemd/system/containerd.service.d/docker-fast.conf
sudo systemctl daemon-reload
sudo systemctl start containerd docker

sudo rm -r /var/lib/docker-fast
```

# Setting up the development VM

Databasus is developed inside an Ubuntu virtual machine (VM). The code, Docker, the tools from [`mise.toml`](../mise.toml), the tests and any coding agents all run in the VM. The editor and the browser run on your own computer and connect to the VM over SSH. A remote Ubuntu server works the same way; only its address in the SSH configuration differs.

Two words are used throughout this guide:

- the **host** is your computer, the one you sit at;
- the **guest** is the Ubuntu system running inside the VM.

## Contents

1. [Why a VM](#why-a-vm)
2. [Requirements](#requirements)
3. [Choose your host](#choose-your-host) and follow its section: [Fedora](#fedora-host), [Ubuntu](#ubuntu-host), [macOS](#macos-host) or [Windows](#windows-host)
4. [Prepare the guest](#preparing-the-guest)
5. [Connect SSH and Zed](#ssh-and-zed-on-the-host)
6. [Check the VM](#checking-the-vm)
7. [When something does not work](#when-something-does-not-work)

## Why a VM

**Databasus needs Linux.** It ships its own database clients, such as `pg_dump` and `mysqldump`, in [`assets/tools`](../assets/tools/README.md). These clients are Linux programs. The backend and its tests therefore cannot run directly on Windows or macOS.

**Every Linux is different, so the project standardizes on one.** Distributions differ in their system libraries and default settings. A setup that works on one can break on another. The project therefore picks stock Ubuntu, the system CI also runs, and builds its development tooling for it.

**Any host works.** The host only shows the editor and the browser. The same guest therefore runs on any computer that can start an Ubuntu VM: Fedora, Ubuntu, macOS or Windows with WSL 2.

**The host stays clean.** Docker's network rules, the kernel settings the tests need and every test container stay inside the guest. None of them touches the host's firewall or its other projects.

```
 host: Fedora, Ubuntu, macOS or Windows
 +----------------------------------------------+
 |  Zed, browser                                |
 |     |  ssh dev, with LocalForward for        |
 |     |  3000, 4005, 5173 and optionally 5437  |
 |     v                                        |
 |  +----------------------------------------+  |
 |  | Ubuntu VM, ext4 root                   |  |
 |  |   checkout, mise tools, Docker         |  |
 |  |   pinned CPUs, capped memory           |  |
 |  +----------------------------------------+  |
 |     VM disk: LVM volume or a file outside    |
 |     btrfs copy-on-write, no snapshots        |
 +----------------------------------------------+
```

This guide ends with a guest ready for work: the project's tools installed, the commit hook in place and the app running. The [commands of daily work](../AGENTS.md#development-environment) are listed in `AGENTS.md`.

## Requirements

### Guest system

- **Ubuntu LTS, 24.04 or later.** CI runs the same family.
- **An ext4 root filesystem.** This is a hard requirement. A filesystem decides how files are laid out on the disk. ext4 is the usual Linux filesystem: it overwrites data in place and handles many small writes quickly, which is what Docker and test databases do all the time. CI runs on ext4 too. [Disk](#disk) explains why another common choice, btrfs, makes the tests slow.
- **A full VM with its own kernel, not a container.** The kernel is the core of the operating system. It manages memory, processes, the network and disks. A container borrows the host's kernel, while a VM runs its own. The [CloudNativePG tests](testing.md#cloudnativepg-physical-backup-tests) change kernel settings and load a kernel module. Inside a container those changes would land on the host.
- **Docker Engine from Docker's own apt repository, with the Compose plugin.** Docker runs rootful, that is with administrator rights. The CloudNativePG tests start privileged containers, and rootless Docker cannot start them.

### CPU

> [!IMPORTANT]
> Give the VM **at least 4** and **no more than about 75%** of the host's logical CPUs. On a machine with 32 logical CPUs that means 24 for the VM.

Both limits hold only on a host with at least 6 logical CPUs. A smaller host cannot give the VM 4 CPUs and keep its own share, so use a remote Ubuntu server instead.

A **logical CPU** is what the system counts as one processor. With SMT, described below, each physical core counts as two.

**Why 75%.** Tests load every CPU the VM has to 100% for minutes at a time. The host still needs CPU time while that happens:

- for the desktop, the browser and the editor;
- for QEMU, the program that runs the VM, which handles the VM's disk and network work on the host's CPUs, outside the VM's share.

When the VM gets every CPU, all of that waits behind the tests. The whole computer then freezes until the run ends, including the SSH connection and the editor.

**Pinning, on Linux hosts.** Pinning means the VM runs only on a fixed list of CPUs. The other CPUs then stay free for the host, even under a full test run. While the VM is idle, the host still uses its CPUs, so pinning costs nothing when no tests run. macOS and WSL let you set only the number of CPUs, not which ones. There the 75% limit alone protects the host.

**Pin whole cores.** Modern processors use SMT, also called Hyper-Threading. Each physical core runs two threads, and the system sees them as two logical CPUs, called siblings. Siblings share one core. If one sibling goes to the host and the other to the VM, both compete for the same core.

- Check which logical CPUs are siblings with `lscpu -e=CPU,CORE`: siblings show the same `CORE` number.
- On an AMD Ryzen 9 7950X, logical CPUs N and N+16 are siblings. `4-15,20-31` gives the VM 12 cores, or 24 logical CPUs. Cores 0 to 3 and their siblings 16 to 19 stay with the host.

**What the count changes.** Go runs as many tests at once as the guest has logical CPUs. The [reference machine](testing.md#reference-machine) has 16 of them. A guest with fewer CPUs, down to the minimum of 4, runs the suites more slowly but does not fail them.

### Memory

**How much.** The [reference machine](testing.md#reference-machine) has about 32 GB for a whole test run. The guest also runs the editor's remote server and the language servers. Give it at least 32 GiB, which is a hard minimum; 40 GiB is better. A host that cannot spare 32 GiB on top of its own needs is too small, so use a remote Ubuntu server instead.

> [!WARNING]
> The host needs its own memory on top of the VM's limit, for the desktop and the browser. The VM's limit and the host's needs together must fit into the physical memory. Otherwise the host starts swapping to disk and freezes, just as with too many CPUs.

**The limit is a ceiling, not a reservation.** The VM takes memory from the host only as the guest uses it:

- When a program in the guest frees memory, Incus gets it back for the host through **free page reporting**: the guest tells the host which memory pages it no longer uses.
- The exception is the **page cache**, the copies of recently read files that Linux keeps in otherwise free memory to speed up the next read. The guest counts the page cache as free, but the host sees it as used until the guest drops it.
- On macOS the VM returns memory less readily, as [the macOS section](#macos-host) describes.

### Disk

The guest's root filesystem must be ext4. The rest of this section explains why and how the host stores the VM's disk.

**What matters is fsync.** Docker and the tests write many small files and call fsync often. fsync is the call that waits until data has physically reached the disk. Databases call it on every committed transaction, so the tests make thousands of such calls. The speed of fsync therefore decides the speed of the tests.

**Why not btrfs.** btrfs, the default filesystem of Fedora, is a copy-on-write filesystem. It does not overwrite a block of data in place. It writes the new version elsewhere and then updates its index. That is cheap for ordinary files. The VM's disk, however, is one huge file that the guest rewrites in small pieces all the time. On btrfs:

- every rewrite moves a piece of the file, so the file fragments into millions of pieces;
- every fsync also has to save btrfs's own metadata;
- as a result, fsync becomes many times slower.

**Why no snapshots.** A snapshot freezes the current blocks. Every later change then copies data again. The space of deleted Docker images also stays taken while a snapshot still refers to it.

> [!NOTE]
> The VM disk lives outside btrfs copy-on-write, and the VM gets no snapshots. A broken VM is recreated from scratch rather than restored.

**Size.** Give the disk a wide margin, 200 GiB or more. Durable test fixtures write database data and WAL, and tests keep backups in temporary directories. The size is a ceiling, not space taken up front: the disk image grows only as the guest writes. [Step 3 of the Fedora host](#step-3-choose-where-the-vm-disk-lives) describes the options for each host.

### Network and ports

The development services listen inside the guest. SSH forwarding makes each of them reachable on the same port of `localhost` on the host, as if it ran there:

| Port | Service |
|---|---|
| 3000 | website |
| 4005 | backend |
| 5173 | frontend |
| 5437 | development database, only needed for a database client on the host |

The frontend calls the backend on port 4005 of the same host name it was opened under. Both ports must therefore be reachable under one name. Forwarding them to `localhost` keeps the addresses that `mise run run` prints and that the OAuth settings use.

### Access

- The guest accepts your SSH key from the host.
- **Do not forward the host's SSH agent into the guest.** The agent holds your unlocked keys. Any process in the guest, including a coding agent, could then use them on any server.
- Give the guest its own key or token with access to this repository only.

## Choose your host

| Host | VM program | CPU pinning | Returns memory to the host | Port forwarding |
|---|---|---|---|---|
| [Fedora](#fedora-host) | Incus | yes | yes, except page cache | SSH `LocalForward` |
| [Ubuntu](#ubuntu-host) | Incus | yes | yes, except page cache | SSH `LocalForward` |
| [macOS](#macos-host) | Lima | no, CPU count only | poorly | automatic |
| [Windows](#windows-host) | WSL 2 | no, CPU count only | yes, after idle | automatic |

After your host's section, continue with [Preparing the guest](#preparing-the-guest).

## Fedora host

Fedora runs the VM with [Incus](https://linuxcontainers.org/incus/docs/main/), a free manager for VMs and containers that is in Fedora's official repositories. The setup takes four steps.

### Step 1. Install Incus

Check that hardware virtualization is turned on. KVM is the part of the Linux kernel that runs VMs. The first command prints its module, and the second shows its device:

```sh
lsmod | grep kvm
ls -l /dev/kvm
```

If either prints nothing, turn on virtualization in the BIOS. AMD calls it SVM Mode, Intel calls it VT-x.

Install Incus with QEMU and the firmware the VM boots with, then give your user access to Incus:

```sh
sudo dnf install incus incus-agent qemu-kvm edk2-ovmf
sudo systemctl enable --now incus
sudo usermod -aG incus-admin "$USER"
```

Log out and in again, so the new group applies.

### Step 2. Set up the network

Create a **bridge** called `incusbr0`, a virtual network switch that connects the VM to the host. The second command connects every new VM to it:

```sh
incus network create incusbr0 ipv4.address=10.77.0.1/24 ipv4.nat=true ipv6.address=none
incus profile device add default eth0 nic network=incusbr0 name=eth0
```

The bridge gets a fixed subnet, so the VM keeps the same address and the SSH configuration never has to change. Incus could pick a subnet itself, but it fails when Docker on the host has already taken many private subnets. If `ip route` already shows `10.77.0.0/24`, choose another subnet and change the addresses in this guide to match.

firewalld is Fedora's firewall. It puts the new bridge into a zone that blocks DHCP and DNS. The VM asks for its address over DHCP, so it would get none. Move the bridge into the trusted zone:

```sh
sudo firewall-cmd --zone=trusted --change-interface=incusbr0 --permanent
sudo firewall-cmd --reload
```

**If Docker runs on the host.** Docker sets the policy of the firewall's `FORWARD` chain to drop and lets through only its own containers' traffic. The VM's traffic to the internet passes through the host as forwarded traffic, so the VM would get an address but reach nothing outside. Docker leaves its `DOCKER-USER` chain to the user. A systemd unit adds the bridge to it every time Docker starts, so the rules survive a reboot:

```sh
sudo tee /etc/systemd/system/incus-docker-forward.service >/dev/null <<'EOF'
[Unit]
Description=Let Incus bridge traffic through Docker's FORWARD drop policy
After=docker.service
PartOf=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/sh -c 'iptables -C DOCKER-USER -i incusbr0 -j ACCEPT 2>/dev/null || iptables -I DOCKER-USER -i incusbr0 -j ACCEPT'
ExecStart=/bin/sh -c 'iptables -C DOCKER-USER -o incusbr0 -j ACCEPT 2>/dev/null || iptables -I DOCKER-USER -o incusbr0 -j ACCEPT'
ExecStop=-/usr/sbin/iptables -D DOCKER-USER -i incusbr0 -j ACCEPT
ExecStop=-/usr/sbin/iptables -D DOCKER-USER -o incusbr0 -j ACCEPT

[Install]
WantedBy=docker.service
EOF
sudo systemctl daemon-reload
sudo systemctl enable --now incus-docker-forward.service
sudo iptables -S DOCKER-USER
```

The last command must list both rules. Docker does not need a restart. Podman does not change the firewall this way and needs no unit.

### Step 3. Choose where the VM disk lives

Inside the VM the disk is always ext4: the guest formats it itself, whatever the host uses. The pool decides only how the host stores that disk.

Check the host's filesystem with `findmnt -no FSTYPE /`, then pick the first option that fits:

| Your situation | Pool | fsync speed |
|---|---|---|
| a free partition or a second disk | LVM | almost bare disk |
| the root is ext4 | `dir` | fast |
| the root is btrfs, no free partition | `dir` without copy-on-write | much better than plain btrfs, slower than LVM |

**Option A: an LVM pool on a free partition or a second disk.** LVM, the Logical Volume Manager, is the standard Linux way to slice a partition into virtual partitions. They are called logical volumes and can be resized. Incus creates one logical volume per VM. The VM writes straight to it as to its own disk, with no host filesystem in between.

> [!CAUTION]
> Creating the pool erases everything on the partition.

```sh
incus storage create vms lvm source=/dev/<partition> lvm.vg_name=incus
```

**Option B: a `dir` pool on an ext4 root.** In a `dir` pool each VM disk is an ordinary file. On ext4 that file is sparse: it takes space only for the parts the guest has written.

```sh
incus storage create vms dir
```

**Option C: a `dir` pool in a btrfs subvolume without copy-on-write.**

- A subvolume is a separately managed directory tree of btrfs. Snapshot tools such as snapper take snapshots of the root but skip other subvolumes. A separate subvolume therefore keeps the disk image out of the snapshots.
- `chattr +C` turns copy-on-write off for every new file in the directory. After it, `lsattr -d` must show the letter `C`.

```sh
sudo btrfs subvolume create /var/lib/incus-vms
sudo chattr +C /var/lib/incus-vms
lsattr -d /var/lib/incus-vms
incus storage create vms dir source=/var/lib/incus-vms
```

### Step 4. Create the VM

Ubuntu's cloud images run **cloud-init** on first boot. This tool applies initial settings from a short file. Here it installs a few packages and adds your SSH key to the `ubuntu` user. Create a key with `ssh-keygen -t ed25519` if you have none.

Before running the commands, adjust the CPU list from [CPU](#cpu) and the memory to your machine.

```sh
cat > /tmp/dev-cloud-init.yaml <<EOF
#cloud-config
package_update: true
packages: [openssh-server, git, curl, build-essential]
ssh_authorized_keys:
  - $(cat ~/.ssh/id_ed25519.pub)
EOF

incus init images:ubuntu/24.04/cloud dev --vm -s vms \
  -c limits.cpu=4-15,20-31 \
  -c limits.memory=40GiB \
  -c boot.autostart=true \
  -c cloud-init.user-data="$(cat /tmp/dev-cloud-init.yaml)" \
  -d root,size=200GiB

incus config device override dev eth0 ipv4.address=10.77.0.10
incus start dev
incus exec dev -- cloud-init status --wait
```

| Option | What it does |
|---|---|
| `--vm` | creates a full VM rather than a container |
| `limits.cpu=4-15,20-31` | pins the VM to these CPUs. With a plain number, such as `24`, the VM would move across all host CPUs, and none would stay reserved for the host |
| `limits.memory=40GiB` | the most memory the VM may take |
| `boot.autostart=true` | starts the VM together with the host |
| `ipv4.address=10.77.0.10` | reserves the address on the bridge, so `ssh dev` always reaches this VM |
| `size=200GiB` | the largest the disk may grow |

**Optional: faster disk at the cost of safety.** Letting QEMU skip the guest's fsync calls makes Docker and builds noticeably faster:

```sh
incus config device set dev root io.cache=unsafe
```

> [!WARNING]
> After a sudden power loss or a host crash, the guest's filesystem may be damaged and the VM has to be recreated. An ordinary shutdown or reboot is safe. Turn this on only if the VM holds nothing you cannot recreate.

**Later changes.**

- Memory changes on a running VM: `incus config set dev limits.memory=48GiB`.
- The CPU list changes only while the VM is stopped: `incus stop dev`, then `incus config set dev limits.cpu=...` and `incus start dev`.
- To start over, run `incus delete -f dev` and repeat this step.

Continue with [Preparing the guest](#preparing-the-guest).

## Ubuntu host

An Ubuntu host also runs the VM with Incus. Follow the [Fedora host](#fedora-host) steps with three differences:

1. **Packages.** Install them with apt: `sudo apt install incus qemu-system`. Ubuntu's own package can lag far behind Incus releases. The [Incus installation guide](https://linuxcontainers.org/incus/docs/main/installing/) names the repository with current ones.
2. **Firewall.** Ubuntu's firewall is ufw rather than firewalld. Allow DHCP, DNS and the VM's internet traffic on the bridge:

   ```sh
   sudo ufw allow in on incusbr0
   sudo ufw route allow in on incusbr0
   sudo ufw route allow out on incusbr0
   ```

3. **Disk.** Ubuntu's root filesystem is usually ext4, so option B fits. Use an LVM pool if a free partition exists.

## macOS host

macOS has no Linux kernel, so every option runs a Linux VM. None of them pins CPUs: you set only how many CPUs the VM gets. Keep to the 75% limit from [CPU](#cpu): on a Mac with 16 cores, give the VM 12.

### Lima (recommended)

Lima with Apple's virtualization framework meets every requirement: a full VM with its own kernel and an ext4 disk.

```sh
limactl start --name dev --vm-type=vz --cpus 12 --memory 40 --disk 200 --mount-none template:ubuntu-lts
limactl shell dev
```

- **Ports.** Lima forwards the guest's ports to `localhost` on the Mac by itself. The SSH forwarding described below is optional.
- **Memory.** The framework returns freed memory to macOS poorly, so set the limit close to what the guest really needs.
- **Home directory.** `--mount-none` keeps the Mac's home directory out of the guest.
- **Docker.** Install it in the guest as [Preparing the guest](#preparing-the-guest) describes, rather than using a Lima template with rootless Docker.
- **SSH.** Lima writes an SSH configuration for each VM. Add `Include ~/.lima/*/ssh.config` as the first line of `~/.ssh/config` and connect to `lima-dev`.

### OrbStack (frontend and website only)

OrbStack is easier to use and returns memory to macOS well. Its Linux machines, however, are containers that share one kernel. Their root filesystem may also not be ext4; check it with `findmnt -no FSTYPE /` inside the machine. OrbStack suits frontend and website work. The backend tests need the requirements above.

## Windows host

WSL 2 runs Ubuntu in a lightweight VM with its own Linux kernel and an ext4 virtual disk. It meets the requirements without another VM program.

1. Install Ubuntu: `wsl --install -d Ubuntu-24.04`.
2. Set the limits in `%UserProfile%\.wslconfig`:

   ```ini
   [wsl2]
   memory=40GB
   processors=24

   [experimental]
   autoMemoryReclaim=gradual
   ```

3. Restart WSL so the limits apply: `wsl --shutdown`.

What to know about WSL:

- **CPU.** `processors` is the number of CPUs. WSL cannot pin them, so keep to the 75% limit from [CPU](#cpu).
- **Memory.** `autoMemoryReclaim` gives the guest's page cache back to Windows once the guest has been idle for a while.
- **Ports.** WSL forwards the guest's ports to `localhost` on Windows by itself. The SSH forwarding described below is optional.
- **Docker.** Install Docker Engine inside the Ubuntu distribution as [Preparing the guest](#preparing-the-guest) describes, rather than Docker Desktop's WSL integration.
- **Checkout location.** Keep it in the distribution's own filesystem, such as `~/databasus`, not under `/mnt/c`. Every access to a file on the Windows drive crosses the VM boundary and is slow.
- **Kernel module.** WSL ships its own kernel, so check that `sudo modprobe br_netfilter` succeeds before running the CloudNativePG tests.

## Preparing the guest

Run these steps inside the guest, in this order. Open a shell there:

| Host | Command |
|---|---|
| Fedora or Ubuntu | `ssh ubuntu@10.77.0.10` or `incus exec dev -- su - ubuntu` |
| macOS | `limactl shell dev` |
| Windows | `wsl -d Ubuntu-24.04`; your user is the one you created at installation rather than `ubuntu` |

### Client libraries

The bundled database clients in [`assets/tools`](../assets/tools/README.md) were built for Debian. They need a few system libraries; install them as CI does. `libpq5` comes from the PostgreSQL repository, as in the runtime image, because the `psql` of the newest bundled major needs a newer `libpq5` than Ubuntu ships. Ubuntu no longer ships the old library names `libncurses.so.5` and `libtinfo.so.5`. The last two commands add them as links to the current versions, which work the same way:

```sh
sudo apt-get update
sudo apt-get install -y postgresql-common
sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
sudo apt-get install -y dpkg-dev libedit2 libncurses6 libtinfo6 libpq5 libmariadb3 libgnutls30 zstd netcat-openbsd
library_directory="/usr/lib/$(dpkg-architecture -qDEB_HOST_MULTIARCH)"
sudo ln -sf "${library_directory}/libncurses.so.6" "${library_directory}/libncurses.so.5"
sudo ln -sf "${library_directory}/libtinfo.so.6" "${library_directory}/libtinfo.so.5"
```

### Docker

1. Install Docker Engine and the Compose plugin from Docker's apt repository, following [Docker's guide for Ubuntu](https://docs.docker.com/engine/install/ubuntu/).
2. Add your user to the `docker` group, so it can use Docker without `sudo`, then log out and in again:

   ```sh
   sudo usermod -aG docker "$USER"
   ```

3. Check that a container starts:

   ```sh
   docker run --rm hello-world
   ```

### inotify limits for CloudNativePG tests

The [CloudNativePG suite](testing.md#cloudnativepg-physical-backup-tests) runs Kubernetes as k3s, a lightweight Kubernetes, inside Docker containers. Kubernetes watches files through **inotify**, a kernel mechanism that notifies a program when a file changes. The kernel limits how many such watchers one user may hold. Its default of 128 instances may not cover several k3s containers at once. CI raises both limits before the suite runs.

Set the same values in the guest and keep them across reboots:

```sh
sudo tee /etc/sysctl.d/90-databasus-k3s.conf >/dev/null <<'EOF'
fs.inotify.max_user_instances = 1024
fs.inotify.max_user_watches = 1048576
EOF
sudo sysctl --system
```

If the guest already has higher values, keep them.

### Bridge netfilter for CloudNativePG tests

Kubernetes routes a request to a service through firewall rules in iptables. Traffic between the pods of one k3s container passes through a virtual network bridge. The kernel applies firewall rules to bridged traffic only while the `br_netfilter` module is loaded.

Without the module, name lookups inside the cluster time out and CloudNativePG never adds the replica. k3s cannot load the module from inside its container. A test fixture stops at startup when the module is missing.

Load it in the guest now and at every boot:

```sh
sudo modprobe br_netfilter
echo br_netfilter | sudo tee /etc/modules-load.d/br_netfilter.conf
```

### mise and the checkout

mise installs the exact tool versions the project pins, such as Go, Node and pnpm. Install it and make it available in two ways:

```sh
curl https://mise.run | sh
echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc
echo 'export PATH="$HOME/.local/bin:$HOME/.local/share/mise/shims:$PATH"' >> ~/.profile
exec bash -l
```

- **`.bashrc`** serves the terminals you open yourself.
- **`.profile`** serves programs that start without a terminal, such as the editor's language servers and its tasks. It adds mise's shims to their path; a shim is a small launcher for one tool. Without it the editor cannot find Go or Node.

Clone the repository and install the project's tools from inside it:

```sh
git clone https://github.com/databasus/databasus.git ~/databasus
cd ~/databasus
mise install --locked --include-task-tools
```

`--locked` installs exactly the versions recorded in `mise.lock`, so every command resolves to the pinned version.

Install the commit hook once per clone:

```sh
hk install --mise
```

The hook comes from [`hk.pkl`](../hk.pkl). It checks formatting and lint on every commit without changing files. Worktrees of the clone share it.

### Settings

The app reads its settings from [`.env.example`](../.env.example). An optional `.env` next to it overrides them key by key, so it holds only the keys you change, and a fresh worktree needs no copy.

### First start

```sh
mise run run
```

The command starts the development database, the test fixtures, the backend, the frontend and the website. On first start it also installs any pinned tools it needs that are still missing. Open `http://localhost:5173` on the host once [SSH and Zed](#ssh-and-zed-on-the-host) are set up.

## SSH and Zed on the host

On macOS, Lima's own SSH entry `lima-dev` replaces the one below. On Windows, WSL forwards the ports itself. On Fedora and Ubuntu, add the VM to `~/.ssh/config` on the host. Each `LocalForward` line makes one guest port reachable on the same port of the host's `localhost` while the SSH connection is open. The ports come from [Network and ports](#network-and-ports):

```
Host dev
  HostName 10.77.0.10
  User ubuntu
  IdentityFile ~/.ssh/id_ed25519
  ServerAliveInterval 30
  LocalForward 3000 127.0.0.1:3000
  LocalForward 4005 127.0.0.1:4005
  LocalForward 5173 127.0.0.1:5173
  LocalForward 5437 127.0.0.1:5437
```

Good to know:

- **`127.0.0.1` instead of `localhost`.** `localhost` can resolve to the IPv6 address `::1`, where the service may not listen.
- **`bind: Address already in use`.** A second `ssh dev` prints it while the first one is still connected. The warning is harmless: the first connection already forwards the ports.
- **A recreated VM.** It has a new host key. SSH refuses to connect until you remove the old one with `ssh-keygen -R 10.77.0.10`.

Open the project in Zed. Zed uses the system `ssh`, so it picks up the `dev` entry with its forwards:

```sh
zed ssh://dev/~/databasus
```

## Checking the VM

| What to check | How | Expected result |
|---|---|---|
| Prerequisites | `mise run doctor` in the guest | the command exits with code 0; each failed line names the section of this guide that fixes it |
| CPUs | `nproc` in the guest | the number of CPUs given to the VM |
| Guest filesystem | `findmnt -no FSTYPE /` in the guest | `ext4` |
| Ports | `mise run run` in the guest, `http://localhost:5173` on the host | the frontend opens and you can sign in |
| Tests | `mise run test` in the guest | every suite passes |

## When something does not work

| Symptom | Cause | Fix |
|---|---|---|
| The VM gets no address | the host firewall blocks DHCP on the bridge | check the firewalld zone or the ufw rules from the host sections |
| The VM gets an address other than the reserved one | the reservation is tied to the VM's MAC address, the hardware address of its network card. The guest asks for an address under another identifier | set `dhcp-identifier: mac` for the interface in the guest's netplan configuration and run `sudo netplan apply` |
| The VM loses its network after Docker is installed on the host | Docker on the host makes the firewall drop forwarded traffic. The VM's traffic is forwarded traffic | add the unit from [If Docker runs on the host](#step-2-set-up-the-network), or keep Docker off the host |
| The host stays short of memory after a test run | the guest holds the memory as page cache | run `sync; echo 1 \| sudo tee /proc/sys/vm/drop_caches` in the guest |
| The VM disk keeps growing after Docker images are removed | the host learns about freed space only after TRIM. This command tells a disk which blocks are no longer used, and Ubuntu runs it weekly | run `docker system prune -af` and `sudo fstrim -av` in the guest |

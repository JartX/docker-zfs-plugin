# Docker ZFS Volume Plugin

A Docker volume plugin for creating persistent volumes as dedicated ZFS
datasets. This plugin is designed to be simple, robust, and easy to
configure.

## Installation

Installation depends on your operating system. Both NixOS (declarative)
and systemd-based systems like Ubuntu (manual) are supported.

### Prerequisites

Before you begin, you must create a parent ZFS dataset that the plugin
will use to store volumes.

``` bash
# 1. Choose a path for your volume data, e.g., /docker
sudo mkdir -p /docker/volumes

# 2. Create the parent ZFS dataset. 
#    Its mountpoint should match the path you chose above.
sudo zfs create -o compression=on -o mountpoint=/docker <your-pool>/docker
```

Note: Compression is not mandatory but is recommended as it's
computationally cheap and can save a significant amount of space.

### Method 1: NixOS

Add the docker-zfs.nix module to your system configuration and enable
the service.

``` nix
# /etc/nixos/configuration.nix
services.dockerZfsPlugin = {
  enable = true;
  volumeBasePath = "/docker";      # Must match the mountpoint from prerequisites
  rootDataset = "<your-pool>/docker"; # The parent dataset you created
};
```

Then, rebuild your system:

``` bash
sudo nixos-rebuild switch
```

### Method 2: Ubuntu / systemd-based systems

Compile the binary:

``` bash
# From the source code directory
go build -o docker-zfs-plugin .
```

Install the binary:

``` bash
sudo install docker-zfs-plugin /usr/local/bin/
```

------------------------------------------------------------------------

## Step 3: 🔧 Create the systemd Service

Create the `/etc/systemd/system/docker-zfs-plugin.service` file and
paste the following content:

``` ini
[Unit]
Description=Docker ZFS Volume Plugin
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=root
Restart=on-failure
ExecStart=/usr/local/bin/docker-zfs-plugin --root-dataset=rpool/docker --volume-base=/docker
DeviceAllow=/dev/zfs rw

[Install]
WantedBy=multi-user.target
```

------------------------------------------------------------------------

## Step 4: ▶️ Enable the Service

Reload systemd, enable the service, and start it:

``` bash
sudo systemctl daemon-reload
sudo systemctl enable --now docker-zfs-plugin.service
sudo systemctl status docker-zfs-plugin.service
```

------------------------------------------------------------------------

## Usage

Note: The driver name for this plugin is `docker-zfs-plugin`. Created
volumes will have a mountpoint under the path you defined in
`volumeBasePath` (e.g., `/docker/volumes/<volume_name>`).

### 1. Create a simple volume:

``` bash
$ docker volume create -d docker-zfs-plugin testVolume
testVolume

$ docker volume ls
DRIVER                VOLUME NAME
docker-zfs-plugin     testVolume
local                 localTestvolume
```

### 2. Create a volume with ZFS attributes:

ZFS attributes can be passed as driver options using the `-o` flag.

``` bash
$ docker volume create -d docker-zfs-plugin -o compression=on -o dedup=on testVolumeWithOpts
testVolumeWithOpts
```

⚠️ I don't advise using the `dedup` option unless you know its
implications, but it is available.\
The `mountpoint` option is **forbidden**, as the driver manages all
mountpoints under a single base path.

### 3. Create a volume under a different parent dataset:

If you want to use a root dataset other than the one specified at
startup, use the `driver_zfsRootDataset` option.

``` bash
$ docker volume create -d docker-zfs-plugin -o driver_zfsRootDataset="<other-pool>/test" testVolume2
testVolume2
```

This will create the dataset `<other-pool>/test/testVolume2`, which will
be mounted under `/docker/volumes/testVolume2`.

------------------------------------------------------------------------

## Docker Compose

The plugin can be used in `docker-compose.yml` files similar to other
volume plugins:

``` yaml
volumes:
  my-data:
    driver: docker-zfs-plugin
    driver_opts:
      driver_zfsAutosnapshot:hourly: "true"
      driver_zfsAutosnapshot:daily: "true"
```

------------------------------------------------------------------------

## Breaking API Changes

I make no guarantees about the compatibility with previous versions of
this plugin.

📋 Docker ZFS Plugin
This visual guide shows you the steps to set up the ZFS volume plugin for Docker on NixOS and on systemd-based systems like Ubuntu.

🛠️ Block 1: Prerequisites
Ensure you have the following environment ready before you start:

Docker and ZFS installed and operational.

A ZFS Dataset created to host Docker volumes.

Example: sudo zfs create rpool/docker

Go (Golang) installed (this is only required for the manual installation on Ubuntu).

🐧 Block 2: Workflow for NixOS (Declarative)
Step 1: 📄 Add the Module
Save the docker-zfs.nix file in a folder within your configuration, such as /etc/nixos/modules/.

Step 2: ⚙️ Update configuration.nix
Import the module and enable the service with your specific configuration.

# /etc/nixos/configuration.nix

{ config, pkgs, ... }:

{
  imports = [ 
    ./modules/docker-zfs.nix  # <-- Import the new module
  ];

  # ...
  
  services.dockerZfsPlugin = {
    enable = true;
    volumeBasePath = "/docker";      # Base path on the host filesystem
    rootDataset = "rpool/docker";    # Parent ZFS dataset
  };

  virtualisation.docker.enable = true;
}

Step 3: 🚀 Deploy the Configuration
Apply the changes to rebuild your NixOS system.

sudo nixos-rebuild switch

📦 Block 3: Workflow for Ubuntu (Manual with systemd)
Step 1: 🔨 Compile the Binary
Navigate to the source code folder and run the build command.

go build -o docker-zfs-plugin .

Step 2: 📂 Install the Binary
Move the executable to a standard system path.

sudo install docker-zfs-plugin /usr/local/bin/

Step 3: 🔧 Create the systemd Service
Create the /etc/systemd/system/docker-zfs-plugin.service file and paste the following content.

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

Step 4: ▶️ Enable the Service
Reload systemd, enable the service, and start it.

sudo systemctl daemon-reload
sudo systemctl enable --now docker-zfs-plugin.service
sudo systemctl status docker-zfs-plugin.service

✅ Block 4: Final Result
Once completed, Docker will use ZFS to manage its volumes, allowing you to leverage all the advantages of ZFS such as snapshots, clones, and optimized performance.

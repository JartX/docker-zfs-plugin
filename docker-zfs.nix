{ config, pkgs, lib, ... }:

let
  cfg = config.services.dockerZfsPlugin;

  dockerZfsPluginPkg = pkgs.buildGoModule rec {
    pname = "docker-zfs-plugin";
    version = "master";

    src = pkgs.fetchFromGitHub {
      owner = "JartX";
      repo = "docker-zfs-plugin";
      rev = "change_default_volume";
      sha256 = "1zicy86xaq7f9p70y1dm9hw5k493vf6lscwyj0jwsqn5dm1l0kfq";
    };

    vendorHash = "sha256-57Hri7EDczLSXkL/1EJmbZOXnXirs0gNbqbwXdgFQXs=";
    subPackages = [ "." ];

    postInstall = ''
      mv $out/bin/docker-volume-zfs-plugin $out/bin/$pname
    '';

    meta = with lib; {
      description = "Docker volume plugin that creates ZFS datasets for volumes";
      license = licenses.mit;
      maintainers = [];
      platforms = platforms.linux;
    };
  };
in
{
  options.services.dockerZfsPlugin = {
    enable = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Enable the docker-zfs volume plugin service";
    };

    package = lib.mkOption {
      type = lib.types.package;
      default = dockerZfsPluginPkg;
      description = "Package providing the docker volume zfs plugin binary";
    };

    volumeBasePath = lib.mkOption {
      type = lib.types.str;
      default = "/docker";
      description = "Base path on the host filesystem where volume mountpoints and state will be stored.";
    };
    
    rootDataset = lib.mkOption {
      type = lib.types.str;
      example = "rpool/docker";
      description = ''
        The root ZFS dataset to manage volumes under.
        This is passed to the --root-dataset flag.
      '';
    };

    extraArgs = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [];
      description = "Extra CLI args to pass to the plugin binary";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.docker-zfs-plugin = {
      description = "Docker ZFS volume plugin";
      after = [ "docker.service" ];
      requires = [ "docker.service" ];
      wantedBy = [ "multi-user.target" ];
      path = [ pkgs.zfs ];
      serviceConfig = {
        Type = "simple";
        ExecStart = ''
          ${cfg.package}/bin/docker-zfs-plugin \
            --root-dataset ${cfg.rootDataset} \
            --volume-base ${cfg.volumeBasePath} \
            ${lib.concatStringsSep " " cfg.extraArgs}
        '';
        Restart = "on-failure";
        RestartSec = "5s";
        User = "root";
        DeviceAllow = [ "/dev/zfs rw" ];
      };

      preStart = ''
        mkdir -p ${cfg.volumeBasePath}/volumes
        chown root:root ${cfg.volumeBasePath}
      '';
    };
  };
}

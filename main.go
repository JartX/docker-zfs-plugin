package main

import (
	zfsdriver "docker-volume-zfs-plugin/zfs"
	"flag"
	"log/slog"
	"os"
	"strconv"

	"github.com/coreos/go-systemd/activation"
	"github.com/docker/go-plugins-helpers/volume"
)

func main() {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	fh, err := os.OpenFile("/docker/docker_zfs.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		println("Failed to create file handler:", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewTextHandler(fh, &slog.HandlerOptions{
		Level: lvl,
	}))

	debug := os.Getenv("DEBUG")
	if ok, _ := strconv.ParseBool(debug); ok {
		lvl.Set(slog.LevelDebug)
	}

	rootDataset := flag.String("root-dataset", "", "The root ZFS dataset to manage volumes under (e.g., rpool/docker)")
	volumeBase := flag.String("volume-base", "/docker", "The base path for volumes and state file")
	flag.Parse()

	if *rootDataset == "" {
		logger.Error("The --root-dataset flag is required")
		os.Exit(1)
	}

	d, err := zfsdriver.NewZfsDriver(logger, *rootDataset, *volumeBase)
	if err != nil {
		logger.Error("Failed to create ZFS driver", slog.Any("err", err))
		os.Exit(1)
	}

	h := volume.NewHandler(d)

	listeners, _ := activation.Listeners()
	if len(listeners) == 0 {
		logger.Debug("launching volume handler.")
		err = h.ServeUnix("/run/docker/plugins/docker-zfs-plugin.sock", 0)
	} else if len(listeners) == 1 {
		l := listeners[0]
		logger.Debug("launching volume handler", "listener", l.Addr().String())
		err = h.Serve(l)
	} else {
		logger.Warn("driver does not support multiple sockets")
	}

	if err != nil {
		logger.Error("Failed to serve volume handler", slog.Any("err", err))
		os.Exit(1)
	}
}

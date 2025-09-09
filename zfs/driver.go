package zfsdriver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.comcom/clinta/go-zfs"
	"github.com/docker/go-plugins-helpers/volume"
)

type VolumeProperties struct {
	DatasetFQN string `json:"datasetFQN"`
}

type ZfsDriver struct {
	volume.Driver

	volumes            map[string]VolumeProperties
	log                *slog.Logger
	defaultRootDataset string
}

const (
	propagatedMountPath = "/var/lib/docker/plugins/pluginHash/propagated-mount/"
	hostRootPath        = propagatedMountPath + "../../../../../.."
	volumeBase          = "/docker"
	statePath           = volumeBase + "/state.json"
)

func NewZfsDriver(logger *slog.Logger, rootDataset string) (*ZfsDriver, error) {
	if !zfs.DatasetExists(rootDataset) {
		return nil, fmt.Errorf("root dataset '%s' does not exist", rootDataset)
	}

	zd := &ZfsDriver{
		volumes:            make(map[string]VolumeProperties),
		log:                logger,
		defaultRootDataset: rootDataset,
	}
	zd.log.Info("Creating ZFS Driver", "rootDataset", rootDataset)

	err := zd.loadDatasetState()
	if err != nil {
		return nil, err
	}

	return zd, nil
}

func (zd *ZfsDriver) loadDatasetState() error {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			zd.log.Debug("No initial state found")
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &zd.volumes); err != nil {
			return err
		}
	}
	return nil
}

func (zd *ZfsDriver) saveDatasetState() {
	data, err := json.Marshal(zd.volumes)
	if err != nil {
		zd.log.Error("Cannot save dataset state", slog.Any("err", err), "Volumes", zd.volumes)
		return
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		zd.log.Error("Cannot write state path file", slog.Any("err", err), "StatePath", statePath)
	}
}

func (zd *ZfsDriver) Create(req *volume.CreateRequest) error {
	zd.log.Debug("Create", "Request", req)
	if req.Options == nil {
		req.Options = make(map[string]string)
	}

	zfsDatasetName := ""
	if req.Options["driver_zfsRootDataset"] != "" {
		zfsDatasetName = req.Options["driver_zfsRootDataset"] + "/" + req.Name
		delete(req.Options, "driver_zfsRootDataset")
	} else {
		zfsDatasetName = zd.defaultRootDataset + "/volumes/" + req.Name
	}
	if zfs.DatasetExists(zfsDatasetName) {
		return errors.New("volume already exists")
	}

	zd.log.Debug("zfsDatasetName", zfsDatasetName)

	if req.Options["mountpoint"] != "" {
		zd.log.Error("mountpoint option is not supported")
		return errors.New("mountpoint option is not supported")
	}
	req.Options["mountpoint"] = volumeBase + "/volumes/" + req.Name

	zd.log.Debug("mountpoint", req.Options["mountpoint"])

	_, err := zfs.CreateDatasetRecursive(zfsDatasetName, req.Options)
	if err != nil {
		zd.log.Error("Cannot create ZFS volume", slog.Any("err", err),
			"zfsDatasetName", zfsDatasetName,
			"Options", req.Options)
		return err
	}

	zd.volumes[req.Name] = VolumeProperties{DatasetFQN: zfsDatasetName}

	zd.saveDatasetState()

	return err
}

func (zd *ZfsDriver) List() (*volume.ListResponse, error) {
	zd.log.Debug("List")
	var vols []*volume.Volume

	for volName := range zd.volumes {
		vol, err := zd.getVolume(volName)
		if err != nil {
			zd.log.Error("Failed to get dataset info", slog.Any("err", err), "Volume Name", volName)
			continue
		}
		vols = append(vols, vol)
	}

	return &volume.ListResponse{Volumes: vols}, nil
}

func (zd *ZfsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	zd.log.Debug("Get", "Request", req)

	v, err := zd.getVolume(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.GetResponse{Volume: v}, nil
}

func (zd *ZfsDriver) scopeMountPath(mountpath string) string {
	return hostRootPath + mountpath
}

func (zd *ZfsDriver) getVolume(name string) (*volume.Volume, error) {
	volProps, ok := zd.volumes[name]
	if !ok {
		zd.log.Error("Volume not found", "name", name)
		return nil, errors.New("volume not found")
	}

	ds, err := zfs.GetDataset(volProps.DatasetFQN)
	if err != nil {
		return nil, err
	}

	mp, err := ds.GetMountpoint()
	if err != nil {
		return nil, err
	}
	mp = zd.scopeMountPath(mp)

	ts, err := ds.GetCreation()
	if err != nil {
		zd.log.Error("Failed to get creation property from zfs dataset", slog.Any("err", err), "Volume name", name)
		return &volume.Volume{Name: name, Mountpoint: mp}, nil
	}

	return &volume.Volume{Name: name, Mountpoint: mp, CreatedAt: ts.Format(time.RFC3339)}, nil
}

func (zd *ZfsDriver) getMP(name string) (string, error) {
	volProps, ok := zd.volumes[name]
	if !ok {
		zd.log.Error("Volume not found", "name", name)
		return "", errors.New("volume not found")
	}

	ds, err := zfs.GetDataset(volProps.DatasetFQN)
	if err != nil {
		return "", err
	}

	mp, err := ds.GetMountpoint()
	if err != nil {
		return "", err
	}

	mp = zd.scopeMountPath(mp)

	return mp, nil
}

func (zd *ZfsDriver) Remove(req *volume.RemoveRequest) error {
	zd.log.Debug("Remove", "Request", req)
	volProps, ok := zd.volumes[req.Name]
	if !ok {
		zd.log.Error("Volume not found", "name", req.Name)
		return errors.New("volume not found")
	}

	ds, err := zfs.GetDataset(volProps.DatasetFQN)
	if err != nil {
		return err
	}

	err = ds.Destroy()
	if err != nil {
		return err
	}

	delete(zd.volumes, req.Name)

	zd.saveDatasetState()

	return nil
}

func (zd *ZfsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	zd.log.Debug("Path", "Request", req)

	mp, err := zd.getMP(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.PathResponse{Mountpoint: mp}, nil
}

func (zd *ZfsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	zd.log.Debug("Mount", "Request", req)

	mp, err := zd.getMP(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.MountResponse{Mountpoint: mp}, nil
}

func (zd *ZfsDriver) Unmount(req *volume.UnmountRequest) error {
	zd.log.Debug("Unmount", "Request", req)

	return nil
}

func (zd *ZfsDriver) Capabilities() *volume.CapabilitiesResponse {
	zd.log.Debug("Capabilities")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}

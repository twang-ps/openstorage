package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/libopenstorage/openstorage/pkg/storageops"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

var diskLabels = map[string]string{
	"source": "openstorage-test",
	"foo":    "bar",
	"Test":   "UPPER_CASE",
}

func RunTest(
	drivers map[string]storageops.Ops,
	diskTemplates map[string]map[string]interface{},
	t *testing.T) {
	for _, d := range drivers {
		name(t, d)

		for _, template := range diskTemplates[d.Name()] {
			disk := create(t, d, template)
			fmt.Printf("Created disk: %v\n", disk)
			diskID := id(t, d, disk)
			require.NotEmpty(t, diskID, "disk ID should not be empty")
			snapshot(t, d, diskID)
			tags(t, d, diskID)
			enumerate(t, d, diskID)
			inspect(t, d, diskID)
			attach(t, d, diskID)
			devicePath(t, d, diskID)
			teardown(t, d, diskID)
			fmt.Printf("Tore down disk: %v\n", disk)
		}
	}
}

func name(t *testing.T, driver storageops.Ops) {
	name := driver.Name()
	require.NotEmpty(t, name, "driver returned empty name")
}

func create(t *testing.T, driver storageops.Ops, template interface{}) interface{} {
	d, err := driver.Create(template, nil)
	require.NoError(t, err, "failed to create disk")
	require.NotNil(t, d, "got nil disk from create api")

	return d
}

func id(t *testing.T, driver storageops.Ops, disk interface{}) string {
	id, err := driver.GetDeviceID(disk)
	require.NoError(t, err, "failed to get disk ID")
	require.NotEmpty(t, id, "got empty disk name/ID")
	return id
}

func snapshot(t *testing.T, driver storageops.Ops, diskName string) {
	snap, err := driver.Snapshot(diskName, true)
	if err == storageops.ErrNotSupported {
		return
	}

	require.NoError(t, err, "failed to create snapshot")
	require.NotEmpty(t, snap, "got empty snapshot from create API")

	snapID, err := driver.GetDeviceID(snap)
	require.NoError(t, err, "failed to get snapshot ID")
	require.NotEmpty(t, snapID, "got empty snapshot name/ID")

	err = driver.SnapshotDelete(snapID)
	require.NoError(t, err, "failed to delete snapshot")
}

func tags(t *testing.T, driver storageops.Ops, diskName string) {
	err := driver.ApplyTags(diskName, diskLabels)
	if err == storageops.ErrNotSupported {
		return
	}

	require.NoError(t, err, "failed to apply tags to disk")

	tags, err := driver.Tags(diskName)
	require.NoError(t, err, "failed to get tags for disk")
	require.Len(t, tags, 3, "invalid number of labels found on disk")

	err = driver.RemoveTags(diskName, diskLabels)
	require.NoError(t, err, "failed to remove tags from disk")

	tags, err = driver.Tags(diskName)
	require.NoError(t, err, "failed to get tags for disk")
	require.Len(t, tags, 0, "invalid number of labels found on disk")

	err = driver.ApplyTags(diskName, diskLabels)
	require.NoError(t, err, "failed to apply tags to disk")
}

func enumerate(t *testing.T, driver storageops.Ops, diskName string) {
	disks, err := driver.Enumerate([]*string{&diskName}, diskLabels, storageops.SetIdentifierNone)
	if err == storageops.ErrNotSupported {
		return
	}

	require.NoError(t, err, "failed to enumerate disk")
	require.Len(t, disks, 1, "enumerate returned invalid length")

	// enumerate with invalid labels
	randomStr := uuid.New()
	randomStr = strings.Replace(randomStr, "-", "", -1)
	invalidLabels := map[string]string{
		fmt.Sprintf("key%s", randomStr): fmt.Sprintf("val%s", randomStr),
	}
	disks, err = driver.Enumerate([]*string{&diskName}, invalidLabels, storageops.SetIdentifierNone)
	require.NoError(t, err, "failed to enumerate disk")
	require.Len(t, disks, 0, "enumerate returned invalid length")
}

func inspect(t *testing.T, driver storageops.Ops, diskName string) {
	disks, err := driver.Inspect([]*string{&diskName})
	if err == storageops.ErrNotSupported {
		return
	}

	require.NoError(t, err, "failed to inspect disk")
	require.Len(t, disks, 1, fmt.Sprintf("inspect returned invalid length: %d", len(disks)))
}

func attach(t *testing.T, driver storageops.Ops, diskName string) {
	devPath, err := driver.Attach(diskName)
	require.NoError(t, err, "disk attach returned error")
	require.NotEmpty(t, devPath, "disk attach returned empty devicePath")

	mappings, err := driver.DeviceMappings()
	require.NoError(t, err, "get device mappings returned error")
	require.NotEmpty(t, mappings, "received empty device mappings")

	err = driver.DetachFrom(diskName, driver.InstanceID())
	require.NoError(t, err, "disk DetachFrom returned error")

	devPath, err = driver.Attach(diskName)
	require.NoError(t, err, "disk attach returned error")
	require.NotEmpty(t, devPath, "disk attach returned empty devicePath")

	mappings, err = driver.DeviceMappings()
	require.NoError(t, err, "get device mappings returned error")
	require.NotEmpty(t, mappings, "received empty device mappings")
}

func devicePath(t *testing.T, driver storageops.Ops, diskName string) {
	devPath, err := driver.DevicePath(diskName)
	require.NoError(t, err, "get device path returned error")
	require.NotEmpty(t, devPath, "received empty devicePath")
}

func teardown(t *testing.T, driver storageops.Ops, diskID string) {
	err := driver.Detach(diskID)
	require.NoError(t, err, "disk detach returned error")

	time.Sleep(3 * time.Second)

	err = driver.Delete(diskID)
	require.NoError(t, err, "failed to delete disk")
}

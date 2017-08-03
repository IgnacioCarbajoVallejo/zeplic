// Package lib contains: cleaner.go - commands.go - consul.go - destroy.go - snapshot.go - sync.go - take.go - tracker.go - uuid.go
//
// UUID sets an uuid to the snapshot
// Search snapshot name from its uuid and vice versa
//
package lib

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nfrance-conseil/zeplic/tools"
	"github.com/IgnacioCarbajoVallejo/go-zfs"
	"github.com/pborman/uuid"
)

// UUID asigns a new uuid
func UUID(snap *zfs.Dataset) error {
	id := uuid.New()
	err := snap.SetProperty(":uuid", id)
	return err
}

// ReceiveUUID asigns an uuid received to snapshot
func ReceiveUUID(id string, SnapshotName string, DestDataset string) error {
	check := tools.Before(SnapshotName, "@")
	var name string
	if check == DestDataset {
		name = SnapshotName
	} else {
		name = strings.Replace(SnapshotName, check, DestDataset, -1)
	}
	ds, err := zfs.GetDataset(name)
	if err != nil {
		w.Err("[ERROR > lib/uuid.go:35] it was not possible to get the snapshot '"+ds.Name+"'.")
	} else {
		err = ds.SetProperty(":uuid", id)
		if err != nil {
			w.Err("[ERROR > lib/uuid.go:39] it was not possible to set the property ':uuid = "+id+"'.")
		}
	}
	return err
}

// SearchName searchs the name of snapshot from its uuid
func SearchName(uuid string) string {
	var snapshot string
	search := fmt.Sprintf("zfs get -rHp -t snapshot -o name,value :uuid | awk '{if ($2 == \"%s\") print $1}'", uuid)
	cmd, err := exec.Command("sh", "-c", search).Output()
	if err != nil {
		w.Err("[ERROR > lib/uuid.go:51] it was not possible to execute the command 'zfs get :uuid'.")
	} else {
		out := bytes.Trim(cmd, "\x0A")
		snapshot = string(out)
	}
	return snapshot
}

// SearchUUID searchs the uuid of snapshot from its name
func SearchUUID(snap *zfs.Dataset) string {
	uuid, err := snap.GetProperty(":uuid")
	if err != nil {
		w.Err("[ERROR > lib/uuid.go:63] it was not possible to find the uuid of the snapshot '"+snap.Name+"'.")
	}
	return uuid
}

// Source returns if a snapshot has the status local or received
func Source(uuid string) string {
	var source string
	search := fmt.Sprintf("zfs get -rHp -t snapshot -o value,source :uuid | awk '{if ($1 == \"%s\") print $2}'", uuid)
	cmd, err := exec.Command("sh", "-c", search).Output()
	if err != nil {
		w.Err("[ERROR > lib/uuid.go:74] it was not possible to execute the command 'zfs get :uuid'.")
	} else {
		out := bytes.Trim(cmd, "\x0A")
		source = string(out)
	}
	return source
}

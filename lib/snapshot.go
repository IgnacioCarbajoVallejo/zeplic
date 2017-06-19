// Package lib contains: commands.go - snapshot.go - uuid.go - written.go
//
// Snapshot makes the structure of snapshot's names
//
package lib

import (
	"fmt"
	"strings"
	"time"

	"github.com/IgnacioCarbajoVallejo/go-zfs"
)

// After gets substring after a string
func After(value string, a string) string {
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

// Before gets substring before a string
func Before(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	return value[0:pos]
}

// Reverse gets substring
func Reverse(value string, a string) string {
	pos := strings.Index(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

// DatasetName returns the dataset name of snapshot
func DatasetName(SnapshotName string) string {
	dataset := Before(SnapshotName, "@")
	return dataset
}

// SnapName defines the name of the snapshot: NAME_yyyy-Month-dd_HH:MM:SS
func SnapName(name string) string {
	year, month, day := time.Now().Date()
	hour, min, sec := time.Now().Clock()
	snapDate := fmt.Sprintf("%s_%d-%s-%02d_%02d:%02d:%02d", name, year, month, day, hour, min, sec)
	return snapDate
}

// SnapBackup defines the name of a backup snapshot: BACKUP_from_yyyy-Month-dd
func SnapBackup(dataset string) string {
	// Get the older snapshot
	list, _ := zfs.Snapshots(dataset)
	oldSnapshot := list[0].Name

	// Get date
	rev := Reverse(oldSnapshot, "_")
	date := Before(rev, "_")
	backup := fmt.Sprintf("%s_%s", "BACKUP_from", date)
	return backup
}

// Renamed returns true if a snapshot was renamed
func Renamed(SnapshotReceived string, SnapshotToCheck string) bool {
	received := After(SnapshotReceived, "@")
	toCheck := After(SnapshotToCheck, "@")
	if received == toCheck {
		return false
	}
	return true
}

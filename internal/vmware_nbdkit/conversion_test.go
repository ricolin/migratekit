package vmware_nbdkit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vexxhost/migratekit/internal/target"
	"github.com/vexxhost/migratekit/internal/vmware"
)

type conversionTarget struct {
	target.Target
	changeID string
	writeErr error
}

func (t *conversionTarget) WriteChangeID(_ context.Context, id *vmware.ChangeID) error {
	if t.writeErr != nil {
		return t.writeErr
	}
	t.changeID = id.Value
	return nil
}

func TestConversionInvalidatesCheckpointOnFailure(t *testing.T) {
	for _, exitCode := range []string{"0", "22"} {
		t.Run("exit="+exitCode, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nfor arg; do disk=$arg; done\ntest \"$LIBGUESTFS_BACKEND\" = direct || exit 77\nprintf converted > \"$disk\"\nexit " + exitCode + "\n"
			if err := os.WriteFile(filepath.Join(dir, "virt-v2v-in-place"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
			t.Setenv("LIBGUESTFS_BACKEND", "parent-backend")
			disk := filepath.Join(dir, "disk")
			target := &conversionTarget{changeID: "snapshot/1"}
			server := &NbdkitServer{Servers: &NbdkitServers{VddkConfig: &VddkConfig{}}}
			err := server.convertTarget(context.Background(), target, disk)
			if (err == nil) != (exitCode == "0") {
				t.Fatalf("unexpected conversion error: %v", err)
			}
			if target.changeID != "" {
				t.Fatal("conversion left a stale checkpoint")
			}
			data, readErr := os.ReadFile(disk)
			if readErr != nil || string(data) != "converted" {
				t.Fatalf("converter did not modify disk: %q, %v", data, readErr)
			}
			if os.Getenv("LIBGUESTFS_BACKEND") != "parent-backend" {
				t.Fatal("conversion modified parent environment")
			}
		})
	}
}

func TestConversionStopsIfCheckpointCannotBeInvalidated(t *testing.T) {
	disk := filepath.Join(t.TempDir(), "disk")
	want := errors.New("metadata update failed")
	target := &conversionTarget{changeID: "snapshot/1", writeErr: want}
	server := &NbdkitServer{Servers: &NbdkitServers{VddkConfig: &VddkConfig{}}}
	if err := server.convertTarget(context.Background(), target, disk); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if _, err := os.Stat(disk); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("conversion ran after checkpoint update failed")
	}
	if target.changeID != "snapshot/1" {
		t.Fatal("failed metadata update changed checkpoint")
	}
}

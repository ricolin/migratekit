package vmware_nbdkit

import (
	"context"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/target"
	"github.com/vexxhost/migratekit/internal/vmware"
)

func (s *NbdkitServer) convertTarget(ctx context.Context, t target.Target, path string) error {
	// Conversion can modify the volume even when it fails. Its contents no
	// longer correspond to a VMware CBT checkpoint once conversion starts.
	if err := t.WriteChangeID(ctx, &vmware.ChangeID{}); err != nil {
		return err
	}
	log.Info("Running virt-v2v-in-place")
	args := []string{"-i", "disk", path}
	if s.Servers.VddkConfig.Debug {
		args = append([]string{"-v", "-x"}, args...)
	}
	cmd := exec.Command("virt-v2v-in-place", args...)
	cmd.Env = append(os.Environ(), "LIBGUESTFS_BACKEND=direct")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// +build linux,!nokmem

package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall" // for Errno type only

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"golang.org/x/sys/unix"
)

const cgroupKernelMemoryLimit = "memory.kmem.limit_in_bytes"

func EnableKernelMemoryAccounting(path string) error {
	// Check if kernel memory is enabled
	// We have to limit the kernel memory here as it won't be accounted at all
	// until a limit is set on the cgroup and limit cannot be set once the
	// cgroup has children, or if there are already tasks in the cgroup.
	for _, i := range []int64{1, -1} {
		if err := setKernelMemory(path, i); err != nil {
			return err
		}
	}
	return nil
}

func setKernelMemory(path string, kernelMemoryLimit int64) error {
	if path == "" {
		return fmt.Errorf("no such directory for %s", cgroupKernelMemoryLimit)
	}
	if !cgroups.PathExists(filepath.Join(path, cgroupKernelMemoryLimit)) {
		// kernel memory is not enabled on the system so we should do nothing
		return nil
	}
	if err := ioutil.WriteFile(filepath.Join(path, cgroupKernelMemoryLimit), []byte(strconv.FormatInt(kernelMemoryLimit, 10)), 0700); err != nil {
		// Check if the error number returned by the syscall is "EBUSY"
		// The EBUSY signal is returned on attempts to write to the
		// memory.kmem.limit_in_bytes file if the cgroup has children or
		// once tasks have been attached to the cgroup
		if pathErr, ok := err.(*os.PathError); ok {
			if errNo, ok := pathErr.Err.(syscall.Errno); ok {
				if errNo == unix.EBUSY {
					return fmt.Errorf("failed to set %s, because either tasks have already joined this cgroup or it has children", cgroupKernelMemoryLimit)
				}
			}
		}
		return fmt.Errorf("failed to write %v to %v: %v", kernelMemoryLimit, cgroupKernelMemoryLimit, err)
	}
	return nil
}

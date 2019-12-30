package internals

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func humanReadableBytes(count uint64) string {
	bytes := float64(count)
	units := []string{"bytes", "KiB", "MiB", "GiB", "TiB", "PiB"}
	for _, unit := range units {
		if bytes < 1024 {
			return fmt.Sprintf(`%.02f %s`, bytes, unit)
		}
		bytes /= 1024
	}
	return fmt.Sprintf(`%.02f EiB`, bytes)
}

func isPermissionError(err error) bool {
	return errors.Is(err, os.ErrPermission)
}

// determineDepth determines the filepath depth of the given filepath.
// For example `a/b` returns 1 and `d/c/b/a` returns 3.
func determineDepth(path string) uint32 {
	// NOTE  This implementation is presumably very inaccurate.
	//       But there is no cross-platform way in golang to do this.
	return uint32(strings.Count(path, string(filepath.Separator))) - 1
}

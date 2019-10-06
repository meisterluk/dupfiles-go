package internals

import (
	"errors"
	"fmt"
	"os"
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

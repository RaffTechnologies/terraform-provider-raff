package provider

import (
	"fmt"
	"strconv"
)

// itoa is a thin alias for strconv.Itoa, used to set int IDs as Terraform
// resource string IDs (volumes, snapshots, backup schedules — anything with
// integer primary keys per the API spec).
func itoa(n int) string {
	return strconv.Itoa(n)
}

// atoi parses a Terraform resource ID back into an int, with a clear
// error message if the state is corrupt (shouldn't happen — itoa is the
// only writer).
func atoi(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid integer ID %q in state: %w", s, err)
	}
	return n, nil
}

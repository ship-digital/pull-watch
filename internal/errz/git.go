package errz

import "fmt"

// ErrNoUpstreamBranch is returned when there is no upstream branch configured
var ErrNoUpstreamBranch = fmt.Errorf("no upstream branch configured")

package buildinfo

import "runtime"

var (
    Version   = "dev"     // e.g., git tag or short SHA
    Commit    = "none"    // short SHA
    BuildTime = "unknown" // RFC3339 UTC
    Go        = runtime.Version()
)
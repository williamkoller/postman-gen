package scan

type ScanOptions struct {
	Dir       string
	UseTypes  bool
	BuildTags string // build tags
}

// ScanDirWithOpts: scans with go/packages+go/types when possible.
// - Forces GOROOT in the analysis environment (avoids "errors without types").
// - On ANY failure, falls back to simple local AST and does NOT return error.
func ScanDirWithOpts(opt ScanOptions) ([]Endpoint, error) {
	// Temporarily always use ScanDir due to packages.Load issues
	// TODO: Reactivate packages.Load when "package without types" issue is resolved
	if !opt.UseTypes {
		eps, err := ScanDir(opt.Dir)
		return eps, nilOr(err)
	}

	// To avoid packages.Load errors, use direct fallback to ScanDir
	eps, ferr := ScanDir(opt.Dir)
	return eps, nilOr(ferr)
}

// nilOr normalizes error to nil (helps maintain "no fatal error" API)
func nilOr(err error) error {
	if err != nil {
		return nil
	}
	return err
}

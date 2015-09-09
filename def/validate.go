package def

// Convenience method that calls all forumla validation.
// Modifies the formula.
func ValidateAll(job *Formula) {
	ValidateBasic(job)
	ValidateConvenience(job)
}

// Checks a formula for irrecoverable errors.
// Will NOT modify the formula, with the exception of correcting uninstantiated variables.
func ValidateBasic(job *Formula) {
	// Note, we don't require non-zero outputs, for obvious reasons :)
	if len(job.Inputs) < 1 {
		panic(ValidationError.New("Formula needs at least one input"))
	}

	if job.Inputs[0].MountPath == "" {
		job.Inputs[0].MountPath = "/"
	} else if job.Inputs[0].MountPath != "/" {
		panic(ValidationError.New("First formula input must be mounted to /"))
	}

	if job.Action.Env == nil {
		job.Action.Env = map[string]string{}
	}
	if job.Action.Entrypoint == nil {
		job.Action.Entrypoint = []string{}
	}
	if job.Action.Cwd == "" {
		job.Action.Cwd = "/"
	}
}

// Modifies a formula with a few tweaks that make them more convenient for human-generated input.
// * If no environment PATH was specified, set the PATH to "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
//   * To disable, set job.Action.Env["PATH"] to any string (including "") as desired.
// * Sets the entrypoint to "/bin/true" if none was specified
//   * To disable, set job.Action.Entrypoint
func ValidateConvenience(job *Formula) {
	// Add a basic PATH if none exists
	if _, ok := job.Action.Env["PATH"]; !ok {
		job.Action.Env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	// Assume a trivial command if none
	if len(job.Action.Entrypoint) < 1 {
		job.Action.Entrypoint = []string{"/bin/true"}
	}
}

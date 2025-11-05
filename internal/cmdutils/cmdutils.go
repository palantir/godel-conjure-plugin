package cmdutils

import (
	"os/exec"

	"github.com/palantir/pkg/safejson"
	"github.com/pkg/errors"
)

// GetCommandOutputAsJSON runs the provided exec.Cmd, unmarshals the command's stdout as JSON into a value of type T,
// and returns the unmarshaled value. Returns an error if an error occurs while running the command or unmarshaling the
// output.
func GetCommandOutputAsJSON[T any](cmd *exec.Cmd) (T, error) {
	// needed to return zero value in case of error
	var returnVal T

	stdout, err := RunCommand(cmd)
	if err != nil {
		return returnVal, err
	}

	if err := safejson.Unmarshal(stdout, &returnVal); err != nil {
		return returnVal, errors.Wrapf(err, "failed to unmarshal output as JSON")
	}
	return returnVal, nil
}

// RunCommand runs the provided exec.Cmd and returns the result of cmd.Output() if the returned error is nil. If the
// returned error is non-nil, returns a wrapped error that includes the command's stdout and stderr.
func RunCommand(cmd *exec.Cmd) ([]byte, error) {
	stdoutOutput, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdoutOutput, errors.Wrapf(err, "failed to execute %v\nstdout:\n%s\nstderr:\n%s", cmd.Args, string(stdoutOutput), string(exitErr.Stderr))
		}
		return stdoutOutput, errors.Wrapf(err, "failed to execute %v\nstdout:\n%s", cmd.Args, string(stdoutOutput))
	}
	return stdoutOutput, nil
}

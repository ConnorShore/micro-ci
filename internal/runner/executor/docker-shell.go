package executor

import (
	"fmt"
	"os"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerShellExecutorOpts struct {
	ExecutorOpts
	ContainerID string
}

type DockerShellExecutor struct {
	client *client.Client
}

func (e *DockerShellExecutor) Execute(opts DockerShellExecutorOpts, onStdOut func(line string)) error {
	cmd := append([]string{"sh", "-c"}, makeSingleLineScript(opts.Script))
	execOpts := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          common.VariablesMapToSlice(opts.Vars),
		Cmd:          cmd,
	}

	execCreateResp, err := e.client.ContainerExecCreate(opts.ctx, opts.ContainerID, execOpts)
	if err != nil {
		return fmt.Errorf("failed to execute script [%v] in container: %v", opts.Script, err)
	}

	execResp, err := e.client.ContainerExecAttach(opts.ctx, execCreateResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("error attaching to exec instance: %v", err)
	}
	defer execResp.Close()

	// Copy output to os.Stdout and os.Stderr
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, execResp.Reader)
	if err != nil {
		return fmt.Errorf("error reading from exec output: %v", err)
	}

	// Inspect exit code to verify successful exit
	inspectResp, err := e.client.ContainerExecInspect(opts.ctx, execCreateResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %v", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("step exited with non-zero status code: %d", inspectResp.ExitCode)
	}
	return nil
}

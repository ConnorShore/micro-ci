package executor

import (
	"fmt"
	"os"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerShellExecutor struct {
	Executor
}

func NewDockerShellExecutor() Executor {
	return &DockerShellExecutor{}
}

func (e *DockerShellExecutor) Execute(opts ExecutorOpts, onStdOut func(line string)) error {
	fmt.Printf("Executing command in docker shell: %v\n", opts.Script)

	// TODO: SEe which approach is better and get log streaming working with it

	// cmd := exec.CommandContext(opts.Ctx, "docker", "exec", environmentId, "sh", "-c", makeSingleLineScript(opts.Script))
	// cmd.Env = common.VariablesMapToSlice(opts.Vars)

	// // Set a process group ID to ensure cleanup of child processes if the context is canceled.
	// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// // Capture stdout and stderr from the `docker exec` command.
	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return err
	// }
	// stderr, err := cmd.StderrPipe()
	// if err != nil {
	// 	return err
	// }

	// // Merge stdout and stderr to process them together.
	// multiReader := io.MultiReader(stdout, stderr)

	// if err := cmd.Start(); err != nil {
	// 	return err
	// }

	// // Use a scanner to read the output line by line and stream it back.
	// scanner := bufio.NewScanner(multiReader)
	// for scanner.Scan() {
	// 	onStdOut(scanner.Text())
	// }

	// // Wait for the command to finish and return its result.
	// return cmd.Wait()

	cmd := append([]string{"sh", "-c"}, makeSingleLineScript(opts.Script))
	execOpts := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          common.VariablesMapToSlice(opts.Vars),
		Cmd:          cmd,
	}

	client, ok := opts.Client.(*client.Client)
	if !ok {
		return fmt.Errorf("opts.client is not of type *client.Client")
	}

	execCreateResp, err := client.ContainerExecCreate(opts.Ctx, opts.EnvironmentId, execOpts)
	if err != nil {
		return fmt.Errorf("failed to execute script [%v] in container: %v", opts.Script, err)
	}

	execResp, err := client.ContainerExecAttach(opts.Ctx, execCreateResp.ID, container.ExecAttachOptions{})
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
	inspectResp, err := client.ContainerExecInspect(opts.Ctx, execCreateResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %v", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("step exited with non-zero status code: %d", inspectResp.ExitCode)
	}
	return nil
}

package executor

import (
	"bufio"
	"io"
	"os/exec"
	"syscall"
)

type DockerShellExecutor struct {
	Executor
}

func NewDockerShellExecutor() Executor {
	return &DockerShellExecutor{}
}

func (e *DockerShellExecutor) Execute(opts ExecutorOpts, onStdOut func(line string)) error {
	commandVars := []string{"exec"}
	for key, val := range opts.Vars {
		ev := key + "=" + val
		commandVars = append(commandVars, "-e", ev)
	}

	commandVars = append(commandVars, opts.EnvironmentId, "sh", "-c", makeSingleLineScript(opts.Script))
	cmd := exec.CommandContext(opts.Ctx, "docker", commandVars...)

	// Set a process group ID to ensure cleanup of child processes if the context is canceled.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stdout and stderr from the `docker exec` command.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Merge stdout and stderr to process them together.
	multiReader := io.MultiReader(stdout, stderr)

	// Execute the command
	if err := cmd.Start(); err != nil {
		return err
	}

	// Use a scanner to read the output line by line and stream it back.
	scanner := bufio.NewScanner(multiReader)
	for scanner.Scan() {
		onStdOut(scanner.Text())
	}

	// Wait for the command to finish and return its result.
	return cmd.Wait()
}

package executor

import (
	"context"
	"strings"

	"github.com/ConnorShore/micro-ci/internal/common"
	"github.com/ConnorShore/micro-ci/internal/pipeline"
)

type ExecutorOpts struct {
	Ctx           context.Context
	EnvironmentId string
	Script        pipeline.Script
	Vars          common.VariableMap
}

type Executor interface {
	Execute(opts ExecutorOpts, onStdOut func(line string)) error
}

// Converts script to a single line
func makeSingleLineScript(s pipeline.Script) string {
	return strings.Join(strings.Split(strings.TrimSpace(string(s)), "\n"), " && ")
}

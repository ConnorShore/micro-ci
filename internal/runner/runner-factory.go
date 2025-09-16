package runner

import (
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/common"
)

func NewRunner(t common.JobType, m *Machine) (Runner, error) {
	switch t {
	case common.TypeBootstrap:
		return NewBootstrapRunner(m), nil
	case common.TypePipeline:
		return NewPipelineRunner(m), nil
	default:
		return nil, fmt.Errorf("no factory runner for job type: %v", t)
	}
}

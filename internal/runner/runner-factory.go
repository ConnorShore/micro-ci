package runner

import (
	"fmt"

	"github.com/ConnorShore/micro-ci/internal/common"
)

// Factory method to create a new runner for a job type
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

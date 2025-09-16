package common

const (
	TypeBootstrap JobType = iota
	TypePipeline
)

type JobType int

type Job interface {
	GetRunId() string
	GetName() string
	GetType() JobType
}

type BaseJob struct {
	RunId string
	Name  string
}

type BootstrapJob struct {
	BaseJob
	RepoURL   string
	CommitSha string
	Branch    string
}

func (j *BootstrapJob) GetRunId() string {
	return j.RunId
}

func (j *BootstrapJob) GetName() string {
	return j.Name
}

func (j *BootstrapJob) GetType() JobType {
	return TypeBootstrap
}

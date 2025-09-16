package common

type MachineState string

type JobStatus string

const (
	StateOffline MachineState = "offline"
	StateIdle    MachineState = "idle"
	StateBusy    MachineState = "busy"

	StatusPending JobStatus = "pending"
	StatusRunning JobStatus = "running"
	StatusFailed  JobStatus = "failed"
	StatusSuccess JobStatus = "success"
)

package models

import "time"

// structure for a code execution request.
type ExecutionRequest struct {
	Language string        `json:"language"`
	Code     string        `json:"code"`
	Input    string        `json:"input"`
	Timeout  time.Duration `json:"timeout"`
}

// structure for execution output.
type ExecutionResult struct {
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	ExitCode   int           `json:"exit_code"`
	TimeTaken  time.Duration `json:"time_taken"`
	MemoryUsed int64         `json:"memory_used"`
	Error      string        `json:"error,omitempty"`
}

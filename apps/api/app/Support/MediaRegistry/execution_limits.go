package mediaregistry

import "time"

const (
	defaultOperationTimeout   = 5 * time.Second
	defaultCallTimeout        = 2 * time.Second
	defaultMaxConcurrentCalls = 64
	hardMaxOperationTimeout   = 30 * time.Second
	hardMaxCallTimeout        = 10 * time.Second
	hardMaxConcurrentCalls    = 1024
)

func DefaultExecutionLimits() ExecutionLimits {
	return ExecutionLimits{
		OperationTimeout:   defaultOperationTimeout,
		CallTimeout:        defaultCallTimeout,
		MaxConcurrentCalls: defaultMaxConcurrentCalls,
	}
}

func normalizeExecutionLimits(input ExecutionLimits) (ExecutionLimits, error) {
	defaults := DefaultExecutionLimits()
	if input.OperationTimeout == 0 {
		input.OperationTimeout = defaults.OperationTimeout
	}
	if input.CallTimeout == 0 {
		input.CallTimeout = defaults.CallTimeout
	}
	if input.MaxConcurrentCalls == 0 {
		input.MaxConcurrentCalls = defaults.MaxConcurrentCalls
	}
	if input.OperationTimeout < time.Millisecond || input.OperationTimeout > hardMaxOperationTimeout ||
		input.CallTimeout < time.Millisecond || input.CallTimeout > hardMaxCallTimeout ||
		input.CallTimeout > input.OperationTimeout ||
		input.MaxConcurrentCalls < 1 || input.MaxConcurrentCalls > hardMaxConcurrentCalls {
		return ExecutionLimits{}, ErrInvalid
	}
	return input, nil
}

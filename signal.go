package arboreal

// The Signal interface is Arboreal's method of controlling execution flow.
type Signal interface {
	Description() string
}

const (
	StateErrorTypeUnknown       = "unknown"
	StateErrorTypeRetryable     = "retryable"
	StateErrorTypeUnrecoverable = "unrecoverable"
	StateErrorTypeLuaSyntax     = "lua_syntax"
)

// An ErrorSignal signals that an error occurred.
// This will bubble out all the way up the call stack, and aborts any further behavior execution
type ErrorSignal struct {
	ErrorMessage string
	ErrorType    string
}

func (e ErrorSignal) Description() string {
	return e.ErrorMessage
}

func (e ErrorSignal) Type() string {
	return e.ErrorType
}

func (e ErrorSignal) Error() string {
	return e.ErrorMessage
}

// CollectUserInputSignal signals that we must pause the current execution context, gather user input, and then resume.
type CollectUserInputSignal struct {
	Reason string
}

func (c CollectUserInputSignal) Description() string {
	return c.Reason
}

// SkipSignal is only relevant to executing a BehaviorTree, and signals that the current branch should be skipped and
// execution should proceed to the next branch.
type SkipSignal struct {
	Reason string
}

func (c SkipSignal) Description() string {
	return c.Reason
}

// TerminalSignal signals that the current execution context should be terminated.
// Execution will flow back out to the parent.
type TerminalSignal struct {
	Reason string
}

func (c *TerminalSignal) Description() string {
	return c.Reason
}

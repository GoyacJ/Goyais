package tools

import "fmt"

type UnknownToolError struct {
	ToolName string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("tool %q is not registered", e.ToolName)
}

type ApprovalRequiredError struct {
	ToolName string
	Reason   string
}

func (e *ApprovalRequiredError) Error() string {
	return fmt.Sprintf("tool %q requires approval: %s", e.ToolName, e.Reason)
}

type DeniedError struct {
	ToolName string
	Reason   string
}

func (e *DeniedError) Error() string {
	return fmt.Sprintf("tool %q denied by safety gate: %s", e.ToolName, e.Reason)
}

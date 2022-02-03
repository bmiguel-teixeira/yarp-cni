package main

import "encoding/json"

type ErrorResult struct {
	CniVersion string    `json:"cniVersion"`
	ExitCode   ErrorCode `json:"code"`
	Message    string    `json:"msg"`
	Details    string    `json:"details"`
}

type ErrorCode int

const ExitCodeGeneric = 1
const ExitCodeMissingCniCommand = 2

func NewErrorResult(exitCode ErrorCode, message string, details string) *ErrorResult {
	return &ErrorResult{
		CniVersion: CniVersion,
		ExitCode:   exitCode,
		Message:    message,
		Details:    details,
	}
}

func (errorResult *ErrorResult) toString() (string, error) {
	content, err := json.Marshal(errorResult)
	return string(content), err
}

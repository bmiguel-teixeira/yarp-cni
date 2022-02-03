package main

import (
	"fmt"
	"os"
)

const CniCommandVar = "CNI_COMMAND"
const CniContainerIdVar = "CNI_CONTAINERID"
const CniNetworkNamespace = "CNI_NETNS"
const CniInterfaceName = "CNI_IFNAME"
const CniExtraArgs = "CNI_ARGS"
const CniPath = "CNI_PATH"

type CniArgs struct {
	Command          string
	ContainerId      string
	NetworkNamespace string
	InterfaceName    string
	ExtraArgs        string
	Path             string
}

func LoadCniEnvironmentValues() (*CniArgs, *ErrorResult) {
	cniRequest := CniArgs{}
	cniCommand, ok := os.LookupEnv(CniCommandVar)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniCommandVar), "")
	}
	cniRequest.Command = cniCommand

	cniContainerIdVar, ok := os.LookupEnv(CniContainerIdVar)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniContainerIdVar), "")
	}
	cniRequest.ContainerId = cniContainerIdVar

	cniNetworkNamespace, ok := os.LookupEnv(CniNetworkNamespace)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniContainerIdVar), "")
	}
	cniRequest.NetworkNamespace = cniNetworkNamespace

	cniInterfaceName, ok := os.LookupEnv(CniInterfaceName)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniContainerIdVar), "")
	}
	cniRequest.InterfaceName = cniInterfaceName

	cniExtraArgs, ok := os.LookupEnv(CniExtraArgs)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniContainerIdVar), "")
	}
	cniRequest.ExtraArgs = cniExtraArgs

	cniPath, ok := os.LookupEnv(CniPath)
	if !ok {
		return nil, NewErrorResult(ExitCodeMissingCniCommand, fmt.Sprintf("Missing %s argument", CniContainerIdVar), "")
	}
	cniRequest.Path = cniPath

	return &cniRequest, nil
}

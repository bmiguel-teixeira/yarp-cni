package main

import (
	"encoding/json"
	"fmt"
	"os"
	"yarp-cni/pkg/cni"
	"yarp-cni/pkg/im"
	"yarp-cni/pkg/ipam"
	"yarp-cni/pkg/router"

	log "github.com/sirupsen/logrus"
)

const CniVersion = "0.3.1"

type PluginMode string

const CniPluginMode = "CNI"
const RouterPluginMode = "ROUTER"

func main() {
	pluginMode, _ := os.LookupEnv("PLUGIN_MODE")
	logger := SetupLogging(pluginMode)

	if pluginMode == RouterPluginMode {
		//start router controller
		router.NewRouteController(logger, "master")
	} else {
		// Eventually load settings
		interfaceSettings := im.InterfaceConfiguration{BridgeName: "yarp0"}
		ipamSettings := &ipam.LocalIpamClientConfig{IpamDbPath: "/etc/cni/ipam.db"}
		dnSettings := cni.Dns{
			Nameservers: []string{"10.96.0.10"},
			Domain:      "",
			Search:      []string{"svc.cluster.local", "cluster.local", "local"},
		}

		ipamClient := ipam.NewLocalIpamClient(logger, ipamSettings)
		interfaceClient := im.NewInterfaceManager(logger, interfaceSettings, ipamClient)

		cniArgs, errorResult := LoadCniEnvironmentValues()
		if errorResult != nil {
			response, err := errorResult.toString()
			if err != nil {
				logger.Error(err)
				os.Exit(255)
			}
			fmt.Println(response)
			os.Exit(0)
		}

		switch cniArgs.Command {
		case "ADD":
			logger.Info(fmt.Sprintf("Received ADD request with %s", cniArgs))
			cniResponse, cniErr := interfaceClient.CreateInterface(cniArgs.ContainerId, cniArgs.NetworkNamespace, cniArgs.InterfaceName)
			cniResponse.Dns = dnSettings
			cniResponse.CniVersion = CniVersion
			if cniErr != nil {
				logger.Error(cniErr)
			}
			response, err := json.Marshal(cniResponse)
			if err != nil {
				fmt.Errorf("error decoding response of [%s]", err)
				os.Exit(1)
			}
			fmt.Println(string(response))
			logger.Info(string(response))
			os.Exit(0)
		case "DEL":
			logger.Info(fmt.Sprintf("Received DEL request with %s", cniArgs))
			cniResponse, cniErr := interfaceClient.DeleteInterface(cniArgs.ContainerId, cniArgs.NetworkNamespace, cniArgs.InterfaceName)
			cniResponse.Dns = dnSettings
			cniResponse.CniVersion = CniVersion
			if cniErr != nil {
				logger.Error(cniErr)
			}
			response, err := json.Marshal(cniResponse)
			if err != nil {
				fmt.Errorf("error decoding response of [%s]", err)
				os.Exit(1)
			}
			fmt.Println(string(response))
			logger.Info(string(response))
			os.Exit(0)
		case "VERSION":
			logger.Info(fmt.Sprintf("Received VERSION request with %s", cniArgs))
			fmt.Sprintln("{\"cniVersion\": %s}", CniVersion)
			os.Exit(0)
		default:
			logger.Info(fmt.Sprintf("Received [%s] request with %s", cniArgs.Command, cniArgs))
			fmt.Errorf("unkown CNI_COMMAND. Expected [ADD, DEL] but go [%s]", cniArgs.Command)
			os.Exit(1)
		}
	}
}

func SetupLogging(pluginMode string) *log.Logger {
	var logger = log.New()
	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetLevel(log.DebugLevel)

	if pluginMode == RouterPluginMode {
		// All other modes, we log to stdout
		logger.Out = os.Stdout
		logger.Info("Logger initilized in stdOut mode")
	} else {
		// If mode is CNI, we log to file
		file, err := os.OpenFile("yarp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logger.Out = file
		} else {
			logger.Info("Failed to log to file, using default stderr")
		}
		logger.Info("Logger initilized in file mode")
	}

	return logger
}

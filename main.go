package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
	"github.com/bitrise-steplib/bitrise-step-disable-iphonesimulator-connect-hardware-keyboard/simpref"
)

type Inputs struct {
	IPhoneSimulatorPreferencesPth string `env:"iphonesimulator_preferences_pth,required"`
	Verbose                       bool   `env:"verbose,opt[yes,no]"`
}

func main() {
	logger := log.NewLogger()

	var inputs Inputs
	if err := stepconf.NewInputParser(env.NewRepository()).Parse(&inputs); err != nil {
		logger.Errorf("Failed to parse inputs: %s", err)
		os.Exit(1)
	}
	stepconf.Print(inputs)

	logger.EnableDebugLog(inputs.Verbose)

	envRepository := env.NewRepository()
	commandFactory := command.NewFactory(envRepository)
	xcodebuildVersionProvider := xcodeversion.NewXcodeVersionProvider(commandFactory)
	xcodeVersion, err := xcodebuildVersionProvider.GetVersion()
	if err != nil { // not fatal error, continuing with empty version
		logger.Warnf("failed to read Xcode version: %s", err)
	}
	deviceFinder := destination.NewDeviceFinder(logger, commandFactory, xcodeVersion)
	pathModifier := pathutil.NewPathModifier()
	fileManager := fileutil.NewFileManager()

	if err := disableConnectHardwareKeyboard(inputs.IPhoneSimulatorPreferencesPth, deviceFinder, pathModifier, fileManager, logger); err != nil {
		logger.Errorf("Failed to disable iPhone Simulator Connect Hardware Keyboard: %s", err)
		os.Exit(1)
	}
}

func disableConnectHardwareKeyboard(pth string, deviceFinder destination.DeviceFinder, pathModifier pathutil.PathModifier, fileManager fileutil.FileManager, logger log.Logger) error {
	logger.Println()
	logger.Infof("Dsiabling iPhone Simulator Connect Hardware Keyboard in preferences: %s", pth)

	prefs, err := simpref.OpenIPhoneSimulatorPreferences(pth, deviceFinder, pathModifier, fileManager, logger)
	if err != nil {
		return fmt.Errorf("failed to open preferences: %s", err)
	}

	if err := prefs.DisableConnectHardwareKeyboard(); err != nil {
		return err
	}

	logger.Infof("Connect Hardware Keyboard disabled")
	return nil
}

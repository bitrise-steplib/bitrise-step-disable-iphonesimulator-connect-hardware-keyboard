package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-steputils/v2/stepenv"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/simulator"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
	"github.com/bitrise-steplib/bitrise-step-disable-iphonesimulator-connect-hardware-keyboard/simpref"
)

const (
	backupIPhoneSimulatorPreferencesPthEnvKey = "BACKUP_IPHONESIMULATOR_PREFERENCES_PATH"
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
		return
	}
	stepconf.Print(inputs)

	logger.EnableDebugLog(inputs.Verbose)

	backupIPhoneSimulatorPreferences(inputs.IPhoneSimulatorPreferencesPth, logger)

	disableConnectHardwareKeyboard(inputs.IPhoneSimulatorPreferencesPth, logger)
}

func disableConnectHardwareKeyboard(pth string, logger log.Logger) {
	logger.Println()
	logger.Infof("Dsiabling iPhone Simulator Connect Hardware Keyboard in preferences: %s", pth)

	envRepository := env.NewRepository()
	commandFactory := command.NewFactory(envRepository)
	xcodebuildVersionProvider := xcodeversion.NewXcodeVersionProvider(commandFactory)
	xcodeVersion, err := xcodebuildVersionProvider.GetVersion()
	if err != nil { // not fatal error, continuing with empty version
		logger.Errorf("failed to read Xcode version: %s", err)
	}
	deviceFinder := destination.NewDeviceFinder(logger, commandFactory, xcodeVersion)
	simulatorManager := simulator.NewManager(logger, commandFactory)
	pathModifier := pathutil.NewPathModifier()
	fileManager := fileutil.NewFileManager()

	prefs, err := simpref.OpenIPhoneSimulatorPreferences(pth, deviceFinder, simulatorManager, pathModifier, fileManager, logger)
	if err != nil {
		logger.Errorf("Failed to open preferences: %s", err)
		os.Exit(1)
	}

	if err := prefs.DisableConnectHardwareKeyboard(); err != nil {
		logger.Errorf("Failed to disable Connect Hardware Keyboard: %s", err)
		os.Exit(1)
	}

	logger.Infof("Connect Hardware Keyboard disabled")
}

func backupIPhoneSimulatorPreferences(pth string, logger log.Logger) {
	logger.Println()
	logger.Infof("Backing up iPhone Simulator preferences: %s", pth)

	absPth, err := pathutil.NewPathModifier().AbsPath(pth)
	if err != nil {
		logger.Errorf("Failed to get absolute path: %s", err)
		os.Exit(1)
	}

	in, err := os.Open(absPth)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("File not found: %s, skipping backup", absPth)
			return
		}

		logger.Errorf("Failed to open file: %s", err)
		os.Exit(1)
	}

	defer func() {
		if err := in.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

	backupPth, err := copyFile(in, logger)
	if err != nil {
		logger.Errorf("Failed to backup preferences: %s", err)
		os.Exit(1)
	}

	if err := stepenv.NewRepository(env.NewRepository()).Set(backupIPhoneSimulatorPreferencesPthEnvKey, backupPth); err != nil {
		logger.Errorf("Failed to set env: %s", err)
		os.Exit(1)
	}

	logger.Printf("Preferences backed up to: $%s=%s", backupIPhoneSimulatorPreferencesPthEnvKey, backupPth)
}

func copyFile(in *os.File, logger log.Logger) (string, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(tmpDir, filepath.Base(in.Name()))

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := out.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return "", err
	}

	return dst, nil
}

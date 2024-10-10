package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-steputils/v2/stepenv"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
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

	prefs, err := simpref.OpenIPhoneSimulatorPreferences(pth, logger)
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

	backupPth, err := copyFile(pth, logger)
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

func copyFile(src string, logger log.Logger) (string, error) {
	absSrc, err := pathutil.NewPathModifier().AbsPath(src)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(tmpDir, filepath.Base(absSrc))

	in, err := os.Open(absSrc)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := in.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

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

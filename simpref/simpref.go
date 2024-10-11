package simpref

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/simulator"
)

const (
	DefaultIPhoneSimulatorPreferencesPth = "~/Library/Preferences/com.apple.iphonesimulator.plist"
	defaultSimulatorDestination          = "platform=iOS Simulator,name=Bitrise iOS default,OS=latest"
)

type IPhoneSimulatorPreferences struct {
	pth         string
	format      int
	preferences map[string]any

	fileManager  fileutil.FileManager
	pathModifier pathutil.PathModifier
	logger       log.Logger
}

func OpenIPhoneSimulatorPreferences(pth string, deviceFinder destination.DeviceFinder, simulatorManager simulator.Manager, pathModifier pathutil.PathModifier, fileManager fileutil.FileManager, logger log.Logger) (*IPhoneSimulatorPreferences, error) {
	absPth, err := pathModifier.AbsPath(pth)
	if err != nil {
		return nil, err
	}

	prefsFile, err := fileManager.Open(absPth)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}

		defaultIPhoneSimulatorPreferencesAbsPth, err := pathModifier.AbsPath(DefaultIPhoneSimulatorPreferencesPth)
		if err != nil {
			return nil, err
		}

		if absPth != defaultIPhoneSimulatorPreferencesAbsPth {
			return nil, fmt.Errorf("file not found: %s", absPth)
		}

		logger.Debugf("Initialising default simulator preferences")

		prefsFile, err = initialiseDefaultSimulatorPreferences(absPth, deviceFinder, simulatorManager, fileManager, logger)
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		if err := prefsFile.Close(); err != nil {
			logger.Warnf("Failed to close file: %s", err)
		}
	}()

	preferencesBytes, err := io.ReadAll(prefsFile)
	if err != nil {
		return nil, err
	}

	var preferences map[string]any
	format, err := plist.Unmarshal(preferencesBytes, &preferences)
	if err != nil {
		return nil, err
	}

	return &IPhoneSimulatorPreferences{
		pth:          absPth,
		format:       format,
		preferences:  preferences,
		fileManager:  fileManager,
		pathModifier: pathModifier,
		logger:       logger,
	}, nil
}

func (prefs *IPhoneSimulatorPreferences) DisableConnectHardwareKeyboard() error {
	devicesPreferences, err := getMap(prefs.preferences, "DevicePreferences")
	if err != nil {
		return err
	}

	for deviceID, _ := range devicesPreferences {
		devicePreferences, err := getMap(devicesPreferences, deviceID)
		if err != nil {
			return err
		}

		originalValue, ok := devicePreferences["ConnectHardwareKeyboard"]
		if ok {
			prefs.logger.Debugf("%s: original value for ConnectHardwareKeyboard: %v", deviceID, originalValue)
		} else {
			prefs.logger.Debugf("%s: ConnectHardwareKeyboard not found", deviceID)
		}

		devicePreferences["ConnectHardwareKeyboard"] = false
		devicesPreferences[deviceID] = devicePreferences

		prefs.logger.Debugf("%s: ConnectHardwareKeyboard disabled", deviceID)
	}

	prefs.preferences["DevicePreferences"] = devicesPreferences

	preferencesBytes, err := plist.Marshal(prefs.preferences, prefs.format)
	if err != nil {
		return err
	}

	if err := os.WriteFile(prefs.pth, preferencesBytes, 0644); err != nil {
		return err
	}

	return nil
}

func initialiseDefaultSimulatorPreferences(pth string, deviceFinder destination.DeviceFinder, simulatorManager simulator.Manager, fileManager fileutil.FileManager, logger log.Logger) (*os.File, error) {
	simulatorDestination, err := destination.NewSimulator(defaultSimulatorDestination)
	if err != nil || simulatorDestination == nil {
		return nil, fmt.Errorf("invalid destination specifier (%s): %w", defaultSimulatorDestination, err)
	}

	device, err := deviceFinder.FindDevice(*simulatorDestination)
	if err != nil {
		return nil, fmt.Errorf("simulator UDID lookup failed: %w", err)
	}

	factory := command.NewFactory(env.NewRepository())
	cmd := factory.Create("open", []string{"-a", "Simulator", "--args", "-CurrentDeviceUDID", device.ID}, nil)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		fmt.Println(out)
		return nil, err
	}

	defer func() {
		if err := simulatorManager.Shutdown(device.ID); err != nil {
			logger.Warnf("Failed to shutdown simulator: %s", err)
		}
	}()

	var prefsFile *os.File

	waitTimeSec := 300
	for waitTimeSec > 0 {
		prefsFile, err = fileManager.Open(pth)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		if prefsFile != nil {
			ok, err := checkPrefsFile(prefsFile)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
			logger.Debugf("Simulator preferences not ready")
		}

		time.Sleep(5 * time.Second)
		waitTimeSec -= 5
	}

	if prefsFile == nil {
		return nil, fmt.Errorf("couldn't initialise iphonesimulator preferences")
	}

	return prefsFile, nil
}

func checkPrefsFile(prefsFile *os.File) (bool, error) {
	preferencesBytes, err := io.ReadAll(prefsFile)
	if err != nil {
		return false, err
	}

	var preferences map[string]any
	_, err = plist.Unmarshal(preferencesBytes, &preferences)
	if err != nil {
		return false, err
	}

	_, err = getMap(preferences, "DevicePreferences")
	if err != nil {
		if err.Error() == "key not found: DevicePreferences" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func getMap(raw map[string]any, key string) (map[string]any, error) {
	rawValue, ok := raw[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	mapValue, ok := rawValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("value is not a map: %s", key)
	}
	return mapValue, nil
}

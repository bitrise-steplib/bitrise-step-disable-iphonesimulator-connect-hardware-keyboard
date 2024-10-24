package simpref

import (
	"fmt"
	"io"
	"os"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
)

const (
	DefaultIPhoneSimulatorPreferencesPth = "~/Library/Preferences/com.apple.iphonesimulator.plist"
	defaultSimulatorDestination          = "platform=iOS Simulator,name=Bitrise iOS default,OS=latest"
)

type IPhoneSimulatorPreferences struct {
	pth         string
	format      int
	preferences map[string]any

	deviceFinder destination.DeviceFinder
	fileManager  fileutil.FileManager
	pathModifier pathutil.PathModifier
	logger       log.Logger
}

func OpenIPhoneSimulatorPreferences(pth string, deviceFinder destination.DeviceFinder, pathModifier pathutil.PathModifier, fileManager fileutil.FileManager, logger log.Logger) (*IPhoneSimulatorPreferences, error) {
	absPth, err := pathModifier.AbsPath(pth)
	if err != nil {
		return nil, err
	}

	prefsFile, err := fileManager.Open(absPth)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}

		logger.Debugf("iphonesimulator preferences file not found at %s, creating one...", absPth)

		prefsFile, err = os.Create(absPth)
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
	format := plist.BinaryFormat
	if len(preferencesBytes) > 0 {
		format, err = plist.Unmarshal(preferencesBytes, &preferences)
		if err != nil {
			return nil, err
		}
	}

	return &IPhoneSimulatorPreferences{
		pth:          absPth,
		format:       format,
		preferences:  preferences,
		deviceFinder: deviceFinder,
		fileManager:  fileManager,
		pathModifier: pathModifier,
		logger:       logger,
	}, nil
}

func (prefs *IPhoneSimulatorPreferences) DisableConnectHardwareKeyboard() error {
	if len(prefs.preferences) == 0 {
		deviceList, err := prefs.deviceFinder.ListDevices()
		if err != nil {
			return err
		}

		devicePreferences := map[string]map[string]any{}
		for _, deviceList := range deviceList.Devices {
			for _, device := range deviceList {
				devicePreference, ok := devicePreferences[device.UDID]
				if !ok {
					devicePreference = map[string]any{}
				} else {
					fmt.Println("devicePreference already set for:", device.UDID)
				}
				devicePreference["ConnectHardwareKeyboard"] = false
				devicePreferences[device.UDID] = devicePreference
			}
		}

		prefs.preferences = map[string]any{
			"ConnectHardwareKeyboard": "0",
			"DevicePreferences":       devicePreferences,
		}
	} else {
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
		prefs.preferences["ConnectHardwareKeyboard"] = "0"
	}

	preferencesBytes, err := plist.Marshal(prefs.preferences, prefs.format)
	if err != nil {
		return err
	}

	if err := os.WriteFile(prefs.pth, preferencesBytes, 0644); err != nil {
		return err
	}

	return nil
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

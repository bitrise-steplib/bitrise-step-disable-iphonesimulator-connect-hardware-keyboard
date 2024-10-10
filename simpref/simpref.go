package simpref

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

const DefaultIPhoneSimulatorPreferencesPth = "~/Library/Preferences/com.apple.iphonesimulator.plist"

type IPhoneSimulatorPreferences struct {
	pth         string
	format      int
	preferences map[string]any

	logger log.Logger
}

func OpenIPhoneSimulatorPreferences(pth string, logger log.Logger) (*IPhoneSimulatorPreferences, error) {
	absPth, err := pathutil.NewPathModifier().AbsPath(pth)
	if err != nil {
		return nil, err
	}

	preferencesBytes, err := os.ReadFile(absPth)
	if err != nil {
		return nil, err
	}

	var preferences map[string]any
	format, err := plist.Unmarshal(preferencesBytes, &preferences)
	if err != nil {
		return nil, err
	}

	return &IPhoneSimulatorPreferences{
		pth:         absPth,
		format:      format,
		preferences: preferences,
		logger:      logger,
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

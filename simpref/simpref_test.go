package simpref

import (
	"testing"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
	"github.com/stretchr/testify/require"
)

func TestIPhoneSimulatorPreferences_DisableConnectHardwareKeyboard(t *testing.T) {
	tests := []struct {
		name   string
		pth    string
		logger log.Logger
	}{
		{
			name:   "ok",
			pth:    "testdata/com.apple.iphonesimulator.plist",
			logger: log.NewLogger(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewLogger()
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

			prefs, err := OpenIPhoneSimulatorPreferences(tt.pth, deviceFinder, pathModifier, fileManager, tt.logger)
			require.NoError(t, err)

			err = prefs.DisableConnectHardwareKeyboard()
			require.NoError(t, err)
		})
	}
}

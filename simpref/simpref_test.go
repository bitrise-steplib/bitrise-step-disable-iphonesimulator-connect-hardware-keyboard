package simpref

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
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
			prefs, err := OpenIPhoneSimulatorPreferences(tt.pth, tt.logger)
			require.NoError(t, err)

			err = prefs.DisableConnectHardwareKeyboard()
			require.NoError(t, err)
		})
	}
}

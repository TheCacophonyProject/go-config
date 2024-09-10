package configtest

import (
	"os"
	"path"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// WriteConfigFromBytes will write the config file in the given fs and return the
// function setting the lockFile and a cleanup function
func WriteConfigFromBytes(t *testing.T, configBytes []byte, fsConfigFile string, fs afero.Fs) (func(string) string, func()) {
	if fs == nil {
		fs = afero.NewMemMapFs()
	}
	require.NoError(t, afero.WriteFile(fs, fsConfigFile, configBytes, 0644))

	lockFile := path.Join(os.TempDir(), fsConfigFile+".lock")
	require.NoError(t, os.MkdirAll(path.Dir(lockFile), 0777))
	return func(p string) string {
			return lockFile
		},
		func() {
			os.Remove(lockFile)
		}
}

func WriteConfigFromFile(t *testing.T, configFile string, fsConfgiFile string, fs afero.Fs) (func(string) string, func()) {
	b := []byte{}
	if configFile != "" {
		var err error
		b, err = os.ReadFile(configFile)
		require.NoError(t, err)
	}
	return WriteConfigFromBytes(t, b, fsConfgiFile, fs)
}

package config

import (
	"context"
	"log"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/TheCacophonyProject/go-config/configtest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wawandco/fako"
)

func TestDefaults(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	location := DefaultWindowLocation()
	assert.NoError(t, conf.Unmarshal(LocationKey, &location))
	assert.Equal(t, DefaultWindowLocation(), location)
}

func TestReadingConfigInDir(t *testing.T) {
	defer newFs(t, "./test-files/test.toml")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	var device Device
	deviceChanges := Device{}
	deviceChanges.ID = 789
	assert.NoError(t, conf.Unmarshal(DeviceKey, &device))
	assert.Equal(t, deviceChanges, device)

	windows := DefaultWindows()
	windowChanges := DefaultWindows()
	windowChanges.PowerOff = "+1s"
	assert.NoError(t, conf.Unmarshal(WindowsKey, &windows))
	assert.Equal(t, windowChanges, windows)

	var location Location
	locationChanges := Location{}
	locationChanges.Accuracy = 543
	assert.NoError(t, conf.Unmarshal(LocationKey, &location))
	assert.Equal(t, locationChanges, location)

	windowLocation := DefaultWindowLocation()
	windowLocationChanges := DefaultWindowLocation()
	windowLocationChanges.Accuracy = 543
	assert.NoError(t, conf.Unmarshal(LocationKey, &windowLocation))
	assert.Equal(t, windowLocationChanges, windowLocation)

	testHosts := DefaultTestHosts()
	testHostsChanges := DefaultTestHosts()
	testHostsChanges.PingRetries = 333
	assert.NoError(t, conf.Unmarshal(TestHostsKey, &testHosts))
	assert.Equal(t, testHostsChanges, testHosts)

	var battery Battery
	batteryChanges := Battery{}
	batteryChanges.NoBattery = 10
	assert.NoError(t, conf.Unmarshal(BatteryKey, &battery))
	assert.Equal(t, batteryChanges, battery)

	modemd := DefaultModemd()
	modemdChanges := DefaultModemd()
	modemdChanges.ConnectionTimeout = time.Second * 21
	assert.NoError(t, conf.Unmarshal(ModemdKey, &modemd))
	assert.Equal(t, modemdChanges, modemd)

	lepton := DefaultLepton()
	leptonChanges := DefaultLepton()
	leptonChanges.SPISpeed = 2
	assert.NoError(t, conf.Unmarshal(LeptonKey, &lepton))
	assert.Equal(t, leptonChanges, lepton)

	thermalRecorder := DefaultThermalRecorder()
	thermalRecorderChanges := DefaultThermalRecorder()
	thermalRecorderChanges.MaxSecs = 321
	assert.NoError(t, conf.Unmarshal(ThermalRecorderKey, &thermalRecorder))
	assert.Equal(t, thermalRecorderChanges, thermalRecorder)

	thermalThrottler := DefaultThermalThrottler()
	thermalThrottlerChanges := DefaultThermalThrottler()
	thermalThrottlerChanges.BucketSize = time.Second * 16
	assert.NoError(t, conf.Unmarshal(ThermalThrottlerKey, &thermalThrottler))
	assert.Equal(t, thermalThrottlerChanges, thermalThrottler)

	ports := DefaultPorts()
	portsChanges := DefaultPorts()
	portsChanges.Managementd = 3
	assert.NoError(t, conf.Unmarshal(PortsKey, &ports))
	assert.Equal(t, portsChanges, ports)

	var secrets Secrets
	secretsChanges := Secrets{}
	secretsChanges.DevicePassword = "pass"
	assert.NoError(t, conf.Unmarshal(SecretsKey, &secrets))
	assert.Equal(t, secretsChanges, secrets)

	thermalMotion := DefaultThermalMotion("")
	thermalMotionChanges := DefaultThermalMotion("")
	thermalMotionChanges.TempThresh = 398
	assert.NoError(t, conf.Unmarshal(ThermalMotionKey, &thermalMotion))
	assert.Equal(t, thermalMotionChanges, thermalMotion)

	audio := DefaultAudioBait()
	audioChanges := DefaultAudioBait()
	audioChanges.Card = 1
	assert.NoError(t, conf.Unmarshal(AudioBaitKey, &audio))
	assert.Equal(t, audioChanges, audio)

	audioRec := DefaultAudioRecording()
	audioRecChanges := DefaultAudioRecording()
	audioRecChanges.Enabled = true
	assert.NoError(t, conf.Unmarshal(AudioRecordingKey, &audioRec))
	assert.Equal(t, audioRecChanges, audioRec)

	setup := DefaultDeviceSetup()
	setupChanges := DefaultDeviceSetup()
	assert.NoError(t, conf.Unmarshal(DeviceSetupKey, &setup))
	assert.Equal(t, setup, setupChanges)

	salt := DefaultSalt()
	saltChanges := DefaultSalt()
	assert.NoError(t, conf.Unmarshal(SaltKey, &salt))
	assert.Equal(t, salt, saltChanges)

}

func TestSettingInvalidKeys(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	w := randomWindows()
	require.NoError(t, conf.Set(WindowsKey, w))
	m := map[string]interface{}{
		"invalid-key": "a value",
	}
	require.Error(t, conf.SetFromMap(WindowsKey, m, false))
	require.NoError(t, conf.SetFromMap(WindowsKey, m, true))
	require.Equal(t, "a value", conf.Get("windows.invalid-key"))
}

func TestWriting(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	conf2, err := New(DefaultConfigDir)
	require.NoError(t, err)

	d := randomDevice()
	w := randomWindows()
	l := randomLocation()
	h := randomTestHosts()
	require.NoError(t, conf.Set(DeviceKey, d))
	require.NoError(t, conf2.Set(WindowsKey, w))
	require.NoError(t, conf.Set(LocationKey, &l))
	require.NoError(t, conf2.Set(TestHostsKey, &h))
	conf, err = New(DefaultConfigDir)
	require.NoError(t, err)
	d2 := Device{}
	require.NoError(t, conf.Unmarshal(DeviceKey, &d2))
	w2 := Windows{}
	require.NoError(t, conf.Unmarshal(WindowsKey, &w2))
	l2 := Location{}
	require.NoError(t, conf.Unmarshal(LocationKey, &l2))
	h2 := TestHosts{}
	require.NoError(t, conf.Unmarshal(TestHostsKey, &h2))

	require.Equal(t, d, d2)
	require.Equal(t, w, w2)
	equalLocation(t, l, l2)
	require.Equal(t, h, h2)
}

func TestClearSection(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	log.Println()
	l := randomLocation()
	w := randomWindows()
	require.NoError(t, conf.Set(LocationKey, &l))
	require.NoError(t, conf.Set(WindowsKey, &w))
	require.NoError(t, conf.Unset(LocationKey+".latitude"))
	require.Error(t, conf.Unset(LocationKey+".latitude.foo"))
	require.NoError(t, conf.Unset(LocationKey+".bar"))
	conf, err = New(DefaultConfigDir)
	require.NoError(t, err)
	l2 := Location{}
	require.NoError(t, conf.Unmarshal(LocationKey, &l2))

	w2 := Windows{}
	require.NoError(t, conf.Unmarshal(WindowsKey, &w2))
	l.Latitude = 0
	equalLocation(t, l, l2)
	require.Equal(t, w, w2)
}

func TestClear(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	l := randomLocation()
	w := randomWindows()
	require.NoError(t, conf.Set(LocationKey, &l))
	require.NoError(t, conf.Set(WindowsKey, &w))
	require.NoError(t, conf.Unset(LocationKey))
	conf, err = New(DefaultConfigDir)
	require.NoError(t, err)
	l2 := Location{}
	require.NoError(t, conf.Unmarshal(LocationKey, &l2))

	w2 := Windows{}
	require.NoError(t, conf.Unmarshal(WindowsKey, &w2))
	equalLocation(t, Location{}, l2)
	require.Equal(t, w, w2)
}

func TestFileLock(t *testing.T) {
	defer newFs(t, "")()
	lockTimeout = time.Millisecond * 100

	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	conf2, err := New(DefaultConfigDir)
	require.NoError(t, err)

	require.NoError(t, conf.getFileLock())
	require.Equal(t, context.DeadlineExceeded, conf2.getFileLock())
	conf.fileLock.Unlock()
	require.NoError(t, conf2.getFileLock())
}

func TestSettingUpdated(t *testing.T) {
	defer newFs(t, "")()
	newNow()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	require.NoError(t, conf.Set(DeviceKey, randomDevice()))
	require.Equal(t, conf.Get(DeviceKey+".updated"), now())
}

func TestMapToLocation(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	newNow()
	locationMap := map[string]interface{}{
		"latitude":  "123.321",
		"timestamp": now().Format(TimeFormat),
	}
	locationExpected := Location{
		Latitude:  123.321,
		Timestamp: now(),
	}
	var location Location
	require.NoError(t, conf.SetFromMap(LocationKey, locationMap, false))
	require.NoError(t, conf.Unmarshal(LocationKey, &location))
	equalLocation(t, locationExpected, location)
}

func TestNotWritingZeroValues(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)

	newNow()
	locationMap := map[string]interface{}{
		"lAtitUde": "123.321",
	}
	locationMapExpected := map[string]interface{}{
		"latitude": float32(123.321),
		"updated":  now(),
	}
	require.NoError(t, conf.SetFromMap(LocationKey, locationMap, false))
	require.Equal(t, locationMapExpected, (conf.v.AllSettings()[LocationKey]))
}

func TestMapToAudio(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	audioMap := map[string]interface{}{
		"directory":      "/audio/directory",
		"card":           "4",
		"volume-control": "audio volume control",
	}
	audioExpected := AudioBait{
		Dir:           "/audio/directory",
		Card:          4,
		VolumeControl: "audio volume control",
	}
	checkWritingMap(t, AudioBaitKey, &AudioBait{}, &audioExpected, audioMap, conf)
}

func TestMapToBattery(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	batteryMap := map[string]interface{}{"enable-voltage-readings": "true"}
	batteryExpected := Battery{EnableVoltageReadings: true}
	checkWritingMap(t, BatteryKey, &Battery{}, &batteryExpected, batteryMap, conf)
}

func TestMapToDevice(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	deviceMap := map[string]interface{}{"Group": "a-group"}
	deviceExpected := Device{Group: "a-group"}
	checkWritingMap(t, DeviceKey, &Device{}, &deviceExpected, deviceMap, conf)
}

func TestMapToModemd(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	modemdMap := map[string]interface{}{
		"test-interval": "10m4s",
		"modems":        []map[string]interface{}{{"name": "modem name"}},
	}
	modemdExpected := Modemd{
		TestInterval: 10*time.Minute + 4*time.Second,
		Modems:       []Modem{{Name: "modem name"}},
	}
	checkWritingMap(t, ModemdKey, &Modemd{}, &modemdExpected, modemdMap, conf)
}

func TestSetField(t *testing.T) {
	defer newFs(t, "")()
	conf, err := New(DefaultConfigDir)
	require.NoError(t, err)
	audio := AudioBait{
		Dir:           "/audio/directory",
		Card:          4,
		VolumeControl: "audio volume control",
	}
	require.NoError(t, conf.Set(AudioBaitKey, audio))

	require.NoError(t, conf.SetField(AudioBaitKey, "card", "5", false))
	require.Error(t, conf.SetField(AudioBaitKey, "not-a-key", "5", false))

	var audio2 AudioBait
	require.NoError(t, conf.Unmarshal(AudioBaitKey, &audio2))

	audioExpected := AudioBait{
		Dir:           "/audio/directory",
		Card:          5,
		VolumeControl: "audio volume control",
	}

	require.Equal(t, audioExpected, audio2)
}

func checkWritingMap(
	t *testing.T,
	key string,
	s, expected interface{},
	m map[string]interface{},
	conf *Config) {
	require.NoError(t, conf.SetFromMap(key, m, false))
	require.NoError(t, conf.Unmarshal(key, s))
	require.Equal(t, expected, s)
}

func newFs(t *testing.T, configFile string) func() {
	fs := afero.NewMemMapFs()
	SetFs(fs)
	fsConfigFile := path.Join(DefaultConfigDir, ConfigFileName)
	lockFileFunc, cleanupFunc := configtest.WriteConfigFromFile(t, configFile, fsConfigFile, fs)
	SetLockFilePath(lockFileFunc)
	return cleanupFunc
}

func newNow() {
	n := time.Now()
	now = func() time.Time {
		return n
	}
}

func randomDevice() (d Device) {
	fako.Fuzz(&d)
	return
}

func randomWindows() (w Windows) {
	fako.Fuzz(&w)
	return
}

func randomLocation() Location {
	return Location{
		Accuracy:  float32(randSrc.Int63()),
		Longitude: float32(randSrc.Int63()),
		Timestamp: now(),
	}
}

func equalLocation(t *testing.T, l1, l2 Location) {
	require.Equal(t, l1.Accuracy, l2.Accuracy)
	require.Equal(t, l1.Altitude, l2.Altitude)
	require.Equal(t, l1.Latitude, l2.Latitude)
	require.Equal(t, l1.Longitude, l2.Longitude)
	require.Equal(t, l1.Timestamp.Unix(), l2.Timestamp.Unix())
}

func randomTestHosts() TestHosts {
	return TestHosts{
		URLs:         []string{randString(10), randString(20), randString(15)},
		PingRetries:  int(randSrc.Int63()),
		PingWaitTime: time.Duration(randSrc.Int63()) * time.Second,
	}
}

// Random string
const (
	chars       = "abcdefghijklmnopqrstuvwxyz0123456789"
	charIdxBits = 6                  // 6 bits to represent a char index
	charIdxMax  = 63 / charIdxBits   // # of char indices fitting in 63 bits
	charIdxMask = 1<<charIdxBits - 1 // All 1-bits, as many as charIdxBits
)

var randSrc = rand.NewSource(time.Now().UnixNano())

func randString(n int) string {
	b := make([]byte, n)
	// A randSrc.Int63() generates 63 random bits, enough for charIdxMax characters!
	for i, cache, remain := n-1, randSrc.Int63(), charIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), charIdxMax
		}
		if idx := int(cache & charIdxMask); idx < len(chars) {
			b[i] = chars[idx]
			i--
		}
		cache >>= charIdxBits
		remain--
	}
	return string(b)
}

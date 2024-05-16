package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"time"

	config "github.com/TheCacophonyProject/go-config"
	"github.com/alexflint/go-arg"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v1"
)

var version = "<not set>"

type Args struct {
	Dir   string `args:"--dir" help:"config directory"`
	Force bool   `args:"--force" help:"will override existing config file"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	args := Args{
		Dir: config.DefaultConfigDir,
	}
	arg.MustParse(&args)
	return args
}

func main() {
	if err := runMain(); err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0)
	log.Printf("running version: %s", version)
	v := viper.New()
	configFile := path.Join(args.Dir, "config.toml")
	if _, err := os.Stat(configFile); err == nil && !args.Force {
		log.Printf("config file '%s' alread exists. Aborting writing new config", configFile)
		return nil
	}
	v.SetConfigFile(configFile)
	for _, f := range configProcessingFuncs {
		s, err := f(args.Dir)
		if err != nil {
			return err
		}
		m, err := interfaceToMap(s)
		if err != nil {
			return err
		}
		if err := v.MergeConfigMap(m); err != nil {
			return err
		}
	}
	log.Printf("all settings: '%v'", v.AllSettings())
	return v.WriteConfig()
}

func interfaceToMap(value interface{}) (m map[string]interface{}, err error) {
	err = mapstructure.Decode(value, &m)
	for k, v := range m {
		if isZeroVal(v) {
			delete(m, k)
		} else if isSlice(v) && isStruct(reflect.ValueOf(v).Index(0)) {
			s := reflect.ValueOf(v)
			sliceMap := make([]map[string]interface{}, 0)
			for i := 0; i < s.Len(); i++ {
				m2, err := interfaceToMap(s.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				sliceMap = append(sliceMap, m2)
			}
			m[k] = sliceMap
		}
	}
	return
}

func isStruct(x interface{}) bool {
	return reflect.ValueOf(x).Kind() == reflect.Struct
}

func isSlice(x interface{}) bool {
	return reflect.ValueOf(x).Kind() == reflect.Slice
}

func isZeroVal(x interface{}) bool {
	switch reflect.ValueOf(x).Kind() {
	case reflect.Slice, reflect.Map:
		return reflect.ValueOf(x).IsNil()
	}
	return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

var configProcessingFuncs = []func(string) (interface{}, error){
	processAttiny,
	processModem,
	processLepton,
	processThermalRecorder,
	processLocation,
	processDevice,
	processManagementd,
	processAudioBait,
	processAudioRecording,
}

func processAttiny(configDir string) (interface{}, error) {
	s := &rawAttinyConfig{}
	return s, yamlToStruct(path.Join(configDir, "attiny.yaml"), s)
}

func yamlToStruct(configFile string, s interface{}) error {
	log.Printf("reading '%v'", configFile)
	buf, err := ioutil.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return yaml.Unmarshal(buf, s)
}

type rawAttinyConfig struct {
	PiWakeUp string         `yaml:"pi-wake-time" mapstructure:"windows.power-on"`
	PiSleep  string         `yaml:"pi-sleep-time" mapstructure:"windows.power-off"`
	Voltages attinyVoltages `yaml:"voltages" mapstructure:"battery"`
}

type attinyVoltages struct {
	Enable      bool   `yaml:"enable" mapstructure:"enable-voltage-readings"`
	NoBattery   uint16 `yaml:"no-battery" mapstructure:"no-battery-reading"`
	LowBattery  uint16 `yaml:"low-battery" mapstructure:"low-battery-reading"`
	FullBattery uint16 `yaml:"full-battery" mapstructure:"full-battery-reading"`
}

func processModem(configDir string) (interface{}, error) {
	s := &rawModemdConfig{}
	return s, yamlToStruct(path.Join(configDir, "modemd.yaml"), s)
}

type rawModemdConfig struct {
	ModemsConfig      []modemConfig `yaml:"modems" mapstructure:"modemd.modems"`
	TestInterval      time.Duration `yaml:"test-interval" mapstructure:"modemd.test-interval"`
	PowerPin          string        `yaml:"power-pin" mapstructure:"gpio.modem-power"`
	InitialOnTime     time.Duration `yaml:"initial-on-time" mapstructure:"modemd.initial-on-duration"`
	FindModemTime     time.Duration `yaml:"find-modem-time" mapstructure:"modemd.find-modem-timeout"`
	ConnectionTimeout time.Duration `yaml:"connection-timeout" mapstructure:"modemd.connection-timeout"`
	RequestOnTime     time.Duration `yaml:"request-on-time" mapstructure:"modemd.request-on-duration"`
}

type modemConfig struct {
	Name          string `yaml:"name" mapstructure:"name"`
	Netdev        string `yaml:"netdev" mapstructure:"net-dev"`
	VendorProduct string `yaml:"vendor-product" mapstructure:"vendor-product-id"`
}

func processLepton(configDir string) (interface{}, error) {
	s := &rawLepton{}
	return s, yamlToStruct(path.Join(configDir, "../leptond.yaml"), s)
}

type rawLepton struct {
	SPISpeed    int64  `yaml:"spi-speed" mapstructure:"lepton.spi-speed"`
	PowerPin    string `yaml:"power-pin" mapstructure:"gpio.thermal-camera-power"`
	FrameOutput string `yaml:"frame-output" mapstructure:"lepton.frame-output"`
}

func processThermalRecorder(configDir string) (interface{}, error) {
	s := &rawThermalRecorder{}
	if err := yamlToStruct(path.Join(configDir, "../thermal-recorder.yaml"), s); err != nil {
		return nil, err
	}
	var start, stop string
	if s.Recorder.UseSunriseSunsetWindow {
		start = (time.Minute * time.Duration(s.Recorder.SunsetOffset)).String()
		stop = (time.Minute * time.Duration(s.Recorder.SunriseOffset)).String()
	} else {
		start = s.Recorder.WindowStart
		stop = s.Recorder.WindowEnd
	}
	s.Recorder.RecordingStart = start
	s.Recorder.RecordingStop = stop
	return s, nil
}

type rawThermalRecorder struct {
	OutputDir    string          `yaml:"output-dir" mapstructure:"thermal-recorder.output-dir"`
	MinDiskSpace uint64          `yaml:"min-disk-space" mapstructure:"thermal-recorder.min-disk-space-mb"`
	Recorder     recorderConfig  `mapstructure:",squash"`
	Motion       motionConfig    `mapstructure:",squash"`
	Throttler    throttlerConfig `mapstructure:",squash"`
}

type recorderConfig struct {
	MinSecs                int    `yaml:"min-secs" mapstructure:"thermal-recorder.min-secs"`
	MaxSecs                int    `yaml:"max-secs" mapstructure:"thermal-recorder.max-secs"`
	PreviewSecs            int    `yaml:"preview-secs" mapstructure:"thermal-recorder.preview-secs"`
	UseSunriseSunsetWindow bool   `yaml:"sunrise-sunset" mapstructure:"-"`
	SunriseOffset          int    `yaml:"sunrise-offset" mapstructure:"-"`
	SunsetOffset           int    `yaml:"sunset-offset" mapstructure:"-"`
	WindowStart            string `yaml:"window-start" mapstructure:"-"`
	WindowEnd              string `yaml:"window-end" mapstructure:"-"`
	RecordingStart         string `yaml:"-" mapstructure:"windows.start-recording"`
	RecordingStop          string `yaml:"-" mapstructure:"windows.stop-recording"`
}

type motionConfig struct {
	DynamicThreshold bool   `yaml:"dynamic-thresh" mapstructure:"thermal-motion.dynamic-threshold"`
	TempThresh       uint16 `yaml:"temp-thresh" mapstructure:"thermal-motion.temp-thresh"`
	DeltaThresh      uint16 `yaml:"delta-thresh" mapstructure:"thermal-motion.delta-thresh"`
	CountThresh      int    `yaml:"count-thresh" mapstructure:"thermal-motion.count-thresh"`
	FrameCompareGap  int    `yaml:"frame-compare-gap" mapstructure:"thermal-motion.frame-compare-gap"`
	UseOneDiffOnly   bool   `yaml:"one-diff-only" mapstructure:"thermal-motion.use-one-diff-only"`
	TriggerFrames    int    `yaml:"trigger-frames" mapstructure:"thermal-motion.trigger-frames"`
	WarmerOnly       bool   `yaml:"warmer-only" mapstructure:"thermal-motion.warmer-only"`
	EdgePixels       int    `yaml:"edge-pixels" mapstructure:"thermal-motion.edge-pixels"`
	Verbose          bool   `yaml:"verbose" mapstructure:"thermal-motion.verbose"`
}

type throttlerConfig struct {
	ApplyThrottling bool          `yaml:"apply-throttling" mapstructure:"thermal-throttler.activate"`
	BucketSize      time.Duration `yaml:"bucket-size" mapstructure:"thermal-throttler.bucket-size"`
	MinRefill       time.Duration `yaml:"min-refill" mapstructure:"thermal-throttler.min-refill"`
}

func processLocation(configDir string) (interface{}, error) {
	s := &rawLocation{}
	return s, yamlToStruct(path.Join(configDir, "location.yaml"), s)
}

type rawLocation struct {
	Latitude     float32   `yaml:"latitude" mapstructure:"location.latitude"`
	Longitude    float32   `yaml:"longitude" mapstructure:"location.longitude"`
	LocTimestamp time.Time `yaml:"timestamp" mapstructure:"location.updated"`
	Altitude     float32   `yaml:"altitude" mapstructure:"location.altitude"`
	Accuracy     float32   `yaml:"accuracy" mapstructure:"location.accuracy"`
}

func processDevice(configDir string) (interface{}, error) {
	s := &rawDeviceConfig{}
	if err := yamlToStruct(path.Join(configDir, "device.yaml"), s); err != nil {
		return nil, err
	}
	return s, yamlToStruct(path.Join(configDir, "device-priv.yaml"), s)
}

type rawDeviceConfig struct {
	ServerURL  string `yaml:"server-url" mapstructure:"device.server"`
	Group      string `yaml:"group" mapstructure:"device.group"`
	DeviceName string `yaml:"device-name" mapstructure:"device.name"`
	Password   string `yaml:"password" mapstructure:"secrets.device-password"`
	DeviceID   int    `yaml:"device-id" mapstructure:"device.id"`
}

func processManagementd(configDir string) (interface{}, error) {
	s := &rawManagementdConfig{}
	return s, yamlToStruct(path.Join(configDir, "managementd.yaml"), s)
}

type rawManagementdConfig struct {
	Port int `yaml:"port" mapstructure:"ports.managementd"`
}

func processAudioBait(configDir string) (interface{}, error) {
	s := &rawAudioBaitConfig{}
	return s, yamlToStruct(path.Join(configDir, "../audiobait.yaml"), s)
}

type rawAudioBaitConfig struct {
	AudioDir      string `yaml:"audio-directory" mapstructure:"audio.directory"`
	Card          int    `yaml:"card" mapstructure:"audio.card"`
	VolumeControl string `yaml:"volume-control" mapstructure:"audio.volume-control"`
}

func processAudioRecording(configDir string) (interface{}, error) {
	s := &rawAudioRecordingConfig{}
	return s, yamlToStruct(path.Join(configDir, "../audio-recording.yaml"), s)
}

type rawAudioRecordingConfig struct {
	IsAudioDevice bool `yaml:"is-audio-device" mapstructure:"audio.is-audio-device"`
}

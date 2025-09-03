/*
cacophony-config-sync - sync device settings with Cacophony Project API.
Copyright (C) 2018, The Cacophony Project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package cacophonyconfigsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	api "github.com/TheCacophonyProject/go-api"
	config "github.com/TheCacophonyProject/go-config"
	"github.com/TheCacophonyProject/go-utils/logging"
	"github.com/TheCacophonyProject/modemd/modemlistener"
	"github.com/alexflint/go-arg"
	"github.com/rjeczalik/notify"
)

const (
	configDir    = config.DefaultConfigDir
	syncInterval = time.Hour * 24
)

func stringToTimeConverter(value interface{}) (interface{}, error) {
	strVal, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string for time conversion, got %T", value)
	}
	parsedTime, err := time.Parse(time.RFC3339, strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %v", err)
	}
	return parsedTime, nil
}

// ConverterFunc defines the function signature for value conversion
type ConverterFunc func(interface{}) (interface{}, error)

// Mapping holds the API key, config key, and optional converter function
type Mapping struct {
	APIKey    string
	ConfigKey string
	MapKey    string
	Converter ConverterFunc
}

// Section holds the section name and its mappings
type Section struct {
	Name     string
	Key      string
	Config   interface{}
	Updated  time.Time
	Mappings []Mapping
}

// Sections is a slice of Section
type Sections []Section

// IMPORTANT: apikey refers to the field from the api, config key is the map structure name, and map key is the mapped key to the config
var ConfigSections = Sections{
	{
		Name:   "thermalRecording",
		Key:    config.ThermalRecorderKey,
		Config: &config.ThermalRecorder{},
		Mappings: []Mapping{
			{
				APIKey:    "useLowPowerMode",
				ConfigKey: "UseLowPowerMode",
				MapKey:    "use-low-power-mode",
			},
			{
				APIKey:    "updated",
				ConfigKey: "Updated",
				MapKey:    "updated",
				Converter: stringToTimeConverter, // Use the converter here
			},
		},
	},
	{
		Name:   "audioRecording",
		Key:    config.AudioRecordingKey,
		Config: &config.AudioRecording{},
		Mappings: []Mapping{
			{
				APIKey:    "audioMode",
				ConfigKey: "AudioMode",
				MapKey:    "audio-mode",
			},
			{
				APIKey:    "audioSeed",
				ConfigKey: "AudioSeed",
				MapKey:    "random-seed",
			},
			{
				APIKey:    "updated",
				ConfigKey: "Updated",
				MapKey:    "updated",
				Converter: stringToTimeConverter,
			},
		},
	},
	{
		Name:   "windows",
		Key:    config.WindowsKey,
		Config: &config.Windows{},
		Mappings: []Mapping{
			{
				APIKey:    "startRecording",
				ConfigKey: "StartRecording",
				MapKey:    "start-recording",
			},
			{
				APIKey:    "stopRecording",
				ConfigKey: "StopRecording",
				MapKey:    "stop-recording",
			},
			{
				APIKey:    "updated",
				ConfigKey: "Updated",
				MapKey:    "updated",
				Converter: stringToTimeConverter,
			},
		},
	},
	{
		Name:   "location",
		Key:    config.LocationKey,
		Config: &config.Location{},
		Mappings: []Mapping{
			{
				APIKey:    "lat",
				ConfigKey: "Latitude",
				MapKey:    "latitude",
			},
			{
				APIKey:    "lng",
				ConfigKey: "Longitude",
				MapKey:    "longitude",
			},
		},
	},
	{
		Name:   "battery",
		Key:    config.BatteryKey,
		Config: &config.Battery{},
		Mappings: []Mapping{
			{
				APIKey:    "chemistry",
				ConfigKey: "Chemistry",
				MapKey:    "chemistry",
			},
			{
				APIKey:    "manualCellCount",
				ConfigKey: "ManualCellCount",
				MapKey:    "manual-cell-count",
			},
			{
				APIKey:    "updated",
				ConfigKey: "Updated",
				MapKey:    "updated",
				Converter: stringToTimeConverter,
			},
		},
	},
}

type CacophonyAPIInterface interface {
	GetDeviceSettings() (map[string]interface{}, error)
	UpdateDeviceSettings(settings map[string]interface{}) (map[string]interface{}, error)
}

type CacophonyConfigInterface interface {
	Unmarshal(key string, rawVal interface{}) error
	SetFromMap(sectionKey string, newConfig map[string]interface{}, force bool) error
	Write() error
}

type SyncService struct {
	apiClient CacophonyAPIInterface
	config    *config.Config
}

func NewSyncService() (*SyncService, error) {
	apiClient, err := api.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %v", err)
	}

	conf, err := config.New(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config handler: %v", err)
	}

	return &SyncService{
		apiClient: apiClient,
		config:    conf,
	}, nil
}

func (s *SyncService) PrintConfig() {
	content, err := os.ReadFile(filepath.Join(configDir, config.ConfigFileName))
	if err != nil {
		log.Errorf("Error reading config file: %v", err)
		return
	}

	inSecretsSection := false
	var out strings.Builder
	for line := range strings.SplitSeq(string(content), "\n") {
		// Just some white space.
		if strings.TrimSpace(line) == "" {
			out.WriteString(line + "\n")
			continue
		}
		// Detect when the secrets section starts [secrets].
		if strings.HasPrefix(line, "[secrets]") {
			inSecretsSection = true
			out.WriteString(line + "\n")
			continue
		}
		// Detect when secrets section ends.
		if strings.HasPrefix(line, "[") {
			inSecretsSection = false
		}
		if inSecretsSection {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				out.WriteString(fmt.Sprintf("%s = ************\n", parts[0]))
			} else {
				log.Warn("Not too sure how to parse line in secrets section")
			}
		} else {
			out.WriteString(line + "\n")
		}
	}

	log.Info("Current Config:\n" + out.String())
}

func (s *SyncService) syncSettings() error {
	deviceSettings, err := s.readCurrentSettings()
	if err != nil {
		return fmt.Errorf("failed to read current settings: %v", err)
	}
	log.Printf("Device Settings: %+v", deviceSettings)

	// Send config to server and get the updated settings
	serverSettings, err := s.uploadSettingsToAPI(deviceSettings)
	if err != nil {
		return fmt.Errorf("failed to synchronize settings with API: %v", err)
	}

	// Map server settings to expected config structure
	mappedSettings := s.mapServerSettingsToConfig(serverSettings)
	// Filter out unchanged settings
	filteredSettings := s.filterUnchangedSettings(mappedSettings, deviceSettings)

	if len(filteredSettings) == 0 {
		log.Println("No settings to update after filtering")
		return nil
	}

	if err := s.updateConfig(filteredSettings); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	return nil
}

func (s *SyncService) filterUnchangedSettings(mappedSettings, deviceSettings map[string]interface{}) map[string]interface{} {
	filteredSettings := make(map[string]interface{})

	log.Println("Starting to filter unchanged settings")

	for _, section := range ConfigSections {
		log.Printf("Checking section: %s", section.Key)

		if mappedSectionSettings, ok := mappedSettings[section.Key]; ok {
			mappedSection := mappedSectionSettings.(map[string]interface{})
			deviceSection, deviceOk := deviceSettings[section.Name].(map[string]interface{})

			if !deviceOk {
				log.Printf("Warning: Device settings for section %s not found or not a map", section.Name)
				filteredSettings[section.Key] = mappedSection
				continue
			}

			// Find the "updated" field for this section
			var updatedField string
			for _, mapping := range section.Mappings {
				if mapping.APIKey == "updated" {
					updatedField = mapping.MapKey
					break
				}
			}

			if updatedField != "" {
				mappedUpdateTime, mappedOk := mappedSection[updatedField].(time.Time)
				deviceUpdateTime, deviceOk := deviceSection["updated"].(time.Time)

				if mappedOk && deviceOk {
					log.Printf("Section %s - Mapped update time: %v, Device update time: %v",
						section.Key, mappedUpdateTime, deviceUpdateTime)

					// If the mapped update time is after the device update time, include this section
					if mappedUpdateTime.After(deviceUpdateTime) {
						log.Printf("Including section %s: mapped time is newer", section.Key)
						filteredSettings[section.Key] = mappedSection
					} else {
						log.Printf("Filtering out section %s: mapped time is not newer", section.Key)
					}
				} else {
					log.Printf("Warning: Couldn't compare update times for section %s. Including it to be safe.", section.Key)
					filteredSettings[section.Key] = mappedSection
				}
			} else {
				log.Printf("No 'updated' field found for section %s. Including it.", section.Key)
				filteredSettings[section.Key] = mappedSection
			}
		} else {
			log.Printf("Section %s not found in mapped settings", section.Key)
		}
	}

	log.Printf("Filtered settings: %+v", filteredSettings)
	return filteredSettings
}

func (s *SyncService) readCurrentSettings() (map[string]interface{}, error) {
	if err := s.config.Reload(); err != nil {
		return nil, err
	}
	settings := make(map[string]interface{})

	for _, section := range ConfigSections {
		err := s.config.Unmarshal(section.Key, section.Config)
		if err != nil {
			return nil, err
		}

		sectionData := reflect.ValueOf(section.Config).Elem()

		sectionSettings := make(map[string]interface{})
		for _, mapping := range section.Mappings {
			field := sectionData.FieldByName(mapping.ConfigKey)
			if field.IsValid() {
				sectionSettings[mapping.APIKey] = field.Interface()
			} else {
				log.Warnf("Field %s not found in section %s\n", mapping.ConfigKey, section.Key)
			}
		}

		settings[section.Name] = sectionSettings
	}

	return settings, nil
}

func (s *SyncService) mapServerSettingsToConfig(serverSettings map[string]interface{}) map[string]interface{} {
	mappedSettings := make(map[string]interface{})

	// Check if serverSettings is nil
	if serverSettings == nil {
		log.Println("Server settings are nil, returning empty mapped settings")
		return mappedSettings
	}

	for _, section := range ConfigSections {
		sectionSettings := make(map[string]interface{})

		// Check if the section exists in serverSettings
		if sectionData, ok := serverSettings[section.Name]; ok {
			// Check if sectionData is a map[string]interface{}
			if sectionMap, ok := sectionData.(map[string]interface{}); ok {
				for _, mapping := range section.Mappings {
					if value, ok := sectionMap[mapping.APIKey]; ok {
						// If this is the "updated" field, convert it to time.Time
						if mapping.APIKey == "updated" {
							if timeStr, ok := value.(string); ok {
								parsedTime, err := time.Parse(time.RFC3339Nano, timeStr)
								if err != nil {
									log.Printf("Error parsing time for %s in section %s: %v", mapping.APIKey, section.Name, err)
									sectionSettings[mapping.MapKey] = value
								} else {
									sectionSettings[mapping.MapKey] = parsedTime
									log.Printf("Converted time for %s in section %s: %v", mapping.APIKey, section.Name, parsedTime)
								}
							} else {
								log.Printf("Expected string for %s in section %s, got %T", mapping.APIKey, section.Name, value)
								sectionSettings[mapping.MapKey] = value
							}
						} else {
							sectionSettings[mapping.MapKey] = value
						}
					}
				}
			}
		}

		// Only add non-empty sections
		if len(sectionSettings) > 0 {
			mappedSettings[section.Key] = sectionSettings
		}
	}

	log.Printf("Final mapped settings: %+v", mappedSettings)
	return mappedSettings
}

func (s *SyncService) updateConfig(settings map[string]interface{}) error {
	fmt.Printf("Settings: %v\n", settings)
	for _, section := range ConfigSections {
		if sectionSettings, ok := settings[section.Key]; ok {
			newConfig := sectionSettings.(map[string]interface{})
			fmt.Printf("New config: %v\n", newConfig)
			err := s.config.SetFromMap(section.Key, newConfig, true)
			if err != nil {
				return fmt.Errorf("failed to set section %s: %v", section.Name, err)
			}
		}
	}

	if err := s.config.Write(); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}
	return nil
}

func (s *SyncService) uploadSettingsToAPI(settings map[string]interface{}) (map[string]interface{}, error) {
	settingsMap := make(map[string]interface{})

	for _, section := range ConfigSections {
		sectionMap := make(map[string]interface{})
		if date, ok := settings[section.Name].(map[string]interface{})["updated"]; ok && !isEmptyValue(date) {
			for _, mapping := range section.Mappings {
				if value, ok := settings[section.Name].(map[string]interface{})[mapping.APIKey]; ok {
					// Check if the value is non-empty before adding it to the map
					if !isEmptyValue(value) {
						log.Printf("Adding value to settings: %v", value)
						sectionMap[mapping.APIKey] = value
					}
				}
			}
		}
		if len(sectionMap) > 0 {
			settingsMap[section.Name] = sectionMap
		}
	}

	updatedSettings, err := s.apiClient.UpdateDeviceSettings(settingsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to update settings on API: %v", err)
	}
	log.Printf("Update Settings: %+v", updatedSettings)
	return updatedSettings, nil
}

// isEmptyValue checks if a value is considered empty
func isEmptyValue(v interface{}) bool {
	switch value := v.(type) {
	case nil:
		return true
	case string:
		return value == ""
	case int, int8, int16, int32, int64:
		return value == 0
	case uint, uint8, uint16, uint32, uint64:
		return value == 0
	case float32, float64:
		return value == float32(0)
	case bool:
		return false
	case time.Time:
		return value.IsZero()
	case []interface{}:
		return len(value) == 0
	case map[string]interface{}:
		return len(value) == 0
	default:
		return reflect.ValueOf(v).IsZero()
	}
}

var (
	log     = logging.NewLogger("info")
	version = "<not set>"
)

type Args struct {
	logging.LogArgs
}

var defaultArgs = Args{}

func procArgs(input []string) (Args, error) {
	args := defaultArgs

	parser, err := arg.NewParser(arg.Config{}, &args)
	if err != nil {
		return Args{}, err
	}
	err = parser.Parse(input)
	if errors.Is(err, arg.ErrHelp) {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}
	if errors.Is(err, arg.ErrVersion) {
		fmt.Println(version)
		os.Exit(0)
	}
	return args, err
}

func Run(inputArgs []string, ver string) error {
	version = ver
	args, err := procArgs(inputArgs)
	if err != nil {
		return fmt.Errorf("failed to parse args: %v", err)
	}
	log = logging.NewLogger(args.LogLevel)

	log.Println("Starting Cacophony Config Sync Service")

	log.Info("Running version: ", version)

	// Making runSyncChannel, string returned is the reason for the sync.
	runSyncChannel := makeSyncChannel()

	syncService, err := NewSyncService()
	if err != nil {
		return fmt.Errorf("failed to initialize sync service: %v", err)
	}

	modemSyncInterval := time.Minute * 30
	lastSyncTime := time.Time{}

	for {
		var syncReason string
		select {
		case syncReason = <-runSyncChannel:
		case <-time.After(syncInterval):
			syncReason = "interval sync"
		}

		if syncReason == "modem connected" && time.Since(lastSyncTime) < modemSyncInterval {
			log.Infof("Skipping sync from modem connect as sync was ran within %s", modemSyncInterval.String())
			continue
		}

		// TODO: If a sync occurs, and the file is changed. This will trigger another sync after the 10 second wait.
		//       This is just a minor issue at the moment and can be fixed later.
		//       Could compare the last config from the API and the local config, if they are the same, then don't sync again

		log.Info("Running sync for reason: ", syncReason)
		log.Println("Config before sync:")
		syncService.PrintConfig()
		// Perform a single sync operation
		if err := syncService.syncSettings(); err != nil {
			return fmt.Errorf("sync operation failed: %v", err)
		}
		log.Println("Config after sync:")
		syncService.PrintConfig()
		lastSyncTime = time.Now()
	}
}

func makeSyncChannel() chan string {
	// Make channel that will trigger a sync, the string is the reason for the sync.
	c := make(chan string, 10)

	// Add initial sync
	c <- "initial sync"

	// Setup modem connected signals to channel
	go func(c chan string) {
		failedCount := 0
		var modemConnectedChan chan time.Time
		for {
			var err error
			modemConnectedChan, err = modemlistener.GetModemConnectedSignalListener()
			if err != nil {
				// Will probably just be a system timing issue, try again in a second
				time.Sleep(time.Second)
				failedCount++
				if failedCount == 10 { // Just log this once
					log.Error("Failed to get modem connected signal listener")
				}
			} else {
				// Got the signal listener, break out of the loop
				break
			}
		}

		for {
			<-modemConnectedChan
			c <- "modem connected"
		}
	}(c)

	// Setup config file changes to channel
	go func(c chan string) {
		configFilePath := filepath.Join(configDir, config.ConfigFileName)
		fsEvents := make(chan notify.EventInfo, 16)
		if err := notify.Watch(configFilePath, fsEvents,
			notify.Write,
			notify.Create,
			notify.Rename,
			notify.Remove,
		); err != nil {
			log.Error("watch:", err)
			return
		}
		defer notify.Stop(fsEvents)

		fileChange := false
		for {
			// This select with the timeout is used so if the config file is changed, it will wait
			// 10 seconds before triggering the sync, and any more config changes will restart the timer.
			// This is so you won't get lots of sync calls when someone is on sidekick and changing the config lots
			select {
			case <-fsEvents:
				fileChange = true
				log.Info("Config file changes, waiting 10 seconds until triggering sync.")
			case <-time.After(10 * time.Second):
				if fileChange {
					log.Info("Triggering sync from config change.")
					c <- "config file changed"
					fileChange = false
				}
			}
		}
	}(c)

	return c
}

func emptyChannel(ch chan string) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	api "github.com/TheCacophonyProject/go-api"
	config "github.com/TheCacophonyProject/go-config"
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

// IMPORTANT:
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
				APIKey:    "powerOn",
				ConfigKey: "PowerOn",
				MapKey:    "power-on",
			},
			{
				APIKey:    "powerOff",
				ConfigKey: "PowerOff",
				MapKey:    "power-off",
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

const (
	configDir    = config.DefaultConfigDir
	syncInterval = time.Second * 10 // adjust as needed
)

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
	config    CacophonyConfigInterface
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

func (s *SyncService) Run(stopCh <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.syncSettings(); err != nil {
				log.Printf("Error syncing settings: %v", err)
			}
		case <-stopCh:
			return
		}
	}
}

func (s *SyncService) syncSettings() error {
	deviceSettings, err := s.readCurrentSettings()
	if err != nil {
		return fmt.Errorf("failed to read current settings: %v", err)
	}

	serverSettings, err := s.uploadSettingsToAPI(deviceSettings)
	if err != nil {
		return fmt.Errorf("failed to fetch settings from API: %v", err)
	}

	// Map server settings to expected config structure
	mappedSettings := s.mapServerSettingsToConfig(serverSettings)

	if err := s.updateConfig(mappedSettings); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	return nil
}

func (s *SyncService) fetchSettingsFromAPI() (map[string]interface{}, error) {
	serverSettings, err := s.apiClient.GetDeviceSettings()
	if err != nil {
		if err.Error() == "no settings found" { // Adjust this condition based on the actual error returned
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to get device settings from API: %v", err)
	}
	return serverSettings, nil
}

func (s *SyncService) readCurrentSettings() (map[string]interface{}, error) {
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
				fmt.Printf("Field %s not found in section %s\n", mapping.ConfigKey, section.Key)
			}
		}

		settings[section.Name] = sectionSettings
	}

	return settings, nil
}

func (s *SyncService) mapServerSettingsToConfig(serverSettings map[string]interface{}) map[string]interface{} {
	mappedSettings := make(map[string]interface{})

	for _, section := range ConfigSections {
		sectionSettings := make(map[string]interface{})
		for _, mapping := range section.Mappings {
			if value, ok := serverSettings[section.Name].(map[string]interface{})[mapping.APIKey]; ok {
				sectionSettings[mapping.MapKey] = value
			}
		}
		mappedSettings[section.Key] = sectionSettings
	}

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
		for _, mapping := range section.Mappings {
			if value, ok := settings[section.Name].(map[string]interface{})[mapping.APIKey]; ok {
				sectionMap[mapping.APIKey] = value
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
	return updatedSettings, nil
}

func main() {
	syncService, err := NewSyncService()
	if err != nil {
		log.Fatalf("Failed to initialize sync service: %v", err)
	}

	stopCh := make(chan struct{})
	go syncService.Run(stopCh, syncInterval)

	// Simulate a stop after a certain duration for demonstration
	time.Sleep(syncInterval + time.Second)
	close(stopCh)
}

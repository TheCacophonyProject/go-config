package main

import (
	"testing"
	"time"

	api "github.com/TheCacophonyProject/go-api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocking the API and Config components

type MockCacophonyAPI struct {
	mock.Mock
}

func (m *MockCacophonyAPI) GetDeviceSettings() (*api.Settings, error) {
	args := m.Called()
	return args.Get(0).(*api.Settings), args.Error(1)
}

func (m *MockCacophonyAPI) UpdateDeviceSettings(settings api.Settings) error {
	args := m.Called(settings)
	return args.Error(0)
}

type MockConfig struct {
	mock.Mock
	settings api.Settings
}

func (m *MockConfig) Unmarshal(prefix string, out interface{}) error {
	args := m.Called(prefix, out)
	*(out.(*api.Settings)) = m.settings
	return args.Error(0)
}

func (m *MockConfig) SetField(section string, key string, value string, quote bool) error {
	args := m.Called(section, key, value, quote)
	return args.Error(0)
}

func (m *MockConfig) Write() error {
	args := m.Called()
	return args.Error(0)
}

func TestSyncService(t *testing.T) {
	mockAPI := new(MockCacophonyAPI)
	mockConfig := new(MockConfig)

	serverSettings := &api.Settings{
		ReferenceImagePOV:            "server_pov",
		ReferenceImagePOVFileSize:    100,
		ReferenceImageInSitu:         "server_insitu",
		ReferenceImageInSituFileSize: 200,
		Success:                      true,
	}

	deviceSettings := api.Settings{
		ReferenceImagePOV:            "device_pov",
		ReferenceImagePOVFileSize:    50,
		ReferenceImageInSitu:         "device_insitu",
		ReferenceImageInSituFileSize: 150,
		Success:                      false,
	}

	mockAPI.On("GetDeviceSettings").Return(serverSettings, nil)
	mockAPI.On("UpdateDeviceSettings", mock.AnythingOfType("api.Settings")).Return(nil)

	mockConfig.settings = deviceSettings
	mockConfig.On("Unmarshal", "", mock.AnythingOfType("*api.Settings")).Return(nil)
	mockConfig.On("SetField", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)
	mockConfig.On("Write").Return(nil)

	syncService := &SyncService{
		apiClient: mockAPI,
		config:    mockConfig,
	}

	err := syncService.syncSettings()
	assert.NoError(t, err)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
}

func TestSyncService_Run(t *testing.T) {
	mockAPI := new(MockCacophonyAPI)
	mockConfig := new(MockConfig)

	serverSettings := &api.Settings{
		ReferenceImagePOV:            "server_pov",
		ReferenceImagePOVFileSize:    100,
		ReferenceImageInSitu:         "server_insitu",
		ReferenceImageInSituFileSize: 200,
		Success:                      true,
	}

	deviceSettings := api.Settings{
		ReferenceImagePOV:            "device_pov",
		ReferenceImagePOVFileSize:    50,
		ReferenceImageInSitu:         "device_insitu",
		ReferenceImageInSituFileSize: 150,
		Success:                      false,
	}

	mockAPI.On("GetDeviceSettings").Return(serverSettings, nil)
	mockAPI.On("UpdateDeviceSettings", mock.AnythingOfType("api.Settings")).Return(nil)

	mockConfig.settings = deviceSettings
	mockConfig.On("Unmarshal", "", mock.AnythingOfType("*api.Settings")).Return(nil)
	mockConfig.On("SetField", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)
	mockConfig.On("Write").Return(nil)

	syncService := &SyncService{
		apiClient: mockAPI,
		config:    mockConfig,
	}

	stopCh := make(chan struct{})
	go syncService.Run(stopCh, time.Millisecond*500) // Short interval for testing

	// Let the Run function execute at least once
	time.Sleep(time.Second * 1)

	close(stopCh) // Stop the ticker

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
}

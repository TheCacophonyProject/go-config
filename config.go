// go-config - Library for reading cacophony config files.
// Copyright (C) 2018, The Cacophony Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/TheCacophonyProject/event-reporter/v3/eventclient"
	"github.com/gofrs/flock"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/cast"
	"github.com/spf13/viper"

	toml "github.com/pelletier/go-toml"
)

type Config struct {
	v         *viper.Viper
	fileLock  *flock.Flock
	AutoWrite bool
	// accessedSections map[string]struct{} //TODO record each section accessed for restarting service purpose
}

const (
	DefaultConfigDir = "/etc/cacophony"
	ConfigFileName   = "config.toml"
	lockRetryDelay   = 678 * time.Millisecond
	TimeFormat       = time.RFC3339
)

type section struct {
	key         string
	mapToStruct func(map[string]interface{}) (interface{}, error)
	validate    func(interface{}) error
}

var (
	allSections               = map[string]section{} // each different section file has an init function that will add to this.
	allSectionDecodeHookFuncs = []mapstructure.DecodeHookFunc{}
)

// Helpers for testing purposes
var (
	fs           = afero.NewOsFs()
	now          = time.Now
	lockFilePath = func(configFile string) string {
		return configFile + ".lock"
	}
)
var (
	lockTimeout         = 10 * time.Second
	mapStrInterfaceType = reflect.TypeOf(map[string]interface{}{})
)

// New created a new config and loads files from the given directory
func New(dir string) (*Config, error) {
	// TODO Take service name and restart service if config changes
	configFile := path.Join(dir, ConfigFileName)
	c := &Config{
		v:         viper.New(),
		fileLock:  flock.New(lockFilePath(configFile)),
		AutoWrite: true,
	}
	c.v.SetFs(fs)
	c.v.SetConfigFile(configFile)
	if err := c.getFileLock(); err != nil {
		return nil, err
	}
	defer c.fileLock.Unlock()
	if err := c.v.ReadInConfig(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Unmarshal(key string, raw interface{}) error {
	return c.v.UnmarshalKey(key, raw)
}

// Set can only update one section at a time.
func (c *Config) Set(key string, value interface{}) error {
	if !checkIfSectionKey(key) {
		return notSectionKeyError(key)
	}
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	if err := c.Update(); err != nil {
		return err
	}
	kind := reflect.ValueOf(value).Kind()
	if kind == reflect.Struct || kind == reflect.Ptr {
		return c.setStruct(key, value)
	}
	c.set(key, value)
	if c.AutoWrite {
		return c.Write()
	}
	return nil
}

// SetFromMap can only update one section at a time.
func (c *Config) SetFromMap(sectionKey string, newConfig map[string]interface{}, force bool) error {
	if !checkIfSectionKey(sectionKey) {
		return notSectionKeyError(sectionKey)
	}
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	if err := c.Update(); err != nil {
		return err
	}

	// Convert to section for type conversion and other checks
	section := allSections[sectionKey]
	newStruct, err := section.mapToStruct(newConfig)
	if err != nil {
		if force {
			// If failed to convert new config map to a struct of that section then
			// try writing to section with a map instead.
			return c.Set(sectionKey, newConfig)
		}
		return err
	}

	// Pull out parts from section for writing to config as to not write zero values
	newMap, err := interfaceToMap(newStruct)
	if err != nil {
		return err
	}
	newMap = copyAndInsensitiviseMap(newMap)
	for key := range newConfig {
		val, ok := newMap[strings.ToLower(key)]
		if !ok {
			return fmt.Errorf("could not find key '%s' in map", key)
		}
		newConfig[key] = val
	}
	return c.Set(sectionKey, newConfig)
}

func (c *Config) SetField(sectionKey, valueKey, value string, force bool) error {
	if !checkIfSectionKey(sectionKey) {
		return notSectionKeyError(sectionKey)
	}

	section := allSections[sectionKey]
	s := map[string]interface{}{}
	c.Unmarshal(section.key, &s)
	s[valueKey] = value
	delete(s, "updated")
	return c.SetFromMap(sectionKey, s, force)
}

func (c *Config) Update() error {
	if err := c.getFileLock(); err != nil {
		return err
	}
	defer c.fileLock.Unlock()
	return c.v.ReadInConfig()
}

func (c *Config) Reload() error {
	configFile := c.v.ConfigFileUsed()
	// Need a new viper instance to clear old settings
	c.v = viper.New()
	c.v.SetFs(fs)
	c.v.SetConfigFile(configFile)
	return c.v.ReadInConfig()
}

// TODO Only update if given time is after the "udpate" field of the section updating and set "update" field to given time if updating
/*
func (c *Config) StrictSet(key string, value interface{}, time time.Time) error {
	return nil
}
*/

func (c *Config) Unset(key string) error {
	configMap := c.v.AllSettings()
	path := strings.Split(key, ".")
	deepestMap, err := deepSearch(configMap, path[0:len(path)-1])
	if err != nil {
		return err
	}
	delete(deepestMap, path[len(path)-1])
	tomlTree, err := toml.TreeFromMap(configMap)
	if err != nil {
		return err
	}
	configFile := c.v.ConfigFileUsed()
	// Need a new viper instance to clear old settings
	c.v = viper.New()
	c.v.SetFs(fs)
	c.v.SetConfigFile(configFile)
	var buf bytes.Buffer
	_, err = tomlTree.WriteTo(&buf)
	if err != nil {
		return err
	}
	if err := c.v.ReadConfig(bytes.NewReader(buf.Bytes())); err != nil {
		return err
	}
	c.v.Set(path[0]+".updated", now())
	if c.AutoWrite {
		return c.Write()
	}
	return nil
}

var errNoFileLock = errors.New("failed to get lock on file")

func (c *Config) getFileLock() error {
	lockCtx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	locked, err := c.fileLock.TryLockContext(lockCtx, lockRetryDelay)
	if err != nil {
		return err
	} else if !locked {
		return errNoFileLock
	}
	return nil
}

func interfaceToMap(value interface{}) (m map[string]interface{}, err error) {
	decodeHookFuncs := mapstructure.ComposeDecodeHookFunc(allSectionDecodeHookFuncs...)
	decoderConfig := mapstructure.DecoderConfig{
		DecodeHook: decodeHookFuncs,
		Result:     &m,
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return nil, err
	}
	err = decoder.Decode(value)
	return
}

func (c *Config) setStruct(key string, value interface{}) error {
	m, err := interfaceToMap(value)
	if err != nil {
		return err
	}
	c.set(key, m)
	if c.AutoWrite {
		return c.Write()
	}
	return nil
}

func (c *Config) Write() error {
	configMap := map[string]interface{}{}
	for key := range allSections {
		if key != SecretsKey {
			configMap[key] = c.Get(key)
		}
	}
	event := eventclient.Event{
		Timestamp: time.Now(),
		Type:      "config",
		Details:   configMap,
	}
	eventclient.AddEvent(event)
	eventclient.UploadEvents()
	return c.v.WriteConfig()
}

func notSectionKeyError(key string) error {
	return fmt.Errorf("'%s' is not a key for a section", key)
}

func checkIfSectionKey(key string) bool {
	_, ok := allSections[key]
	return ok
}

func (c *Config) set(key string, value interface{}) {
	c.v.Set(key, value)
	c.v.Set(strings.Split(key, ".")[0]+".updated", now())
}

func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

func SetFs(f afero.Fs) {
	fs = f
}

func SetLockFilePath(f func(string) string) {
	lockFilePath = f
}

func decodeStructFromMap(s interface{}, m map[string]interface{}, decodeHook interface{}) error {
	decoderConfig := mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(stringToDuration, stringToTime),
		Result:           s,
		WeaklyTypedInput: true,
		ErrorUnused:      true,
	}
	if decodeHook != nil {
		decoderConfig.DecodeHook = mapstructure.ComposeDecodeHookFunc(decodeHook, decoderConfig.DecodeHook)
	}
	decoder, err := mapstructure.NewDecoder(&decoderConfig)
	if err != nil {
		return err
	}
	return decoder.Decode(m)
}

func stringToDuration(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(time.Second) || f.Kind() != reflect.String {
		return data, nil
	}
	return time.ParseDuration(data.(string))
}

func stringToTime(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(time.Time{}) || f.Kind() != reflect.String {
		return data, nil
	}
	return time.Parse(TimeFormat, data.(string))
}

// Slightly edited from viper library: https://github.com/spf13/viper/blob/master/util.go#L199
// deepSearch scans deep maps, following the key indexes listed in the
// sequence "path".
// The last value is expected to be another map, and is returned.
//
// In case intermediate keys do not exist, or map to a non-map value, an error is returned
func deepSearch(m map[string]interface{}, path []string) (map[string]interface{}, error) {
	for _, k := range path {
		m2, ok := m[k]
		if !ok {
			return m, errors.New("error with following path in map")
		}
		m3, ok := m2.(map[string]interface{})
		if !ok {
			return m, errors.New("error with following path in map")
		}
		m = m3
	}
	return m, nil
}

// From viper library: https://github.com/spf13/viper/blob/master/util.go#L51
// copyAndInsensitiviseMap behaves like insensitiviseMap, but creates a copy of
// any map it makes case insensitive.
func copyAndInsensitiviseMap(m map[string]interface{}) map[string]interface{} {
	nm := make(map[string]interface{})

	for key, val := range m {
		lkey := strings.ToLower(key)
		switch v := val.(type) {
		case map[interface{}]interface{}:
			nm[lkey] = copyAndInsensitiviseMap(cast.ToStringMap(v))
		case map[string]interface{}:
			nm[lkey] = copyAndInsensitiviseMap(v)
		default:
			nm[lkey] = v
		}
	}

	return nm
}

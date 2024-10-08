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
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

func init() {
	allSections[LocationKey] = section{
		key:         LocationKey,
		mapToStruct: mapToLocation,
		validate:    validateLocation,
		defaultValue: func() interface{} {
			return nil
		},
		pointerValue: func() interface{} {
			return &Location{}
		},
	}
	allSectionDecodeHookFuncs = append(allSectionDecodeHookFuncs, locationToMap)
}

const LocationKey = "location"

type Location struct {
	Timestamp time.Time
	Accuracy  float32
	Altitude  float32
	Latitude  float32
	Longitude float32
}

// Default location used when setting windows relative to sunset/sunrise
func DefaultWindowLocation() Location {
	return Location{
		Latitude:  -43.5321,
		Longitude: 172.6362,
	}
}

func locationToMap(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != mapStrInterfaceType {
		return data, nil
	}
	switch f {
	case reflect.TypeOf(&Location{}):
		data = *(data.(*Location)) // follow the pointer
		fallthrough
	case reflect.TypeOf(Location{}):
		m := map[string]interface{}{}
		err := mapstructure.Decode(data, &m)
		m["Timestamp"] = data.(Location).Timestamp.Truncate(time.Second)
		return m, err
	default:
		return data, nil
	}
}

func mapToLocation(m map[string]interface{}) (interface{}, error) {
	var l Location
	if err := decodeStructFromMap(&l, m, stringToTime); err != nil {
		return nil, err
	}
	if err := validateLocation(&l); err != nil {
		return nil, err
	}
	return l, nil
}

// TODO
func validateLocation(l interface{}) error {
	return nil
}

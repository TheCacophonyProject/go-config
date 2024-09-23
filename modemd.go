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

const ModemdKey = "modemd"

func init() {
	allSections[ModemdKey] = section{
		key:         ModemdKey,
		mapToStruct: modemdMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultModemd()
		},
		pointerValue: func() interface{} {
			return &Modemd{}
		},
	}
	allSectionDecodeHookFuncs = append(allSectionDecodeHookFuncs, modemdToMap)
}

type Modemd struct {
	TestInterval           time.Duration `mapstructure:"test-interval"`
	InitialOnDuration      time.Duration `mapstructure:"initial-on-duration"`
	FindModemTimeout       time.Duration `mapstructure:"find-modem-timeout"`
	ConnectionTimeout      time.Duration `mapstructure:"connection-timeout"`
	RequestOnDuration      time.Duration `mapstructure:"request-on-duration"`
	RetryInterval          time.Duration `mapstructure:"retry-interval"`
	RetryFindModemInterval time.Duration `mapstructure:"retry-find-modem-interval"`
	MinConnDuration        time.Duration `mapstructure:"min-connection-duration"`
	MaxOffDuration         time.Duration `mapstructure:"max-off-duration"`
	Modems                 []Modem       `mapstructure:"modems"`
}

type Modem struct {
	Name            string `mapstructure:"name"`
	NetDev          string `mapstructure:"net-dev"`
	VendorProductID string `mapstructure:"vendor-product-id"`
}

func DefaultModemd() Modemd {
	return Modemd{
		TestInterval:           time.Minute * 5,
		InitialOnDuration:      time.Minute * 20,
		FindModemTimeout:       time.Minute * 3,
		ConnectionTimeout:      time.Minute * 3,
		RequestOnDuration:      time.Minute * 2,
		RetryInterval:          time.Minute * 20,
		RetryFindModemInterval: time.Hour * 3,
		MinConnDuration:        time.Minute,
		MaxOffDuration:         time.Hour * 24,
		Modems: []Modem{
			{Name: "Huawei 4G modem", NetDev: "eth1", VendorProductID: "12d1:14db"},
			{Name: "Spark 3G modem", NetDev: "usb0", VendorProductID: "19d2:1405"},
			{Name: "Qualcomm", NetDev: "usb0", VendorProductID: "1e0e:9011"},
		},
	}
}

func modemdToMap(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != mapStrInterfaceType {
		return data, nil
	}
	switch f {
	case reflect.TypeOf(&Modemd{}):
		data = *(data.(*Modemd)) // follow the pointer
		fallthrough
	case reflect.TypeOf(Modemd{}):
		m := map[string]interface{}{}
		if err := mapstructure.Decode(data, &m); err != nil {
			return nil, err
		}
		modems := []map[string]interface{}{}
		if err := mapstructure.Decode(data.(Modemd).Modems, &modems); err != nil {
			return nil, err
		}
		m["modems"] = modems
		return m, nil
	default:
		return data, nil
	}
}

func modemdMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Modemd
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}

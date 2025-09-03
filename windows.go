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
	"fmt"
	"time"
)

func init() {
	allSections[WindowsKey] = section{
		key:         WindowsKey,
		mapToStruct: windowsMapToStruct,
		validate:    noValidateFunc,
		defaultValue: func() interface{} {
			return DefaultWindows()
		},
		pointerValue: func() interface{} {
			return &Windows{}
		},
	}
}

const WindowsKey = "windows"

type Windows struct {
	StartRecording string    `mapstructure:"start-recording"`
	StopRecording  string    `mapstructure:"stop-recording"`
	Updated        time.Time `mapstructure:"updated"`
}

func DefaultWindows() Windows {
	return Windows{
		StartRecording: "-30m",
		StopRecording:  "+30m",
		Updated:        time.Now(),
	}
}

func noValidateFunc(s interface{}) error {
	return nil
}

func windowsMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s Windows
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}

	timeDurs := []string{s.StartRecording, s.StopRecording}
	for _, timeDur := range timeDurs {
		if timeDur != "" && !checkIfTimeOrDuration(timeDur) {
			return nil, fmt.Errorf("could not parse '%s' as a time or duration", timeDur)
		}
	}

	return s, nil
}

func checkIfTimeOrDuration(timeStr string) bool {
	if _, err := time.Parse("15:04", timeStr); err == nil {
		return true
	}
	if _, err := time.ParseDuration(timeStr); err == nil {
		return true
	}
	return false
}

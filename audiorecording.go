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

const AudioRecordingKey = "audio-recording"

func init() {
	allSections[AudioRecordingKey] = section{
		key:         AudioRecordingKey,
		mapToStruct: audioRecordingMapToStruct,
		validate:    noValidateFunc,
	}
}

type AudioRecording struct {
	AudioMode string `mapstructure:"audio-mode"`
	AudioSeed uint32 `mapstructure:"random-seed"`
}

func DefaultAudioRecording() AudioRecording {
	return AudioRecording{
		AudioMode: "Disabled",
		AudioSeed: 0,
	}
}

func audioRecordingMapToStruct(m map[string]interface{}) (interface{}, error) {
	var s AudioRecording
	if err := decodeStructFromMap(&s, m, nil); err != nil {
		return nil, err
	}
	return s, nil
}

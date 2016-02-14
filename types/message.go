package types

import (
	"encoding/json"
)

type EnumMessageType int

const (
	EnumMessageControl EnumMessageType = 1
	EnumMessageMove    EnumMessageType = 2
	//EnumMessage EnumMessageType = 2
)

const (
	EnumControlNewInstance        int = 1
	EnumControlStartClientSession int = 2
	EnumControlStartServerSession int = 3
	EnumControlNewPlayerConnected int = 4
	EnumControlCreateSession      int = 5
	EnumControlConnectSession     int = 6
)

type Message struct {
	Type     EnumMessageType `json:"t"`
	SubType  int             `json:"s"`
	Message  string          `json:"m"`
	ClientId string          `json:"c,omitempty"`
}

func (m *Message) ToString() (string, error) {
	res, err := json.Marshal(m)
	if nil == err {
		return string(res), nil
	}
	return "", err
}

func MessageFromString(inString string) (*Message, error) {
	m := new(Message)
	err := json.Unmarshal([]byte(inString), m)
	return m, err
}

type NewInstanceConfig struct {
	Websocket   bool `json:"w"` // Set to false to switch to WebRtc
	Multiplayer bool `json:"m"` // Multiple player can connect
}

func NewInstanceConfigFromString(inString string) (*NewInstanceConfig, error) {
	c := new(NewInstanceConfig)
	err := json.Unmarshal([]byte(inString), c)
	return c, err
}

package types

import (
	"encoding/json"
)

type EnumMessageType int

const (
	EnumMessageControl EnumMessageType = 0
	EnumMessageMove    EnumMessageType = 1
	//EnumMessage EnumMessageType = 2
)

const (
	EnumControlNewInstance        int = 0
	EnumControlStartClientSession int = 1
	EnumControlStartServerSession int = 2
	EnumControlNewPlayerConnected int = 3
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
	var m Message
	err := json.Unmarshal([]byte(inString), &m)
	if nil != err {
		return nil, err
	}
	return &m, nil
}

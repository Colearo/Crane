package utils

import (
	"encoding/json"
)

const (
	JOIN_REQUEST        = "join_reqeust"
	TOPO_SUBMISSION     = "topo_submission"
	TOPO_SUBMISSION_RES = "topo_submission_response"
	BOLT_DISPATCH       = "bolt_dispatch"
	CONN_NOTIFY         = "conn_notify"
	GROUPING_BY_FIELD   = "grouping_by_field"
	GROUPING_BY_SHUFFLE = "grouping_by_shuffle"
)

type PayloadHeader struct {
	Type string
}

type PayloadMessage struct {
	Header  PayloadHeader
	Content []byte
}

type JoinRequest struct {
	Name string
}

type BoltTaskMessage struct {
	Name                 string
	Port                 string
	PrevBoltAddr         []string
	PrevBoltGroupingHind string
	PrevBoltFieldIndex   int
	SuccBoltGroupingHint string
	SuccBoltFieldIndex   int
	PluginFile           string
	PluginSymbol         string
}



func Marshal(contentType string, content interface{}) ([]byte, error) {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	msg := PayloadMessage{
		PayloadHeader{Type: contentType},
		contentBytes,
	}

	return json.Marshal(msg)
}

func CheckType(raw []byte) *PayloadMessage {
	payload := &PayloadMessage{}
	json.Unmarshal(raw, payload)
	return payload
}

func Unmarshal(raw []byte, content interface{}) {
	json.Unmarshal(raw, content)
}

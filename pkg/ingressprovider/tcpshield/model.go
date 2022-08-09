package tcpshield

import "time"

type DomainDescriptor struct {
	Name         string `json:"name"`
	BackendSetId uint32 `json:"backend_set_id,omitempty"`
	BAC          bool   `json:"bac"`
}

type Domain struct {
	Id        uint32    `json:"id"`
	Verified  bool      `json:"verified"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	DomainDescriptor
}

type DomainList []*Domain

type DomainResponse struct {
	Data *Domain `json:"data"`
}

type BackendSetDescriptor struct {
	Name          string   `json:"name"`
	ProxyProtocol bool     `json:"proxy_protocol"`
	Backends      []string `json:"backends"`
}

type BackendSet struct {
	Id        uint32     `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	BackendSetDescriptor
}

type BackendSetList []*BackendSet

type BackendSetResponse struct {
	Data *struct {
		Id uint32 `json:"id"`
	} `json:"data"`
}

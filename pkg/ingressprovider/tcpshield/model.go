/*
 *     Singularity is an open-source game server orchestration framework
 *     Copyright (C) 2022 Innit Incorporated
 *
 *     This program is free software: you can redistribute it and/or modify
 *     it under the terms of the GNU Affero General Public License as published
 *     by the Free Software Foundation, either version 3 of the License, or
 *     (at your option) any later version.
 *
 *     This program is distributed in the hope that it will be useful,
 *     but WITHOUT ANY WARRANTY; without even the implied warranty of
 *     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *     GNU Affero General Public License for more details.
 *
 *     You should have received a copy of the GNU Affero General Public License
 *     along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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

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

package ingressprovider

import "net"

type Provider interface {
	// Create creates an ingress and return the id
	Create(hostName string, backendSet []*Backend) (string, error)

	// Update updates an existing ingress' backends
	Update(hostName string, backendSet []*Backend) error

	// Delete deletes an existing ingress
	Delete(id string) error
}

// Backend represents a Minecraft server's connection details
type Backend struct {
	IP   net.IP
	Port uint16
}

// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"errors"
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
)

var (
	ErrHostInvalid = errors.New("invalid host")
	ErrShutdown    = errors.New("listener shut down")
)

// HostNode is the coordinator node for host data
type HostNode struct {
	*host.Host
	version interface{}
}

// ID implements zzk.Node
func (node *HostNode) GetID() string {
	return node.ID
}

// Create implements zzk.Node
func (node *HostNode) Create(conn client.Connection) error {
	return AddHost(conn, node.Host)
}

// Update implements zzk.Node
func (node *HostNode) Update(conn client.Connection) error {
	return UpdateHost(conn, node.Host)
}

// Version implements client.Node
func (node *HostNode) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *HostNode) SetVersion(version interface{}) {
	node.version = version
}

// SyncHosts synchronizes the hosts on the coordinator with the provided list
func SyncHosts(conn client.Connection, hosts []host.Host) error {
	nodes := make([]zzk.Node, len(hosts))
	for i := range hosts {
		nodes[i] = &HostNode{Host: &hosts[i]}
	}
	return zzk.Sync(conn, nodes, "/hosts")
}

func AddHost(conn client.Connection, host *host.Host) error {
	hpth := path.Join("/hosts", host.ID)
	hnode := HostNode{Host: host}
	return conn.Create(hpth, &hnode)
}

func UpdateHost(conn client.Connection, host *host.Host) error {
	hpth := path.Join("/hosts", host.ID)
	hnode := HostNode{Host: host}
	return conn.Set(hpth, &hnode)
}

func RemoveHost(cancel <-chan interface{}, conn client.Connection, hostID string) error {
	hpth := path.Join("/hosts", hostID)
	if ok, err := conn.Exists(hpth); err != nil {
		return err
	} else if !ok {
		return nil
	}

	// stop all the instances running on that host
	ch, err := conn.Children(path.Join(hpth, "instances"))
	if err != nil && err != client.ErrNoNode {
		return err
	}
	for _, stateID := range ch {
		if err := StopServiceInstance(conn, hostID, stateID); err != nil {
			return err
		}
	}

	stop := make(chan struct{})
	defer func() { close(stop) }()

	// wait for all the instances to die
	for {
		ch, ev, err := conn.ChildrenW(path.Join(hpth, "instances"), stop)
		if err != nil {
			return err
		}
		if len(ch) == 0 {
			break
		}
		select {
		case <-ev:
		case <-cancel:
			return ErrShutdown
		}
		close(stop)
		stop = make(chan struct{})
	}

	return conn.Delete(hpth)
}

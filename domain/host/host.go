// Copyright 2014 The Serviced Authors.
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

package host

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/servicedversion"
	"github.com/zenoss/glog"
)

//Host that runs the control center agent.
type Host struct {
	ID             string // Unique identifier, default to hostid
	Name           string // A label for the host, eg hostname, role
	PoolID         string // Pool that the Host belongs to
	IPAddr         string // The IP address the host can be reached at from a serviced master
	RPCPort        int    // The RPC port of the host
	Cores          int    // Number of cores available to serviced
	Memory         uint64 // Amount of RAM (bytes) available to serviced
	PrivateNetwork string // The private network where containers run, eg 172.16.42.0/24
	CreatedAt      time.Time
	UpdatedAt      time.Time
	IPs            []HostIPResource // The static IP resources available on the host
	KernelVersion  string
	KernelRelease  string
	ServiceD       struct {
		Version   string
		Date      string
		Gitcommit string
		Gitbranch string
		Giturl    string
		Buildtag  string
		Release   string
	}
	MonitoringProfile domain.MonitorProfile
	datastore.VersionedEntity
}

// Equals verifies whether two host objects are equal
func (a *Host) Equals(b *Host) bool {
	if a.ID != b.ID {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.PoolID != b.PoolID {
		return false
	}
	if a.IPAddr != b.IPAddr {
		return false
	}
	if a.RPCPort != b.RPCPort {
		return false
	}
	if a.Cores != b.Cores {
		return false
	}
	if a.Memory != b.Memory {
		return false
	}
	if a.PrivateNetwork != b.PrivateNetwork {
		return false
	}
	if a.KernelVersion != b.KernelVersion {
		return false
	}
	if a.KernelRelease != b.KernelRelease {
		return false
	}
	if !reflect.DeepEqual(a.IPs, b.IPs) {
		return false
	}
	if a.CreatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	if a.UpdatedAt.Unix() != b.UpdatedAt.Unix() {
		return false
	}
	if a.ServiceD != b.ServiceD {
		return false
	}
	if !a.MonitoringProfile.Equals(&b.MonitoringProfile) {
		return false
	}

	return true
}

//HostIPResource contains information about a specific IP available as a resource
type HostIPResource struct {
	HostID        string
	IPAddress     string
	InterfaceName string
	MACAddress    string
}

// New creates a new empty host
func New() *Host {
	host := &Host{}
	return host
}

// Build creates a Host type from the current host machine, filling out fields using the current machines attributes.
// The IP param is a routable IP used to connect to to the Host, if empty an IP from the available IPs will be used.
// The poolid param is the pool the host should belong to.  Optional list of IP address strings to set as available IP
// resources, if not set the IP used for the host will be given as an IP Resource. If any IP is not a valid IP on the
// machine return error.
func Build(ip string, rpcport string, poolid string, ipAddrs ...string) (*Host, error) {
	if strings.TrimSpace(poolid) == "" {
		return nil, errors.New("empty poolid not allowed")
	}

	rpcPort, err := strconv.Atoi(rpcport)
	if err != nil {
		return nil, err
	}
	host, err := currentHost(ip, rpcPort, poolid)
	if err != nil {
		return nil, err
	}

	if len(ipAddrs) == 0 {
		// use the default IP of the host if specific IPs have not been requested
		ipAddrs = append(ipAddrs, host.IPAddr)
	}
	glog.Infof("building with ipsAddrs: %v [%d]", ipAddrs, len(ipAddrs))
	hostIPs, err := getIPResources(host.ID, ipAddrs...)
	if err != nil {
		return nil, err
	}
	host.IPs = hostIPs

	// get embedded host information
	host.ServiceD.Version = servicedversion.Version
	host.ServiceD.Gitbranch = servicedversion.Gitbranch
	host.ServiceD.Gitcommit = servicedversion.Gitcommit
	host.ServiceD.Giturl = servicedversion.Giturl
	host.ServiceD.Date = servicedversion.Date
	host.ServiceD.Buildtag = servicedversion.Buildtag
	host.ServiceD.Release = servicedversion.Release

	*host = *host

	return host, nil
}

//UpdateHostInfo returns a new host with updated hardware and software info. Does not update poor or IP information
func UpdateHostInfo(h Host) (Host, error) {
	currentHost, err := currentHost(h.IPAddr, h.RPCPort, h.PoolID)
	if err != nil {
		return Host{}, err
	}

	//update the passed in *copy* so we don't have to deal with new non hardware fields later on
	h.Memory = currentHost.Memory
	h.Cores = currentHost.Cores
	h.KernelRelease = currentHost.KernelRelease
	h.KernelVersion = currentHost.KernelVersion
	h.PrivateNetwork = currentHost.PrivateNetwork
	h.ServiceD = currentHost.ServiceD

	return h, nil
}

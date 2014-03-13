// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// BlkioIoServiced stores data from /sys/fs/cgroup/blkio/blkio.io_serviced.
type BlkioIoServiced struct {
	Total int64
}

// ReadBlkioIoServiced fills out and returns a BlkioIoServiced struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/blkio/blkio.io_serviced is used.
func ReadBlkioIoServiced(fileName string) BlkioIoServiced {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/blkio/blkio.io_serviced"
	}
	stat := BlkioIoServiced{}
	kv, _ := parseSSKVint64(fileName)
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}

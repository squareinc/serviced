// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/cli/cmd"
	"github.com/control-center/serviced/servicedversion"
)

const conffile = "/etc/default/serviced"

var Version string
var Date string
var Gitcommit string
var Gitbranch string
var Giturl string
var Buildtag string

func main() {
	servicedversion.Version = Version
	servicedversion.Date = Date
	servicedversion.Gitcommit = Gitcommit
	servicedversion.Gitbranch = Gitbranch
	servicedversion.Giturl = Giturl
	servicedversion.Buildtag = Buildtag

	config, err := getConfigs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: could not read default configs: %s\n", err)
	}

	// DEBUG: KWW: Restore original code
	//cmd.New(api.New(), config).Run(os.Args)
	mycmd := cmd.New(api.New(), config)
	fmt.Println("--- os.Args ---")
	for _, osarg := range os.Args {
		fmt.Println(osarg)
	}
	mycmd.Run(os.Args)
}

func getConfigs(args []string) (*cmd.EnvironConfigReader, error) {
	var filename string
	for i, arg := range args {
		if strings.HasPrefix(arg, "--config-file") {
			if idx := strings.IndexByte(arg, '='); idx >= 0 {
				filename = arg[idx+1:]
			} else if i+1 < len(args) {
				if !strings.HasPrefix(args[i+1], "-") {
					filename = args[i+1]
				}
			}
			filename = strings.Trim(filename, "\"")
			break
		}
	}
	if filename = strings.TrimSpace(filename); filename == "" {
		filename = conffile
	}
	return cmd.NewEnvironConfigReader(filename, "SERVICED_")
}

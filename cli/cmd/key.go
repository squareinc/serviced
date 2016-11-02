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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
)

func (c *ServicedCli) initKey() {
	key_command := cli.Command{
		Name:        "key",
		Usage:       "Displays host's public key",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Shows host public key",
				Description:  "serviced key list HostID",
				BashComplete: c.printHostsFirst,
				Action:       c.cmdHostKey,
			}, {
				Name:         "reset",
				Usage:        "Regenerate host key",
				Description:  "serviced key reset HostID",
				BashComplete: c.printHostsFirst,
				Action:       c.cmdKeyReset,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "key-file, k",
						Value: "",
						Usage: "Name of the output host key file",
					},
					cli.BoolFlag{
						Name:  "register, r",
						Usage: "Register delegate keys on the host via ssh",
					},
				},
			},
		},
	}

	c.app.Commands = append(c.app.Commands, key_command)
}

func (c *ServicedCli) cmdHostKey(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list")
		return
	}
	key, err := c.driver.GetHostPublicKey(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not retrieve host's public key. ", err.Error())
		return
	}
	fmt.Printf(string(key))
}

func (c *ServicedCli) cmdKeyReset(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list")
		return
	}

	hostID := args[0]

	host, err := c.driver.GetHost(hostID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get host %s: %s", hostID, err.Error())
		return
	}

	key, err := c.driver.ResetHostKey(hostID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not reset host's public key. ", err.Error())
		return
	}

	keyfileName := ctx.String("key-file")
	registerHost := ctx.Bool("register")
	c.outputDelegateKey(host, key, keyfileName, registerHost)
}

func (c *ServicedCli) outputDelegateKey(host *host.Host, keyData []byte, keyfileName string, register bool) {
	writeKeyFile := false
	if register {
		prompt := utils.Isatty(os.Stdin) && utils.Isatty(os.Stdout)
		if err := c.driver.RegisterRemoteHost(host, keyData, prompt); err != nil {
			fmt.Fprintf(os.Stderr, "Error registering host: %s\n", err.Error())
			writeKeyFile = true
		} else {
			fmt.Println("Registered host at", host.IPAddr)
		}
	} else {
		writeKeyFile = true
	}

	if keyfileName != "" {
		writeKeyFile = true
	}

	if writeKeyFile == true {
		if keyfileName == "" {
			keyfileName = fmt.Sprintf("IP-%s.delegate.key", strings.Replace(host.IPAddr, ".", "-", -1))
		}
		if keyfileName, err := filepath.Abs(keyfileName); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing delegate key file \"%s\": %s\n", keyfileName, err.Error())
			return
		}
		if err := c.driver.WriteDelegateKey(keyfileName, keyData); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing delegate key file \"%s\": %s\n", keyfileName, err.Error())
		}
		fmt.Println("Wrote delegate key file to", keyfileName)
	}
	fmt.Println(host.ID)
}

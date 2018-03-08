/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"github.com/GoogleCloudPlatform/k8s-container-builder/pkg/constants"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

type RunCommand struct {
	cmd *instructions.RunCommand
}

func (r *RunCommand) ExecuteCommand() error {
	var newCommand []string
	if r.cmd.PrependShell {
		// This is the default shell on Linux
		shell := []string{"/bin/sh", "-c"}
		newCommand = append(shell, strings.Join(r.cmd.CmdLine, " "))
	} else {
		newCommand = r.cmd.CmdLine
	}

	logrus.Infof("cmd: %s", newCommand[0])
	logrus.Infof("args: %s", newCommand[1:])

	cmd := exec.Command(newCommand[0], newCommand[1:]...)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// FilesToSnapshot returns nil for this command because we don't know which files
// have changed, so we snapshot the entire system.
func (r *RunCommand) FilesToSnapshot() []string {
	return nil
}

// Author returns some information about the command for the image config
func (r *RunCommand) Author() string {
	author := []string{constants.Author}
	cmdLine := strings.Join(r.cmd.CmdLine, " ")
	if r.cmd.PrependShell {
		shell := []string{"/bin/sh", "-c"}
		author = append(author, shell...)
		return strings.Join(author, " ") + " " + cmdLine
	}
	author = append(author, cmdLine)
	return strings.Join(author, " ")
}

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
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type EntrypointCommand struct {
	cmd *instructions.EntrypointCommand
}

// ExecuteCommand handles command processing similar to CMD and RUN,
func (e *EntrypointCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	var newCommand []string
	if e.cmd.PrependShell {
		// This is the default shell on Linux
		var shell []string
		if len(config.Shell) > 0 {
			shell = config.Shell
		} else {
			shell = append(shell, "/bin/sh", "-c")
		}

		newCommand = append(shell, strings.Join(e.cmd.CmdLine, " "))
	} else {
		newCommand = e.cmd.CmdLine
	}

	config.Entrypoint = newCommand
	return nil
}

// FilesToSnapshot returns an empty array since this is a metadata command
func (e *EntrypointCommand) FilesToSnapshot() []string {
	return []string{}
}

// String returns some information about the command for the image config history
func (e *EntrypointCommand) String() string {
	return e.cmd.String()
}

// CacheCommand returns false since this command shouldn't be cached
func (e *EntrypointCommand) CacheCommand() bool {
	return false
}

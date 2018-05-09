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
	"github.com/docker/docker/builder/dockerfile"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
	"strings"
)

type ArgCommand struct {
	cmd *instructions.ArgCommand
}

// ExecuteCommand only needs to add this ARG key/value as seen
func (r *ArgCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("ARG")
	buildArgs.AddArg(r.cmd.Key, r.cmd.Value)
	return nil
}

// FilesToSnapshot returns an empty array since this command only touches metadata.
func (r *ArgCommand) FilesToSnapshot() []string {
	return []string{}
}

// CreatedBy returns some information about the command for the image config history
func (r *ArgCommand) CreatedBy() string {
	return strings.Join([]string{r.cmd.Name(), r.cmd.Key}, " ")
}

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
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
)

type ArgCommand struct {
	cmd *instructions.ArgCommand
}

// ExecuteCommand only needs to add this ARG key/value as seen
func (r *ArgCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedKey, err := util.ResolveEnvironmentReplacement(r.cmd.Key, replacementEnvs, false)
	if err != nil {
		return err
	}
	var resolvedValue *string
	if r.cmd.Value != nil {
		value, err := util.ResolveEnvironmentReplacement(*r.cmd.Value, replacementEnvs, false)
		if err != nil {
			return err
		}
		resolvedValue = &value
	}
	buildArgs.AddArg(resolvedKey, resolvedValue)
	return nil
}

// FilesToSnapshot returns an empty array since this command only touches metadata.
func (r *ArgCommand) FilesToSnapshot() []string {
	return []string{}
}

// String returns some information about the command for the image config history
func (r *ArgCommand) String() string {
	return r.cmd.String()
}

// CacheCommand returns false since this command shouldn't be cached
func (r *ArgCommand) CacheCommand() bool {
	return false
}

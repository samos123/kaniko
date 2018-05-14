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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
	"github.com/sirupsen/logrus"
)

type WorkdirCommand struct {
	cmd           *instructions.WorkdirCommand
	snapshotFiles []string
}

func (w *WorkdirCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: workdir")
	workdirPath := w.cmd.Path
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedWorkingDir, err := util.ResolveEnvironmentReplacement(workdirPath, replacementEnvs, true)
	if err != nil {
		return err
	}
	if filepath.IsAbs(resolvedWorkingDir) {
		config.WorkingDir = resolvedWorkingDir
	} else {
		config.WorkingDir = filepath.Join(config.WorkingDir, resolvedWorkingDir)
	}
	logrus.Infof("Changed working directory to %s", config.WorkingDir)
	w.snapshotFiles = []string{config.WorkingDir}
	return os.MkdirAll(config.WorkingDir, 0755)
}

// FilesToSnapshot returns the workingdir, which should have been created if it didn't already exist
func (w *WorkdirCommand) FilesToSnapshot() []string {
	return w.snapshotFiles
}

// CreatedBy returns some information about the command for the image config history
func (w *WorkdirCommand) CreatedBy() string {
	return w.cmd.Name() + " " + w.cmd.Path
}

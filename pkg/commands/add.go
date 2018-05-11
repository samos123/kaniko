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
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/v1"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

type AddCommand struct {
	cmd           *instructions.AddCommand
	buildcontext  string
	snapshotFiles []string
}

// ExecuteCommand executes the ADD command
// Special stuff about ADD:
// 	1. If <src> is a remote file URL:
// 		- destination will have permissions of 0600
// 		- If remote file has HTTP Last-Modified header, we set the mtime of the file to that timestamp
// 		- If dest doesn't end with a slash, the filepath is inferred to be <dest>/<filename>
// 	2. If <src> is a local tar archive:
// 		-If <src> is a local tar archive, it is unpacked at the dest, as 'tar -x' would
func (a *AddCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	srcs := a.cmd.SourcesAndDest[:len(a.cmd.SourcesAndDest)-1]
	dest := a.cmd.SourcesAndDest[len(a.cmd.SourcesAndDest)-1]

	logrus.Infof("cmd: Add %s", srcs)
	logrus.Infof("dest: %s", dest)

	// First, resolve any environment replacement
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedEnvs, err := util.ResolveEnvironmentReplacementList(a.cmd.SourcesAndDest, replacementEnvs, true)
	if err != nil {
		return err
	}
	dest = resolvedEnvs[len(resolvedEnvs)-1]
	// Resolve wildcards and get a list of resolved sources
	srcs, err = util.ResolveSources(resolvedEnvs, a.buildcontext)
	if err != nil {
		return err
	}
	var unresolvedSrcs []string
	// If any of the sources are local tar archives:
	// 	1. Unpack them to the specified destination
	// If any of the sources is a remote file URL:
	//	1. Download and copy it to the specified dest
	// Else, add to the list of unresolved sources
	for _, src := range srcs {
		fullPath := filepath.Join(a.buildcontext, src)
		if util.IsSrcRemoteFileURL(src) {
			urlDest := util.URLDestinationFilepath(src, dest, config.WorkingDir)
			logrus.Infof("Adding remote URL %s to %s", src, urlDest)
			if err := util.DownloadFileToDest(src, urlDest); err != nil {
				return err
			}
			a.snapshotFiles = append(a.snapshotFiles, urlDest)
		} else if util.IsFileLocalTarArchive(fullPath) {
			logrus.Infof("Unpacking local tar archive %s to %s", src, dest)
			if err := util.UnpackLocalTarArchive(fullPath, dest); err != nil {
				return err
			}
			// Add the unpacked files to the snapshotter
			filesAdded, err := util.Files(dest)
			if err != nil {
				return err
			}
			logrus.Debugf("Added %v from local tar archive %s", filesAdded, src)
			a.snapshotFiles = append(a.snapshotFiles, filesAdded...)
		} else {
			unresolvedSrcs = append(unresolvedSrcs, src)
		}
	}
	// With the remaining "normal" sources, create and execute a standard copy command
	if len(unresolvedSrcs) == 0 {
		return nil
	}

	copyCmd := CopyCommand{
		cmd: &instructions.CopyCommand{
			SourcesAndDest: append(unresolvedSrcs, dest),
		},
		buildcontext: a.buildcontext,
	}
	if err := copyCmd.ExecuteCommand(config, buildArgs); err != nil {
		return err
	}
	a.snapshotFiles = append(a.snapshotFiles, copyCmd.snapshotFiles...)
	return nil
}

// FilesToSnapshot should return an empty array if still nil; no files were changed
func (a *AddCommand) FilesToSnapshot() []string {
	return a.snapshotFiles
}

// CreatedBy returns some information about the command for the image config
func (a *AddCommand) CreatedBy() string {
	return strings.Join(a.cmd.SourcesAndDest, " ")
}

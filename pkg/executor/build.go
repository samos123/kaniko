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

package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/kaniko/pkg/cache"
	"github.com/GoogleContainerTools/kaniko/pkg/commands"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"
	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
)

// This is the size of an empty tar in Go
const emptyTarSize = 1024

// stageBuilder contains all fields necessary to build one stage of a Dockerfile
type stageBuilder struct {
	stage           config.KanikoStage
	image           v1.Image
	cf              *v1.ConfigFile
	snapshotter     *snapshot.Snapshotter
	baseImageDigest string
	opts            *config.KanikoOptions
}

// newStageBuilder returns a new type stageBuilder which contains all the information required to build the stage
func newStageBuilder(opts *config.KanikoOptions, stage config.KanikoStage) (*stageBuilder, error) {
	sourceImage, err := util.RetrieveSourceImage(stage, opts.BuildArgs, opts)
	if err != nil {
		return nil, err
	}
	imageConfig, err := util.RetrieveConfigFile(sourceImage)
	if err != nil {
		return nil, err
	}
	if err := resolveOnBuild(&stage, &imageConfig.Config); err != nil {
		return nil, err
	}
	hasher, err := getHasher(opts.SnapshotMode)
	if err != nil {
		return nil, err
	}
	l := snapshot.NewLayeredMap(hasher, util.CacheHasher())
	snapshotter := snapshot.NewSnapshotter(l, constants.RootDir)

	digest, err := sourceImage.Digest()
	if err != nil {
		return nil, err
	}
	return &stageBuilder{
		stage:           stage,
		image:           sourceImage,
		cf:              imageConfig,
		snapshotter:     snapshotter,
		baseImageDigest: digest.String(),
		opts:            opts,
	}, nil
}

func (s *stageBuilder) build() error {
	// Unpack file system to root
	if _, err := util.GetFSFromImage(constants.RootDir, s.image); err != nil {
		return err
	}
	// Take initial snapshot
	if err := s.snapshotter.Init(); err != nil {
		return err
	}

	// Set the initial cache key to be the base image digest, the build args and the SrcContext.
	compositeKey := NewCompositeCache(s.baseImageDigest)
	compositeKey.AddKey(s.opts.BuildArgs...)

	cmds := []commands.DockerCommand{}
	for _, cmd := range s.stage.Commands {
		command, err := commands.GetCommand(cmd, s.opts.SrcContext)
		if err != nil {
			return err
		}
		cmds = append(cmds, command)
	}

	layerCache := &cache.RegistryCache{
		Opts: s.opts,
	}
	if s.opts.Cache {
		// Possibly replace commands with their cached implementations.
		for i, command := range cmds {
			if command == nil {
				continue
			}
			ck, err := compositeKey.Hash()
			if err != nil {
				return err
			}
			img, err := layerCache.RetrieveLayer(ck)
			if err != nil {
				logrus.Infof("No cached layer found for cmd %s", command.String())
				break
			}

			if cacheCmd := command.CacheCommand(img); cacheCmd != nil {
				logrus.Infof("Using caching version of cmd: %s", command.String())
				cmds[i] = cacheCmd
			}
		}
	}

	args := dockerfile.NewBuildArgs(s.opts.BuildArgs)
	for index, command := range cmds {
		if command == nil {
			continue
		}

		// Add the next command to the cache key.
		compositeKey.AddKey(command.String())

		// If the command uses files from the context, add them.
		files, err := command.FilesUsedFromContext(&s.cf.Config, args)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := compositeKey.AddPath(f); err != nil {
				return err
			}
		}
		logrus.Info(command.String())

		if err := command.ExecuteCommand(&s.cf.Config, args); err != nil {
			return err
		}
		files = command.FilesToSnapshot()

		if !s.shouldTakeSnapshot(index, files) {
			continue
		}

		tarPath, err := s.takeSnapshot(files)
		if err != nil {
			return err
		}

		ck, err := compositeKey.Hash()
		if err != nil {
			return err
		}
		if err := s.saveSnapshotToImage(command.String(), ck, tarPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *stageBuilder) takeSnapshot(files []string) (string, error) {
	if files == nil || s.opts.SingleSnapshot {
		return s.snapshotter.TakeSnapshotFS()
	}
	// Volumes are very weird. They get created in their command, but snapshotted in the next one.
	// Add them to the list of files to snapshot.
	for v := range s.cf.Config.Volumes {
		files = append(files, v)
	}
	return s.snapshotter.TakeSnapshot(files)
}

func (s *stageBuilder) shouldTakeSnapshot(index int, files []string) bool {
	isLastCommand := index == len(s.stage.Commands)-1

	// We only snapshot the very end of intermediate stages.
	if !s.stage.Final {
		return isLastCommand
	}

	// We only snapshot the very end with single snapshot mode on.
	if s.opts.SingleSnapshot {
		return isLastCommand
	}

	// nil means snapshot everything.
	if files == nil {
		return true
	}

	// Don't snapshot an empty list.
	if len(files) == 0 {
		return false
	}
	return true
}

func (s *stageBuilder) saveSnapshotToImage(createdBy string, ck string, tarPath string) error {
	if tarPath == "" {
		return nil
	}
	fi, err := os.Stat(tarPath)
	if err != nil {
		return err
	}
	if fi.Size() <= emptyTarSize {
		logrus.Info("No files were changed, appending empty layer to config. No layer added to image.")
		return nil
	}

	layer, err := tarball.LayerFromFile(tarPath)
	if err != nil {
		return err
	}
	// Push layer to cache now along with new config file
	if s.opts.Cache {
		if err := pushLayerToCache(s.opts, ck, layer, createdBy); err != nil {
			return err
		}
	}
	s.image, err = mutate.Append(s.image,
		mutate.Addendum{
			Layer: layer,
			History: v1.History{
				Author:    constants.Author,
				CreatedBy: createdBy,
			},
		},
	)
	return err

}

// DoBuild executes building the Dockerfile
func DoBuild(opts *config.KanikoOptions) (v1.Image, error) {
	// Parse dockerfile and unpack base image to root
	stages, err := dockerfile.Stages(opts)
	if err != nil {
		return nil, err
	}
	for index, stage := range stages {
		sb, err := newStageBuilder(opts, stage)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("getting stage builder for stage %d", index))
		}
		if err := sb.build(); err != nil {
			return nil, errors.Wrap(err, "error building stage")
		}
		reviewConfig(stage, &sb.cf.Config)
		sourceImage, err := mutate.Config(sb.image, sb.cf.Config)
		if err != nil {
			return nil, err
		}
		if stage.Final {
			sourceImage, err = mutate.CreatedAt(sourceImage, v1.Time{Time: time.Now()})
			if err != nil {
				return nil, err
			}
			if opts.Reproducible {
				sourceImage, err = mutate.Canonical(sourceImage)
				if err != nil {
					return nil, err
				}
			}
			if opts.Cleanup {
				if err = util.DeleteFilesystem(); err != nil {
					return nil, err
				}
			}
			return sourceImage, nil
		}
		if stage.SaveStage {
			if err := saveStageAsTarball(index, sourceImage); err != nil {
				return nil, err
			}
			if err := extractImageToDependecyDir(index, sourceImage); err != nil {
				return nil, err
			}
		}
		// Delete the filesystem
		if err := util.DeleteFilesystem(); err != nil {
			return nil, err
		}
	}
	return nil, err
}

func extractImageToDependecyDir(index int, image v1.Image) error {
	dependencyDir := filepath.Join(constants.KanikoDir, strconv.Itoa(index))
	if err := os.MkdirAll(dependencyDir, 0755); err != nil {
		return err
	}
	logrus.Debugf("trying to extract to %s", dependencyDir)
	_, err := util.GetFSFromImage(dependencyDir, image)
	return err
}

func saveStageAsTarball(stageIndex int, image v1.Image) error {
	destRef, err := name.NewTag("temp/tag", name.WeakValidation)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(constants.KanikoIntermediateStagesDir, 0750); err != nil {
		return err
	}
	tarPath := filepath.Join(constants.KanikoIntermediateStagesDir, strconv.Itoa(stageIndex))
	logrus.Infof("Storing source image from stage %d at path %s", stageIndex, tarPath)
	return tarball.WriteToFile(tarPath, destRef, image)
}

func getHasher(snapshotMode string) (func(string) (string, error), error) {
	if snapshotMode == constants.SnapshotModeTime {
		logrus.Info("Only file modification time will be considered when snapshotting")
		return util.MtimeHasher(), nil
	}
	if snapshotMode == constants.SnapshotModeFull {
		return util.Hasher(), nil
	}
	return nil, fmt.Errorf("%s is not a valid snapshot mode", snapshotMode)
}

func resolveOnBuild(stage *config.KanikoStage, config *v1.Config) error {
	if config.OnBuild == nil {
		return nil
	}
	// Otherwise, parse into commands
	cmds, err := dockerfile.ParseCommands(config.OnBuild)
	if err != nil {
		return err
	}
	// Append to the beginning of the commands in the stage
	stage.Commands = append(cmds, stage.Commands...)
	logrus.Infof("Executing %v build triggers", len(cmds))

	// Blank out the Onbuild command list for this image
	config.OnBuild = nil
	return nil
}

// reviewConfig makes sure the value of CMD is correct after building the stage
// If ENTRYPOINT was set in this stage but CMD wasn't, then CMD should be cleared out
// See Issue #346 for more info
func reviewConfig(stage config.KanikoStage, config *v1.Config) {
	entrypoint := false
	cmd := false

	for _, c := range stage.Commands {
		if c.Name() == constants.Cmd {
			cmd = true
		}
		if c.Name() == constants.Entrypoint {
			entrypoint = true
		}
	}
	if entrypoint && !cmd {
		config.Cmd = nil
	}
}

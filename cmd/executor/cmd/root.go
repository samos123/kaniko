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

package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/options"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/genuinetools/amicontained/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opts     = &options.KanikoOptions{}
	logLevel string
	force    bool
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", constants.DefaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
	RootCmd.PersistentFlags().BoolVarP(&force, "force", "", false, "Force building outside of a container")
	addKanikoOptionsFlags(RootCmd)
	addHiddenFlags(RootCmd)
}

var RootCmd = &cobra.Command{
	Use: "executor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := util.SetLogLevel(logLevel); err != nil {
			return err
		}
		if !opts.NoPush && len(opts.Destinations) == 0 {
			return errors.New("You must provide --destination, or use --no-push")
		}
		if err := resolveSourceContext(); err != nil {
			return errors.Wrap(err, "error resolving source context")
		}
		return resolveDockerfilePath()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if !checkContained() {
			if !force {
				return errors.New("kaniko should only be run inside of a container, run with the --force flag if you are sure you want to continue")
			}
			logrus.Warn("kaniko is being run outside of a container. This can have dangerous effects on your system")
		}
		if err := os.Chdir("/"); err != nil {
			return errors.Wrap(err, "error changing to root dir")
		}
		image, err := executor.DoBuild(opts)
		if err != nil {
			return errors.Wrap(err, "error building image")
		}
		return executor.DoPush(image, opts)
	},
}

// addKanikoOptionsFlags configures opts
func addKanikoOptionsFlags(cmd *cobra.Command) {
	RootCmd.PersistentFlags().StringVarP(&opts.DockerfilePath, "dockerfile", "f", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&opts.SrcContext, "context", "c", "/workspace/", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().StringVarP(&opts.Bucket, "bucket", "b", "", "Name of the GCS bucket from which to access build context as tarball.")
	RootCmd.PersistentFlags().VarP(&opts.Destinations, "destination", "d", "Registry the final image should be pushed to. Set it repeatedly for multiple destinations.")
	RootCmd.PersistentFlags().StringVarP(&opts.SnapshotMode, "snapshotMode", "", "full", "Change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().VarP(&opts.BuildArgs, "build-arg", "", "This flag allows you to pass in ARG values at build time. Set it repeatedly for multiple values.")
	RootCmd.PersistentFlags().BoolVarP(&opts.DockerInsecureSkipTLSVerify, "insecure-skip-tls-verify", "", false, "Push to insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().StringVarP(&opts.TarPath, "tarPath", "", "", "Path to save the image in as a tarball instead of pushing")
	RootCmd.PersistentFlags().BoolVarP(&opts.SingleSnapshot, "single-snapshot", "", false, "Take a single snapshot at the end of the build.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Reproducible, "reproducible", "", false, "Strip timestamps out of the image to make it reproducible")
	RootCmd.PersistentFlags().StringVarP(&opts.Target, "target", "", "", "Set the target build stage to build")
	RootCmd.PersistentFlags().BoolVarP(&opts.NoPush, "no-push", "", false, "Do not push the image to the registry")
}

// addHiddenFlags marks certain flags as hidden from the executor help text
func addHiddenFlags(cmd *cobra.Command) {
	// This flag is added in a vendored directory, hide so that it doesn't come up via --help
	RootCmd.PersistentFlags().MarkHidden("azure-container-registry-config")
	// Hide this flag as we want to encourage people to use the --context flag instead
	RootCmd.PersistentFlags().MarkHidden("bucket")
}

func checkContained() bool {
	_, err := container.DetectRuntime()
	return err == nil
}

// resolveDockerfilePath resolves the Dockerfile path to an absolute path
func resolveDockerfilePath() error {
	if util.FilepathExists(opts.DockerfilePath) {
		abs, err := filepath.Abs(opts.DockerfilePath)
		if err != nil {
			return errors.Wrap(err, "getting absolute path for dockerfile")
		}
		opts.DockerfilePath = abs
		return nil
	}
	// Otherwise, check if the path relative to the build context exists
	if util.FilepathExists(filepath.Join(opts.SrcContext, opts.DockerfilePath)) {
		abs, err := filepath.Abs(filepath.Join(opts.SrcContext, opts.DockerfilePath))
		if err != nil {
			return errors.Wrap(err, "getting absolute path for src context/dockerfile path")
		}
		opts.DockerfilePath = abs
		return nil
	}
	return errors.New("please provide a valid path to a Dockerfile within the build context with --dockerfile")
}

// resolveSourceContext unpacks the source context if it is a tar in a bucket
// it resets srcContext to be the path to the unpacked build context within the image
func resolveSourceContext() error {
	if opts.SrcContext == "" && opts.Bucket == "" {
		return errors.New("please specify a path to the build context with the --context flag or a bucket with the --bucket flag")
	}
	if opts.SrcContext != "" && !strings.Contains(opts.SrcContext, "://") {
		return nil
	}
	if opts.Bucket != "" {
		if !strings.Contains(opts.Bucket, "://") {
			opts.SrcContext = constants.GCSBuildContextPrefix + opts.Bucket
		} else {
			opts.SrcContext = opts.Bucket
		}
	}
	// if no prefix use Google Cloud Storage as default for backwards compability
	contextExecutor, err := buildcontext.GetBuildContext(opts.SrcContext)
	if err != nil {
		return err
	}
	logrus.Debugf("Getting source context from %s", opts.SrcContext)
	opts.SrcContext, err = contextExecutor.UnpackTarFromBuildContext()
	if err != nil {
		return err
	}
	logrus.Debugf("Build context located at %s", opts.SrcContext)
	return nil
}

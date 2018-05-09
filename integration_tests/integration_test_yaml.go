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

package main

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	executorImage           = "executor-image"
	dockerImage             = "gcr.io/cloud-builders/docker"
	ubuntuImage             = "ubuntu"
	structureTestImage      = "gcr.io/gcp-runtimes/container-structure-test"
	testRepo                = "gcr.io/kaniko-test/"
	dockerPrefix            = "docker-"
	kanikoPrefix            = "kaniko-"
	daemonPrefix            = "daemon://"
	containerDiffOutputFile = "container-diff.json"
	kanikoTestBucket        = "kaniko-test-bucket"
	buildcontextPath        = "/workspace/integration_tests"
	dockerfilesPath         = "/workspace/integration_tests/dockerfiles"
	onbuildBaseImage        = testRepo + "onbuild-base:latest"
)

var fileTests = []struct {
	description         string
	dockerfilePath      string
	configPath          string
	dockerContext       string
	kanikoContext       string
	kanikoContextBucket bool
	repo                string
	snapshotMode        string
	args                []string
}{
	{
		description:    "test extract filesystem",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_extract_fs",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_extract_fs.json",
		dockerContext:  dockerfilesPath,
		kanikoContext:  dockerfilesPath,
		repo:           "extract-filesystem",
		snapshotMode:   "time",
	},
	{
		description:    "test run",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_run",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_run.json",
		dockerContext:  dockerfilesPath,
		kanikoContext:  dockerfilesPath,
		repo:           "test-run",
		args: []string{
			"file=/file",
		},
	},
	{
		description:    "test run no files changed",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_run_2",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_run_2.json",
		dockerContext:  dockerfilesPath,
		kanikoContext:  dockerfilesPath,
		repo:           "test-run-2",
		snapshotMode:   "time",
	},
	{
		description:    "test copy",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_copy",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_copy.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-copy",
		snapshotMode:   "time",
	},
	{
		description:         "test bucket build context",
		dockerfilePath:      "/workspace/integration_tests/dockerfiles/Dockerfile_test_copy",
		configPath:          "/workspace/integration_tests/dockerfiles/config_test_bucket_buildcontext.json",
		dockerContext:       buildcontextPath,
		kanikoContext:       kanikoTestBucket,
		kanikoContextBucket: true,
		repo:                "test-bucket-buildcontext",
	},
	{
		description:    "test workdir",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_workdir",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_workdir.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-workdir",
		args: []string{
			"workdir=/arg/workdir",
		},
	},
	{
		description:    "test volume",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_volume",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_volume.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-volume",
	},
	{
		description:    "test add",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_add",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_add.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-add",
		args: []string{
			"file=context/foo",
		},
	},
	{
		description:    "test mv add",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_mv_add",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_mv_add.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-mv-add",
	},
	{
		description:    "test registry",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_registry",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_registry.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-registry",
	},
	{
		description:    "test onbuild",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_onbuild",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_onbuild.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-onbuild",
		args: []string{
			"file=/tmp/onbuild",
		},
	},
	{
		description:    "test scratch",
		dockerfilePath: "/workspace/integration_tests/dockerfiles/Dockerfile_test_scratch",
		configPath:     "/workspace/integration_tests/dockerfiles/config_test_scratch.json",
		dockerContext:  buildcontextPath,
		kanikoContext:  buildcontextPath,
		repo:           "test-scratch",
		args: []string{
			"hello=hello-value",
			"file=context/foo",
			"file3=context/b*",
		},
	},
}

var structureTests = []struct {
	description           string
	dockerfilePath        string
	structureTestYamlPath string
	dockerBuildContext    string
	kanikoContext         string
	repo                  string
}{
	{
		description:           "test env",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_env",
		repo:                  "test-env",
		dockerBuildContext:    dockerfilesPath,
		kanikoContext:         dockerfilesPath,
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_env.yaml",
	},
	{
		description:           "test metadata",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_metadata",
		repo:                  "test-metadata",
		dockerBuildContext:    dockerfilesPath,
		kanikoContext:         dockerfilesPath,
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_metadata.yaml",
	},
	{
		description:           "test user command",
		dockerfilePath:        "/workspace/integration_tests/dockerfiles/Dockerfile_test_user_run",
		repo:                  "test-user",
		dockerBuildContext:    dockerfilesPath,
		kanikoContext:         dockerfilesPath,
		structureTestYamlPath: "/workspace/integration_tests/dockerfiles/test_user.yaml",
	},
}

type step struct {
	Name string
	Args []string
	Env  []string
}

type testyaml struct {
	Steps   []step
	Timeout string
}

func main() {

	// First, copy container-diff in
	containerDiffStep := step{
		Name: "gcr.io/cloud-builders/gsutil",
		Args: []string{"cp", "gs://container-diff/latest/container-diff-linux-amd64", "."},
	}
	containerDiffPermissions := step{
		Name: ubuntuImage,
		Args: []string{"chmod", "+x", "container-diff-linux-amd64"},
	}
	GCSBucketTarBuildContext := step{
		Name: ubuntuImage,
		Args: []string{"tar", "-C", "/workspace/integration_tests/", "-zcvf", "/workspace/context.tar.gz", "."},
	}
	uploadTarBuildContext := step{
		Name: "gcr.io/cloud-builders/gsutil",
		Args: []string{"cp", "/workspace/context.tar.gz", "gs://kaniko-test-bucket/"},
	}

	// Build executor image
	buildExecutorImage := step{
		Name: dockerImage,
		Args: []string{"build", "-t", executorImage, "-f", "deploy/Dockerfile", "."},
	}

	// Build and push onbuild base images
	buildOnbuildImage := step{
		Name: dockerImage,
		Args: []string{"build", "-t", onbuildBaseImage, "-f", "/workspace/integration_tests/dockerfiles/Dockerfile_onbuild_base", "."},
	}
	pushOnbuildBase := step{
		Name: dockerImage,
		Args: []string{"push", onbuildBaseImage},
	}
	y := testyaml{
		Steps: []step{containerDiffStep, containerDiffPermissions, GCSBucketTarBuildContext,
			uploadTarBuildContext, buildExecutorImage, buildOnbuildImage, pushOnbuildBase},
		Timeout: "1200s",
	}
	for _, test := range fileTests {
		// First, build the image with docker
		dockerImageTag := testRepo + dockerPrefix + test.repo
		var buildArgs []string
		buildArgFlag := "--build-arg"
		for _, arg := range test.args {
			buildArgs = append(buildArgs, buildArgFlag)
			buildArgs = append(buildArgs, arg)
		}
		dockerBuild := step{
			Name: dockerImage,
			Args: append([]string{"build", "-t", dockerImageTag, "-f", test.dockerfilePath, test.dockerContext}, buildArgs...),
		}
		// Then, buld the image with kaniko
		kanikoImage := testRepo + kanikoPrefix + test.repo
		snapshotMode := ""
		if test.snapshotMode != "" {
			snapshotMode = "--snapshotMode=" + test.snapshotMode
		}
		contextFlag := "--context"
		if test.kanikoContextBucket {
			contextFlag = "--bucket"
		}
		kaniko := step{
			Name: executorImage,
			Args: append([]string{"--destination", kanikoImage, "--dockerfile", test.dockerfilePath, contextFlag, test.kanikoContext, snapshotMode}, buildArgs...),
		}

		// Pull the kaniko image
		pullKanikoImage := step{
			Name: dockerImage,
			Args: []string{"pull", kanikoImage},
		}

		daemonDockerImage := daemonPrefix + dockerImageTag
		daemonKanikoImage := daemonPrefix + kanikoImage
		// Run container diff on the images
		args := "container-diff-linux-amd64 diff " + daemonDockerImage + " " + daemonKanikoImage + " --type=file -j >" + containerDiffOutputFile
		containerDiff := step{
			Name: ubuntuImage,
			Args: []string{"sh", "-c", args},
			Env:  []string{"PATH=/workspace:/bin"},
		}

		catContainerDiffOutput := step{
			Name: ubuntuImage,
			Args: []string{"cat", containerDiffOutputFile},
		}
		compareOutputs := step{
			Name: ubuntuImage,
			Args: []string{"cmp", "-b", test.configPath, containerDiffOutputFile},
		}

		y.Steps = append(y.Steps, dockerBuild, kaniko, pullKanikoImage, containerDiff, catContainerDiffOutput, compareOutputs)
	}

	for _, test := range structureTests {

		// First, build the image with docker
		dockerImageTag := testRepo + dockerPrefix + test.repo
		dockerBuild := step{
			Name: dockerImage,
			Args: []string{"build", "-t", dockerImageTag, "-f", test.dockerfilePath, test.dockerBuildContext},
		}

		// Build the image with kaniko
		kanikoImage := testRepo + kanikoPrefix + test.repo
		kaniko := step{
			Name: executorImage,
			Args: []string{"--destination", kanikoImage, "--dockerfile", test.dockerfilePath, "--context", test.kanikoContext},
		}
		// Pull the kaniko image
		pullKanikoImage := step{
			Name: dockerImage,
			Args: []string{"pull", kanikoImage},
		}
		// Run structure tests on the kaniko and docker image
		kanikoStructureTest := step{
			Name: structureTestImage,
			Args: []string{"test", "--image", kanikoImage, "--config", test.structureTestYamlPath},
		}
		dockerStructureTest := step{
			Name: structureTestImage,
			Args: []string{"test", "--image", dockerImageTag, "--config", test.structureTestYamlPath},
		}
		y.Steps = append(y.Steps, dockerBuild, kaniko, pullKanikoImage, kanikoStructureTest, dockerStructureTest)
	}

	d, _ := yaml.Marshal(&y)
	fmt.Println(string(d))
}

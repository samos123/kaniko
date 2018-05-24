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
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
	"github.com/docker/docker/builder/dockerfile/instructions"
	"github.com/google/go-containerregistry/v1"
)

var userTests = []struct {
	user        string
	expectedUid string
	shouldError bool
}{
	{
		user:        "root",
		expectedUid: "root",
		shouldError: false,
	},
	{
		user:        "0",
		expectedUid: "0",
		shouldError: false,
	},
	{
		user:        "fakeUser",
		expectedUid: "",
		shouldError: true,
	},
	{
		user:        "root:root",
		expectedUid: "root:root",
		shouldError: false,
	},
	{
		user:        "0:root",
		expectedUid: "0:root",
		shouldError: false,
	},
	{
		user:        "root:0",
		expectedUid: "root:0",
		shouldError: false,
	},
	{
		user:        "0:0",
		expectedUid: "0:0",
		shouldError: false,
	},
	{
		user:        "root:fakeGroup",
		expectedUid: "",
		shouldError: true,
	},
	{
		user:        "$envuser",
		expectedUid: "root",
		shouldError: false,
	},
	{
		user:        "root:$envgroup",
		expectedUid: "root:root",
		shouldError: false,
	},
}

func TestUpdateUser(t *testing.T) {
	for _, test := range userTests {
		cfg := &v1.Config{
			Env: []string{
				"envuser=root",
				"envgroup=root",
			},
		}
		cmd := UserCommand{
			&instructions.UserCommand{
				User: test.user,
			},
		}
		buildArgs := dockerfile.NewBuildArgs([]string{})
		err := cmd.ExecuteCommand(cfg, buildArgs)
		testutil.CheckErrorAndDeepEqual(t, test.shouldError, err, test.expectedUid, cfg.User)
	}
}

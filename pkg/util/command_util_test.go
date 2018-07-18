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

package util

import (
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kaniko/testutil"
)

var testUrl = "https://github.com/GoogleContainerTools/runtimes-common/blob/master/LICENSE"

var testEnvReplacement = []struct {
	path         string
	command      string
	envs         []string
	isFilepath   bool
	expectedPath string
}{
	{
		path:    "/simple/path",
		command: "WORKDIR /simple/path",
		envs: []string{
			"simple=/path/",
		},
		isFilepath:   true,
		expectedPath: "/simple/path",
	},
	{
		path:    "/simple/path/",
		command: "WORKDIR /simple/path/",
		envs: []string{
			"simple=/path/",
		},
		isFilepath:   true,
		expectedPath: "/simple/path/",
	},
	{
		path:    "${a}/b",
		command: "WORKDIR ${a}/b",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b",
	},
	{
		path:    "/$a/b",
		command: "COPY ${a}/b /c/",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b",
	},
	{
		path:    "/$a/b/",
		command: "COPY /${a}/b /c/",
		envs: []string{
			"a=/path/",
			"b=/path2/",
		},
		isFilepath:   true,
		expectedPath: "/path/b/",
	},
	{
		path:    "\\$foo",
		command: "COPY \\$foo /quux",
		envs: []string{
			"foo=/path/",
		},
		isFilepath:   true,
		expectedPath: "$foo",
	},
	{
		path:    "8080/$protocol",
		command: "EXPOSE 8080/$protocol",
		envs: []string{
			"protocol=udp",
		},
		expectedPath: "8080/udp",
	},
}

func Test_EnvReplacement(t *testing.T) {
	for _, test := range testEnvReplacement {
		actualPath, err := ResolveEnvironmentReplacement(test.path, test.envs, test.isFilepath)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedPath, actualPath)

	}
}

var buildContextPath = "../../integration/"

var destinationFilepathTests = []struct {
	src              string
	dest             string
	cwd              string
	expectedFilepath string
}{
	{
		src:              "context/foo",
		dest:             "/foo",
		cwd:              "/",
		expectedFilepath: "/foo",
	},
	{
		src:              "context/foo",
		dest:             "/foodir/",
		cwd:              "/",
		expectedFilepath: "/foodir/foo",
	},
	{
		src:              "context/foo",
		cwd:              "/",
		dest:             "foo",
		expectedFilepath: "/foo",
	},
	{
		src:              "context/bar/",
		cwd:              "/",
		dest:             "pkg/",
		expectedFilepath: "/pkg/bar",
	},
	{
		src:              "context/bar/",
		cwd:              "/newdir",
		dest:             "pkg/",
		expectedFilepath: "/newdir/pkg/bar",
	},
	{
		src:              "./context/empty",
		cwd:              "/",
		dest:             "/empty",
		expectedFilepath: "/empty",
	},
	{
		src:              "./context/empty",
		cwd:              "/dir",
		dest:             "/empty",
		expectedFilepath: "/empty",
	},
	{
		src:              "./",
		cwd:              "/",
		dest:             "/dir",
		expectedFilepath: "/dir",
	},
	{
		src:              "context/foo",
		cwd:              "/test",
		dest:             ".",
		expectedFilepath: "/test/foo",
	},
}

func Test_DestinationFilepath(t *testing.T) {
	for _, test := range destinationFilepathTests {
		actualFilepath, err := DestinationFilepath(test.src, test.dest, test.cwd)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFilepath, actualFilepath)
	}
}

var urlDestFilepathTests = []struct {
	url          string
	cwd          string
	dest         string
	expectedDest string
}{
	{
		url:          "https://something/something",
		cwd:          "/test",
		dest:         ".",
		expectedDest: "/test/something",
	},
	{
		url:          "https://something/something",
		cwd:          "/cwd",
		dest:         "/test",
		expectedDest: "/test",
	},
	{
		url:          "https://something/something",
		cwd:          "/test",
		dest:         "/dest/",
		expectedDest: "/dest/something",
	},
}

func Test_UrlDestFilepath(t *testing.T) {
	for _, test := range urlDestFilepathTests {
		actualDest := URLDestinationFilepath(test.url, test.dest, test.cwd)
		testutil.CheckErrorAndDeepEqual(t, false, nil, test.expectedDest, actualDest)
	}
}

var matchSourcesTests = []struct {
	srcs          []string
	files         []string
	expectedFiles []string
}{
	{
		srcs: []string{
			"pkg/*",
			testUrl,
		},
		files: []string{
			"pkg/a",
			"pkg/b",
			"/pkg/d",
			"pkg/b/d/",
			"dir/",
		},
		expectedFiles: []string{
			"pkg/a",
			"pkg/b",
			testUrl,
		},
	},
}

func Test_MatchSources(t *testing.T) {
	for _, test := range matchSourcesTests {
		actualFiles, err := matchSources(test.srcs, test.files)
		sort.Strings(actualFiles)
		sort.Strings(test.expectedFiles)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedFiles, actualFiles)
	}
}

var isSrcValidTests = []struct {
	srcsAndDest     []string
	resolvedSources []string
	shouldErr       bool
}{
	{
		srcsAndDest: []string{
			"context/foo",
			"context/bar",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: true,
	},
	{
		srcsAndDest: []string{
			"context/foo",
			"context/bar",
			"dest/",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: false,
	},
	{
		srcsAndDest: []string{
			"context/bar/bam",
			"dest",
		},
		resolvedSources: []string{
			"context/bar/bam",
		},
		shouldErr: false,
	},
	{
		srcsAndDest: []string{
			"context/foo",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
		},
		shouldErr: false,
	},
	{
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			"dest/",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: false,
	},
	{
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
			"context/bar",
		},
		shouldErr: true,
	},
	{
		srcsAndDest: []string{
			"context/foo",
			"context/doesntexist*",
			"dest",
		},
		resolvedSources: []string{
			"context/foo",
		},
		shouldErr: false,
	},
	{
		srcsAndDest: []string{
			"context/",
			"dest",
		},
		resolvedSources: []string{
			"context/",
		},
		shouldErr: false,
	},
}

func Test_IsSrcsValid(t *testing.T) {
	for _, test := range isSrcValidTests {
		err := IsSrcsValid(test.srcsAndDest, test.resolvedSources, buildContextPath)
		testutil.CheckError(t, test.shouldErr, err)
	}
}

var testResolveSources = []struct {
	srcsAndDest  []string
	expectedList []string
}{
	{
		srcsAndDest: []string{
			"context/foo",
			"context/b*",
			testUrl,
			"dest/",
		},
		expectedList: []string{
			"context/foo",
			"context/bar",
			testUrl,
		},
	},
}

func Test_ResolveSources(t *testing.T) {
	for _, test := range testResolveSources {
		actualList, err := ResolveSources(test.srcsAndDest, buildContextPath)
		testutil.CheckErrorAndDeepEqual(t, false, err, test.expectedList, actualList)
	}
}

var testRemoteUrls = []struct {
	name  string
	url   string
	valid bool
}{
	{
		name:  "Valid URL",
		url:   testUrl,
		valid: true,
	},
	{
		name:  "Invalid URL",
		url:   "not/real/",
		valid: false,
	},
	{
		name:  "URL which fails on GET",
		url:   "https://thereisnowaythiswilleverbearealurlrightrightrightcatsarethebest.com/something/not/real",
		valid: false,
	},
}

func Test_RemoteUrls(t *testing.T) {
	for _, test := range testRemoteUrls {
		t.Run(test.name, func(t *testing.T) {
			valid := IsSrcRemoteFileURL(test.url)
			testutil.CheckErrorAndDeepEqual(t, false, nil, test.valid, valid)
		})
	}

}

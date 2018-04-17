# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash
set -ex

if [ -f "$KOKORO_GFILE_DIR"/common.sh ]; then
    echo "Installing dependencies..."
    source "$KOKORO_GFILE_DIR/common.sh"
    mkdir -p /usr/local/go/src/github.com/GoogleContainerTools/
    cp -r github/kaniko /usr/local/go/src/github.com/GoogleContainerTools/
    pushd /usr/local/go/src/github.com/GoogleContainerTools/kaniko
fi

echo "Running integration tests..."
make out/executor
go run integration_tests/integration_test_yaml.go | gcloud container builds submit --config /dev/fd/0 .

#!/bin/bash -eu
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Based on approach at https://github.com/hugojosefson/find-node-or-install.

function die() {
  (>&2 echo $1)
  exit 1
}

cd $(dirname $0)

readonly NODE_VERSION='lts/*'
export NVM_DIR=${PWD}/nvm

if [[ ! -d "${NVM_DIR}" ]]; then
  git clone git://github.com/creationix/nvm.git ${NVM_DIR} &> /dev/null || die "failed to install nvm"
fi

. ${NVM_DIR}/nvm.sh &> /dev/null || die "failed to source nvm.sh"
nvm install ${NODE_VERSION} &> /dev/null || die "failed to install node"
nvm use ${NODE_VERSION} &> /dev/null || die "failed to activate node"

dirname $(nvm which ${NODE_VERSION})

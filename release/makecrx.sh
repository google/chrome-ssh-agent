#!/bin/bash -eu
#
# Copyright 2017 Google LLC
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

# Purpose: Pack a Chromium extension directory into crx format
#
# Based on example at https://developer.chrome.com/extensions/crx#scripts.

readonly KEY=$1; shift

# Start in top-level source directory
cd "$(dirname $0)/.."

readonly BUILD_TMP=$(mktemp -d)
trap "rm -rf '${BUILD_TMP}'" EXIT

readonly OUTPUT_DIR="${PWD}/bin"

readonly NAME=$(basename "${PWD}")
readonly CRX="${OUTPUT_DIR}/${NAME}.crx"
readonly PUB="${BUILD_TMP}/${NAME}.pub"
readonly SIG="${BUILD_TMP}/${NAME}.sig"
readonly ZIP="${BUILD_TMP}/${NAME}.zip"

# Zip all output files.
zip -qr -9 -X "${ZIP}" . --include \
	manifest.json \
	\*.html \
	\*.js \
	\*CONTRIBUTING* \
	\*README* \
	\*LICENCE*

# Sign the contents of the zip file.
openssl sha1 -sha1 -binary -sign "${KEY}" < "${ZIP}" > "${SIG}"

# Create public key.
openssl rsa -pubout -outform DER < "${KEY}" > "${PUB}" 2>/dev/null

byte_swap () {
  # Take "abcdefgh" and return it as "ghefcdab"
  echo "${1:6:2}${1:4:2}${1:2:2}${1:0:2}"
}

readonly CRMAGIC_HEX="4372 3234" # Cr24
readonly VERSION_HEX="0200 0000" # 2
readonly PUB_LEN_HEX=$(byte_swap $(printf '%08x\n' $(ls -l "${PUB}" | awk '{print $5}')))
readonly SIG_LEN_HEX=$(byte_swap $(printf '%08x\n' $(ls -l "${SIG}" | awk '{print $5}')))
(
  echo "${CRMAGIC_HEX} ${VERSION_HEX} ${PUB_LEN_HEX} ${SIG_LEN_HEX}" | xxd -r -p
  cat "${PUB}" "${SIG}" "${ZIP}"
) > "${CRX}"


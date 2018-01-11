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

# Usage:
#   (1) Build a new zip file (e.g., with 'make zip')
#   (2) Run deploy.sh
#
# This script requires CLIENT_ID, CLIENT_SECRET, and WEBSTORE_REFRESH_TOKEN to
# be defined as environment variables.  See
# https://developer.chrome.com/webstore/using_webstore_api for details.

readonly EXTENSION_ID=eechpbnaifiimgajnomdipfaamobdfha
readonly FILE_NAME=bin/chrome-ssh-agent.zip

readonly TOKEN_REQUEST="\
client_id=${WEBSTORE_CLIENT_ID}\
&client_secret=${WEBSTORE_CLIENT_SECRET}\
&refresh_token=${WEBSTORE_REFRESH_TOKEN}\
&grant_type=refresh_token\
&redirect_uri=urn:ietf:wg:oauth:2.0:oob"
readonly TOKEN_RESPONSE=$(curl "https://accounts.google.com/o/oauth2/token" -d "${TOKEN_REQUEST}")

readonly TOKEN=$(echo "${TOKEN_RESPONSE}" | python -c "import sys, json; print json.load(sys.stdin)['access_token']")

curl \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-goog-api-version: 2" \
  -X PUT \
  -T ${FILE_NAME} \
  -v \
  https://www.googleapis.com/upload/chromewebstore/v1.1/items/${EXTENSION_ID}

curl \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-goog-api-version: 2" \
  -H "Content-Length: 0" \
  -H "publishTarget: trustedTesters" \
  -X POST \
  -v \
  https://www.googleapis.com/chromewebstore/v1.1/items/${EXTENSION_ID}/publish

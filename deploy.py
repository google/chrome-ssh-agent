#!/usr/bin/env python3
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

import os
import os.path
import requests
import sys


def defaultHeaders(token):
  return {
    'Authorization': 'Bearer %s' % token,
    'x-goog-api-version': '2',
  }


def getToken():
  r = requests.post('https://accounts.google.com/o/oauth2/token', data={
      'client_id': os.environ['WEBSTORE_CLIENT_ID'],
      'client_secret': os.environ['WEBSTORE_CLIENT_SECRET'],
      'refresh_token': os.environ['WEBSTORE_REFRESH_TOKEN'],
      'grant_type': 'refresh_token',
      'redirect_uri': 'urn:ietf:wg:oauth:2.0:oob',
    })
  token = r.json().get('access_token')
  if not token:
    raise RuntimeError('Failed to get access token: %s' % r.text)
  return token


def upload(token, extension_id, file_path):
  url = 'https://www.googleapis.com/upload/chromewebstore/v1.1/items/%(id)s' % {
      'id': extension_id,
    }
  headers = defaultHeaders(token)
  with open(file_path, 'rb') as f:
    r = requests.put(url, headers=headers, data=f)
    r.raise_for_status()
    if r.json().get('uploadState') != 'SUCCESS':
      raise RuntimeError('Upload failed: %s' % r.text)


def publish(token, extension_id, publish_target):
  url = 'https://www.googleapis.com/chromewebstore/v1.1/items/%(id)s/publish' % {
      'id': extension_id,
    }
  params = {'uploadType': 'media'}
  headers = defaultHeaders(token)
  headers['publishTarget'] = publish_target
  r = requests.post(url, params=params, headers=headers)
  r.raise_for_status()
  statuses = set(r.json().get('status', []))
  errors = statuses - frozenset(['OK'])
  if errors:
    raise RuntimeError('Publish failed: %s' % r.text)
  

def main():
  extension_id = os.environ['EXTENSION_ID']
  file_path = os.environ['FILE_NAME']
  publish_target = os.environ['PUBLISH_TARGET']

  token = getToken()
  upload(token, extension_id, file_path)
  publish(token, extension_id, publish_target)


if __name__ == "__main__":
  main()

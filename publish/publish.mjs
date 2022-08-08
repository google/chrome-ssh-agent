// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import fs from 'fs';
import chromeWebstoreUpload from 'chrome-webstore-upload';
import {HTTPError} from 'got';  // chrome-webstore-upload dependency; included automatically.
import {runfiles} from '@bazel/runfiles';

const extensionId = process.env.EXTENSION_ID;
const clientId = process.env.WEBSTORE_CLIENT_ID;
const clientSecret = process.env.WEBSTORE_CLIENT_SECRET;
const refreshToken = process.env.WEBSTORE_REFRESH_TOKEN;
const publishTarget = process.env.PUBLISH_TARGET || 'default';

const extensionPath = runfiles.resolveWorkspaceRelative('chrome-ssh-agent.zip');
const extensionFile = fs.createReadStream(extensionPath);

console.log('Creating client');
const store = chromeWebstoreUpload({
  extensionId: extensionId,
  clientId: clientId,
  clientSecret: clientSecret,
  refreshToken: refreshToken,
});

try {
  console.log('Uploading extension');
  const uploadResult = await store.uploadExisting(extensionFile);
  if (uploadResult.uploadState !== 'SUCCESS') {
    throw `Upload failed with errors:\n${uploadResult.itemError.join('\n')}`;
  }

  console.log('Publishing extension');
  const publishResult = await store.publish(publishTarget);
  const publishErrors = new Set(publishResult.status);
  publish.delete('OK');
  if (publishErrors.size > 0) {
    throw `Publish failed with errors:\n${publishResult.statusDetail.join('\n')}`;
  }
} catch (e) {
  if (e instanceof HTTPError) {
    throw `HTTP Error:\n  Details: ${e}\n  Response Body: ${e.response.body}`;
  } else {
    throw e;
  }
}

console.log('Finished');

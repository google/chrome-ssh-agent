[![Build status](https://api.travis-ci.org/google/chrome-ssh-agent.svg?branch=master)](https://travis-ci.org/google/chrome-ssh-agent)

# SSH Agent for Google Chrome™

This is a bare-bones SSH agent extension for Google Chrome™.  It provides an
SSH agent implementation that can be used with the
[Secure Shell Chrome extension](http://chrome.google.com/webstore/detail/secure-shell/pnhechapfaindjhompbnflcldabbghjo).

# Getting Started

## Installation

Install the extension from the 
[Chrome Web Store](https://chrome.google.com/webstore/detail/chrome-ssh-agent/eechpbnaifiimgajnomdipfaamobdfha).

## Adding and Using Keys

1. Click on the SSH Agent extension's icon in to Chrome toolbar.
   ![List keys](https://github.com/google/chrome-ssh-agent/raw/master/img/screenshot-list.png)
2. Configure a new private key by clicking the 'Add' button.  Give it a name
   and enter the PEM-encoded private key.
   ![Add key](https://github.com/google/chrome-ssh-agent/raw/master/img/screenshot-add.png)
   If you use Chrome Sync, configured keys will be synced to your account so
   they are available across your devices.
3. Load the key into the SSH agent by clicking the 'Load' button and providing
   the key's passphrase.
   ![Enter passphrase](https://github.com/google/chrome-ssh-agent/raw/master/img/screenshot-passphrase.png)
4. When creating a new connection in the Secure Shell extension, add
   `--ssh-agent=eechpbnaifiimgajnomdipfaamobdfha` to "SSH Relay Server
   Options" field to indicate that it should use the SSH Agent for keys.
   ![Connect](https://github.com/google/chrome-ssh-agent/raw/master/img/screenshot-connect.png)

# Current Limitations

## Unencrypted Keys Are Not Supported

The extension currently only supports encrypted private keys.

# Credits

Portions of the code and approach are heavily based on the
[MacGyver](http://github.com/stripe/macgyver) Chrome extension. In
particular, the following:

* Usage of GopherJS, which makes it easy to use Go's existing
  [SSH Agent implementation](http://godoc.org/golang.org/x/crypto/ssh/agent).
* Code translating between the SSH Agent protocol used by the secure Shell
  extension and the actual SSH agent protocol
  ([details](http://github.com/stripe/macgyver#chrome-ssh-agent-protocol)).

# Disclaimer

This is not an officially supported Google product.

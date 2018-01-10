![Build status](https://api.travis-ci.org/google/chrome-ssh-agent.svg?branch=master)

# chrome-ssh-agent

This is a bare-bones SSH agent extension for Google Chrome.  It provides an
SSH agent implementation that can be used with the
[Secure Shell Chrome extension](http://chrome.google.com/webstore/detail/secure-shell/pnhechapfaindjhompbnflcldabbghjo).

# Getting Started

## Installation from Chrome Web Store

Visit the extension in the
[Chrome Web Store](https://chrome.google.com/webstore/detail/chrome-ssh-agent/eechpbnaifiimgajnomdipfaamobdfha)
and install it.

## Managing Keys in the SSH Agent

The extension allows you to configure keys which are then stored in Chrome and
synced across your devices.

Once configured, key can be loaded into the SSH agent by providing the key's
passphrase. Loaded keys are available for use by the Secure Shell extension.

## Using Keys in the Secure Shell Extension

The Secure Shell extension must be instructed to use the SSH Agent. This is
done by adding `--ssh-agent=eechpbnaifiimgajnomdipfaamobdfha` to the
"SSH Relay Server Options" in the properties for a SSH connection.

# Current Limitations

## Unencrypted Keys Are Not Supported

The extension currently only supports encrypted private keys.

# Credits

Portions of the code and approach are heavily based on the
[MacGyver](http://github.com/stripe/macgyver) Chrome extension. In
particular, the following:

* Usage of GopherJS, which makes it easy to use Go's existing
  [SSH Agent implementation](http://godoc.org/golang.org/x/crypto/ssh/agent).
* Code translating between the Chrome SSH Agent protocol and the actual SSH
  agent protocol ([details](http://github.com/stripe/macgyver#chrome-ssh-agent-protocol)).

# Disclaimer

This is not an officially supported Google product.

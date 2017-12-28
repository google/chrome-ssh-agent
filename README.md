# chrome-ssh-agent

This is a bare-bones SSH agent extension for Google Chrome.  It provides an
SSH agent implementation that can be used with the
[Secure Shell Chrome extension](http://chrome.google.com/webstore/detail/secure-shell/pnhechapfaindjhompbnflcldabbghjo).

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

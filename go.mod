module github.com/google/chrome-ssh-agent

go 1.21.1

toolchain go1.21.5

require (
	github.com/google/go-cmp v0.6.0
	// https://github.com/tebeka/selenium/commit/e617f9870cec59a6f6e234017e45d36ef0444a04 required to support CRX3 format
	github.com/tebeka/selenium v0.9.10-0.20211105214847-e9100b7f5ac1
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a
	golang.org/x/crypto v0.16.0
)

require (
	github.com/bazelbuild/rules_go v0.44.0
	github.com/norunners/vert v0.0.0-20221203075838-106a353d42dd
	golang.org/x/tools v0.16.1
)

require (
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/mediabuyerbot/go-crx3 v1.3.1 // indirect
	golang.org/x/sys v0.15.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

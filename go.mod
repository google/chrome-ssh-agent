module github.com/google/chrome-ssh-agent

go 1.18

require (
	github.com/google/go-cmp v0.5.6
	// https://github.com/tebeka/selenium/commit/e617f9870cec59a6f6e234017e45d36ef0444a04 required to support CRX3 format
	github.com/tebeka/selenium v0.9.10-0.20211105214847-e9100b7f5ac1
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
)

require (
	github.com/bazelbuild/rules_go v0.34.0
	github.com/norunners/vert v0.0.0-20211229045251-b4c39e2856da
	golang.org/x/tools v0.0.0-20190624190245-7f2218787638
)

require (
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/mediabuyerbot/go-crx3 v1.3.1 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

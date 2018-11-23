// Copyright 2018 Google LLC
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

package testdata

type TestKey struct {
	Private    string
	Passphrase string
	Blob       string
	Type       string
}

var (
	WithPassphrase = TestKey{
		Private: `
-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,3F17234B07052C56268A529F4C96A478

XeoG0rjwABbZUIKitYRmzYCdLd91HN9zu4d32MkKe+qFNueGp/vJyuFSLYthmw24
RAXxAJ3JE01QGmbeXDrhJR1uuUG/2GtsT2L3jhntxfgMepUOfXuO9YxcO9dd7Qan
P0Zr9Ry59i4IadHYs6ZXpD06nWHHSM/JPxsosfwdrDJTDebo1eVwHVpoizRsL5Wp
2V1a5fBQstGXA55X1fjKlDutrtPsin1picqF5/24A6k1WoeMWhX4UpCkDxj0o+Y6
KMfCipkm/vqKlLHYTLiRruYp7q6zyTKb3Mf+6cVRBwO7DVv1D95nOHkTjymnIsrX
VgdG5z4OcrRjzw9MM4qwTiNv3Ba42veextWWw6spyiE0uYPVbmPFjzLwSCiAXDqY
ckZ9MtD3RoyPjFBz/D4YbtDR7miXR1dzQz55ibk9aZp3PY0JIk3Tc7vPN7hLhmtd
2rgIy4jj8D6SKxWxcMX5suceOcUG7DN1LVPl0K1LNrBe3a9uUsL10W6BsyrqHZF2
X1KiOYI5x2tdlxqSsGUQkmTEIOMPwnb5u/w3d2TvD6p1sgx2z6cRYm1sg/KX1eBy
wz9zQQXZzvo09kh06XSJECWJ+f7nxHj8vh0LREZpHjU16fq2WE0EqMITcHxBDaaE
Aql4BahWuSOEDHGOmMnjMmlBQRSfdDXkHB9WbxD+e4I0guGyQesP347fCIjqhFBJ
RyVjAuvqXNVyqjTBhCxqV5aHRuJFIF7e+drdx0Wn3NWSIelSBxe3zJsARd46xeNL
BqSjKS2Mdetxu2jHLV9RKv0shgeeMzNTl2vXnB6LLeK9grMEHJZfAa8soLtMNb+d
S2M7O9XCX97iwRa0TdxTuLtETOL8JgUnD8wVlJSG7RmqOXLcpA3rbz45kYHDApLR
TtLlRrfTBRLYse12j9i8LxkgefME/nqX5YY8ZUGXCIoifFz/fimj9aIwzIHyvvJG
KlwRQMIg83qTHVqzzgLzfIhmUhFYaB6WaQjW08/9bQUMVh5FnrqHeKAONQMlYbq4
LF0GsvDjBNbN3sbPPDtYP7skj8zhTcfEKznsQkIlFI98fY7tYIsWDG/B5KXEaJ7h
W56PKWnYZ4SWoq2k/FFvnFQT5683bpu5Vl0QJDxUYS8h+6KqGIghmYgR9EGsTbsf
UsJLYJ8Zsh1ZUyOxBvYpqua3tfn3PDIg+UAYf9NI3LkroW7xmkW1CONahGvf1KFn
xvWhPgmazIpHuzb09dlxO6q6OF6yVDPsatJXNnSFTkRKwrRbQR/hVg+oGwjFtWFS
dW5FfsFY8ftVQIozCnxUEw3TwDztRkbVXxQ1O4tOjbzhkkFA3H/NukrbLLCEMjDI
6YoJpONNVsjTVE3Pn+bOc9vuQEpMhITMfvCiI2vqsXMDSYkQNOa/nrvpDGTIEm/w
o038XwjZI5MEPLZlhh69e6jbL3kSsVARD5ahkarLuvUyKpqiInrnUWLHZETwiSH2
wy+03hJeDFxANxHpz25t87FwzHR4FceteqJXHWoR6XiH805u+2KHCHhv+6nvQCe2
SQv684pfXIhZ8Lfr13deSx5G8h+ULUDyfHzgheSXWOPyve+wdehAOyh71npHjyXe
-----END RSA PRIVATE KEY-----`,
		Passphrase: "secret",
		Blob:       "AAAAB3NzaC1yc2EAAAADAQABAAABAQC8c/qTG/jF0SFloU74KvKEYxYlPpxplKXfd4NXtIx578iuKzbX1HQSgEpr2aWUXoPQNMqNpkhNFaDU3nVLtD74vEn2Yn3QuzRUgMeOybqImN5v2TvAmpUt2YOHO3FraDQaYSGBS5FXp2eulvgZ2KnQyMFBo+R1m2VIfuq2rQZPEgyaq/DYbLLKpmgH2Ud8csVo+2RcnzBx2ZpOppFQ+EjgHljwYPpHf93LNX4Q/auU6+RA8Z0JpH/hw0US4d5eNvdifHTuvSAj3bIjTeyfQGZnfHzrwfk2FvtsBFS/bLwEUlD/htZCcW6zaDxEYAsXKPizW5dNDt77C9QIWy+kZy7X",
		Type:       "ssh-rsa",
	}

	WithoutPassphrase = TestKey{
		Private: `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqvs4Aur/N3tFjDvuAqcLQ4BJVHpoqzO/RbwbXSBA5bCmd7rO
4cy2inJK0oGphOTn6KRxpRJM8Wwl67iZrRYMTgHC357ymzOurMRXN1L1IZNRn4QO
xBoamuX98pMRpPMyBs6dEQoVJAsLaG7ZTdWGXumvGVkGDdit+6waGBqD7XpVl2Q2
iNZHTuhWvSPPoTyhjiv7Nll7zQVpwmwvu/7qdlGES2SH1P4HmZ3E2Qe2KZ1Yj4RE
sig4WoUPLg+xv+qA0gUNZEHxiAKy6Rs787msGS3biUXYhmKUG8aRHxjmxZcqhl62
UnFCzeDQtpovZtlTqtoDZQsZOb3z/1TlxJA8CQIDAQABAoIBAF5FYN6K/uhyOShW
qqYfv+AZzVScoTUztNQYIOY5sE50FXSSNRreKg8vcP2brAGvzAXDFT20V2QNAuNy
xphePa6M3gs5sf3MgxSStJu2S52Vgj13LEUHN4AMKvYiDGpsBDsolAUfEATtaf7M
j1eQ0SNnqLlLEkF0JIlMnJ6JkA/QqrGKbDsIbq0/R2YmF10gS/PAnRnp18vIJ7TX
N201vBGmI9DsTzqTgJ4dnCY8FNxi0Y6dBpxuPpSRE6v5W221Xf56ce60Zi1JAyjq
NrUyrhiVu24SOKwIZ5w1e7ZCLvZjSi3hlezhQgCq2fKl4xf2LpbOdh8hkrXqzSub
yIr5mQECgYEA1v8COltnwnqHip536W4hrjbuiS+w/VznbuU2eAj3zi4DPugib3lX
Xrar8RjTJMPSzV6olIR1D2VyKd3Veiyvi+H+mVPZpfd8OPJRuDU98L26AS0/YAqS
FbGwRZxLPD9x5VfKm5zO4JoSbveCzXhoPpkBulQt9nE4XjKsf/bzhFECgYEAy5c6
JGFQzrnIRcEy87VspOXv1UeMNrNWZGtvwP3R+uOsMK23ySiUOx5FSJRK4IuG20RA
hhIXdWVa7oQfeCbtZSk1yms+3x0T/39p6nLZ2b3nP8WXIsGpjwCk7O52vesAjZgH
i2HFXJ7AWYR8zzxhBj3OKW2l0cBrmtOi+D1H5jkCgYBzUguO49KPFYw4hXHKawFz
4hEm0sb7z+ZvrFEAJ8dL95BUIM2/v3Vm31LxGqC+2q7q67g/GaF0pbSL0mqcgvWS
caFP+xMGm+4s2YWN6jkUNaBc2zlgOatMKahkXkZYxatBGksaFw08mkgC745gyhIY
aZfsqxSQWQCkPkgax4qtUQKBgQCoEWGoIsYowmm4W/OKCN11i3Rf5z6y8X2CTMbm
1SKBMW42iVJNN7iWzTh44CKoF8buP/vcMhc3jMJyYJPyBoC3oDuNrNcsLL8TjsWL
C+EXxZOfq6hGwwUMzoVYKsvPoK7GNRkVUVMyUMONore+BKQ8GM2WmbPn4idymv/Q
WhZ+0QKBgFaSJH2os/hjuyjtHAXOU8ktvudy7IegEUlNUzX0Xzk4eToDvDQNevad
SkxRM2/9n4E6QAADUWlLjVgl92W+lLHylDV5baWe+QKMut3vyXjUJYe1ZKYe6zZV
3wx1s/evfKXpd2Vs4ulNEaVs4nDmZ5zyS7TUp/ByabdkAJ5JnUpR
-----END RSA PRIVATE KEY-----`,
		Blob: "AAAAB3NzaC1yc2EAAAADAQABAAABAQCq+zgC6v83e0WMO+4CpwtDgElUemirM79FvBtdIEDlsKZ3us7hzLaKckrSgamE5OfopHGlEkzxbCXruJmtFgxOAcLfnvKbM66sxFc3UvUhk1GfhA7EGhqa5f3ykxGk8zIGzp0RChUkCwtobtlN1YZe6a8ZWQYN2K37rBoYGoPtelWXZDaI1kdO6Fa9I8+hPKGOK/s2WXvNBWnCbC+7/up2UYRLZIfU/geZncTZB7YpnViPhESyKDhahQ8uD7G/6oDSBQ1kQfGIArLpGzvzuawZLduJRdiGYpQbxpEfGObFlyqGXrZScULN4NC2mi9m2VOq2gNlCxk5vfP/VOXEkDwJ",
		Type: "ssh-rsa",
	}

	OpenSSHFormat = TestKey{
		Private: `
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAskcoyCcwRb7++BmO1WKA8U0maoU5AgdN2/kmoGjFqEgAAAJiQRlYEkEZW
BAAAAAtzc2gtZWQyNTUxOQAAACAskcoyCcwRb7++BmO1WKA8U0maoU5AgdN2/kmoGjFqEg
AAAECi8uJyNV03/YUAxiMNV5myM1d8Yrc2iWTPTLS+x09/0yyRyjIJzBFvv74GY7VYoDxT
SZqhTkCB03b+SagaMWoSAAAAEnJhbGltaUB3b3Jrc3RhdGlvbgECAw==
-----END OPENSSH PRIVATE KEY-----`,
		Blob: "AAAAC3NzaC1lZDI1NTE5AAAAICyRyjIJzBFvv74GY7VYoDxTSZqhTkCB03b+SagaMWoS",
		Type: "ssh-ed25519",
	}
)

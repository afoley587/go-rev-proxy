package main

type RegistrationRequest struct {
	OutsideHost string // Outside host address. The hostname or IP on the public network. For example foo.bar.com or foo.localhost
	InsideHost  string // Inside host address. The hostname or IP on the internal network. For example 192.168.10.5
}

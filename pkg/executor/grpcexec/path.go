package grpcexec

import "fmt"

// @symbol: {package}.{service}/{rpc}
type Path struct {
	Package string `json:"package" yaml:"package"`

	Service string `json:"service" yaml:"service"`

	// RPC actually function in service which we are calling
	RPC string `json:"rpc" yaml:"rpc"`
}

func (g *Path) String() string {
	return fmt.Sprintf("%s.%s/%s", g.Package, g.Service, g.RPC)
}

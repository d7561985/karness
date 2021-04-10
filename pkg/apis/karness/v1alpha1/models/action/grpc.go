package action

type GRPC struct {
	// required: true
	Addr string `json:"addr"`

	// Proto package name
	// required: true
	Package string `json:"package"`

	// required: true
	Service string `json:"service"`

	// rpc command
	RPC string `json:"rpc"`
}

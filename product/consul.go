package product

const consulName = "consul"

var Consul = Product{
	Name:              consulName,
	BinaryName:        binaryName(consulName),
	GetVersion:        nil,
	BuildInstructions: nil,
}

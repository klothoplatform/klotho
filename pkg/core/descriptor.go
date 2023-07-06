package core

type (
	Descriptor    string
	Functionality Descriptor
)

const (
	Compute Functionality = "compute"
	Cluster Functionality = "cluster"
	Unknown Functionality = "Unknown"
)

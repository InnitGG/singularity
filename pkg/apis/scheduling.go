package apis

const (
	Packed      SchedulingStrategy = "Packed"
	Distributed SchedulingStrategy = "Distributed"
)

// SchedulingStrategy determines how Singularity should schedule Pods across the cluster.
type SchedulingStrategy string

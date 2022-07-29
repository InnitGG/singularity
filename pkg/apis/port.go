package apis

const (
	Internal PortPolicy         = "Internal"
	Dynamic  SchedulingStrategy = "Dynamic"
)

// PortPolicy determines how Singularity should expose the game server.
type PortPolicy string

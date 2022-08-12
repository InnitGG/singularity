package gameserverset

import (
	singularityv1 "innit.gg/singularity/pkg/apis/singularity/v1"
	"sync"
)

// parallelize processes a channel of game server objects, invoking the provided callback for items in the channel with the specified degree of parallelism up to a limit.
// Returns nil if all callbacks returned nil or one of the error responses, not necessarily the first one.
func parallelize(gameServers chan *singularityv1.GameServer, parallelism int, work func(gs *singularityv1.GameServer) error) error {
	errch := make(chan error, parallelism)

	var wg sync.WaitGroup

	for i := 0; i < parallelism; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for it := range gameServers {
				err := work(it)
				if err != nil {
					errch <- err
					break
				}
			}
		}()
	}
	wg.Wait()
	close(errch)

	for range gameServers {
		// drain any remaining game servers in the channel, in case we did not consume them all
	}

	// return first error from the channel, or nil if all successful.
	return <-errch
}

// newGameServersChannel returns a channel producing n amount of GameServers
func newGameServersChannel(n int, gsSet *singularityv1.GameServerSet) chan *singularityv1.GameServer {
	gameServers := make(chan *singularityv1.GameServer)
	go func() {
		defer close(gameServers)

		for i := 0; i < n; i++ {
			gameServers <- gsSet.GameServer()
		}
	}()

	return gameServers
}

// gameServerListToChannel returns a channel of GameServers from list
func gameServerListToChannel(list []*singularityv1.GameServer) chan *singularityv1.GameServer {
	gameServers := make(chan *singularityv1.GameServer)
	go func() {
		defer close(gameServers)

		for _, gs := range list {
			gameServers <- gs
		}
	}()

	return gameServers
}

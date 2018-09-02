package alternation

import (
	"gocoin/libraries/chain/space/reputation"
	"gocoin/libraries/common"
	"time"
)

const (
	serviceNone                = 0
	serviceReputationAll       = 1
	serviceReputationGroup     = 2
	serviceReputationConsensus = 3
)

var (
	// CFGS wallet configure
	CFGS ExtServices
)

//ExtServices structure for wallet configure
type ExtServices struct {
	id      [8]byte
	address common.Address

	reputationService int
	epochService      int
	settleService     int
}

// Run function for service manager interface
func Run() {
	var loop = func(f func()) {
		/* node define listenning services */

		if serviceNone != CFGS.reputationService {
			switch CFGS.reputationService {
			case serviceReputationGroup:
				{
					// start reputation group service
					go reputation.RunGroup()
					break
				}
			case serviceReputationAll:
				{
					// start the reputation function
					go reputation.RunAll()
					break
				}
			default:

				break
			}
		}

		if serviceNone != CFGS.settleService {
			// start settle service
		}

		f()
	}

	time.Sleep(3 * time.Second)
	loop(Run)
}

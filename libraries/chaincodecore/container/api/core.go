package api


import (
	"io"
	"golang.org/x/net/context"

	"gocoin/libraries/chaincodecore/container/ccintf"
)

type BuildSpecFactory func() (io.Reader, error)
type PrelaunchFunc func() error

//abstract virtual image for supporting arbitrary virual machines
type VM interface {
	Deploy(ctxt context.Context, ccid ccintf.CCID, args []string, env []string, reader io.Reader) error
	Start(ctxt context.Context, ccid ccintf.CCID, args []string, env []string, builder BuildSpecFactory, preLaunchFunc PrelaunchFunc) error
	Stop(ctxt context.Context, ccid ccintf.CCID, timeout uint, dontkill bool, dontremove bool) error
	Destroy(ctxt context.Context, ccid ccintf.CCID, force bool, noprune bool) error
	GetVMName(ccID ccintf.CCID) (string, error)
}


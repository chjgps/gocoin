// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

// package mclock is a wrapper for a monotonic clock source
package mclock

import (
	"time"

	"github.com/aristanetworks/goarista/monotime"
)

type AbsTime time.Duration // absolute monotonic time

func Now() AbsTime {
	return AbsTime(monotime.Now())
}

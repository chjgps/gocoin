// Copyright 2011 Google Inc.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// Array is a pass-by-value UUID that can be used as an effecient key in a map.
type Array [16]byte

// UUID converts uuid into a slice.
func (uuid Array) UUID() UUID {
	return uuid[:]
}

type UUID []byte

// A Version represents a UUIDs version.
type Version byte

// A Variant represents a UUIDs variant.
type Variant byte

// Constants returned by Variant.
const (
	Invalid   = Variant(iota) // Invalid UUID
	RFC4122                   // The variant specified in RFC4122
	Reserved                  // Reserved, NCS backward compatibility.
	Microsoft                 // Reserved, Microsoft Corporation backward compatibility.
	Future                    // Reserved for future definition.
)

var rander = rand.Reader // random function

// Parse decodes s into a UUID or returns nil.  Both the UUID form of
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx and
// urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx are decoded.
func Parse(s string) UUID {
	if len(s) == 36+9 {
		if strings.ToLower(s[:9]) != "urn:uuid:" {
			return nil
		}
		s = s[9:]
	} else if len(s) != 36 {
		return nil
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return nil
	}
	var uuid [16]byte
	for i, x := range [16]int{
		0, 2, 4, 6,
		9, 11,
		14, 16,
		19, 21,
		24, 26, 28, 30, 32, 34} {
		if v, ok := xtob(s[x:]); !ok {
			return nil
		} else {
			uuid[i] = v
		}
	}
	return uuid[:]
}

func (uuid UUID) String() string {
	if len(uuid) != 16 {
		return ""
	}
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:])
}

func encodeHex(dst []byte, uuid UUID) {
	hex.Encode(dst[:], uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}

package guid

import (
	realrand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

// base32 space; case insensitive; ascii-ordered; my least favorite characters (a bunch that can all look like vertical lines, and 'u' for looking too much like 'v') removed with prejudice.
var pushChars = [32]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'k', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}

const radix = 32

const randLen = 16

// timexxxx-randpt1x-randpt2x
// each 8 chars of random is (2^5)^8 a trillion numbers per millisecond.
// 8 chars of time is short enough to roll... about every 34 years.
const size = 8 + 1 + 8 + 1 + 8

// derived from https://github.com/kjk/betterguid/

var (
	lastPushTimeMs int64
	// We generate 80-bits of randomness which get turned into 16 characters and appended to the
	// timestamp to prevent collisions.  Multiple requests in the same millisecond get incremented.
	lastRandChars [randLen]byte
	mu            sync.Mutex
	rnd           *rand.Rand
)

func init() {
	var seed int64
	binary.Read(realrand.Reader, binary.LittleEndian, &seed)
	rnd = rand.New(rand.NewSource(seed))
	for i := 0; i < randLen; i++ {
		lastRandChars[i] = byte(rnd.Intn(radix))
	}
}

func New() string {
	var id [size]byte
	id[17] = '-'
	id[8] = '-'
	mu.Lock()
	timeMs := time.Now().UTC().UnixNano() / 1e6
	if timeMs == lastPushTimeMs {
		// increment lastRandChars
		for i := 0; i < randLen; i++ {
			lastRandChars[i]++
			if lastRandChars[i] < radix {
				break
			}
			// increment the next byte
			lastRandChars[i] = 0
		}
	} else {
		lastPushTimeMs = timeMs
		for i := 0; i < randLen; i++ {
			lastRandChars[i] = byte(rnd.Intn(radix))
		}
	}
	// put random as the third part
	for i := 0; i < 8; i++ {
		id[size-i-1] = pushChars[lastRandChars[i]]
	}
	// put random as the second part
	for i := 8; i < 16; i++ {
		id[size-i-2] = pushChars[lastRandChars[i]]
	}
	mu.Unlock()

	// put current time at the beginning
	for i := 7; i >= 0; i-- {
		n := int(timeMs % radix)
		id[i] = pushChars[n]
		timeMs = timeMs / radix
	}
	// there's actually still a '1' (at the moment) leftover on the left.
	// this'll roll over sometime in 2039.
	// which means this is good enough to keep things from jittering at random, but should not be considered truly sortable.
	// if timeMs != 0 { panic(fmt.Errorf("time still %d", timeMs)) }

	return string(id[:])
}

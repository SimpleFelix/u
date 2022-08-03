package u

import (
	"crypto/rand"
	"math/big"
	"time"
)

//todo generic
func MinInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func MaxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func MinInt64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func MaxInt64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func MinDuration(x, y time.Duration) time.Duration {
	if x < y {
		return x
	}
	return y
}

func MaxDuration(x, y time.Duration) time.Duration {
	if x > y {
		return x
	}
	return y
}

func RandomInRange(from, to int64) int64 {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(to-from))
	n := nBig.Int64()
	return from + n
}

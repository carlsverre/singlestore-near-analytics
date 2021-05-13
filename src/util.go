package src

import (
	"fmt"
	"math/big"
)

func ParseBigInt(i string) *big.Int {
	out, ok := (&big.Int{}).SetString(i, 10)
	if !ok {
		panic(fmt.Sprintf("failed to parse big.Int: %s", i))
	}
	return out
}

package helper

import (
	"crypto/rand"
	"math/big"
)

func GenRandomCode(length int) (string, error) {
	var res string
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		res += num.String()
	}
	return res, nil
}

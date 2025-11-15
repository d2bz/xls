package helper

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type (
	TokenOptions struct {
		AccessSecret string
		AccessExpire int64
		UserID       int
	}

	// claims struct {
	// 	jwt.RegisteredClaims
	// 	UserID int
	// }

	TokenMsg struct {
		AccessToken string `json:"access_token"`
		ExpireAt    int64  `json:"expire_at"`
	}
)

func BuildToken(opts *TokenOptions) (tmsg TokenMsg, err error) {
	claims := make(jwt.MapClaims)
	iat := time.Now().Unix()
	claims["exp"] = iat + opts.AccessExpire
	claims["iat"] = iat
	claims["userid"] = opts.UserID
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	tmsg.AccessToken, err = token.SignedString([]byte(opts.AccessSecret))
	if err != nil {
		return
	}
	tmsg.ExpireAt = iat + opts.AccessExpire
	return tmsg, nil
}

// func BuildToken(opts *TokenOptions) (token Token, err error) {
// 	claims := &claims{
// 		UserID: opts.UserID,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(opts.AccessExpire) * time.Second)),
// 			IssuedAt:  jwt.NewNumericDate(time.Now()),
// 		},
// 	}
// 	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	token.AccessToken, err = t.SignedString([]byte(opts.AccessSecret))
// 	if err != nil {
// 		return
// 	}
// 	token.ExpireAt = time.Now().Unix() + opts.AccessExpire
// 	return token, nil
// }

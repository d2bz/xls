package test

import (
	"testing"
	"xls/app/core/internal/helper"
)

func TestToken(t *testing.T) {
	opts := &helper.TokenOptions{
		AccessSecret: "abc123",
		AccessExpire: 3600,
		UserID:       1,
	}
	token, err := helper.BuildToken(opts)
	if err != nil {
		t.Fatalf("Failed to build token: %v", err)
	}
	t.Logf("Token: %s", token.AccessToken)
}

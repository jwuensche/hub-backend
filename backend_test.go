package main

import (
	"fmt"
	"testing"
)

func TestTokenVerification(t *testing.T) {
	fmt.Println(verifyToken("DDD45DD2DF0879EB"))
}

func TestFeedInit(t *testing.T) {
	verifyFeeds()
}

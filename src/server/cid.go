package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"

	"github.com/quic-go/quic-go"
)

type CIDgen struct {
	ConfigurationID byte
	ServerID        byte
}

func (cidg CIDgen) ConnectionIDLen() int {
	return 12
}

func (cidg CIDgen) GenerateConnectionID() (quic.ConnectionID, error) {
	buf := make([]byte, cidg.ConnectionIDLen())
	if _, err := rand.Read(buf); err != nil {
		return quic.ConnectionIDFromBytes(buf), err
	}

	buf[0] = cidg.ConfigurationID
	buf[1] = cidg.ServerID
	log.Printf("[Server %d] Generated CID: %s", cidg.ServerID, strings.ToUpper(hex.EncodeToString(buf)))

	return quic.ConnectionIDFromBytes(buf), nil
}

func CreateCIDgen(ConfID byte, ServID byte) *CIDgen {
	return &CIDgen{
		ConfigurationID: ConfID,
		ServerID:        ServID,
	}
}

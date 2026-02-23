package main

import (
	"encoding/hex"
	"log"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

var pool *nostr.RelayPool

func initNostr() {
	pool = nostr.NewRelayPool()

	for relay, policy := range config.Relays {
		cherr := pool.Add(relay, nostr.SimplePolicy{
			Read:  policy.Read,
			Write: policy.Write,
		})
		err := <-cherr
		if err != nil {
			log.Printf("error adding relay '%s': %s", relay, err.Error())
		}
	}

	hasRelays := false
	pool.Relays.Range(func(_ string, _ *nostr.Relay) bool {
		hasRelays = true
		return false
	})
	if !hasRelays {
		log.Printf("You have zero relays configured, everything will probably fail.")
	}

	go func() {
		for notice := range pool.Notices {
			msg := strings.ToLower(notice.Message)
			if strings.Contains(msg, "bad signature") {
				continue
			}
			log.Printf("%s has sent a notice: '%s'\n", notice.Relay, notice.Message)
		}
	}()

	if config.PrivateKey != "" {
		// config.PrivateKey is stored as raw bytes; RelayPool expects hex.
		skHex := hex.EncodeToString([]byte(config.PrivateKey))
		pool.SecretKey = &skHex
	}
}

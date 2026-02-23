package main

import (
	"fmt"

	"github.com/docopt/docopt-go"
	"github.com/nbd-wtf/go-nostr"
)

func addRelay(opts docopt.Opts) {
	addr := opts["<url>"].(string)
	addRelayURL(addr)
	fmt.Printf("Added relay %s.\n", addr)
}

// addRelayURL adds a relay to config and, if pool exists, to the pool. Caller must save config.
func addRelayURL(addr string) {
	config.Relays[addr] = Policy{Read: true, Write: true}
	if pool != nil {
		cherr := pool.Add(addr, nostr.SimplePolicy{Read: true, Write: true})
		go func() { <-cherr }()
	}
}

func removeRelay(opts docopt.Opts) {
	if addr, _ := opts.String("<url>"); addr != "" {
		removeRelayURL(addr)
		fmt.Printf("Removed relay %s.\n", addr)
	}

	if all, _ := opts.Bool("--all"); all {
		config.Relays = map[string]Policy{}
		fmt.Println("Removed all relays.")
	}
}

// removeRelayURL removes a relay from config. Caller must save config. Pool is not updated until next initNostr.
func removeRelayURL(addr string) {
	delete(config.Relays, addr)
}

func recommendRelay(opts docopt.Opts) {
	addr := opts["<url>"].(string)

	// TODO

	fmt.Printf("Published a relay recommendation for %s.", addr)
}

func listRelays(opts docopt.Opts) {
	for relay, policy := range config.Relays {
		fmt.Printf("%s: %s\n", relay, policy)
	}
}

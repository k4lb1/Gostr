package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"

	"github.com/docopt/docopt-go"
	"github.com/nbd-wtf/go-nostr/nip06"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func decodeKey(keyraw string) ([]byte, error) {
	// raw 64-char hex
	if len(keyraw) == 64 && !strings.HasPrefix(keyraw, "nsec") {
		keyval, err := hex.DecodeString(keyraw)
		if err != nil {
			return nil, fmt.Errorf("decoding key from hex: %w", err)
		}
		return keyval, nil
	}

	// NIP-19 bech32-encoded (nsec...)
	raw, prefix, err := nip19.Decode(keyraw)
	if err != nil {
		return nil, fmt.Errorf("decoding key from bech32: %w", err)
	}
	if prefix != "nsec" {
		return nil, fmt.Errorf("unexpected bech32 prefix %q, want \"nsec\"", prefix)
	}
	// raw already contains the 32-byte private key.
	return raw, nil
}

func setPrivateKey(opts docopt.Opts) {
	keyraw := opts["<key>"].(string)
	if err := setPrivateKeyString(keyraw); err != nil {
		log.Printf("Failed to parse private key: %s\n", err.Error())
		return
	}
}

// setPrivateKeyString decodes nsec/hex and sets config.PrivateKey. Caller must save config.
func setPrivateKeyString(keyraw string) error {
	keyval, err := decodeKey(keyraw)
	if err != nil {
		return err
	}
	config.PrivateKey = string(keyval)
	return nil
}

func showPublicKey(opts docopt.Opts) {
	if config.PrivateKey == "" {
		log.Printf("No private key set.\n")
		return
	}

	pubkey := getPubKey(config.PrivateKey)
	if pubkey != "" {
		fmt.Printf("%s\n", pubkey)

		nip19pubkey, _ := nip19.EncodePublicKey(pubkey, "")
		fmt.Printf("%s\n", nip19pubkey)
	}
}

func getPubKey(privateKey string) string {
	// privateKey is stored as raw bytes in the config.
	keyb := []byte(privateKey)
	_, pubkey := btcec.PrivKeyFromBytes(keyb)
	return hex.EncodeToString(schnorr.SerializePubKey(pubkey))
}

func keyGen(opts docopt.Opts) {
	seedWords, err := nip06.GenerateSeedWords()
	if err != nil {
		log.Println(err)
		return
	}

	seed := nip06.SeedFromWords(seedWords)

	sk, err := nip06.PrivateKeyFromSeed(seed)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("seed:", seedWords)
	fmt.Println("private key:", sk)
}

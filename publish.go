package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/nbd-wtf/go-nostr"
)

func publish(opts docopt.Opts) {
	if config.PrivateKey == "" {
		log.Printf("Can't publish. Private key not set.\n")
		return
	}

	initNostr()

	var event nostr.Event

	if file, _ := opts.String("--file"); file != "" {
		jsonb, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Failed reading content from file: %s", err)
			return
		}
		if err := json.Unmarshal(jsonb, &event); err != nil {
			log.Printf("Failed unmarshaling json from file: %s", err)
			return
		}
	} else {
		references, err := optSlice(opts, "--reference")
		if err != nil {
			return
		}

		var tags nostr.Tags
		for _, ref := range references {
			tags = append(tags, nostr.Tag{"e", ref})
		}

		profiles, err := optSlice(opts, "--profile")
		if err != nil {
			return
		}

		for _, profile := range profiles {
			tags = append(tags, nostr.Tag{"p", profile})
		}

		content, _ := opts.String("<content>")
		if content == "" {
			log.Printf("Content must not be empty")
			return
		}
		if content == "-" {
			content, err = readContentStdin(4096)
			if err != nil {
				log.Printf("Failed reading content from stdin: %s", err)
				return
			}
		}

		event = nostr.Event{
			CreatedAt: time.Now(),
			Kind:      nostr.KindTextNote,
			Tags:      tags,
			Content:   content,
		}
	}

	publishEvent, statuses, err := pool.PublishEvent(&event)
	if err != nil {
		log.Printf("Error publishing: %s.\n", err.Error())
		return
	}

	printPublishStatus(publishEvent, statuses)
}

// PublishReaction sends a kind-7 reaction (like) to the given event. NIP-25.
// Returns the created event ID on success.
func PublishReaction(evID, authorPubkey string) (string, error) {
	if config.PrivateKey == "" {
		return "", errors.New("private key not set")
	}
	initNostr()
	tags := nostr.Tags{
		{"e", evID},
		{"p", authorPubkey},
	}
	ev := nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindReaction,
		Tags:      tags,
		Content:   "+",
	}
	pub, _, err := pool.PublishEvent(&ev)
	if err != nil {
		return "", err
	}
	return pub.ID, nil
}

// PublishBoost sends a kind-6 boost (repost) for the given event. NIP-18.
// eventJSON is the stringified JSON of the boosted event (recommended by NIP-18).
// Returns the created event ID on success.
func PublishBoost(evID, authorPubkey, eventJSON string) (string, error) {
	if config.PrivateKey == "" {
		return "", errors.New("private key not set")
	}
	initNostr()
	tags := nostr.Tags{
		{"e", evID},
		{"p", authorPubkey},
	}
	ev := nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindBoost,
		Tags:      tags,
		Content:   eventJSON,
	}
	pub, _, err := pool.PublishEvent(&ev)
	if err != nil {
		return "", err
	}
	return pub.ID, nil
}

// PublishDeletion publishes a kind-5 deletion for the given event ID. NIP-09.
func PublishDeletion(evID string) error {
	if config.PrivateKey == "" {
		return errors.New("private key not set")
	}
	initNostr()
	_, _, err := pool.PublishEvent(&nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindDeletion,
		Tags:      nostr.Tags{{"e", evID}},
		Content:   "",
	})
	return err
}

func optSlice(opts docopt.Opts, key string) ([]string, error) {
	if v, ok := opts[key]; ok {
		vals, ok := v.([]string)
		if ok {
			return vals, nil
		}
	}

	return []string{}, errors.New("unable to find opt")
}

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/nbd-wtf/go-nostr"
)

func viewEvent(opts docopt.Opts) {
	gopher, _ := opts.Bool("--gopher")
	verbose, _ := opts.Bool("--verbose")
	jsonformat, _ := opts.Bool("--json")
	id := opts["<id>"].(string)
	if id == "" {
		log.Println("provided event ID was empty")
		return
	}
	initNostr()

	_, all := pool.Sub(nostr.Filters{{IDs: []string{id}}})
	for event := range nostr.Unique(all) {
		if event.ID != id {
			log.Printf("got unexpected event %s.\n", event.ID)
			continue
		}

		if gopher {
			printGostrHeader()
			for _, line := range formatAsGopher(event, nil) {
				fmt.Printf("%s\r\n", line)
			}
		} else {
			printEvent(event, nil, verbose, jsonformat)
		}
		break
	}
}

func deleteEvent(opts docopt.Opts) {
	initNostr()

	id := opts["<id>"].(string)
	if id == "" {
		log.Println("Event id is empty! Exiting.")
		return
	}

	event, statuses, err := pool.PublishEvent(&nostr.Event{
		CreatedAt: time.Now(),
		Kind:      nostr.KindDeletion,
		Tags:      nostr.Tags{nostr.Tag{"e", id}},
	})
	if err != nil {
		log.Printf("Error publishing: %s.\n", err.Error())
		return
	}

	printPublishStatus(event, statuses)
}

// iterEventsWithTimeout returns a channel of events; this channel will be
// closed once events have stopped arriving for timeoutDuration
func iterEventsWithTimeout(events chan nostr.Event, timeoutDuration time.Duration) chan nostr.Event {
	resultsChan := make(chan nostr.Event)

	go func() {
		defer close(resultsChan)

		timer := time.NewTimer(timeoutDuration)
		defer timer.Stop()

		for {
			select {
			case ev, ok := <-events:
				if !ok {
					// upstream closed: we're done
					return
				}

				// forward event
				resultsChan <- ev

				// reset inactivity timer
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(timeoutDuration)

			case <-timer.C:
				// no events for timeoutDuration: stop
				return
			}
		}
	}()

	return resultsChan
}

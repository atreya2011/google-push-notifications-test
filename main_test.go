package main

import (
	"testing"

	"github.com/sanity-io/litter"
)

func TestGetEvents(t *testing.T) {
	events, nst, err := getEvents()
	if err == nil {
		litter.Dump(events)
		litter.Dump(nst)
	}
	if err != nil {
		t.Errorf("got error %v", err)
	}
}

package hindsight

import "time"

type Storage interface {
	Store(evts ...*Event) error
	Fetch(from, until time.Time, filter *Filter) ([]*Event, error)
}

type Filter struct {
	HostList []string
	// what else?
}

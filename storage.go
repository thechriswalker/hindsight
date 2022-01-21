package hindsight

import "time"

type Storage interface {
	Store(evts []*Event) error
	Fetch(day time.Time, filter *Filter) ([]*Event, error)
}

type Filter struct {
	HostList []string
	// what else?
}

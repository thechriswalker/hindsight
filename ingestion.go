package hindsight

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

func ListenForIngestion(ctx context.Context, c *Config, store Storage) error {
	l, err := net.Listen("tcp", c.ListenIngestion)
	if err != nil {
		return fmt.Errorf("could not start ingestion listener: %w", err)
	}

	go func() {
		// we need to close the listener in the case of context.Done
		<-ctx.Done()
		l.Close()
	}()

	for {
		sock, err := l.Accept()
		if err != nil {
			// log it
			// but only if we didn't cancel it ourselves
			select {
			case <-ctx.Done():
				// yep, just quit
				return nil
			default:
				return fmt.Errorf("error accepting connection: %w", err)
			}
		}
		// enter a goroutine
		go func(conn net.Conn) {
			// new shadowed context with a cancel function
			ctx, cancel := context.WithCancel(ctx)
			// first if the ctx is done, we need to quit
			// but don't block here!
			defer cancel()

			go func() {
				<-ctx.Done()
				conn.Close()
			}()

			sc := bufio.NewScanner(conn)
			for sc.Scan() {
				in := &InboundEvent{}
				err := json.Unmarshal(sc.Bytes(), in)
				if err != nil {
					// bad producer.
					log.Warn().Err(err).Str("line", sc.Text()).Msg("bad event from producer")
					return
				}
				ev := mapInboundEvent(c, in)
				err = store.Store(ev)
				if err != nil {
					// this one is our fault!
					log.Error().Err(err).Msg("failed to store event")
					// we should continue though...
				} else {
					// should we log anyway?
					log.Trace().Interface("evt", ev).Msg("ingested")
				}
			}
			if err := sc.Err(); err != nil {
				// nothing we can do about it now...
				log.Warn().Err(err).Msg("error reading event from producer")
			}
		}(sock)
	}
}

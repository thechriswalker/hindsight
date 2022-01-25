package hindsight

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ListenIngestion string // host:port for API - should not be public
	ListenUI        string // host:port for UI - could be exposed to internet
	DatabasePath    string // path to DB
	RandomSaltSeed  string // A random string to use to generate the daily hashes
}

func LoadConfig(filename string) (*Config, error) {
	// set defaults
	c := &Config{
		ListenIngestion: "127.0.0.1:8765",
		ListenUI:        "127.0.0.1:8080",
		DatabasePath:    "hindsight.db",
		RandomSaltSeed:  "", // leave this empty until after toml unmarshalling
	}
	_, err := toml.DecodeFile(filename, c)
	if err != nil {
		return nil, fmt.Errorf("could not load config file from %q: %w", filename, err)
	}
	if c.RandomSaltSeed == "" {
		// generate some random data.
		buf := make([]byte, 16)
		_, err = rand.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("could not generate any randomness: %w", err)
		}
		c.RandomSaltSeed = base64.RawURLEncoding.EncodeToString(buf)
		// we should write that back to the file...
		// but I don't want to "blat" any comments.
		//so we will read the entire file and replace that little bit
		existing, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("could not read config file to add randomness: %w", err)
		}
		// we will rewrite the whole file
		b := bytes.Buffer{}
		index := bytes.Index(existing, []byte(`random_salt_seed`))
		if index == -1 {
			// just append
			b.Write(existing)
			fmt.Fprintf(&b, "\nrandom_salt_seed = %q # AUTO-GENERATED\n", c.RandomSaltSeed)
		} else {
			// replace, write data before
			b.Write(existing[0:index])
			// insert new value (we don't need the newline here)
			fmt.Fprintf(&b, "random_salt_seed = %q # AUTO-GENERATED\n", c.RandomSaltSeed)
			// find the original next newline
			newline := bytes.IndexByte(existing[index:], '\n')
			if newline != -1 {
				// if it was -1, then there was no trailing newline, so we don't have any more file.
				// but if not, then we should write the rest.
				b.Write(existing[index+newline+1:])
			}
		}
		err = ioutil.WriteFile(filename, b.Bytes(), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to rewrite config file: %w", err)
		}
	}
	return c, nil
}

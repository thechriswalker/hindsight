# Hindsight

Hindsight is a privacy focused analytics package for **small** virtual-hosted environments. That is, I built it to provide simple analytics for my own small server and the services I host there.

I have no idea how it would cope with large traffic sites.

It is named `hindsight` as it is about looking back on data to gain knowledge. Also, because of Captain Hindsight (as I was going to call this when I first came up with the idea...).

## Privacy

Captain Hindsight works on pretty much just the data available in the Combined Log Format, as well as the virtual hosts.

To keep privacy the logs are allocated to a "user" with the following anonymisation:

```
SHA256(
    <VirtualHost> ||
    <Remote IP Address> ||
    <User Agent> ||
    <Daily Random Salt> ||
)
```

The daily random salt is rotated on a daily basis and calculated based on the "day" using UTC.

This is almost exactly how "plausible.io" does it. We the data we do store about each hit is similar to plausible, and does not contain any personally identifiable data:

- Request: Host, Path, Method
- Derived from UA:
  - Device Kind: Desktop/Mobile/Tablet...
  - Browser: including some version info
  - OS: including some version info
- Location and ASN: derived from remote IP, we only take the ISO country code.
- Response: StatusCode, Duration, Content-Length

The response data is useful and not generally available via client-side javascript tracking.

No cookies are used and no javascript.

Because we do not implement JS there is no click tracking, or "events" only "PageViews", however you could perform this via your own code a "beacon" (https://developer.mozilla.org/en-US/docs/Web/API/Navigator/sendBeacon) or with certain browsers via the link `ping` attribute (https://developer.mozilla.org/en-US/docs/Web/HTML/Element/a#attr-ping)

These requests would be tracked just like any other.

### Installation and Requirements

Captain Hindsight is written in Go and deploys as a single binary, piece of cake. Build with:

```
make
```

To produce a binary in `bin/`.

However it requires a database, and by default that will be SQLite. That is totally inadequate for storing and querying large scale analytics, but my sites get minimal traffic. I may add an option for Postgres (maybe with Timescale), but for now I just want something that works for me.

### Usage

Captain Hindsight can ingest information in 2 different ways:

- From Log files is Combined/Common Log Format.

  We can point the captain to a directory/file and ingest the
  data from there. This can be a one-off ingestion process or
  long running `tail` style ingestion.

- From Applications directly i.e. plugin via a middleware or log library

  This allows adding events via an API. There is a client library
  for Go `github.com/0x6377/captain-hindsight/client`. I may make
  one for NodeJS/Browser

  Or, you know, `curl`:

  ```
  curl http://hindsight/api/ingest \
    -H "content-type: application/json" \
    -H "authorisation: Bearer $API_TOKEN" \
    -data-binary '{"hindsight:"1.0", ...}'
  ```

  Or in batch:

  ```
  curl http://hindsight/api/ingest \
    -H "content-type: application/x-ndjson" \
    -H "authorisation: Bearer $API_TOKEN" \
    -data-binary @lines.ndjson
  ```

#### JSON Format For Events

The canonical format for a log line is in JSON and the ingest endpoint can accept multiple object via Newline Delimited JSON, which is basically: keep newlines out of your objects then one object per line (https://github.com/ndjson/ndjson-spec).

The actual data should be keyed as follows:

```json
{
  "Hindsight": "1.0", // this is so I can change later.
  "Time": "RFC3339 Timestamp, in UTC",
  "IP": "remote IP address",
  "Host": "virtual.host.com",
  "Method": "GET, POST, etc...",
  "Path": "/path/of/request?and=query", //
  "UserAgent": "UA string from 'user-agent' header",
  "StatusCode": 200, // or whatever
  "BytesWritten": 1234, // or whatever
  "DurationMS": 1234 // or however long
}
```

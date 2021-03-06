# trello-calendar-proxy
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue)](https://godoc.org/github.com/pgaskin/trello-calendar-proxy) [![Go Report Card](https://goreportcard.com/badge/github.com/pgaskin/trello-calendar-proxy)](https://goreportcard.com/report/github.com/pgaskin/trello-calendar-proxy) [![Drone (cloud)](https://img.shields.io/drone/build/pgaskin/trello-calendar-proxy)](https://cloud.drone.io/pgaskin/trello-calendar-proxy) [![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/geek1011/trello-calendar-proxy)](https://hub.docker.com/r/geek1011/trello-calendar-proxy) [![Deploy](https://img.shields.io/badge/heroku-deploy-blueviolet)](https://heroku.com/deploy) [![Demo](https://img.shields.io/website/https/trellocal.geek1011.net?down_color=lightgrey&down_message=offline&label=demo&up_color=blue&up_message=up)](https://trellocal.geek1011.net)

```
NAME
    trello-calendar-proxy - Adds additional features to the Trello Calendar Power-Up

SYNOPSIS
    /                                 - Shows this message
    /calendar/{uid}/{cid}/{token}.ics - Proxies a Trello calendar URL

DESCRIPTION
    trello-calendar-proxy modifies the calendars returned from the Trello Calendar
    Power-Up to add additional features. To use this with a Trello Calendar URL,
    just replace https://trello.com/calendar/ with the URL of this proxy.

FEATURES
    Refresh Interval
    -- The refresh interval is reduced from 1 hour to 15 minutes.

    Event Duration
    -- A custom event duration can be set by adding "Calendar::Duration=dur" on
       its own line (without the quotes) at the top of the card description.
       Replace dur with a valid Go time.Duration between 1s and 7d, for example
       1d, 2h, 1h30m, 15m30s, and so on. If not specified, the Trello default of
       1h is used.

    Event Location
    -- A custom event location can be set by adding "Calendar::Location=loc" on
       its own line (without the quotes) at the top of the card description.
       Replace loc with the double-quoted location.

    Event Recurrence
    -- Event recurrence will be implemented in a future version.
```

```
Usage of trello-calendar-proxy:
  -a, --addr string   The address to bind to (env TRELLO_CALPROXY_ADDR) (default ":8080")
      --help          Show this help text
```

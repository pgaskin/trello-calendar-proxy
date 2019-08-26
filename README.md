# trello-calendar-proxy

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

    Event Recurrence
    -- Event recurrence will be implemented in a future version.
```

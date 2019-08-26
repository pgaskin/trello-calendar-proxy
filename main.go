package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"
)

var rev = "git"

func main() {
	addr := pflag.StringP("addr", "a", ":8080", "The address to bind to")
	help := pflag.Bool("help", false, "Show this help text")

	envmap := map[string]string{
		"addr": "TRELLO_CALPROXY_ADDR",
	}

	if val, ok := os.LookupEnv("PORT"); ok {
		val = ":" + val
		fmt.Printf("Setting --addr from PORT to %#v\n", val)
		if err := pflag.Set("addr", val); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(2)
		}
	}

	pflag.VisitAll(func(flag *pflag.Flag) {
		if env, ok := envmap[flag.Name]; ok {
			flag.Usage += fmt.Sprintf(" (env %s)", env)
			if val, ok := os.LookupEnv(env); ok {
				if err := flag.Value.Set(val); err != nil {
					fmt.Fprintf(os.Stderr, "Error: env var %s (--%s) = %#v: %v\n", env, flag.Name, val, err)
					os.Exit(2)
				}
			}
		}
	})

	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(2)
	}

	run(*addr)
}

func run(addr string) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.GetHead)
	r.Use(middleware.SetHeader("Server", fmt.Sprintf("trello-calendar-proxy (%s)", rev)))

	r.Get("/", readme)
	r.Get("/calendar/{uid}/{cid}/{token}.ics", transformCalendar)

	fmt.Printf("trello-calendar-proxy (%s)\n", rev)
	fmt.Printf("Listening on http://%s\n", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func readme(w http.ResponseWriter, r *http.Request) {
	buf := []byte(fmt.Sprintf(`NAME
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

ABOUT
    If using this with a private calendar, I recommend hosting your own instance.

    Copyright 2019 Patrick Gaskin
    MIT License

    GitHub - https://github.com/geek1011/trello-calendar-proxy
    Revision - %s
`, rev))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.WriteHeader(http.StatusOK)
	if r.Method != "HEAD" {
		w.Write(buf)
	}
}

func transformCalendar(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")
	cid := chi.URLParam(r, "cid")
	token := chi.URLParam(r, "token")

	req, err := http.NewRequest("GET", fmt.Sprintf("https://trello.com/calendar/%s/%s/%s.ics", uid, cid, token), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error creating Trello request: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error executing Trello request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "calendar not found")
		return
	} else if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Trello returned status %s:\n", resp.Status)
		io.Copy(w, resp.Body)
		return
	}

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/calendar") {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Trello returned invalid content type %#v", resp.Header.Get("Content-Type"))
		return
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error executing Trello request: %v", err)
		return
	}

	ical, err := ParseICal(buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error parsing calendar: %v", err)
		return
	}

	setRefreshTime(ical, time.Minute*15)
	addDurations(ical)
	// TODO: RRULE(only daily/weekly/monthly+count/until)?

	nbuf := ical.Bytes()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(nbuf)))
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)

	if r.Method != "HEAD" {
		w.Write(nbuf)
	}
}

func setRefreshTime(ical ICal, dur time.Duration) {
	idur, err := ICalDuration(dur)
	if err != nil {
		panic(err)
	}
	for _, calendar := range ical {
		for _, node := range calendar.Inner {
			if node.Name == "X-PUBLISHED-TTL" || node.Name == "REFRESH-INTERVAL" {
				node.Value = idur
			}
		}
	}
}

var (
	errDurationNotFound = xerrors.New("duration not specified in event description")
	durationRe          = regexp.MustCompile(`(?:^|\s+)Calendar::Duration=(\S+)\s*`)
)

func addDurations(ical ICal) {
	for _, calendar := range ical {
		for _, node := range calendar.Inner {
			if node.Name == "BEGIN" && node.Value == "VEVENT" {
				var newInner []*Node
				var idur string
				var err error
				for _, innerNode := range node.Inner {
					if innerNode.Name == "DESCRIPTION" {
						if idur, err = parseDuration(innerNode.Value); err != nil && !xerrors.Is(err, errDurationNotFound) {
							innerNode.Value += fmt.Sprintf("\n\nwarning: trello-calendar-proxy: parse duration: %v", err)
						}
						innerNode.Value = durationRe.ReplaceAllString(innerNode.Value, "")
					}
					if !innerNode.NamePrefix("DTEND") && !innerNode.NamePrefix("DURATION") {
						newInner = append(newInner, innerNode)
					}
				}
				if idur == "" {
					idur = "PT1H"
				}
				newInner = append(newInner, &Node{
					Name:  "DURATION",
					Value: idur,
				})
				node.Inner = newInner
			}
		}
	}
}

func parseDuration(desc string) (string, error) {
	m := durationRe.FindAllStringSubmatch(desc, -1)
	switch len(m) {
	case 0:
		return "", errDurationNotFound
	case 1:
		dur, err := time.ParseDuration(m[0][1])
		if err != nil {
			return "", xerrors.Errorf("invalid duration %#v: %w", m[0][1], err)
		}
		idur, err := ICalDuration(dur)
		if err != nil {
			return "", xerrors.Errorf("invalid duration %#v: %w", m[0][1], err)
		}
		return idur, nil
	default:
		return "", xerrors.New("multiple durations specified")
	}
}

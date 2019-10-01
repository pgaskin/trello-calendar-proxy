package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ICal is a simple AST for an iCalendar. The main focus is to be able to
// reproduce the same output as the input and to be tolerant of errors (but
// garbage in, garbage out). Note that the line ending is normalized to \r\n for
// the output, and the lines are wrapped at 75 chars (according to RFC5545
// section 3.1). Extra blank lines *may* also be removed.
type ICal []*Node

// Node represents a part of the iCalendar. For convenience, lines with a name
// of BEGIN have their block's contents put into the Inner slice until the
// matching END is reached.
type Node struct {
	Name  string
	Value string
	Inner []*Node
}

// NamePrefix checks if the node name has a prefix. This is useful for names
// like DTEND, which could be suffixed with a timezone.
func (n Node) NamePrefix(prefix string) bool {
	return strings.HasPrefix(n.Name, prefix)
}

// ParseICal parses an AST from a raw iCalendar.
func ParseICal(buf []byte) (ICal, error) {
	nodes, err := parse(denormalize(buf))
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if (node.Name != "BEGIN") || node.Value != "VCALENDAR" {
			return nil, fmt.Errorf("expected only VCALENDAR objects in root, got %s:%s", node.Name, node.Value)
		}
	}
	return ICal(nodes), nil
}

// denormalize normalizes the bytes for an iCalendar for parsing.
func denormalize(buf []byte) []string {
	buf = bytes.ReplaceAll(buf, []byte("\r\n"), []byte("\n")) // CRLF -> LF
	buf = bytes.ReplaceAll(buf, []byte("\n"), []byte("\r\n")) // LF -> CRLF
	buf = bytes.ReplaceAll(buf, []byte("\r\n "), []byte{})    // Unwrap lines
	return strings.Split(string(buf), "\r\n")                 // Split into lines
}

// parse parses a normalized slice of nodes.
func parse(lines []string) ([]*Node, error) {
	var nodes []*Node
	for i := 0; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}

		node, err := parseNode(lines[i])
		if err != nil {
			return nil, err
		}

		// Nested nodes of the same type are not supported.
		switch node.Name {
		case "BEGIN":
			// TODO: don't parse everything twice, rewrite the whole parser using loops and stacks instead of recursion
			var found bool
			for j := i + 1; j < len(lines); j++ {
				if tmp, err := parseNode(lines[j]); err != nil {
					return nil, err
				} else if tmp.Name == "END" && tmp.Value == node.Value {
					node.Inner, err = parse(lines[i+1 : j])
					if err != nil {
						return nil, err
					}
					i = j // set cursor to end line
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("could not find end for node (%d, %s)", i, lines[i])
			}
		case "END":
			return nil, fmt.Errorf("node nesting mismatch (%d, %s)", i, lines[i]) // if wrong end node or unexpected end node
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// parseNode parses a single normalized Node.
func parseNode(line string) (*Node, error) {
	var node Node
	spl := strings.SplitN(line, ":", 2)
	if len(spl) != 2 {
		return &node, fmt.Errorf("malformed iCal: expected key-value pair, got %#v", line)
	}
	node.Name, node.Value = spl[0], strings.NewReplacer(
		"\\r", "",
		"\\n", "\n",
	).Replace(spl[1])
	return &node, nil
}

// Bytes encodes the ICal into a byte slice. It will not fail.
func (ical ICal) Bytes() []byte {
	return renormalize(encode([]*Node(ical)))
}

// renormalize normalizes the lines for an iCalendar.
func renormalize(lines []string) []byte {
	const ending, indent = "\r\n", " "
	buf := bytes.NewBuffer(nil)
	for _, line := range lines {
		if len(line) <= 75 {
			buf.WriteString(line)
			buf.WriteString(ending)
			continue
		}
		buf.WriteString(line[:75])
		for i, c := range line[75:] {
			if i%(75-len(indent)) == 0 {
				buf.WriteString(ending)
				buf.WriteString(indent)
			}
			buf.WriteRune(c)
		}
		buf.WriteString(ending)
	}
	return buf.Bytes()
}

// encode encodes a slice of nodes.
func encode(nodes []*Node) []string {
	var lines []string
	for _, node := range nodes {
		if len(node.Inner) > 0 && node.Name != "BEGIN" {
			panic("BEGIN must be the name of a node with children")
		} else if node.Name == "END" {
			panic("END is automatically generated and should not be in the AST")
		}

		lines = append(lines, encodeNode(node))
		if node.Name == "BEGIN" {
			lines = append(lines, encode(node.Inner)...)
			lines = append(lines, encodeNode(&Node{
				Name:  "END",
				Value: node.Value,
			}))
		}
	}
	return lines
}

// encodeNode encodes a node.
func encodeNode(node *Node) string {
	return fmt.Sprintf("%s:%s", node.Name, strings.NewReplacer(
		"\r", "",
		"\n", "\\n",
	).Replace(node.Value))
}

// ICalDuration formats a time.Duration according to https://icalendar.org/iCalendar-RFC-5545/3-3-6-duration.html.
func ICalDuration(t time.Duration) (string, error) {
	if t >= time.Hour*24*7 || t <= -(time.Hour*24*7) {
		return "", errors.New("duration too large (must be under 1 week)")
	} else if t >= -time.Second && t <= time.Second {
		return "", errors.New("duration too small (must be at least 1 second)")
	}

	var neg bool
	var d, h, m, s int
	if t < 0 {
		t *= -1
		neg = true
	}
	d, t = int(t/(time.Hour*24)), t%(time.Hour*24)
	h, t = int(t/time.Hour), t%time.Hour
	m, t = int(t/time.Minute), t%time.Minute
	s = int(t / time.Second)

	buf := bytes.NewBuffer(nil)
	if neg {
		buf.WriteRune('-')
	}
	buf.WriteRune('P')
	if d > 0 {
		buf.WriteString(strconv.Itoa(d))
		buf.WriteRune('D')
	}
	if h > 0 || m > 0 || s > 0 {
		buf.WriteRune('T')
		switch {
		case h > 0:
			buf.WriteString(strconv.Itoa(h))
			buf.WriteRune('H')
			fallthrough
		case m > 0:
			buf.WriteString(strconv.Itoa(m))
			buf.WriteRune('M')
			fallthrough
		case s > 0:
			buf.WriteString(strconv.Itoa(s))
			buf.WriteRune('S')
		}
	}
	return buf.String(), nil
}

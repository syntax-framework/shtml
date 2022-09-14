package cmn

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type ServerTimingMetric struct {
	Name        string
	Duration    string
	Description string
	start       *time.Time
}

func (t *ServerTimingMetric) Start() *ServerTimingMetric {
	now := time.Now()
	t.start = &now
	return t
}

func (t *ServerTimingMetric) Stop() {
	if t.start == nil {
		return
	}
	t.Duration = fmt.Sprintf("%.3f", float64(time.Now().Sub(*t.start).Microseconds())/float64(time.Microsecond))
}

// ServerTiming http Server Timing implementation
//
// https://www.w3.org/TR/server-timing/
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing
type ServerTiming struct {
	Timers []*ServerTimingMetric
}

func (t *ServerTiming) Metric(name string, description string) *ServerTimingMetric {
	timer := &ServerTimingMetric{
		Name:        name,
		Description: description,
	}
	t.Timers = append(t.Timers, timer)
	return timer
}

func (t *ServerTiming) String() string {
	buf := bytes.Buffer{}
	i := 0
	for _, timer := range t.Timers {

		// db;dur=123;desc="Database", tmpl;dur=56;desc="Template processing"
		name := strings.TrimSpace(timer.Name)
		if name != "" {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(name)
			if timer.Duration != "" {
				buf.WriteString(";dur=")
				buf.WriteString(timer.Duration)
			}
			description := strings.ReplaceAll(strings.TrimSpace(timer.Description), `"`, "")
			if description != "" {
				buf.WriteString(`;desc="`)
				buf.WriteString(description)
				buf.WriteByte('"')
			}
			i++
		}
	}
	return buf.String()
}

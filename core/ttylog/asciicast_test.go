package ttylog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeConversions(t *testing.T) {
	cases := map[string]struct {
		microseconds int64
		seconds      float64
	}{
		"precision": {
			microseconds: 1,
			seconds:      1e-6,
		},
		"negative": {
			microseconds: -631119539e6,
			seconds:      -631119539,
		},
		"positive": {
			microseconds: 631119539e6,
			seconds:      631119539,
		},
		"bigprecise": {
			microseconds: 123456789987654,
			seconds:      123456789.987654,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			s2m := secondsToMicroseconds(tc.seconds)
			m2s := microsecondsToSeconds(tc.microseconds)

			// Only allow delta to be to the NS
			assert.InDelta(t, m2s, tc.seconds, float64(time.Nanosecond)/float64(time.Second))
			assert.Equal(t, s2m, tc.microseconds)
		})
	}
}

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRequestsByCountry_IncrementsPerLabelSet(t *testing.T) {
	RequestsByCountry.Reset()

	RequestsByCountry.WithLabelValues("US", "NA").Inc()
	RequestsByCountry.WithLabelValues("US", "NA").Inc()
	RequestsByCountry.WithLabelValues("GB", "EU").Inc()

	assert.Equal(t, float64(2), testutil.ToFloat64(RequestsByCountry.WithLabelValues("US", "NA")))
	assert.Equal(t, float64(1), testutil.ToFloat64(RequestsByCountry.WithLabelValues("GB", "EU")))
}

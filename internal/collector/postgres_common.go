package collector

import (
	"github.com/barcodepro/pgscv/internal/log"
	"github.com/barcodepro/pgscv/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

const (
	// Postgres server versions numeric representations.
	PostgresV96 = 90600
	PostgresV10 = 100000
)

// parsePostgresStats extracts values from query result, generates metrics using extracted values and passed
// labels and send them to Prometheus.
func parsePostgresStats(r *store.QueryResult, ch chan<- prometheus.Metric, descs []typedDesc, labelNames []string) error {
	for _, row := range r.Rows {
		for i, colname := range r.Colnames {
			// Column's values act as metric values or as labels values.
			// If column's name is NOT in the labelNames, process column's values as values for metrics. If column's name
			// is in the labelNames, skip that column.
			if !stringsContains(labelNames, string(colname.Name)) {
				var labelValues = make([]string, len(labelNames))

				// Get values from columns which are specified in labelNames. These values will be attached to the metric.
				for j, lname := range labelNames {
					// Get the index of the column in QueryResult, using that index fetch the value from row's values.
					for idx, cname := range r.Colnames {
						if lname == string(cname.Name) {
							labelValues[j] = row[idx].String
						}
					}
				}

				// Skip empty (NULL) values.
				if row[i].String == "" {
					log.Debug("got empty (NULL) value, skip")
					continue
				}

				// Get data value and convert it to float64 used by Prometheus.
				v, err := strconv.ParseFloat(row[i].String, 64)
				if err != nil {
					log.Errorf("skip collecting metric: %s", err)
					continue
				}

				// Get index of the descriptor from 'descs' slice using column's name. This index will be needed below when need
				// to tie up extracted data values with suitable metric descriptor - column's name here is the key.
				idx, err := lookupByColname(descs, string(colname.Name))
				if err != nil {
					log.Debugf("skip collecting metric: %s", err)
					continue
				}

				// Generate metric and throw it to Prometheus.
				ch <- descs[idx].mustNewConstMetric(v, labelValues...)
			}
		}
	}

	return nil
}

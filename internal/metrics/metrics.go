package metrics

import "github.com/prometheus/client_golang/prometheus/promhttp"

// Expose metrics handler for /metrics via router
var Handler = promhttp.Handler

func Init() {}
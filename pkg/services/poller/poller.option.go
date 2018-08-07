package poller

// Option functional options for Poller
type Option func(*Poller)

// WithPollInterval allows customization of pollInterval
func WithPollInterval(interval int) Option {
	return func(w *Poller) {
		w.pollInterval = interval
	}
}

// WithJobBatchSize allows customization of jobBatchSize
func WithJobBatchSize(size *int64) Option {
	return func(w *Poller) {
		w.jobBatchSize = size
	}
}

// WithQueryParam allows customization of queryParam
func WithQueryParam(params map[string]*string) Option {
	return func(w *Poller) {
		w.queryParam = params
	}
}

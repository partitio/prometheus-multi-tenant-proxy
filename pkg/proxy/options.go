package proxy

import (
	"net/url"
)

type Option func(o *options)

type options struct {
	prometheusServerURL   *url.URL
	alertmanagerServerURL *url.URL
	label                 string
	labelsAPIEnabled      bool
}

func WithPrometheus(url *url.URL) Option {
	return func(o *options) {
		o.prometheusServerURL = url
	}
}

func WithAlertmanager(url *url.URL) Option {
	return func(o *options) {
		o.alertmanagerServerURL = url
	}
}

func WithLabel(label string) Option {
	return func(o *options) {
		o.label = label
	}
}

func WithLabelsAPI(enabled bool) Option {
	return func(o *options) {
		o.labelsAPIEnabled = enabled
	}
}

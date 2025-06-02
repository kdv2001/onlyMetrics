package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlers_CollectMetric(t *testing.T) {
	t.Parallel()
	type fields struct {
		metricUseCases useCases
	}
	type expected struct {
		status int
	}
	type args struct {
		w *httptest.ResponseRecorder
		r func() *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected expected
	}{
		{
			name: "ok add gauge metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: nil,
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
						"gauge",
						"something",
						"100",
					), nil)
					req.SetPathValue(MetricTypePathKey, "gauge")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "100")

					return req
				},
			},
			expected: expected{
				status: http.StatusOK,
			},
		},
		{
			name: "ok add counter metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: nil,
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
						"counter",
						"something",
						"100",
					), nil)
					req.SetPathValue(MetricTypePathKey, "counter")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "100")

					return req
				},
			},
			expected: expected{
				status: http.StatusOK,
			},
		},
		{
			name: "unknown metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: nil,
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequestWithContext(context.Background(),
						http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
							"unknownMetric",
							"something",
							"100",
						), nil)
					req.SetPathValue(MetricTypePathKey, "unknownMetric")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "100")

					return req
				},
			},
			expected: expected{
				status: http.StatusBadRequest,
			},
		},
		{
			name: "err parse gauge metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: nil,
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequestWithContext(context.Background(),
						http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
							"gauge",
							"something",
							"badValue",
						), nil)
					req.SetPathValue(MetricTypePathKey, "gauge")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "bad value")

					return req
				},
			},
			expected: expected{
				status: http.StatusBadRequest,
			},
		},
		{
			name: "err parse counter metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: nil,
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequestWithContext(context.Background(),
						http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
							"counter",
							"something",
							"badValue",
						), nil)
					req.SetPathValue(MetricTypePathKey, "counter")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "badValue")

					return req
				},
			},
			expected: expected{
				status: http.StatusBadRequest,
			},
		},
		{
			name: "err update metric",
			fields: fields{
				metricUseCases: &metricUseCaseMock{
					err: errors.New("some error"),
				},
			},
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					req := httptest.NewRequestWithContext(context.Background(),
						http.MethodPost, fmt.Sprintf("/update/%s/%s/%s",
							"counter",
							"something",
							"100",
						), nil)
					req.SetPathValue(MetricTypePathKey, "counter")
					req.SetPathValue(MetricNamePathKey, "something")
					req.SetPathValue(ValuePathKey, "100")

					return req
				},
			},
			expected: expected{
				status: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &Handlers{
				metricUseCases: tt.fields.metricUseCases,
			}
			h.CollectMetric(tt.args.w, tt.args.r())

			if tt.args.w.Code != tt.expected.status {
				t.Errorf("got %d, want %d", tt.args.w.Code, tt.expected.status)
			}
		})
	}
}

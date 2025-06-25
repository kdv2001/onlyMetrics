package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

func TestUseCases_UpdateMetric(t *testing.T) {
	t.Parallel()
	type fields struct {
		metricStorage metricStorage
	}
	type args struct {
		in0   context.Context
		value domain.MetricValue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					err: nil,
				},
			},
			args: args{
				in0: context.Background(),
				value: domain.MetricValue{
					Type:         domain.GaugeMetricType,
					Name:         "some",
					CounterValue: 0,
					GaugeValue:   100,
				},
			},
			wantErr: false,
		},
		{
			name: "err gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					err: errors.New("some error"),
				},
			},
			args: args{
				in0: context.Background(),
				value: domain.MetricValue{
					Type:         domain.GaugeMetricType,
					Name:         "some",
					CounterValue: 0,
					GaugeValue:   100,
				},
			},
			wantErr: true,
		},
		{
			name: "success counter metric",
			fields: fields{
				metricStorage: &mockMetric{
					err: nil,
				},
			},
			args: args{
				in0: context.Background(),
				value: domain.MetricValue{
					Type:         domain.GaugeMetricType,
					Name:         "some",
					CounterValue: 0,
					GaugeValue:   100,
				},
			},
			wantErr: false,
		},
		{
			name: "err counter metric",
			fields: fields{
				metricStorage: &mockMetric{
					err: errors.New("some error"),
				},
			},
			args: args{
				in0: context.Background(),
				value: domain.MetricValue{
					Type:         domain.CounterMetricType,
					Name:         "some",
					CounterValue: 0,
					GaugeValue:   100,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc := &UseCases{
				metricStorage: tt.fields.metricStorage,
			}
			if err := uc.UpdateMetric(tt.args.in0, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

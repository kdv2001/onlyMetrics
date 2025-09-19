package metrics

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

func TestUseCases_UpdateMetric(t *testing.T) {
	t.Parallel()
	type fields struct {
		metricStorage MetricStorage
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

func TestUseCases_GetMetric(t *testing.T) {
	t.Parallel()
	type fields struct {
		metricStorage MetricStorage
	}
	type args struct {
		ctx   context.Context
		value domain.MetricType
		name  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.MetricValue
		wantErr bool
	}{
		{
			name: "success get gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					gaugeValue: 100,
					err:        nil,
				},
			},
			args: args{
				ctx:   context.Background(),
				value: domain.GaugeMetricType,
				name:  "gauge",
			},
			want: domain.MetricValue{
				Type:       domain.GaugeMetricType,
				Name:       "gauge",
				GaugeValue: 100,
			},
			wantErr: false,
		},
		{
			name: "err get gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					gaugeValue: 0,
					err:        errors.New("some error"),
				},
			},
			args: args{
				ctx:   context.Background(),
				value: domain.GaugeMetricType,
				name:  "gauge",
			},
			want:    domain.MetricValue{},
			wantErr: true,
		},
		{
			name: "success get gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					counterValue: 100,
					err:          nil,
				},
			},
			args: args{
				ctx:   context.Background(),
				value: domain.CounterMetricType,
				name:  "counter",
			},
			want: domain.MetricValue{
				Type:         domain.CounterMetricType,
				Name:         "counter",
				CounterValue: 100,
			},
			wantErr: false,
		},
		{
			name: "err get gauge metric",
			fields: fields{
				metricStorage: &mockMetric{
					counterValue: 0,
					err:          errors.New("some error"),
				},
			},
			args: args{
				ctx:   context.Background(),
				value: domain.CounterMetricType,
				name:  "counter",
			},
			want:    domain.MetricValue{},
			wantErr: true,
		},
		{
			name:   "err unknown metric",
			fields: fields{},
			args: args{
				ctx:   context.Background(),
				value: "unknown",
				name:  "unknown",
			},
			want:    domain.MetricValue{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uc := &UseCases{
				metricStorage: tt.fields.metricStorage,
			}
			got, err := uc.GetMetric(tt.args.ctx, tt.args.value, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMetric() got = %v, want %v", got, tt.want)
			}
		})
	}
}

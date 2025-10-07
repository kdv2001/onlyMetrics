package agent

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

func Test_recursiveGetMetrics(t *testing.T) {
	t.Parallel()
	type args struct {
		v reflect.Value
	}
	tests := []struct {
		name string
		args args
		want []domain.MetricValue
	}{
		{
			name: "",
			args: args{
				v: reflect.ValueOf(struct {
					metricUint64  uint64
					metricUint32  uint32
					metricFloat64 float64
					metricStruct  struct {
						metricUint64 uint64
					}
				}{
					metricUint64:  1,
					metricUint32:  2,
					metricFloat64: 3,
					metricStruct: struct {
						metricUint64 uint64
					}{
						metricUint64: 4,
					},
				}),
			},
			want: []domain.MetricValue{
				{
					Type:       domain.GaugeMetricType,
					Name:       "metricUint64",
					GaugeValue: 1,
				},
				{
					Type:       domain.GaugeMetricType,
					Name:       "metricUint32",
					GaugeValue: 2,
				},
				{
					Type:       domain.GaugeMetricType,
					Name:       "metricFloat64",
					GaugeValue: 3,
				},
				{
					Type:       domain.GaugeMetricType,
					Name:       "metricUint64",
					GaugeValue: 4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := recursiveGetMetrics(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("recursiveGetMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsUpdater_updateMetrics(t *testing.T) {
	t.Parallel()
	type fields struct {
		stats []domain.MetricValue
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				stats: nil,
			},
			args: args{
				in0: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &MetricsUpdater{
				mu:          sync.RWMutex{},
				stats:       tt.fields.stats,
				pollCount:   atomic.Int64{},
				randomValue: Container[float64]{},
			}
			if err := m.updateMetrics(tt.args.in0); (err != nil) != tt.wantErr {
				t.Errorf("updateMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkRecursiveGetMetrics(b *testing.B) {
	b.Run("recursiveGetMetrics", func(b *testing.B) {
		_ = recursiveGetMetrics(reflect.ValueOf(struct {
			metricUint64  uint64
			metricUint32  uint32
			metricFloat64 float64
			metricStruct  struct {
				metricUint64 uint64
			}
		}{
			metricUint64:  1,
			metricUint32:  2,
			metricFloat64: 3,
			metricStruct: struct {
				metricUint64 uint64
			}{
				metricUint64: 4,
			},
		}))
	})
}

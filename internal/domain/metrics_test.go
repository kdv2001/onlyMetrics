package domain

import "testing"

func TestNewMetricTypeFromString(t *testing.T) {
	t.Parallel()
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    MetricType
		wantErr bool
	}{
		{
			name: "GaugeMetricType",
			args: args{
				s: GaugeMetricType.String(),
			},
			want:    GaugeMetricType,
			wantErr: false,
		},
		{
			name: "CounterMetricType",
			args: args{
				s: CounterMetricType.String(),
			},
			want:    CounterMetricType,
			wantErr: false,
		},
		{
			name: "UnknownMetricType",
			args: args{
				s: "Unknown",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewMetricTypeFromString(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMetricTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewMetricTypeFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

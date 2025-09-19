package domain

import "testing"

func TestNewMetricTypeFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    MetricType
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

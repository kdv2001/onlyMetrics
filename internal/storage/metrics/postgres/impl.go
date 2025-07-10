package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	"github.com/kdv2001/onlyMetrics/internal/pkg/logger"
)

// Storage ...
type Storage struct {
	dbConn *pgx.Conn
}

// NewStorage ...
func NewStorage(conn *pgx.Conn) *Storage {
	return &Storage{
		dbConn: conn,
	}
}

func (s *Storage) Close(ctx context.Context) {
	err := s.dbConn.Close(ctx)
	if err != nil {
		logger.Errorf(ctx, "error close db conn")
	}
}

// Ping ...
func (s *Storage) Ping(ctx context.Context) error {
	return s.dbConn.Ping(ctx)
}

func (s *Storage) UpdateGauge(_ context.Context, value domain.MetricValue) error {
	return nil
}
func (s *Storage) UpdateCounter(_ context.Context, value domain.MetricValue) error {
	return nil
}
func (s *Storage) GetGaugeValue(_ context.Context, name string) (float64, error) {
	return 0, nil
}
func (s *Storage) GetCounterValue(_ context.Context, name string) (int64, error) {
	return 0, nil
}
func (s *Storage) GetAllValues(_ context.Context) ([]domain.MetricValue, error) {
	return nil, nil
}

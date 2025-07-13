package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kdv2001/onlyMetrics/internal/domain"
	"github.com/kdv2001/onlyMetrics/internal/pkg/logger"
)

// Storage ...
type Storage struct {
	dbConn *pgx.Conn
}

// NewStorage ...
func NewStorage(ctx context.Context, conn *pgx.Conn) (*Storage, error) {
	s := &Storage{
		dbConn: conn,
	}

	err := s.createTables(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) createTables(ctx context.Context) error {
	tx, err := s.dbConn.Begin(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, metricValuesTable)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
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

// UpdateGauge ...
func (s *Storage) UpdateGauge(ctx context.Context, value domain.MetricValue) error {
	_, err := s.dbConn.Exec(ctx, `insert into values (metric_name, gauge_value, agent_name, created_at)
values ($1,   $2, $3, $4);`,
		value.Name, value.GaugeValue, "single agent", time.Now().UTC())
	if err != nil {
		return err
	}

	return nil
}

// UpdateCounter ...
func (s *Storage) UpdateCounter(ctx context.Context, value domain.MetricValue) error {
	_, err := s.dbConn.Exec(ctx, `insert into values (metric_name, counter_value, agent_name, created_at) 
values ($1,   $2, $3, $4);`,
		value.Name, value.CounterValue, "single agent", time.Now().UTC())
	if err != nil {
		return err
	}

	return nil
}

// metricValue postgres представление
type metricValue struct {
	ID           sql.NullInt64   `db:"id"`
	MetricName   sql.NullString  `db:"metric_name"`
	GaugeValue   sql.NullFloat64 `db:"gauge_value"`
	CounterValue sql.NullInt64   `db:"counter_value"`
	AgentName    sql.NullString  `db:"agent_name"`
	CreatedAt    sql.NullTime    `db:"created_at"`
}

// GetGaugeValue ...
func (s *Storage) GetGaugeValue(ctx context.Context, name string) (float64, error) {
	res := new(metricValue)
	err := s.dbConn.QueryRow(ctx,
		`select * from values where metric_name = $1 order by created_at desc limit 1;`,
		name).Scan(&res.ID, &res.MetricName, &res.GaugeValue, &res.CounterValue, &res.AgentName, &res.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, domain.ErrNotFound
		}

		return 0, err
	}

	return res.GaugeValue.Float64, nil
}
func (s *Storage) GetCounterValue(ctx context.Context, name string) (int64, error) {
	res := sql.NullInt64{}
	err := s.dbConn.QueryRow(ctx,
		`select sum(counter_value)  as counter_value from values where metric_name = $1;`,
		name).Scan(&res)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return 0, domain.ErrNotFound
		}

		return 0, err
	}

	return res.Int64, nil
}

// GetAllValues ...
func (s *Storage) GetAllValues(ctx context.Context) ([]domain.MetricValue, error) {
	rowsGauge, err := s.dbConn.Query(ctx, `
select metric_name, gauge_value
from values
where (metric_name, created_at) in
      (select metric_name, max(created_at)
       from values
       group by values.metric_name) and gauge_value notnull;
`)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// 29 максимальное кол-во уникальных метрик
	res := make([]domain.MetricValue, 0, 29)
	for rowsGauge.Next() {
		mv := new(metricValue)
		err = rowsGauge.Scan(&mv.MetricName, &mv.GaugeValue)
		if err != nil {
			return nil, err
		}

		res = append(res, domain.MetricValue{
			Type:       domain.GaugeMetricType,
			Name:       mv.MetricName.String,
			GaugeValue: mv.GaugeValue.Float64,
		})
	}
	rowsGauge.Close()

	rowsCounter, err := s.dbConn.Query(ctx, `
select metric_name, sum(counter_value) as counter_value
from values where counter_value notnull
group by metric_name;
`)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	for rowsCounter.Next() {
		mv := new(metricValue)
		err = rowsCounter.Scan(&mv.MetricName, &mv.CounterValue)
		if err != nil {
			return nil, err
		}

		res = append(res, domain.MetricValue{
			Type:         domain.CounterMetricType,
			Name:         mv.MetricName.String,
			CounterValue: mv.CounterValue.Int64,
		})
	}
	rowsCounter.Close()

	if len(res) == 0 {
		return nil, domain.ErrNotFound
	}

	return res, nil
}

// UpdateMetrics ...
func (s *Storage) UpdateMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	for _, metric := range metrics {
		switch metric.Type {
		case domain.GaugeMetricType:
			err := s.UpdateGauge(ctx, metric)
			if err != nil {
				return err
			}
		case domain.CounterMetricType:
			err := s.UpdateCounter(ctx, metric)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

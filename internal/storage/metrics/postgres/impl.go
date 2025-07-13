package postgres

import (
	"context"
	"database/sql"
	"errors"

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
	_, err := s.dbConn.Exec(ctx, metricValuesTable)

	return err
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

func (s *Storage) UpdateGauge(ctx context.Context, value domain.MetricValue) error {
	_, err := s.dbConn.Exec(ctx, `insert into values (metric_name, gauge_value, agent_name) values ($1,   $2, $3);`,
		value.Name, value.GaugeValue, "single agent")
	if err != nil {
		return err
	}

	return nil
}
func (s *Storage) UpdateCounter(ctx context.Context, value domain.MetricValue) error {
	_, err := s.dbConn.Exec(ctx, `insert into values (metric_name, counter_value, agent_name) values ($1,   $2, $3);`,
		value.Name, value.CounterValue, "single agent")
	if err != nil {
		return err
	}

	return nil
}

type metricValue struct {
	ID           sql.NullInt64   `db:"id"`
	MetricName   sql.NullString  `db:"metric_name"`
	GaugeValue   sql.NullFloat64 `db:"gauge_value"`
	CounterValue sql.NullInt64   `db:"counter_value"`
	AgentName    sql.NullString  `db:"agent_name"`
	CreatedAt    sql.NullTime    `db:"created_at"`
}

func (s *Storage) GetGaugeValue(ctx context.Context, name string) (float64, error) {
	res, err := s.getValue(ctx, name)
	if err != nil {
		return 0, err
	}

	return res.GaugeValue.Float64, nil
}
func (s *Storage) GetCounterValue(ctx context.Context, name string) (int64, error) {
	res, err := s.getValue(ctx, name)
	if err != nil {
		return 0, err
	}

	return res.CounterValue.Int64, nil
}

func (s *Storage) getValue(ctx context.Context, name string) (*metricValue, error) {
	res := new(metricValue)
	err := s.dbConn.QueryRow(ctx,
		`select * from values where metric_name = $1 order by created_at desc limit 1;`,
		name).Scan(res)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	return res, nil
}

func (s *Storage) GetAllValues(ctx context.Context) ([]domain.MetricValue, error) {
	rows, err := s.dbConn.Query(ctx, `
select *
from values
where (metric_name, created_at) in
      (select metric_name, max(created_at)
       from values
       group by values.metric_name) ;
`)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, domain.ErrNotFound
		}

		return nil, err
	}

	// 29 максимальное кол-во уникальных метрик
	res := make([]domain.MetricValue, 0, 29)
	for rows.Next() {
		mv := new(metricValue)

		err = rows.Scan(&mv.ID, &mv.MetricName, &mv.GaugeValue,
			&mv.CounterValue, &mv.AgentName, &mv.CreatedAt)
		if err != nil {
			return nil, err
		}

		mType := domain.GaugeMetricType
		switch {
		case mv.CounterValue.Valid:
			mType = domain.CounterMetricType
		}

		res = append(res, domain.MetricValue{
			Type:         mType,
			Name:         mv.MetricName.String,
			CounterValue: mv.CounterValue.Int64,
			GaugeValue:   mv.GaugeValue.Float64,
		})
	}

	return res, nil
}

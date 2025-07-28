package postgres

const metricValuesTable = `
		create table if not exists values (	
    	id            bigint GENERATED ALWAYS AS IDENTITY primary key,
    	metric_name   varchar                     NOT NULL,
    	gauge_value   double precision,
    	counter_value integer,
    	agent_name    varchar                     NOT NULL,
    	created_at    timestamp WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
	);`

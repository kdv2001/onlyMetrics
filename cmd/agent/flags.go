package main

import (
	"dario.cat/mergo"

	"github.com/kdv2001/onlyMetrics/pkg/config"
)

type flags struct {
	ServerAddr             *config.URL      `env:"ADDRESS" flag:"a;;metric server address" default:"localhost:8080" json:"address"`
	ReportInterval         *config.Duration `env:"REPORT_INTERVAL" flag:"r;;report interval Duration" default:"10" json:"report_interval"`
	PollInterval           *config.Duration `env:"POLL_INTERVAL" flag:"p;;report poll Duration" default:"2" json:"poll_interval"`
	CryptKey               string           `env:"KEY" flag:"k;;crypt request key" json:"key"`
	MaxGoroutineNum        int64            `env:"RATE_LIMIT" flag:"l;;max goroutine sender num" default:"0" json:"max_goroutine_num"`
	SymmetricEncryptionKey string           `env:"CRYPTO_KEY" flag:"crypto-key;;path to CERTIFICATE.pem and PRIVATE_KEY.pem" json:"symmetric_encryption_key"`
	Config                 string           `env:"CONFIG" flag:"config;;config file path" json:"config"`
}

func makeFlags() *flags {
	return &flags{
		ServerAddr:     new(config.URL),
		ReportInterval: new(config.Duration),
		PollInterval:   new(config.Duration),
	}
}

func initFlags() (*flags, error) {
	resultFlags := makeFlags()

	// парсим аргументы программы
	env := makeFlags()
	if err := config.UnmarshalEnv(env); err != nil {
		return nil, err
	}

	if err := mergo.Merge(resultFlags, env); err != nil {
		return nil, err
	}

	// парсим переменные окружения
	parsedFlags := makeFlags()
	if err := config.UnmarshalFlags(parsedFlags); err != nil {
		return nil, err
	}

	if err := mergo.Merge(resultFlags, parsedFlags); err != nil {
		return nil, err
	}

	// парсим значения из конфига
	configFilePath := resultFlags.Config

	configFileFlags := makeFlags()
	if configFilePath != "" {
		err := config.UnmarshalJSONFile(configFileFlags, configFilePath)
		if err != nil {
			return nil, err
		}
	}

	if err := mergo.Merge(resultFlags, configFileFlags); err != nil {
		return nil, err
	}

	// устанавливаем дефолт значения
	if err := config.SetDefaultValues(resultFlags); err != nil {
		return nil, err
	}

	return resultFlags, nil
}

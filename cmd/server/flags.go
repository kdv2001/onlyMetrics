package main

import (
	"dario.cat/mergo"

	"github.com/kdv2001/onlyMetrics/pkg/config"
)

type flags struct {
	ServerAddr             string           `default:":8080" env:"ADDRESS" flag:"a;;The address to bind the server to" json:"address"`
	StoreInterval          *config.Duration `default:"300" env:"STORE_INTERVAL" flag:"i;;The interval to save data to file" json:"store_interval"`
	FileStoragePath        string           `default:"data.txt" env:"FILE_STORAGE_PATH" flag:"f;;The address to metric file" json:"store_file"`
	RestoreData            bool             `env:"RESTORE" flag:"r;;The flag to restore data from file" json:"restore"`
	PostgresDSN            string           `env:"DATABASE_DSN" flag:"d;;The flag to Postgres DSN" json:"database_dsn"`
	CryptKey               string           `env:"KEY" flag:"k;;crypt request key" json:"crypto_key"`
	SymmetricEncryptionKey string           `env:"CRYPTO_KEY" flag:"crypto-key;;symmetric encryption key" json:"symmetric_encryption_key"`
	ConfigFilePath         string           `env:"CONFIG" flag:"config;;config file path" `
}

func makeFlags() *flags {
	return &flags{
		StoreInterval: new(config.Duration),
	}
}

func initFlags() (*flags, error) {
	resultFlags := makeFlags()

	// парсим переменные окружения
	parsedFlags := makeFlags()
	if err := config.UnmarshalFlags(parsedFlags); err != nil {
		return nil, err
	}

	if err := mergo.Merge(resultFlags, parsedFlags); err != nil {
		return nil, err
	}

	// парсим аргументы программы
	env := makeFlags()
	if err := config.UnmarshalEnv(env); err != nil {
		return nil, err
	}

	if err := mergo.Merge(resultFlags, env); err != nil {
		return nil, err
	}

	// парсим значения из конфига
	configFilePath := resultFlags.ConfigFilePath

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

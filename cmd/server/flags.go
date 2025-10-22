package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type flags struct {
	ServerAddr             string   `json:"address"`
	StoreInterval          duration `json:"store_interval"`
	FileStoragePath        string   `json:"store_file"`
	RestoreData            bool     `json:"restore"`
	PostgresDSN            string   `json:"database_dsn"`
	CryptKey               string   `json:"crypto_key"`
	SymmetricEncryptionKey string   `json:"symmetric_encryption_key"`
}

type duration time.Duration

func (d *duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func (d *duration) asTimeDuration() time.Duration {
	if d == nil {
		return time.Duration(0)
	}
	return time.Duration(*d)
}

func initFlags() (flags, error) {
	serverAddr := flag.String("a", ":8080", "The address to bind the server to")
	storeInterval := flag.Int64("i", 300, "The interval to save data to file")
	fileStoragePath := flag.String("f", "data.txt", "The address to metric file")
	restore := flag.Bool("r", false, "The flag to restore data from file")
	postgresDSN := flag.String("d", "", "The flag to Postgres DSN")
	cryptKey := flag.String("k", "", "crypt request key")
	symmetricEncryptionKey := flag.String("crypto-key", "", "symmetric encryption key")
	configFilePath := flag.String("config", "", "config file path")

	flag.Parse()

	if value := os.Getenv("ADDRESS"); value != "" {
		serverAddr = &value
	}

	storeIntervalKey := "STORE_INTERVAL"
	if value, exist := os.LookupEnv(storeIntervalKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", storeIntervalKey)
		}

		val, err := parseIntervalValue(value)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", storeIntervalKey, err)
		}
		storeInterval = &val
	}

	fileStoragePathKey := "FILE_STORAGE_PATH"
	if value, exist := os.LookupEnv(fileStoragePathKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", fileStoragePathKey)
		}

		fileStoragePath = &value
	}

	restoreKey := "RESTORE"
	if value, exist := os.LookupEnv(restoreKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", restoreKey)
		}

		res, err := strconv.ParseBool(value)
		if err != nil {
			return flags{}, fmt.Errorf("can not parse %s environment variable: %w", restoreKey, err)
		}

		restore = &res
	}

	dataBaseDSNKey := "DATABASE_DSN"
	if value, exist := os.LookupEnv(dataBaseDSNKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", dataBaseDSNKey)
		}

		postgresDSN = &value
	}

	cryptKeyKey := "KEY"
	if value, exist := os.LookupEnv(cryptKeyKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", cryptKeyKey)
		}

		cryptKey = &value
	}

	symmetricEncryptionKeyKey := "CRYPTO_KEY"
	if value, exist := os.LookupEnv(symmetricEncryptionKeyKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", symmetricEncryptionKeyKey)
		}

		symmetricEncryptionKey = &value
	}

	configFilePathKey := "CONFIG"
	if value, exist := os.LookupEnv(configFilePathKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", configFilePathKey)
		}

		configFilePath = &value
	}

	configFileFlags := flags{}
	if *configFilePath != "" {
		var err error
		configFileFlags, err = getFileConfig(*configFilePath)
		if err != nil {
			return flags{}, err
		}
	}

	// если есть пустые значения и задан конфиг, заполняем значениями из конфига
	storeIntervalDur := duration(time.Duration(*storeInterval) * time.Second)
	return flags{
		ServerAddr: opIf(*serverAddr != "", *serverAddr, configFileFlags.ServerAddr),
		StoreInterval: opIf(
			storeIntervalDur != duration(time.Duration(0)),
			storeIntervalDur,
			configFileFlags.StoreInterval),
		FileStoragePath:        opIf(*fileStoragePath != "", *fileStoragePath, configFileFlags.FileStoragePath),
		RestoreData:            opIf(*restore, *restore, configFileFlags.RestoreData),
		PostgresDSN:            opIf(*postgresDSN != "", *postgresDSN, configFileFlags.PostgresDSN),
		CryptKey:               opIf(*cryptKey != "", *cryptKey, configFileFlags.CryptKey),
		SymmetricEncryptionKey: opIf(*symmetricEncryptionKey != "", *symmetricEncryptionKey, configFileFlags.SymmetricEncryptionKey),
	}, nil
}

func parseIntervalValue(value string) (int64, error) {
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %w", value, err)
	}
	if intValue <= 0 {
		return 0, fmt.Errorf("invalid POLL_INTERVAL: %s", value)
	}

	return intValue, nil
}

func getFileConfig(path string) (flags, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return flags{}, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	f := flags{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return flags{}, fmt.Errorf("failed to unmarshal %s: %w", path, err)
	}

	return f, nil
}

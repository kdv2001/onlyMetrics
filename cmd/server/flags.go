package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type flags struct {
	serverAddr             string
	storeInterval          time.Duration
	fileStoragePath        string
	restoreData            bool
	postgresDSN            string
	cryptKey               string
	symmetricEncryptionKey string
}

func initFlags() (flags, error) {
	serverAddr := flag.String("a", ":8080", "The address to bind the server to")
	storeInterval := flag.Int64("i", 300, "The interval to save data to file")
	fileStoragePath := flag.String("f", "data.txt", "The address to metric file")
	restore := flag.Bool("r", false, "The flag to restore data from file")
	postgresDSN := flag.String("d", "", "The flag to Postgres DSN")
	cryptKey := flag.String("k", "", "crypt request key")
	symmetricEncryptionKey := flag.String("crypto-key", "", "symmetric encryption key")

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

	return flags{
		serverAddr:             *serverAddr,
		storeInterval:          time.Duration(*storeInterval) * time.Second,
		fileStoragePath:        *fileStoragePath,
		restoreData:            *restore,
		postgresDSN:            *postgresDSN,
		cryptKey:               *cryptKey,
		symmetricEncryptionKey: *symmetricEncryptionKey,
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

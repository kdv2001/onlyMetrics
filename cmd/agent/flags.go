package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type flags struct {
	ServerAddr             url.URL  `json:"address"`
	ReportInterval         duration `json:"report_interval"`
	PollInterval           duration `json:"poll_interval"`
	CryptKey               string   `json:"crypto_key"`
	MaxGoroutineNum        int64    `json:"max_goroutine_num"`
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

var defaultServerAddr = url.URL{
	Host: "localhost:8080",
}

func initFlags() (flags, error) {
	serverAddr := url.URL{}
	flag.Func("a", "metric server address", func(address string) error {
		if address == "" {
			return nil
		}

		serverAddr = url.URL{
			Host: address,
		}

		return nil
	})
	reportInterval := flag.Int64("r", 10, "report interval duration")
	pollInterval := flag.Int64("p", 2, "report poll duration")
	cryptKey := flag.String("k", "", "crypt request key")
	maxGoroutineNum := flag.Int64("l", 0, "max goroutine sender num")
	symmetricEncryptionKey := flag.String("crypto-key", "", "path to CERTIFICATE.pem and PRIVATE_KEY.pem")
	configFilePath := flag.String("config", "", "config file path")

	flag.Parse()

	if value, exist := os.LookupEnv("ADDRESS"); exist {
		if value == "" {
			return flags{}, fmt.Errorf("ADDRESS environment variable not set")
		}

		serverAddr = url.URL{
			Host: value,
		}
	}

	reportIntervalKey := "REPORT_INTERVAL"
	if value, exist := os.LookupEnv(reportIntervalKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", reportIntervalKey)
		}

		val, err := parseIntervalValue(value)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", reportIntervalKey, err)
		}
		reportInterval = &val
	}

	poolIntervalKey := "POLL_INTERVAL"
	if value, exist := os.LookupEnv(poolIntervalKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", poolIntervalKey)
		}

		val, err := parseIntervalValue(value)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", poolIntervalKey, err)
		}
		pollInterval = &val
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

	maxGoroutineNumKey := "RATE_LIMIT"
	if value, exist := os.LookupEnv(maxGoroutineNumKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", maxGoroutineNumKey)
		}

		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", maxGoroutineNumKey, err)
		}

		maxGoroutineNum = &intValue
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

	finalAddress := opIf(serverAddr.String() != "", serverAddr, configFileFlags.ServerAddr)
	if finalAddress.String() == "" {
		finalAddress = defaultServerAddr
	}

	reportIntervalDur := duration(time.Duration(*reportInterval) * time.Second)
	pollIntervalDur := duration(time.Duration(*pollInterval) * time.Second)
	return flags{
		ServerAddr: finalAddress,
		ReportInterval: opIf(
			reportIntervalDur != duration(time.Duration(0)),
			reportIntervalDur,
			configFileFlags.ReportInterval),
		PollInterval: opIf(
			pollIntervalDur != duration(time.Duration(0)),
			pollIntervalDur,
			configFileFlags.PollInterval),
		CryptKey:               opIf(*cryptKey != "", *cryptKey, configFileFlags.CryptKey),
		MaxGoroutineNum:        opIf(*maxGoroutineNum != 0, *maxGoroutineNum, configFileFlags.MaxGoroutineNum),
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

type flagsJSON struct {
	ServerAddr             string   `json:"address"`
	ReportInterval         duration `json:"report_interval"`
	PollInterval           duration `json:"poll_interval"`
	CryptKey               string   `json:"crypto_key"`
	MaxGoroutineNum        int64    `json:"max_goroutine_num"`
	SymmetricEncryptionKey string   `json:"symmetric_encryption_key"`
}

func getFileConfig(path string) (flags, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return flags{}, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	f := flagsJSON{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return flags{}, fmt.Errorf("failed to unmarshal %s: %w", path, err)
	}

	return flags{
		ServerAddr: url.URL{
			Host: f.ServerAddr,
		},
		ReportInterval:         f.ReportInterval,
		PollInterval:           f.PollInterval,
		CryptKey:               f.CryptKey,
		MaxGoroutineNum:        f.MaxGoroutineNum,
		SymmetricEncryptionKey: f.SymmetricEncryptionKey,
	}, nil
}

//go:build !integration

package app

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	logger "github.com/abredov/golang_fin/internal/logger"
	redisclient "github.com/abredov/golang_fin/internal/storage/redis"
	storageData "github.com/abredov/golang_fin/internal/storage/storageData"
	storageSQLMock "github.com/abredov/golang_fin/internal/storage/storageSQLMock"
	"github.com/stretchr/testify/require"
)

const localhost string = "127.0.0.1"

type ConfigTest struct{}

func (config *ConfigTest) Init(_ string) error {
	return nil
}

func (config *ConfigTest) GetServerURL() string {
	return "127.0.0.1:4000"
}

func (config *ConfigTest) GetAddress() string {
	return localhost
}

func (config *ConfigTest) GetPort() string {
	return "4000"
}

func (config *ConfigTest) GetServerShutdownTimeout() time.Duration {
	return 5 * time.Second
}

func (config *ConfigTest) GetDBName() string {
	return "OTUSAntibf"
}

func (config *ConfigTest) GetDBUser() string {
	return "postgres"
}

func (config *ConfigTest) GetDBPassword() string {
	return "SecurePass"
}

func (config *ConfigTest) GetDBMaxConnectionLifetime() time.Duration {
	return 5 * time.Second
}

func (config *ConfigTest) GetDBMaxIdleConnections() int {
	return 20
}

func (config *ConfigTest) GetDBMaxOpenConnections() int {
	return 20
}

func (config *ConfigTest) GetDBTimeout() time.Duration {
	return 5 * time.Second
}

func (config *ConfigTest) GetDBAddress() string {
	return localhost
}

func (config *ConfigTest) GetDBPort() string {
	return "5432"
}

func (config *ConfigTest) GetRedisAddress() string {
	return localhost
}

func (config *ConfigTest) GetRedisPort() string {
	return "6379"
}

func (config *ConfigTest) GetLimitLogin() int {
	return 10
}

func (config *ConfigTest) GetLimitPassword() int {
	return 100
}

func (config *ConfigTest) GetLimitIP() int {
	return 20
}

func (config *ConfigTest) GetLimitTimeCheck() time.Duration {
	return 1 * time.Minute
}

func initAppWMock(t *testing.T) *App {
	t.Helper()
	logg, _ := logger.New("debug")
	config := ConfigTest{}
	storage := storageSQLMock.New()
	ctxStorage, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
	defer cancel()
	err := storage.Init(ctxStorage, logg, &config)
	require.NoError(t, err)
	redis := redisclient.New()
	err = redis.InitAsMock(ctxStorage, logg)
	require.NoError(t, err)
	antibf := New(logg, storage, redis, &config)
	return antibf
}

func TestSimpleRequestValidator(t *testing.T) {
	t.Parallel()
	t.Run("PositiveRequestValidator", func(t *testing.T) {
		t.Parallel()
		_, err := SimpleRequestValidate("user0", "root", "192.168.64.12")
		require.NoError(t, err)
	})
	t.Run("NegativeErrVoidLogin", func(t *testing.T) {
		t.Parallel()
		_, err := SimpleRequestValidate("", "root", "192.168.64.12")
		require.Truef(t, errors.Is(err, ErrVoidLogin), "actual error %q", err)
	})
	t.Run("NegativeErrVoidPassword", func(t *testing.T) {
		t.Parallel()
		_, err := SimpleRequestValidate("user0", "", "192.168.64.12")
		require.Truef(t, errors.Is(err, ErrVoidPassword), "actual error %q", err)
	})
	t.Run("NegativeErrVoidIP", func(t *testing.T) {
		t.Parallel()
		_, err := SimpleRequestValidate("user0", "root", "")
		require.Truef(t, errors.Is(err, ErrBadIP), "actual error %q", err)
	})
}

func TestSimpleIPDataValidator(t *testing.T) {
	t.Parallel()
	t.Run("PositiveIPDataValidator", func(t *testing.T) {
		t.Parallel()
		testData := storageData.StorageIPData{
			IP:   "192.168.64.0",
			Mask: 25,
		}
		err := SimpleIPDataValidate(testData, false)
		require.NoError(t, err)
	})
	t.Run("PositiveIPDataValidatorALL", func(t *testing.T) {
		t.Parallel()
		testData := storageData.StorageIPData{
			IP:   "ALL",
			Mask: 0,
		}
		err := SimpleIPDataValidate(testData, true)
		require.NoError(t, err)
	})
	t.Run("NegativeErrVoidIP", func(t *testing.T) {
		t.Parallel()
		testData := storageData.StorageIPData{
			IP:   "",
			Mask: 25,
		}
		err := SimpleIPDataValidate(testData, false)
		require.Truef(t, errors.Is(err, ErrBadIP), "actual error %q", err)
	})
	t.Run("NegativeErrVoidMask", func(t *testing.T) {
		t.Parallel()
		testData := storageData.StorageIPData{
			IP:   "192.168.64.0",
			Mask: 0,
		}
		err := SimpleIPDataValidate(testData, false)
		require.Truef(t, errors.Is(err, ErrVoidMask), "actual error %q", err)
	})
}

func TestAppNegativeAddIPCrossAdding(t *testing.T) {
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newData := storageData.StorageIPData{
		IP:   "192.168.64.0",
		Mask: 25,
	}
	_, err = app.IPAddToList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	ok, err := app.IPIsInList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in whitelist", ok)
	_, err = app.IPAddToList(context.Background(), "blacklist", newData)
	require.Truef(t, errors.Is(err, ErrIPinWL), "actual error %q", err)
	newData = storageData.StorageIPData{
		IP:   "10.0.0.0",
		Mask: 8,
	}
	_, err = app.IPAddToList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	ok, err = app.IPIsInList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in blacklist", ok)
	_, err = app.IPAddToList(context.Background(), "whitelist", newData)
	require.Truef(t, errors.Is(err, ErrIPinBL), "actual error %q", err)
}

// WHITELIST

func TestAppPositiveAddIPToWhiteListAndIsIPInWhiteList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newData := storageData.StorageIPData{
		IP:   "192.168.64.0",
		Mask: 25,
	}
	_, err = app.IPAddToList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	ok, err := app.IPIsInList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in whitelist", ok)
}

func TestAppPositiveRemoveIPInWhiteListAndIsIPInWhiteList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newData := storageData.StorageIPData{
		IP:   "192.168.64.0",
		Mask: 25,
	}
	_, err = app.IPAddToList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	ok, err := app.IPIsInList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in whitelist", ok)
	err = app.IPRemoveFromList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	ok, err = app.IPIsInList(context.Background(), "whitelist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == false, "IP is still in whitelist after removing", ok)
}

func TestAppPositiveGetAllIPInWhiteList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newDataSl := make([]storageData.StorageIPData, 2)
	newDataSl[0] = storageData.StorageIPData{
		ID:   0,
		IP:   "192.168.64.0",
		Mask: 25,
	}
	newDataSl[1] = storageData.StorageIPData{
		ID:   1,
		IP:   "10.0.0.0",
		Mask: 8,
	}
	for _, curData := range newDataSl {
		_, err = app.IPAddToList(context.Background(), "whitelist", curData)
		require.NoError(t, err)
	}

	controlDataSl, err := app.IPGetAllFromList(context.Background(), "whitelist")
	require.NoError(t, err)
	require.Equal(t, newDataSl, controlDataSl)
}

// BLACKLIST

func TestAppPositiveAddIPToBlackListAndIsIPInBlackList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newData := storageData.StorageIPData{
		IP:   "192.168.64.0",
		Mask: 25,
	}
	_, err = app.IPAddToList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	ok, err := app.IPIsInList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in blacklist", ok)
}

func TestAppPositiveRemoveIPInBlackListAndIsIPInBlackList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newData := storageData.StorageIPData{
		IP:   "192.168.64.0",
		Mask: 25,
	}
	_, err = app.IPAddToList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	ok, err := app.IPIsInList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == true, "IP is not in blacklist", ok)
	err = app.IPRemoveFromList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	ok, err = app.IPIsInList(context.Background(), "blacklist", newData)
	require.NoError(t, err)
	require.Truef(t, ok == false, "IP is still in blacklist after removing", ok)
}

func TestAppPositiveGetAllIPInBlackList(t *testing.T) { //nolint: dupl, nolintlint
	app := initAppWMock(t)
	config := ConfigTest{}
	err := app.InitStorage(context.Background(), &config)
	require.NoError(t, err)
	defer app.CloseStorage(context.Background())
	newDataSl := make([]storageData.StorageIPData, 2)
	newDataSl[0] = storageData.StorageIPData{
		ID:   0,
		IP:   "192.168.64.0",
		Mask: 25,
	}
	newDataSl[1] = storageData.StorageIPData{
		ID:   1,
		IP:   "10.0.0.0",
		Mask: 8,
	}
	for _, curData := range newDataSl {
		_, err = app.IPAddToList(context.Background(), "blacklist", curData)
		require.NoError(t, err)
	}

	controlDataSl, err := app.IPGetAllFromList(context.Background(), "blacklist")
	require.NoError(t, err)
	require.Equal(t, newDataSl, controlDataSl)
}

// REQUEST AUTH

func TestRequestAuth(t *testing.T) {
	t.Parallel()
	t.Run("PositiveRequestAuth", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, true, ok)
		require.Equal(t, "Check successful", message)
	})

	t.Run("PositiveRequestAuthInWhiteList", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		newData := storageData.StorageIPData{
			IP:   "192.168.64.0",
			Mask: 24,
		}
		_, err := app.IPAddToList(context.Background(), "whitelist", newData)
		require.NoError(t, err)
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, true, ok)
		require.Equal(t, "IP is in whitelist", message)
	})
	t.Run("PositiveRequestAuthInBlackList", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		newData := storageData.StorageIPData{
			IP:   "192.168.64.0",
			Mask: 24,
		}
		_, err := app.IPAddToList(context.Background(), "blacklist", newData)
		require.NoError(t, err)
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, "IP is in blacklist", message)
	})
	t.Run("PositiveRequestAuthRateLimitByTag", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		for i := 0; i < 10; i++ {
			ok, message, err := app.CheckRequest(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, true, ok)
			require.Equal(t, "Check successful", message)
		}
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, "Limited by login rate", message)
	})
	t.Run("PositiveRequestAuthRateLimitByTagAndClearLoginBucket", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		for i := 0; i < 10; i++ {
			ok, message, err := app.CheckRequest(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, true, ok)
			require.Equal(t, "Check successful", message)
		}
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, "Limited by login rate", message)
		err = app.ClearBucketForLogin(context.Background(), "user")
		require.NoError(t, err)
		ok, message, err = app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, true, ok)
		require.Equal(t, "Check successful", message)
	})
	t.Run("PositiveRequestAuthRateLimitByTagAndClearLoginBucket", func(t *testing.T) {
		t.Parallel()
		app := initAppWMock(t)
		req := storageData.RequestAuth{
			Login:    "user",
			Password: "PassGood",
			IP:       "192.168.64.32",
		}
		for i := 0; i < 20; i++ {
			req := storageData.RequestAuth{
				Login:    strconv.Itoa(i),
				Password: "PassGood",
				IP:       "192.168.64.32",
			}
			ok, message, err := app.CheckRequest(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, true, ok)
			require.Equal(t, "Check successful", message)
		}
		ok, message, err := app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, false, ok)
		require.Equal(t, "Limited by IP rate", message)
		err = app.ClearBucketForIP(context.Background(), "192.168.64.32")
		require.NoError(t, err)
		ok, message, err = app.CheckRequest(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, true, ok)
		require.Equal(t, "Check successful", message)
	})
}

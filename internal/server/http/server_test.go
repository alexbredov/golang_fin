//go:build !integration

package httpinternal

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abredov/golang_fin/internal/app"
	logger "github.com/abredov/golang_fin/internal/logger"
	RedisStorage "github.com/abredov/golang_fin/internal/storage/redis"
	storageData "github.com/abredov/golang_fin/internal/storage/storageData"
	storageSQLMock "github.com/abredov/golang_fin/internal/storage/storageSQLMock"
	"github.com/stretchr/testify/require"
)

const (
	correctOutJSONAnswer string = `{"Text":"Everything is OK","Code":0}`
	localhost            string = "127.0.0.1"
)

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
	return 1000
}

func (config *ConfigTest) GetLimitTimeCheck() time.Duration {
	return 1 * time.Minute
}

func TestServer_RESTWhiteList(t *testing.T) { //nolint:dupl
	t.Parallel()
	t.Run("IPAddToWL", func(t *testing.T) {
		t.Parallel()
		data := bytes.NewBufferString(`{
			"IP": "192.168.64.0",
			"Mask":8
		}`)
		server := createServer(t)
		r := httptest.NewRequest("POST", "/whitelist/", data)
		w := httptest.NewRecorder()
		server.RESTWhiteList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPRemoveFromWL", func(t *testing.T) {
		t.Parallel()
		ctrldataTestIP := "192.168.64.0"
		newData := storageData.StorageIPData{
			IP:   ctrldataTestIP,
			Mask: 8,
		}
		data := bytes.NewBufferString(`{
			"IP": "192.168.64.0",
			"Mask":8
		}`)
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "whitelist", newData)
		require.NoError(t, err)
		ctrldataSlice, err := server.app.IPGetAllFromList(context.Background(), "whitelist")
		require.NoError(t, err)
		flag := false
		for _, currentctrldata := range ctrldataSlice {
			if currentctrldata.IP == ctrldataTestIP && currentctrldata.Mask == 8 {
				flag = true
				break
			}
		}
		require.Equal(t, flag, true)
		r := httptest.NewRequest("DELETE", "/whitelist/", data)
		w := httptest.NewRecorder()
		server.RESTWhiteList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPIsInWhiteList", func(t *testing.T) {
		t.Parallel()
		newData := storageData.StorageIPData{
			IP:   "192.168.64.12",
			Mask: 8,
		}
		data := bytes.NewBufferString(`{
			"IP":"192.168.64.12",
			"Mask":8
		}`)
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "whitelist", newData)
		require.NoError(t, err)
		r := httptest.NewRequest("GET", "/whitelist/", data)
		w := httptest.NewRecorder()
		server.RESTWhiteList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := `{"IPList":[],"Message":{"Text":"Yes","Code":0}}`
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPGetAllFromWL", func(t *testing.T) {
		t.Parallel()
		data := bytes.NewBufferString(`{
			"IP":"ALL",
			"Mask":0
		}`)
		newData := storageData.StorageIPData{
			IP:   "192.168.64.12",
			Mask: 8,
		}
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "whitelist", newData)
		require.NoError(t, err)
		newData = storageData.StorageIPData{
			IP:   "192.168.0.5",
			Mask: 24,
		}
		_, err = server.app.IPAddToList(context.Background(), "whitelist", newData)
		require.NoError(t, err)
		r := httptest.NewRequest("GET", "/whitelist/", data)
		w := httptest.NewRecorder()
		server.RESTWhiteList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := `{"IPList":[{"IP":"192.168.64.12","Mask":8,"ID":0},` +
			`{"IP":"192.168.0.5","Mask":24,"ID":1}],"Message":{"Text":"Everything is OK","Code":0}}`
		require.Equal(t, respExpect, string(respBody))
	})
}

func TestServer_RESTBlackList(t *testing.T) { //nolint:dupl
	t.Parallel()
	t.Run("IPAddToBL", func(t *testing.T) {
		t.Parallel()
		data := bytes.NewBufferString(`{
			"IP": "192.168.64.0",
			"Mask":8
		}`)
		server := createServer(t)
		r := httptest.NewRequest("POST", "/blacklist/", data)
		w := httptest.NewRecorder()
		server.RESTBlackList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPRemoveFromBL", func(t *testing.T) {
		t.Parallel()
		ctrldataTestIP := "192.168.64.0"
		newData := storageData.StorageIPData{
			IP:   ctrldataTestIP,
			Mask: 8,
		}
		data := bytes.NewBufferString(`{
			"IP": "192.168.64.0",
			"Mask":8
		}`)
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "blacklist", newData)
		require.NoError(t, err)
		ctrldataSlice, err := server.app.IPGetAllFromList(context.Background(), "blacklist")
		require.NoError(t, err)
		flag := false
		for _, currentctrldata := range ctrldataSlice {
			if currentctrldata.IP == ctrldataTestIP && currentctrldata.Mask == 8 {
				flag = true
				break
			}
		}
		require.Equal(t, flag, true)
		r := httptest.NewRequest("DELETE", "/blacklist/", data)
		w := httptest.NewRecorder()
		server.RESTBlackList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPIsInBlackList", func(t *testing.T) {
		t.Parallel()
		newData := storageData.StorageIPData{
			IP:   "192.168.64.12",
			Mask: 8,
		}
		data := bytes.NewBufferString(`{
			"IP":"192.168.64.12",
			"Mask":8
		}`)
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "blacklist", newData)
		require.NoError(t, err)
		r := httptest.NewRequest("GET", "/blacklist/", data)
		w := httptest.NewRecorder()
		server.RESTBlackList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := `{"IPList":[],"Message":{"Text":"Yes","Code":0}}`
		require.Equal(t, respExpect, string(respBody))
	})
	t.Run("IPGetAllFromBL", func(t *testing.T) {
		t.Parallel()
		data := bytes.NewBufferString(`{
			"IP":"ALL",
			"Mask":0
		}`)
		newData := storageData.StorageIPData{
			IP:   "192.168.64.12",
			Mask: 8,
		}
		server := createServer(t)
		_, err := server.app.IPAddToList(context.Background(), "blacklist", newData)
		require.NoError(t, err)
		newData = storageData.StorageIPData{
			IP:   "192.168.0.5",
			Mask: 24,
		}
		_, err = server.app.IPAddToList(context.Background(), "blacklist", newData)
		require.NoError(t, err)
		r := httptest.NewRequest("GET", "/blacklist/", data)
		w := httptest.NewRecorder()
		server.RESTBlackList(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := `{"IPList":[{"IP":"192.168.64.12","Mask":8,"ID":0},` +
			`{"IP":"192.168.0.5","Mask":24,"ID":1}],"Message":{"Text":"Everything is OK","Code":0}}`
		require.Equal(t, respExpect, string(respBody))
	})
}

func TestServer_AuthorizationRequest(t *testing.T) {
	t.Run("AuthorizationRequest", func(t *testing.T) {
		data := bytes.NewBufferString(`{
			"Login":"user",
			"Password":"PassGood",
			"IP":"192.168.64.12"
		}`)
		server := createServer(t)
		r := httptest.NewRequest("GET", "/request/", data)
		w := httptest.NewRecorder()
		server.AuthorizationRequest(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := `{"Message":"Check successful","OK":true}`
		require.Equal(t, respExpect, string(respBody))
	})
}

func TestServer_ClearBucketForLogin(t *testing.T) {
	t.Run("ClearBucketForLogin", func(t *testing.T) {
		data := bytes.NewBufferString(`{
			"Tag":"user"
		}`)
		server := createServer(t)
		r := httptest.NewRequest("DELETE", "/clearbucketforlogin/", data)
		w := httptest.NewRecorder()
		server.ClearBucketForLogin(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
}

func TestServer_ClearBucketForIP(t *testing.T) {
	t.Run("ClearBucketForIP", func(t *testing.T) {
		data := bytes.NewBufferString(`{
			"Tag":"192.168.64.12"
		}`)
		server := createServer(t)
		r := httptest.NewRequest("DELETE", "/clearbucketforip/", data)
		w := httptest.NewRecorder()
		server.ClearBucketForIP(w, r)
		result := w.Result()
		defer result.Body.Close()
		respBody, err := io.ReadAll(result.Body)
		require.NoError(t, err)
		respExpect := correctOutJSONAnswer
		require.Equal(t, respExpect, string(respBody))
	})
}

func createServer(t *testing.T) *Server {
	t.Helper()
	t.Helper()
	logg, _ := logger.New("debug")
	config := ConfigTest{}
	storage := storageSQLMock.New()
	ctxStorage, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
	defer cancel()
	err := storage.Init(ctxStorage, logg, &config)
	require.NoError(t, err)
	redis := RedisStorage.New()
	err = redis.InitAsMock(ctxStorage, logg)
	require.NoError(t, err)
	antibf := app.New(logg, storage, redis, &config)
	server := NewServer(logg, antibf, &config)
	return server
}

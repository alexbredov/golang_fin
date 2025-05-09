//nolint:gosec,goconst,noctx,nolintlint
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"

	"github.com/alexbredov/golang_fin/helpers"
	"github.com/alexbredov/golang_fin/internal/logger"
	storageData "github.com/alexbredov/golang_fin/internal/storage/storageData"
	_ "github.com/jackc/pgx/stdlib" // db driver
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var (
	configFilePath string
	pgSQLDB        *sql.DB
	reddb          *redis.Client
	config         Config
	log            *logger.LogWrapper
)

type AuthorizationRequestAnswer struct {
	Message string
	OK      bool
}
type outputJSON struct {
	Text string
	Code int
}
type IPListResult struct {
	IPList  []storageData.StorageIPData
	Message outputJSON
}
type InputTag struct {
	Tag string
}

func init() {
	flag.StringVar(&configFilePath, "config", "../configs/", "Path to config file")
}

func TestMain(m *testing.M) {
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	config = NewConfig()
	if err := config.Init(configFilePath); err != nil {
		fmt.Println(err)
		os.Exit(1) //nolint:gocritic
	}

	var err error
	log, err = logger.New(config.Logger.Level)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pgSQLDB, err = InitAndConnectDB(ctx, log, &config)
	if err != nil {
		log.Error("PGSQL InitAndConnectDB err: " + err.Error())
		os.Exit(1)
	}
	log.Info("PGSQL InitAndConnectDB success")

	reddb, err = InitAndConnectRedis(ctx, log, &config)
	if err != nil {
		log.Error("Redis InitAndConnect err: " + err.Error())
		os.Exit(1)
	}
	log.Info("Redis InitAndConnect success")

	log.Info("Integration tests are up and running")

	exitCode := m.Run()

	// Очистка после тестов
	if err := cleanDBandRedis(ctx, log); err != nil {
		log.Error("Error cleaning DB and Redis: " + err.Error())
	}
	if err := closeDBandRedis(ctx, log); err != nil {
		log.Error("Error closing DB and Redis: " + err.Error())
	}
	log.Info("Integration tests complete")
	os.Exit(exitCode)
}

func TestAddToWL(t *testing.T) { //nolint:dupl
	t.Run("AddToWhiteList_Success", func(t *testing.T) {
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `SELECT IP,mask FROM whitelist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("AddToWhiteList_Success done")
	})
	t.Run("AddToWhiteList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Internal error: "+"IP is already in blacklist")
		script = `SELECT IP,mask FROM whitelist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("AddToWhiteList_Failure done")
	})
}

func TestRemoveFromWL(t *testing.T) {
	t.Run("RemoveFromWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM whitelist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		script = `SELECT IP,mask FROM whitelist WHERE IP = '192.168.64.0' AND mask=24`
		row = pgSQLDB.QueryRowContext(ctx, script)
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("RemoveFromWhiteList_Success done")
	})
	t.Run("RemoveFromWhiteList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Internal error: "+storageData.ErrNoRecord.Error())
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("RemoveFromWhiteList_Failure done")
	})
}

func TestIPIsInWL(t *testing.T) {
	t.Run("IPIsInWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP, mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM whitelist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "Yes")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPIsInWhiteList_Success done")
	})
	t.Run("IPIsInWhiteList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "No")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPIsInWhiteList_Failure done")
	})
}

func TestIPGetAllInWL(t *testing.T) {
	t.Run("IPGetAllInWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `INSERT INTO whitelist(IP,mask) VALUES ('10.0.0.0',8)`
		_, err = pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"ALL","Mask":0}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		result := make([]string, 0)
		for _, currentIPSubnet := range answer.IPList {
			result = append(result, helpers.StringBuild(currentIPSubnet.IP, "/", strconv.Itoa(currentIPSubnet.Mask)))
		}
		require.Equal(t, len(result), 2)
		require.Equal(t, result[0], "192.168.64.0/24")
		require.Equal(t, result[1], "10.0.0.0/8")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPGetAllInWhiteList_Success done")
	})
}

func TestAddToBL(t *testing.T) { //nolint:dupl
	t.Run("AddToBlackList_Success", func(t *testing.T) {
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `SELECT IP,mask FROM blacklist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("AddToBlackList_Success done")
	})
	t.Run("AddToBlackList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Internal error: "+"IP is already in whitelist")
		script = `SELECT IP,mask FROM blacklist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("AddToBlackList_Failure done")
	})
}

func TestRemoveFromBL(t *testing.T) {
	t.Run("RemoveFromBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM blacklist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		script = `SELECT IP,mask FROM blacklist WHERE IP = '192.168.64.0' AND mask=24`
		row = pgSQLDB.QueryRowContext(ctx, script)
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("RemoveFromBlackList_Success done")
	})
	t.Run("RemoveFromBlackList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Internal error: "+storageData.ErrNoRecord.Error())
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("RemoveFromBlackList_Failure done")
	})
}

func TestIPIsInBL(t *testing.T) {
	t.Run("IPIsInBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP, mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM blacklist WHERE IP = '192.168.64.0' AND mask=24`
		row := pgSQLDB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "Yes")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPIsInBlackList_Success done")
	})
	t.Run("IPIsInBlackList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"192.168.64.0","Mask":24}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "No")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPIsInBlackList_Failure done")
	})
}

func TestIPGetAllInBL(t *testing.T) {
	t.Run("IPGetAllInBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ('192.168.64.0',24)`
		_, err := pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `INSERT INTO blacklist(IP,mask) VALUES ('10.0.0.0',8)`
		_, err = pgSQLDB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"ALL","Mask":0}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		result := make([]string, 0)
		for _, currentIPSubnet := range answer.IPList {
			result = append(result, helpers.StringBuild(currentIPSubnet.IP, "/", strconv.Itoa(currentIPSubnet.Mask)))
		}
		require.Equal(t, len(result), 2)
		require.Equal(t, result[0], "192.168.64.0/24")
		require.Equal(t, result[1], "10.0.0.0/8")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("IPGetAllInBlackList_Success done")
	})
}

func TestClearBucketForLogin(t *testing.T) {
	t.Run("ClearBucketForLogin_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		err := reddb.Set(ctx, "l_user", "10", 0).Err()
		require.NoError(t, err)
		value, err := reddb.Get(ctx, "l_user").Result()
		require.NoError(t, err)
		require.Equal(t, value, "10")
		url := helpers.StringBuild("http://", config.GetServerURL(), "/clearLogin/")
		jsonStr := []byte(`{"Tag":"user"}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		log.Info("Response from ClearBucketForLogin" + string(respBody))
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		value, err = reddb.Get(ctx, "l_user").Result()
		require.NoError(t, err)
		require.Equal(t, value, "0")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("ClearBucketForLogin_Success done")
	})
}

func TestClearBucketForIP(t *testing.T) {
	t.Run("ClearBucketForIP_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		err := reddb.Set(ctx, "i_192.168.64.12", "100", 0).Err()
		require.NoError(t, err)
		value, err := reddb.Get(ctx, "i_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "100")
		url := helpers.StringBuild("http://", config.GetServerURL(), "/clearIP/")
		jsonStr := []byte(`{"Tag":"192.168.64.12"}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		log.Info("Response from ClearBucketForIP" + string(respBody))
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		value, err = reddb.Get(ctx, "i_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "0")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("ClearBucketForIP_Success done")
	})
}

func TestAuthorizationRequest(t *testing.T) {
	t.Run("AuthorizationRequest_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/request/")
		jsonStr := []byte(`{
			"Login":"user",
			"Password":"PassGood",
			"IP":"192.168.64.12"
		}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := AuthorizationRequestAnswer{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.OK, true)
		require.Equal(t, answer.Message, "Check successful")
		value, err := reddb.Get(ctx, "l_user").Result()
		require.NoError(t, err)
		require.Equal(t, value, "1")
		value, err = reddb.Get(ctx, "p_PassGood").Result()
		require.NoError(t, err)
		require.Equal(t, value, "1")
		value, err = reddb.Get(ctx, "i_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "1")
		err = cleanDBandRedis(ctx, log)
		require.NoError(t, err)
		log.Info("AuthorizationRequest_Success done")
	})
}

func InitAndConnectDB(ctx context.Context, logger storageData.Logger, config storageData.Config) (*sql.DB, error) {
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		defer recover()
		var err error
		dsn := helpers.StringBuild("postgres://", config.GetDBUser(), ":", config.GetDBPassword(), "@",
			config.GetDBAddress(), ":", config.GetDBPort(), "/", config.GetDBName(), "?sslmode=disable")
		pgSQLDBint, err := sql.Open("pgx", dsn)
		if err != nil {
			logger.Error("SQL Open connection failed:" + err.Error())
			return nil, err
		}
		pgSQLDBint.SetConnMaxLifetime(config.GetDBMaxConnectionLifetime())
		pgSQLDBint.SetMaxOpenConns(config.GetDBMaxOpenConnections())
		pgSQLDBint.SetMaxIdleConns(config.GetDBMaxIdleConnections())
		err = pgSQLDBint.PingContext(ctx)
		if err != nil {
			logger.Error("SQL DB ping failed:" + err.Error())
			return nil, err
		}
		return pgSQLDBint, nil
	}
}

func InitAndConnectRedis(ctx context.Context, logger storageData.Logger, config storageData.Config) (*redis.Client, error) { //nolint:lll
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		defer recover()
		var err error
		reddb = redis.NewClient(&redis.Options{
			Addr:     config.GetRedisAddress() + ":" + config.GetRedisPort(),
			Password: "",
			DB:       0,
		})
		_, err = reddb.Ping(ctx).Result()
		if err != nil {
			logger.Error("Redis Ping err:" + err.Error())
			return nil, err
		}
		reddb.FlushDB(ctx)
		return reddb, nil
	}
}

func cleanDBandRedis(ctx context.Context, logger storageData.Logger) error {
	reddb.FlushDB(ctx)
	script := `TRUNCATE TABLE whitelist`
	_, err := pgSQLDB.ExecContext(ctx, script)
	if err != nil {
		logger.Error("SQL DB truncate whitelist failed:" + err.Error())
		return err
	}
	script = `TRUNCATE TABLE blacklist`
	_, err = pgSQLDB.ExecContext(ctx, script)
	if err != nil {
		logger.Error("SQL DB truncate blacklist failed:" + err.Error())
		return err
	}
	return err
}

func closeDBandRedis(_ context.Context, logger storageData.Logger) error {
	err := reddb.Close()
	if err != nil {
		logger.Error("Redis Close err:" + err.Error())
		return err
	}
	err = pgSQLDB.Close()
	return err
}

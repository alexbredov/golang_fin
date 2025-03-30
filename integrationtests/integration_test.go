//go:build integration

package integrationtests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/abredov/golang_fin/helpers"
	"github.com/abredov/golang_fin/internal/logger"
	storageData "github.com/abredov/golang_fin/internal/storage/storageData"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
)

var (
	configFilePath string
	pgSQL_DB       *sql.DB
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
	flag.StringVar(&configFilePath, "config", "./configs/docker/", "Path to config file")
}

func TestMain(m *testing.M) {
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	config = NewConfig()
	err := config.Init(configFilePath)
	if err != nil {
		fmt.Println(err)
	}
	log, err = logger.New(config.Logger.Level)
	if err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case <-ctx.Done():
			log.Info("Integration tests failed")
			os.Exit(1)
		default:
			pgSQL_DB, err = InitAndConnectDB(ctx, log, &config)
			if err != nil {
				log.Error("PGSQL InitAndConnectDB err: " + err.Error())
				cancel()
			}
			reddb, err = InitAndConnectRedis(ctx, log, &config)
			log.Info("Integration tests are up and running")
			exitCode := m.Run()
			log.Info("Exit code:" + strconv.Itoa(exitCode))
			err = cleanDBandRedis(ctx)
			if err != nil {
				cancel()
			}
			err = closeDBandRedis(ctx)
			if err != nil {
				cancel()
			}
			log.Info("Integration tests complete")
			os.Exit(exitCode)
		}
	}
}

func TestAddToWL(t *testing.T) {
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
		script := `SELECT IP,mask FROM whitelist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
	t.Run("AddToWhiteList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.Error(t, err)
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
		require.Equal(t, answer.Text, "IP is already in blacklist")
		script = `SELECT IP,mask FROM whitelist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestRemoveFromWL(t *testing.T) {
	t.Run("RemoveFromWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM whitelist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		script = `SELECT IP,mask FROM whitelist WHERE IP = "192.168.64.0" AND mask=24`
		row = pgSQL_DB.QueryRowContext(ctx, script)
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, storageData.ErrNoRecord.Error())
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestIPIsInWL(t *testing.T) {
	t.Run("IPIsInWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP, mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM whitelist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "Yes")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "No")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestIPGetAllInWL(t *testing.T) {
	t.Run("IPGetAllInWhiteList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `INSERT INTO whitelist(IP,mask) VALUES ("10.0.0.0",8)`
		_, err = pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/whitelist/")
		jsonStr := []byte(`{"IP":"ALL","Mask":0}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
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
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestAddToBL(t *testing.T) {
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
		script := `SELECT IP,mask FROM blacklist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.NoError(t, err)
		require.Equal(t, IP, "192.168.64.0")
		require.Equal(t, mask, 24)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
	t.Run("AddToBlackList_Failure", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO whitelist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.Error(t, err)
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
		require.Equal(t, answer.Text, "IP is already in whitelist")
		script = `SELECT IP,mask FROM blacklist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
		var IP string
		var mask int
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestRemoveFromBL(t *testing.T) {
	t.Run("RemoveFromBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM blacklist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		script = `SELECT IP,mask FROM blacklist WHERE IP = "192.168.64.0" AND mask=24`
		row = pgSQL_DB.QueryRowContext(ctx, script)
		err = row.Scan(&IP, &mask)
		require.Truef(t, errors.Is(err, sql.ErrNoRows), "actual error %q", err)
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, storageData.ErrNoRecord.Error())
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestIPIsInBL(t *testing.T) {
	t.Run("IPIsInBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP, mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `SELECT IP,mask FROM blacklist WHERE IP = "192.168.64.0" AND mask=24`
		row := pgSQL_DB.QueryRowContext(ctx, script)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "Yes")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
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
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := IPListResult{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Message.Text, "No")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestIPGetAllInBL(t *testing.T) {
	t.Run("IPGetAllInBlackList_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		script := `INSERT INTO blacklist(IP,mask) VALUES ("192.168.64.0",24)`
		_, err := pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		script = `INSERT INTO blacklist(IP,mask) VALUES ("10.0.0.0",8)`
		_, err = pgSQL_DB.ExecContext(ctx, script)
		require.NoError(t, err)
		url := helpers.StringBuild("http://", config.GetServerURL(), "/blacklist/")
		jsonStr := []byte(`{"IP":"ALL","Mask":0}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
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
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
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
		url := helpers.StringBuild("http://", config.GetServerURL(), "/clearbucketforlogin/")
		jsonStr := []byte(`{"Tag":"user"}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		value, err = reddb.Get(ctx, "l_user").Result()
		require.NoError(t, err)
		require.Equal(t, value, "0")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestClearBucketForIP(t *testing.T) {
	t.Run("ClearBucketForIP_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		err := reddb.Set(ctx, "ip_192.168.64.12", "100", 0).Err()
		require.NoError(t, err)
		value, err := reddb.Get(ctx, "ip_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "100")
		url := helpers.StringBuild("http://", config.GetServerURL(), "/clearbucketforip/")
		jsonStr := []byte(`{"Tag":"192.168.64.12"}`)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		answer := outputJSON{}
		err = json.Unmarshal(respBody, &answer)
		require.NoError(t, err)
		require.Equal(t, answer.Text, "Everything is OK")
		value, err = reddb.Get(ctx, "ip_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "0")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}
func TestAuthorizationRequest(t *testing.T) {
	t.Run("AuthorizationRequest_Success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), config.GetDBTimeout())
		defer cancel()
		url := helpers.StringBuild("http://", config.GetServerURL(), "/request/")
		jsonStr := []byte(`{
			"Login":"user"
			"Password":"PassGood"
			"IP":"192.168.64.12"
		}`)
		req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err = client.Do(req)
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
		value, err = reddb.Get(ctx, "ip_192.168.64.12").Result()
		require.NoError(t, err)
		require.Equal(t, value, "1")
		err = cleanDBandRedis(ctx)
		require.NoError(t, err)
	})
}

func InitAndConnectDB(ctx context.Context, logger storageData.Logger, config storageData.Config) (*sql.DB, error) {
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		defer recover()
		var err error
		dsn := helpers.StringBuild(config.GetDBUser(), ":", config.GetDBPassword(), "@tcp(", config.GetDBAddress(), ":", config.GetDBPort(), ")/", config.GetDBName())
		pgSQL_DBint, err := sql.Open("pgx", dsn)
		if err != nil {
			logger.Error("SQL Open connection failed:" + err.Error())
			return nil, err
		}
		pgSQL_DBint.SetConnMaxLifetime(config.GetDBMaxConnectionLifetime())
		pgSQL_DBint.SetMaxOpenConns(config.GetDBMaxOpenConnections())
		pgSQL_DBint.SetMaxIdleConns(config.GetDBMaxIdleConnections())
		err = pgSQL_DBint.PingContext(ctx)
		if err != nil {
			logger.Error("SQL DB ping failed:" + err.Error())
			return nil, err
		}
		return pgSQL_DBint, nil
	}
}
func InitAndConnectRedis(ctx context.Context, logger storageData.Logger, config storageData.Config) (*redis.Client, error) {
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
func cleanDBandRedis(ctx context.Context) error {
	reddb.FlushDB(ctx)
	script := "TRUNCATE TABLE OTUSAntibf.whitelist"
	_, err := pgSQL_DB.ExecContext(ctx, script)
	if err != nil {
		return err
	}
	script = "TRUNCATE TABLE OTUSAntibf.blacklist"
	_, err = pgSQL_DB.ExecContext(ctx, script)
	if err != nil {
		return err
	}
}

func closeDBandRedis(ctx context.Context) error {
	err := reddb.Close()
	if err != nil {
		return err
	}
	err = pgSQL_DB.Close()
	return err
}

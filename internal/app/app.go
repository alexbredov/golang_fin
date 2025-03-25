package app

import (
	"antibf/helpers"
	"antibf/internal/storage/storageData"
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var (
	ErrIPinBL = errors.New("IP is already in blacklist")
	ErrIPinWL = errors.New("IP is already in whitelist")
)

type App struct {
	logger        Logger
	storage       Storage
	bucketStorage BucketStorage
	ticker        *time.Ticker
	period        time.Duration
	limitLogin    int
	limitPassword int
	limitIP       int
}

type Logger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	Fatal(msg string)
	GetZapLogger() *zap.SugaredLogger
}

type Storage interface {
	Init(ctx context.Context, logger storageData.Logger, config storageData.Config) error
	Close(ctx context.Context, logger storageData.Logger) error
	IPAddToList(ctx context.Context, listname string, logger storageData.Logger, IPData storageData.StorageIPData) (int, error)
	IPRemoveFromList(ctx context.Context, listname string, logger storageData.Logger, IPData storageData.StorageIPData) error
	IPIsInList(ctx context.Context, listname string, logger storageData.Logger, IPData storageData.StorageIPData) (bool, error)
	IPGetAllFromList(ctx context.Context, listname string, logger storageData.Logger) ([]storageData.StorageIPData, error)
}
type BucketStorage interface {
	Init(ctx context.Context, logger storageData.Logger, config storageData.Config) error
	SetBucketValue(ctx context.Context, logger storageData.Logger, key string, value int) error
	IncreaseAndGetBucketValue(ctx context.Context, logger storageData.Logger, key string) (int, error)
	Close(ctx context.Context, logger storageData.Logger) error
	FlushStorage(ctx context.Context, logger storageData.Logger) error
}

func New(logger Logger, storage Storage, bucketStorage BucketStorage, config storageData.Config) *App {
	app := App{
		logger:        logger,
		storage:       storage,
		bucketStorage: bucketStorage,
		limitIP:       config.GetLimitIP(),
		limitLogin:    config.GetLimitLogin(),
		limitPassword: config.GetLimitPassword(),
		period:        config.GetLimitTimeCheck(),
	}
	return &app
}

func (a *App) InitBucketStorageAndLimits(ctx context.Context, config storageData.Config) error {
	return a.bucketStorage.Init(ctx, a.logger, config)
}

func (a *App) CloseBucketStorage(ctx context.Context) error {
	return a.bucketStorage.Close(ctx, a.logger)
}

func (a *App) CheckRequest(ctx context.Context, req storageData.RequestAuth) (bool, string, error) {
	ok, err := a.IPIsInList(ctx, "blacklist", req.IP)
	if err != nil {
		message := helpers.StringBuild("CheckRequest IPIsInList error: ", err.Error())
		a.logger.Error(message)
		return false, "", err
	}
	if ok {
		return false, "IP is in blacklist", nil
	}
	ok, err = a.IPIsInList(ctx, "whitelist", req.IP)
	if err != nil {
		message := helpers.StringBuild("CheckRequest IPIsInList error: ", err.Error())
		a.logger.Error(message)
		return false, "", err
	}
	if ok {
		return true, "IP is in whitelist", nil
	}
	countLogin, err := a.bucketStorage.IncreaseAndGetBucketValue(ctx, a.logger, "l_"+req.Login)
	if err != nil {
		errBaseText := "CheckRequest IncreaseAndGetBucketValue - Login error: "
		message := helpers.StringBuild(errBaseText, err.Error(), ", key: ", "l_"+req.Login)
		a.logger.Error(message)
		return false, "", err
	}
	if countLogin > int(a.limitLogin) {
		return false, "Limited by login rate", nil
	}
	countPassword, err := a.bucketStorage.IncreaseAndGetBucketValue(ctx, a.logger, "p_"+req.Password)
	if err != nil {
		errBaseText := "CheckRequest IncreaseAndGetBucketValue - Password error: "
		message := helpers.StringBuild(errBaseText, err.Error(), ", key: ", "p_"+req.Password)
		a.logger.Error(message)
		return false, "", err
	}
	if countPassword > int(a.limitPassword) {
		return false, "Limited by password rate", nil
	}
	countIP, err := a.bucketStorage.IncreaseAndGetBucketValue(ctx, a.logger, "i_"+req.IP)
	if err != nil {
		errBaseText := "CheckRequest IncreaseAndGetBucketValue - IP error: "
		message := helpers.StringBuild(errBaseText, err.Error(), ", key:", "i_"+req.IP)
		a.logger.Error(message)
		return false, "", err
	}
	if countIP > int(a.limitIP) {
		return false, "Limited by IP rate", nil
	}
	return true, "Check successful", nil
}

func (a *App) RateLimitTicker(ctx context.Context) {
	a.logger.Info("rate limiter ticker started")
	a.ticker = time.NewTicker(a.period)
	go func() {
		for {
			select {
			case <-ctx.Done():
				a.logger.Info("rate limiter ticker stopped")
				break
			case <-a.ticker.C:
				a.bucketStorage.FlushStorage(ctx, a.logger)
				a.logger.Info("Buckets flushed")
			}
		}
	}()
}

func (a *App) ClearBucketForLogin(ctx context.Context, login string) error {
	err := a.bucketStorage.SetBucketValue(ctx, a.logger, "l_"+login, 0)
	if err != nil {
		message := helpers.StringBuild("ClearBucketForLogin error: ", err.Error(), " Login: ", login)
		a.logger.Error(message)
		return err
	}
	a.logger.Info("Bucket cleared for login " + login)
	return nil
}
func (a *App) ClearBucketForIP(ctx context.Context, ipData string) error {
	err := a.bucketStorage.SetBucketValue(ctx, a.logger, "i_"+ipData, 0)
	if err != nil {
		message := helpers.StringBuild("ClearBucketForIP error: ", err.Error(), " IP: ", ipData)
		a.logger.Error(message)
		return err
	}
	a.logger.Info("Bucket cleared for ip " + ipData)
	return nil
}
func (a *App) InitStorage(ctx context.Context, config storageData.Config) error {
	return a.storage.Init(ctx, a.logger, config)
}
func (a *App) CloseStorage(ctx context.Context) error {
	return a.storage.Close(ctx, a.logger)
}
func (a *App) IPAddToList(ctx context.Context, listname string, ipData storageData.StorageIPData) (int, error) {
	err := SimpleIPDataValidate(ipData, false)
	if err != nil {
		message := helpers.StringBuild("IPAddToList validation failed", err.Error())
		a.logger.Error(message)
		return 0, err
	}
	var secondlistname string
	switch listname {
	case storageData.WhiteListName:
		secondlistname = storageData.BlackListName
	case storageData.BlackListName:
		secondlistname = storageData.WhiteListName
	default:
		return 0, storageData.ErrBadListType
	}
	ok, err := a.storage.IPIsInList(ctx, secondlistname, a.logger, ipData)
	if err != nil {
		message := helpers.StringBuild("IPAddToList validation in second list failed", err.Error())
		a.logger.Error(message)
		return 0, err
	}
	if ok {
		switch listname {
		case storageData.WhiteListName:
			return 0, ErrIPinBL
		case storageData.BlackListName:
			return 0, ErrIPinWL
		default:
			return 0, storageData.ErrBadListType
		}
	}
	id, err := a.storage.IPAddToList(ctx, listname, a.logger, ipData)
	if err != nil {
		message := helpers.StringBuild("IPAddToList IP storage error", err.Error())
		a.logger.Error(message)
		return 0, err
	}
	message := helpers.StringBuild("IP added to ", listname, "(IP: ", ipData.IP, "/", strconv.Itoa(ipData.Mask), ")")
	a.logger.Info(message)
	return id, nil
}
func (a *App) IPRemoveFromList(ctx context.Context, listname string, ipData storageData.StorageIPData) error {
	err := checkListName(listname)
	if err != nil {
		message := helpers.StringBuild("IPRemoveFromList checkListName failed: ", err.Error())
		a.logger.Error(message)
		return err
	}
	err = SimpleIPDataValidate(ipData, false)
	if err != nil {
		message := helpers.StringBuild("IPRemoveFromList IPData validation failed: ", err.Error())
		a.logger.Error(message)
		return err
	}
	err = a.storage.IPRemoveFromList(ctx, listname, a.logger, ipData)
	if err != nil {
		message := helpers.StringBuild("IPRemoveFromList app failed (IP: ", ipData.IP, "), ", err.Error())
		a.logger.Error(message)
		return err
	}
	message := helpers.StringBuild("IP successfully removed from "+listname+"(IP: ", ipData.IP, "/", strconv.Itoa(ipData.Mask), ")")
	a.logger.Info(message)
	return nil
}
func (a *App) IPIsInList(ctx context.Context, listname string, ipData storageData.StorageIPData) (bool, error) {
	err := checkListName(listname)
	if err != nil {
		message := helpers.StringBuild("IPIsInList checkListName failed: ", err.Error())
		a.logger.Error(message)
		return false, err
	}
	err = SimpleIPDataValidate(ipData, false)
	if err != nil {
		message := helpers.StringBuild("IPIsInList IPData validation failed: ", err.Error())
		a.logger.Error(message)
		return false, err
	}
	ok, err := a.storage.IPIsInList(ctx, listname, a.logger, ipData)
	if err != nil {
		message := helpers.StringBuild("IPIsInList app failed: ", err.Error())
		a.logger.Error(message)
		return false, err
	}
	return ok, err
}
func (a *App) IPGetAllFromList(ctx context.Context, listname string) ([]storageData.StorageIPData, error) {
	err := checkListName(listname)
	if err != nil {
		message := helpers.StringBuild("IPGetAllFromList checkListName failed: ", err.Error())
		a.logger.Error(message)
		return nil, err
	}
	list, err := a.storage.IPGetAllFromList(ctx, listname, a.logger)
	if err != nil {
		message := helpers.StringBuild("IPGetAllFromList app failed: ", err.Error())
		a.logger.Error(message)
		return nil, err
	}
	return list, err
}
func (a *App) IPIsInSubnetCheck(ctx context.Context, listname string, ip string) (bool, error) {
	err := checkListName(listname)
	if err != nil {
		message := helpers.StringBuild("IPIsInSubnetCheck checkListName failed: ", err.Error())
		a.logger.Error(message)
		return false, err
	}
	canIP := net.ParseIP(ip)
	list, err := a.storage.IPGetAllFromList(ctx, listname, a.logger)
	if err != nil {
		return false, err
	}
	for _, currentIPData := range list {
		currentIPString := currentIPData.IP + "/" + strconv.Itoa(currentIPData.Mask)
		_, subnet, err := net.ParseCIDR(currentIPString)
		if err != nil {
			return false, err
		}
		if subnet.Contains(canIP) {
			return true, nil
		}
	}
	return false, nil
}
func checkListName(listname string) error {
	if listname != storageData.WhiteListName && listname != storageData.BlackListName {
		return storageData.ErrBadListType
	}
	return nil
}

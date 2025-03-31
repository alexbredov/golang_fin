package storagesqlmock

import (
	"context"
	"sort"
	"strconv"
	"sync"

	storageData "github.com/alexbredov/golang_fin/internal/storage/storageData"
)

type StorageSQLMock struct {
	mu        sync.RWMutex
	whitelist map[string]storageData.StorageIPData
	blacklist map[string]storageData.StorageIPData
	idWhite   int
	idBlack   int
}

func New() *StorageSQLMock {
	return &StorageSQLMock{}
}

func (mock *StorageSQLMock) Init(_ context.Context, _ storageData.Logger, _ storageData.Config) error {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.whitelist = make(map[string]storageData.StorageIPData)
	mock.blacklist = make(map[string]storageData.StorageIPData)
	mock.idWhite = 0
	mock.idBlack = 0
	return nil
}

func (mock *StorageSQLMock) Close(_ context.Context, _ storageData.Logger) error {
	return nil
}

func (mock *StorageSQLMock) IPAddToList(ctx context.Context, listname string, _ storageData.Logger, ipData storageData.StorageIPData) (int, error) { //nolint:lll
	select {
	case <-ctx.Done():
		return 0, storageData.ErrStorageTimeout
	default:
		tag := ipData.IP + "/" + strconv.Itoa(ipData.Mask)
		mock.mu.Lock()
		defer mock.mu.Unlock()
		switch listname {
		case storageData.WhiteListName:
			ipData.ID = mock.idWhite
			mock.whitelist[tag] = ipData
			mock.idWhite++
		case storageData.BlackListName:
			ipData.ID = mock.idBlack
			mock.blacklist[tag] = ipData
			mock.idBlack++
		default:
			return 0, storageData.ErrBadListType
		}
		return ipData.ID, nil
	}
}

func (mock *StorageSQLMock) IPIsInList(ctx context.Context, listname string, _ storageData.Logger, ipData storageData.StorageIPData) (bool, error) { //nolint:lll
	select {
	case <-ctx.Done():
		return false, storageData.ErrStorageTimeout
	default:
		tag := ipData.IP + "/" + strconv.Itoa(ipData.Mask)
		mock.mu.RLock()
		defer mock.mu.RUnlock()
		var err error
		var ok bool
		switch listname {
		case storageData.WhiteListName:
			_, ok = mock.whitelist[tag]
		case storageData.BlackListName:
			_, ok = mock.blacklist[tag]
		default:
			return false, storageData.ErrBadListType
		}
		return ok, err
	}
}

func (mock *StorageSQLMock) IPRemoveFromList(ctx context.Context, listname string, _ storageData.Logger, ipData storageData.StorageIPData) error { //nolint:lll
	select {
	case <-ctx.Done():
		return storageData.ErrStorageTimeout
	default:
		var ok bool
		tag := ipData.IP + "/" + strconv.Itoa(ipData.Mask)
		switch listname {
		case storageData.WhiteListName:
			_, ok = mock.whitelist[tag]
		case storageData.BlackListName:
			_, ok = mock.blacklist[tag]
		default:
			return storageData.ErrBadListType
		}
		if !ok {
			return storageData.ErrNoRecord
		}
		mock.mu.Lock()
		defer mock.mu.Unlock()
		switch listname {
		case storageData.WhiteListName:
			delete(mock.whitelist, tag)
		case storageData.BlackListName:
			delete(mock.blacklist, tag)
		default:
			return storageData.ErrBadListType
		}
		return nil
	}
}

func (mock *StorageSQLMock) IPGetAllFromList(ctx context.Context, listname string, _ storageData.Logger) ([]storageData.StorageIPData, error) { //nolint:lll
	resultIPData := make([]storageData.StorageIPData, 0)
	select {
	case <-ctx.Done():
		return nil, storageData.ErrStorageTimeout
	default:
		mock.mu.RLock()
		switch listname {
		case storageData.WhiteListName:
			for _, currentIPData := range mock.whitelist {
				resultIPData = append(resultIPData, currentIPData)
			}
		case storageData.BlackListName:
			for _, currentIPData := range mock.blacklist {
				resultIPData = append(resultIPData, currentIPData)
			}
		default:
			return nil, storageData.ErrBadListType
		}
		mock.mu.RUnlock()
		sort.SliceStable(resultIPData, func(i, j int) bool {
			return resultIPData[i].ID < resultIPData[j].ID
		})
		return resultIPData, nil
	}
}

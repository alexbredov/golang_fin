package app

import (
	storageData "antibf/internal/storage/storageData"
	"errors"
	"strconv"
	"strings"
)

var (
	ErrVoidLogin     = errors.New("void login")
	ErrVoidPassword  = errors.New("void password")
	ErrVoidIP        = errors.New("void IP")
	ErrVoidMask      = errors.New("void mask")
	ErrIncorrectMask = errors.New("incorrect mask")
	ErrBadIP         = errors.New("IP struct is bad")
)

func SimpleRequestValidate(login string, password string, ip string) (storageData.RequestAuth, error) {
	request := storageData.RequestAuth{Login: login, Password: password, IP: ip}
	err := checkIP(ip, 0, 255)
	switch {
	case err != nil:
		return storageData.RequestAuth{}, err
	case request.Login == "":
		return storageData.RequestAuth{}, ErrVoidLogin
	case request.Password == "":
		return storageData.RequestAuth{}, ErrVoidPassword
	default:
	}
	return request, nil
}
func SimpleIPDataValidate(ipData storageData.StorageIPData, isAllRequest bool) error {
	var err error
	if !isAllRequest {
		err = checkIP(ipData.IP, 0, 255)
	}
	switch {
	case err != nil:
		return err
	case ipData.IP == "":
		return ErrVoidIP
	case ipData.Mask == 0 && !isAllRequest:
		return ErrVoidMask
	case ipData.Mask < 0 || ipData.Mask > 31:
		return ErrIncorrectMask
	default:
	}
	return nil
}

func checkIP(ip string, minimal int, maximal int) error {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ErrBadIP
	}
	for _, currentPart := range parts {
		intPart, err := strconv.Atoi(currentPart)
		if err != nil {
			return err
		}
		if intPart < minimal || intPart > maximal {
			return ErrBadIP
		}
	}
	return nil
}

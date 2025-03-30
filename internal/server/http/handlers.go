package httpinternal

import (
	"antibf/helpers"
	storageData "antibf/internal/storage/storageData"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

const correctAnswerText string = "Everything is OK"

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

var (
	ErrInputJSONBad      = errors.New("input JSON is bad")
	ErrOutputJSONBad     = errors.New("output JSON is bad")
	ErrMethodUnsupported = errors.New("method is unsupported")
	ErrNoIDInIP          = errors.New("no ID in IP handler")
)

func apiErrHandler(err error, w *http.ResponseWriter) {
	W := *w
	newMsg := outputJSON{}
	newMsg.Text = err.Error()
	newMsg.Code = 1
	jsonstr, err := json.Marshal(newMsg)
	if err != nil {
		errMsg := helpers.StringBuild(http.StatusText(http.StatusInternalServerError), " (", err.Error(), ")")
		http.Error(W, errMsg, http.StatusInternalServerError)
	}
	_, err = W.Write(jsonstr)
	if err != nil {
		errMsg := helpers.StringBuild(http.StatusText(http.StatusInternalServerError), " (", err.Error(), ")")
		http.Error(W, errMsg, http.StatusInternalServerError)
	}
}
func (server *Server) helloWorld(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello World"))
}
func (server *Server) AuthorizationRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()
	switch r.Method {
	case http.MethodGet:
		newRequest := storageData.RequestAuth{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		err = json.Unmarshal(body, &newRequest)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		answer := &AuthorizationRequestAnswer{}
		ok, msg, errInternal := server.app.CheckRequest(ctx, newRequest)
		if errInternal != nil {
			answer.Message = "Internal error: " + errInternal.Error()
			answer.OK = false
			w.Header().Add("X-Request-Error", errInternal.Error())
		}
		answer.Message = msg
		answer.OK = ok
		jsonstr, err := json.Marshal(answer)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		_, err = w.Write(jsonstr)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		return
	default:
		apiErrHandler(ErrMethodUnsupported, &w)
		return
	}
}
func (server *Server) ClearBucketForIP(w http.ResponseWriter, r *http.Request) {
	server.clearBucketByTag(w, r, "ip")
}
func (server *Server) ClearBucketForLogin(w http.ResponseWriter, r *http.Request) {
	server.clearBucketByTag(w, r, "login")
}
func (server *Server) clearBucketByTag(w http.ResponseWriter, r *http.Request, tagType string) {
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	defer cancel()
	switch r.Method {
	case http.MethodDelete:
		newMsg := outputJSON{}
		inputTag := InputTag{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		err = json.Unmarshal(body, &inputTag)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		switch tagType {
		case "login":
			err = server.app.ClearBucketForLogin(ctx, inputTag.Tag)
		case "ip":
			err = server.app.ClearBucketForIP(ctx, inputTag.Tag)
		default:
			apiErrHandler(ErrBadBucketTypeTag, &w)
			return
		}
		if err != nil {
			newMsg.Text = err.Error()
			newMsg.Code = 1
			w.Header().Add("X-Request-Error", err.Error())
		} else {
			newMsg.Text = correctAnswerText
			newMsg.Code = 0
		}
		jsonstr, err := json.Marshal(newMsg)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		_, err = w.Write(jsonstr)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		return
	default:
		apiErrHandler(ErrMethodUnsupported, &w)
		return
	}
}

func (server *Server) RESTWhiteList(w http.ResponseWriter, r *http.Request) {
	server.listRest(w, r, "whitelist")
}
func (server *Server) RESTBlackList(w http.ResponseWriter, r *http.Request) {
	server.listRest(w, r, "blacklist")
}
func (server *Server) listRest(w http.ResponseWriter, r *http.Request, listname string) {
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), server.Config.GetDBTimeout())
	defer cancel()
	switch r.Method {
	case http.MethodGet:
		IPListRes := IPListResult{}
		newIPData := storageData.StorageIPData{}
		newMsg := outputJSON{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		err = json.Unmarshal(body, &newIPData)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		if newIPData.IP == "ALL" {
			IPList, errInternal := server.app.IPGetAllFromList(ctx, listname)
			if errInternal != nil {
				newMsg.Text = "Internal error: " + errInternal.Error()
				newMsg.Code = 1
				w.Header().Add("X-Request-Error", errInternal.Error())
			} else {
				newMsg.Text = correctAnswerText
				newMsg.Code = 0
			}
			IPListRes.IPList = make([]storageData.StorageIPData, len(IPList))
			IPListRes.IPList = IPList
			IPListRes.Message = newMsg
			jsonstr, err := json.Marshal(IPListRes)
			if err != nil {
				apiErrHandler(err, &w)
				return
			}
			_, err = w.Write(jsonstr)
			if err != nil {
				apiErrHandler(err, &w)
				return
			}
			return
		}
		ok, errInternal := server.app.IPIsInList(ctx, listname, newIPData)
		if errInternal != nil {
			newMsg.Text = "Internal error: " + errInternal.Error()
			newMsg.Code = 1
			w.Header().Add("X-Request-Error", errInternal.Error())
		} else {
			if ok {
				newMsg.Text = "Yes"
			} else {
				newMsg.Text = "No"
			}
			newMsg.Code = 0
		}
		IPListRes.IPList = make([]storageData.StorageIPData, 0)
		IPListRes.Message = newMsg
		jsonstr, err := json.Marshal(IPListRes)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		_, err = w.Write(jsonstr)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		return
	case http.MethodPost:
		newIPData := storageData.StorageIPData{}
		newMsg := outputJSON{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		err = json.Unmarshal(body, &newIPData)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		id, errInternal := server.app.IPAddToList(ctx, listname, newIPData)
		if errInternal != nil {
			newMsg.Text = "Internal error: " + errInternal.Error()
			newMsg.Code = 1
			w.Header().Add("X-Request-Error", errInternal.Error())
		} else {
			newMsg.Text = correctAnswerText
			newMsg.Code = id
		}
		jsonstr, err := json.Marshal(newMsg)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		_, err = w.Write(jsonstr)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		return
	case http.MethodDelete:
		deleteData := storageData.StorageIPData{}
		newMsg := outputJSON{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		err = json.Unmarshal(body, &deleteData)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		errInternal := server.app.IPRemoveFromList(ctx, listname, deleteData)
		if errInternal != nil {
			newMsg.Text = "Internal error: " + errInternal.Error()
			newMsg.Code = 1
			w.Header().Add("X-Request-Error", errInternal.Error())
		} else {
			newMsg.Text = correctAnswerText
			newMsg.Code = 0
		}
		jsonstr, err := json.Marshal(newMsg)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		_, err = w.Write(jsonstr)
		if err != nil {
			apiErrHandler(err, &w)
			return
		}
		return
	default:
		apiErrHandler(ErrMethodUnsupported, &w)
		return
	}
}

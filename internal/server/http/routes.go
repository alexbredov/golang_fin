package http_internal

import "net/http"

func (server *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", loggingMiddleware(server.helloWorld, server.logger))
	mux.HandleFunc("/request/", loggingMiddleware(server.AuthorizationRequest, server.logger))
	mux.HandleFunc("/clearLogin/", loggingMiddleware(server.ClearBucketForLogin, server.logger))
	mux.HandleFunc("/clearIP/", loggingMiddleware(server.ClearBucketForIP, server.logger))
	mux.HandleFunc("/whitelist/", loggingMiddleware(server.RESTWhiteList, server.logger))
	mux.HandleFunc("/blacklist/", loggingMiddleware(server.RESTBlackList, server.logger))
	return mux
}

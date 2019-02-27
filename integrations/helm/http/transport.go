package http

import (
	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.NewRoute().Name(SyncGit).Methods("POST").Path("/v1/sync-git")
	return r
}

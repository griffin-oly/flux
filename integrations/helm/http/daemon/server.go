package daemon

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/weaveworks/flux/integrations/helm/api"
	transport "github.com/weaveworks/flux/integrations/helm/http"
	"net/http"
	"time"
)

// ListenAndServe starts a HTTP server instrumented with Prometheus on the specified address
func ListenAndServe(listenAddr string, apiServer api.Server, logger log.Logger, stopCh <-chan struct{}) {
	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	handler := NewHandler(apiServer, transport.NewRouter())
	mux.Handle("/api/", http.StripPrefix("/api", handler))

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  15 * time.Second,
	}

	logger.Log("info", fmt.Sprintf("Starting HTTP server on %s", listenAddr))

	// run server in background
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Log("error", fmt.Sprintf("HTTP server crashed %v", err))
		}
	}()

	// wait for close signal and attempt graceful shutdown
	<-stopCh
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log("warn", fmt.Sprintf("HTTP server graceful shutdown failed %v", err))
	} else {
		logger.Log("info", "HTTP server stopped")
	}
}

func NewHandler(s api.Server, r *mux.Router) http.Handler {
	handle := HTTPServer{s}
	r.Get(transport.SyncGit).HandlerFunc(handle.SyncGit)
	return r
}

type HTTPServer struct {
	server api.Server
}

func (s HTTPServer) SyncGit(w http.ResponseWriter, r *http.Request) {
	go s.server.SyncMirrors()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

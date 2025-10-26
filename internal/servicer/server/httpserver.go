package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/hasmcp/sser/internal/servicer/config"
	zlog "github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/acme/autocert"
)

type (
	Servicer interface {
		ListenAndServe() error
		Shutdown() error
	}

	servicer struct {
		cfg     serverConfig
		server  *fasthttp.Server
		acmesrv *http.Server
	}

	Params struct {
		Config  config.Servicer
		Handler fasthttp.RequestHandler
	}

	serverConfig struct {
		Name                         string        `yaml:"name"`
		Addr                         string        `yaml:"addr"`
		SSLAddr                      string        `yaml:"sslAddr"`
		IdleTimeout                  time.Duration `yaml:"idleTimeout"`
		ReadTimeout                  time.Duration `yaml:"readTimeout"`
		WriteTimeout                 time.Duration `yaml:"writeTimeout"`
		MaxRequestBodySize           int           `yaml:"maxRequestBodySize"`
		TCPKeepalivePeriod           time.Duration `yaml:"tcpKeepalivePeriod"`
		TCPKeepalive                 bool          `yaml:"tcpKeepalive"`
		NoDefaultDate                bool          `yaml:"noDefaultDate"`
		CloseOnShutdown              bool          `yaml:"closeOnShutdown"`
		DisablePreParseMultipartForm bool          `yaml:"disablePreParseMultipartForm"`
		SSL                          SslConfig     `yaml:"ssl"`
	}

	SslConfig struct {
		Enabled          bool   `yaml:"enabled"`
		DomainName       string `yaml:"domainName"`
		LetsencryptEmail string `yaml:"letsencryptEmail"`
		CacheDir         string `yaml:"cacheDir"`
	}
)

const (
	cfgKey = "server"

	logPrefix = "[httpserver] "
)

func New(p Params) (Servicer, error) {
	var cfg serverConfig
	if err := p.Config.Populate(cfgKey, &cfg); err != nil {
		return nil, err
	}

	var srv *http.Server
	server := &fasthttp.Server{
		Handler:                      p.Handler,
		Name:                         cfg.Name,
		IdleTimeout:                  cfg.IdleTimeout,
		ReadTimeout:                  cfg.ReadTimeout,
		WriteTimeout:                 cfg.WriteTimeout,
		MaxRequestBodySize:           cfg.MaxRequestBodySize,
		TCPKeepalivePeriod:           cfg.TCPKeepalivePeriod,
		TCPKeepalive:                 cfg.TCPKeepalive,
		NoDefaultDate:                cfg.NoDefaultDate,
		CloseOnShutdown:              cfg.CloseOnShutdown,
		DisablePreParseMultipartForm: cfg.DisablePreParseMultipartForm,
	}
	if cfg.SSL.Enabled {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS, // Automatically agree to the Let's Encrypt TOS
			HostPolicy: autocert.HostWhitelist(cfg.SSL.DomainName),
			Cache:      autocert.DirCache(cfg.SSL.CacheDir), // This enables certificate persistence and initial validity check
			Email:      cfg.SSL.LetsencryptEmail,
		}

		addr := cfg.Addr
		if !strings.HasPrefix(addr, ":") {
			addr = ":" + addr
		}
		srv = &http.Server{
			Addr:    addr,
			Handler: m.HTTPHandler(http.HandlerFunc(redirectHTTP)), // nil means default redirect to HTTPS
		}

		tlsConfig := &tls.Config{
			GetCertificate: m.GetCertificate,
			// Secure configuration recommended by Mozilla:
			MinVersion: tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{
				tls.CurveP521,
				tls.CurveP384,
				tls.CurveP256,
			},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}

		server.TLSConfig = tlsConfig
	}

	return &servicer{
		cfg:     cfg,
		server:  server,
		acmesrv: srv,
	}, nil
}

func (s *servicer) ListenAndServe() error {
	addr := s.cfg.Addr
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	if s.cfg.SSL.Enabled {
		zlog.Info().Msg(logPrefix + "ssl enabled")
		sslAddr := s.cfg.SSLAddr
		if !strings.HasPrefix(sslAddr, ":") {
			sslAddr = ":" + sslAddr
		}

		go func() {
			// Create a standard HTTP handler that serves the ACME challenge requests
			// and redirects everything else to HTTPS.
			zlog.Info().Str("domainName", s.cfg.SSL.DomainName).Str("addr", addr).
				Msg(logPrefix + "starting ACME challenge HTTP listener")

			if err := s.acmesrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zlog.Fatal().Err(err).Msg(logPrefix + "ACME HTTP listener failed")
			}
			zlog.Info().Msg(logPrefix + "ACME HTTP listener shut down.")
		}()

		// Serve the fasthttp server using the TLS listener
		if err := s.server.ListenAndServeTLS(sslAddr, "", ""); err != nil {
			zlog.Error().Err(err).Msg(logPrefix + "fasthttp HTTPS listener failed")
			return err
		}
	}
	return s.server.ListenAndServe(addr)
}

func redirectHTTP(w http.ResponseWriter, r *http.Request) {
	// Skip redirect for ACME challenge requests
	if r.URL.Path == "/.well-known/acme-challenge/" {
		return
	}
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func (s *servicer) Shutdown() error {
	if s.acmesrv != nil {
		s.acmesrv.Shutdown(context.Background())
	}
	return s.server.Shutdown()
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

var domain string

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	target := "https://" + domain + r.RequestURI
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func main() {
	// Получаем домен из аргумента
	flag.StringVar(&domain, "domain", "", "Domain name to process HTTP/s server")
	flag.Parse()
	if domain == "" {
		fmt.Println("The --domain parameter is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Println("Starting server for domain:", domain)

	// Let's Encrypt
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("certs"), // Кэшируем сертификаты
		Prompt:     autocert.AcceptTOS,         // Автоматическое согласие с условиями Let's Encrypt
		HostPolicy: autocert.HostWhitelist(domain),
	}

	// Запускаем HTTP -> HTTPS
	go func() {
		httpMux := http.NewServeMux()
		httpMux.HandleFunc("/", redirectToHTTPS)

		log.Println("Starting HTTP redirect server on :80")
		if err := http.ListenAndServe(":80", httpMux); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	httpsSrv := &http.Server{
		Addr:      ":443",
		TLSConfig: manager.TLSConfig(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, TLS user! Your config: %+v", r.TLS)
		}),
	}

	// отключение
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		<-stop
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpsSrv.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown error: %v", err)
		}
		log.Println("Server stopped.")
	}()

	// Запуск HTTPS сервера
	log.Println("Starting HTTPS server on :443")
	if err := httpsSrv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTPS server error: %v", err)
	}
}

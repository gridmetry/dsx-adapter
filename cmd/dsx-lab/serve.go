package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gridmetry/dsx-adapter/internal/bridge"
	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
	"gridmetry/dsx-adapter/internal/transport"
)

// runServe wires lab config, the MQTT subscriber, the processing
// pipeline, and the authenticated SSE HTTP server, then blocks until
// SIGINT/SIGTERM for a graceful shutdown.
func runServe(args []string) int {
	cfg, err := lab.LoadBridgeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab serve: %v\n", err)
		return 1
	}

	pubKeyBytes, err := lab.ReadBoundedFile(cfg.PublicKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab serve: reading signing public key: %v\n", err)
		return 1
	}
	pubKey, err := flex.DecodePublicKeyPEM(pubKeyBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab serve: decoding signing public key: %v\n", err)
		return 1
	}

	trust := flex.NewTrust(flex.TrustedSigner{KeyID: cfg.SignerKeyID, PublicKey: pubKey, Source: cfg.SignerSource})
	policy := flex.Policy{
		FeedTag:      cfg.FeedTag,
		MaxWatts:     cfg.MaxWatts,
		MaxEventAge:  cfg.MaxEventAge,
		MaxClockSkew: cfg.MaxClockSkew,
	}
	codec := &flex.Codec{Trust: trust, Policy: policy, Dedup: flex.NewDedup(1024, 10*time.Minute)}

	stats := &lab.Stats{}
	hub := transport.NewHub(64)
	pipeline := bridge.NewPipeline(codec, hub, stats, 256)

	done := make(chan struct{})
	go pipeline.Run(done)

	sub := bridge.NewMQTTSubscriber(bridge.MQTTConfig{
		BrokerURL: cfg.MQTTBrokerURL,
		ClientID:  cfg.MQTTClientID,
		Username:  cfg.MQTTUsername,
		Password:  cfg.MQTTPassword,
		Topic:     cfg.MQTTTopic,
	}, pipeline)
	if err := sub.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab serve: connecting to MQTT broker: %v\n", err)
		close(done)
		return 1
	}

	server := &transport.Server{
		Hub:         hub,
		Token:       cfg.SSEToken,
		AllowOrigin: cfg.CORSAllowOrigin,
		Ready:       sub.Ready,
		Stats:       stats,
	}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: server.Mux()}

	serveErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()
	fmt.Printf("dsx-lab serve: listening on %s, MQTT broker %s, topic %s\n", cfg.HTTPAddr, cfg.MQTTBrokerURL, cfg.MQTTTopic)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Printf("dsx-lab serve: received %s, shutting down\n", sig)
	case err := <-serveErr:
		if err != nil {
			fmt.Fprintf(os.Stderr, "dsx-lab serve: HTTP server error: %v\n", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	sub.Close(250)
	close(done)
	return 0
}

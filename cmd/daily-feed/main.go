package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ryosukesatoh/daily-feed/internal/config"
	"github.com/ryosukesatoh/daily-feed/internal/fetcher"
	"github.com/ryosukesatoh/daily-feed/internal/publisher"
	"github.com/ryosukesatoh/daily-feed/internal/runner"
	"github.com/ryosukesatoh/daily-feed/internal/summarizer"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	once := flag.Bool("once", false, "run the pipeline once and exit")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build fetcher
	var f fetcher.Fetcher
	switch cfg.Fetcher.Type {
	case "arxiv":
		f = fetcher.NewArxivFetcher()
	default:
		log.Fatalf("Unknown fetcher type: %s", cfg.Fetcher.Type)
	}

	// Build summarizer
	var s summarizer.Summarizer
	switch cfg.Summarizer.Type {
	case "anthropic":
		s = summarizer.NewAnthropicSummarizer(
			cfg.Summarizer.APIKey,
			cfg.Summarizer.Model,
			cfg.Summarizer.MaxTokens,
			cfg.TopN,
			cfg.Topic,
		)
	default:
		log.Fatalf("Unknown summarizer type: %s", cfg.Summarizer.Type)
	}

	// Build publishers
	var pubs []publisher.Publisher
	var webPub *publisher.WebPublisher

	switch cfg.Publisher.Type {
	case "stdout":
		pubs = append(pubs, publisher.NewStdoutPublisher())
	case "email":
		pubs = append(pubs, publisher.NewEmailPublisher(
			cfg.Publisher.Email.SMTPHost,
			cfg.Publisher.Email.SMTPPort,
			cfg.Publisher.Email.Username,
			cfg.Publisher.Email.Password,
			cfg.Publisher.Email.From,
			cfg.Publisher.Email.To,
		))
	case "web":
		webPub = publisher.NewWebPublisher(cfg.Publisher.Web.Addr)
		pubs = append(pubs, webPub)
	case "discord":
		pubs = append(pubs, publisher.NewDiscordPublisher(cfg.Publisher.Discord.WebhookURL))
	default:
		log.Fatalf("Unknown publisher type: %s", cfg.Publisher.Type)
	}

	// Start web server if configured
	if webPub != nil {
		if err := webPub.Start(); err != nil {
			log.Fatalf("Failed to start web publisher: %v", err)
		}
	}

	// Build runner
	r := runner.New(cfg.Topic, cfg.MaxResults, f, s, pubs)

	// Single-run mode: run the pipeline once and exit
	if *once {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		log.Println("Running digest (once mode)...")
		if err := r.Run(ctx); err != nil {
			log.Fatalf("Pipeline failed: %v", err)
		}
		log.Println("Done")
		return
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run immediately on startup if configured
	if cfg.RunOnStart {
		log.Println("Running initial digest...")
		if err := r.Run(ctx); err != nil {
			log.Printf("Initial run failed: %v", err)
		}
	}

	// Set up cron scheduler
	c := cron.New()
	_, err = c.AddFunc(cfg.Schedule, func() {
		log.Println("Cron triggered, running digest...")
		if err := r.Run(ctx); err != nil {
			log.Printf("Scheduled run failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to set up cron schedule %q: %v", cfg.Schedule, err)
	}
	c.Start()
	log.Printf("Scheduled digest with cron expression: %s", cfg.Schedule)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	cancel()
	c.Stop()

	if webPub != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := webPub.Shutdown(shutdownCtx); err != nil {
			log.Printf("Web server shutdown error: %v", err)
		}
	}

	log.Println("Shutdown complete")
}

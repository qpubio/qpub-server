package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qpubio/qpub-server/internal/bootstrap"
	logType "github.com/qpubio/qpub-server/internal/shared/type/log"
)

// qpub-server data-plane entrypoint.
func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fatalColor := logType.GetColorForLevel(logType.Fatal)

	app, err := bootstrap.New()
	if err != nil {
		log.Fatalf("%s[FATAL]\033[0m Failed to initialize application: %v", fatalColor, err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("%s[FATAL]\033[0m Failed to start application: %v", fatalColor, err)
	}

	<-sigChan

	if err := app.Shutdown(); err != nil {
		log.Fatalf("%s[FATAL]\033[0m Failed to stop application gracefully: %v", fatalColor, err)
	}
}

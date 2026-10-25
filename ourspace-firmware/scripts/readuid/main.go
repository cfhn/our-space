package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cfhn/our-space/ourspace-firmware/internal/cardreader"
)

var ErrInvalidParameter = errors.New("invalid value")

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}

func run() error {
	flag.Parse()

	flagMode := flag.String("mode", "oneshot", "oneshot|listen: oneshot waits for one card to be scanned, prints the uid and exits. listen keeps waiting until stopped via ctrl+c")

	if *flagMode != "oneshot" && *flagMode != "listen" {
		return fmt.Errorf("%w for -mode: %s", ErrInvalidParameter, *flagMode)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	reader, err := cardreader.New(logger)
	if err != nil {
		fmt.Println(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL)

	go func() {
		err = reader.StartScanning(ctx)
		if err != nil {
			fmt.Println(err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-reader.Events:
			if event.Type == cardreader.CardEventTypeAdd {
				fmt.Printf("%s", strings.ToUpper(hex.EncodeToString(event.UID)))

				if *flagMode == "oneshot" {
					return nil
				}

				fmt.Println()
			}
		}
	}
}

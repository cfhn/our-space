package cardreader

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"paepcke.de/pcscid/pcsc"
)

var ErrShortUID = errors.New("response shorter than uid length")

type CardType int

const (
	cardTypeMifareClassic1K CardType = iota
	cardTypeMifareDesfireEv1
	cardTypeMifareUltralight
)

type CardInfo struct {
	UIDLength uint32
}

type CardEventType int

const (
	CardEventTypeAdd CardEventType = iota
	CardEventTypeRemove
)

type CardEvent struct {
	Type CardEventType
	UID  []byte
}

//nolint:gochecknoglobals // quasi-constants, since Go does not allow slice/map consts
var (
	mifareClassicATR = []byte{
		0x3B, 0x8F, 0x80, 0x01, 0x80, 0x4F, 0x0C, 0xA0, 0x00, 0x00, 0x03, 0x06, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x6A,
	}
	mifareDesfireEv1ATR = []byte{
		0x3B, 0x81, 0x80, 0x01, 0x80, 0x80,
	}
	mifareUltralightATR = []byte{
		0x3B, 0x8F, 0x80, 0x01, 0x80, 0x4F, 0x0C, 0xA0, 0x00, 0x00, 0x03, 0x06, 0x03, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x68,
	}

	cardInfo = map[CardType]CardInfo{
		cardTypeMifareClassic1K: {
			UIDLength: 4,
		},
		cardTypeMifareDesfireEv1: {
			UIDLength: 7,
		},
		cardTypeMifareUltralight: {
			UIDLength: 7,
		},
	}
)

type Reader struct {
	client *pcsc.Client
	logger *slog.Logger

	eventsPublisher chan<- CardEvent
	Events          <-chan CardEvent
}

func New(logger *slog.Logger) (*Reader, error) {
	client, err := pcsc.New("", logger.With(slog.String("module", "pcsc")))
	if err != nil {
		return nil, err
	}

	eventsChan := make(chan CardEvent, 1)

	return &Reader{
		client:          client,
		logger:          logger,
		eventsPublisher: eventsChan,
		Events:          eventsChan,
	}, nil
}

func (r *Reader) StartScanning(ctx context.Context) error {
	readerStates := make(map[string]pcsc.ReaderState)
	notificationSent := make(map[string]bool)

	states, err := r.client.States()
	if err != nil {
		return err
	}

	for _, readerState := range states {
		readerStates[readerState.Reader] = readerState
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := r.client.WaitChange(1 * time.Minute)
		if errors.Is(err, pcsc.ErrTimeout) {
			continue
		}

		if err != nil {
			r.logger.ErrorContext(ctx, "waiting for card change", slog.Any("error", err))
			continue
		}

		states, err := r.client.States()
		if err != nil {
			r.logger.ErrorContext(ctx, "getting card reader states", slog.Any("error", err))
			continue
		}

		for _, readerState := range states {
			if readerStates[readerState.Reader].EventCounter != readerState.EventCounter {
				readerStates[readerState.Reader] = readerState

				if readerState.State&pcsc.ReaderPresent != 0 {
					var cardType CardType

					switch {
					case bytes.Equal(readerState.ATR, mifareClassicATR):
						cardType = cardTypeMifareClassic1K
					case bytes.Equal(readerState.ATR, mifareDesfireEv1ATR):
						cardType = cardTypeMifareDesfireEv1
					case bytes.Equal(readerState.ATR, mifareUltralightATR):
						cardType = cardTypeMifareUltralight
					default:
						r.logger.WarnContext(ctx, "unknown ATR", slog.String("atr", hex.EncodeToString(readerState.ATR)))
						continue
					}

					cardSerial, err := r.readCardSerial(readerState.Reader, cardType)
					if err != nil {
						r.logger.ErrorContext(ctx, "reading card serial", slog.Any("error", err))
						continue
					}

					notificationSent[readerState.Reader] = true

					r.eventsPublisher <- CardEvent{
						Type: CardEventTypeAdd,
						UID:  cardSerial,
					}
				}

				if readerState.State&pcsc.ReaderAbsent != 0 {
					if !notificationSent[readerState.Reader] {
						continue
					}

					notificationSent[readerState.Reader] = false

					r.eventsPublisher <- CardEvent{
						Type: CardEventTypeRemove,
					}
				}
			}
		}
	}
}

func (r *Reader) readCardSerial(reader string, cardType CardType) ([]byte, error) {
	info := cardInfo[cardType]

	card, err := r.client.Connect(reader, pcsc.ProtocolAny)
	if err != nil {
		return nil, err
	}
	defer card.Disconnect(pcsc.ResetCard) //nolint:errcheck // can't do anything about this anyway

	resp, err := card.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00}, 0)
	if err != nil {
		return nil, err
	}

	if uint64(len(resp)) < uint64(info.UIDLength) {
		return nil, fmt.Errorf("%w: %d < %d", ErrShortUID, len(resp), info.UIDLength)
	}

	return resp[:info.UIDLength], nil
}

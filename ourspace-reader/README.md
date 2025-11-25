# NFC Reader

The reader sends the UID of the presented card to the backend and lights up differently, depending on the response.

The hardware is a repurposed NFC reader consisting of a Freetronics EtherTen (Arduino Uno + W5100 Ethernet + PoE), SK6812 LED ring and a "behrens elektronik" PN5180 NFC reader module (simply prints the read UID via Serial).

The software is mostly copied from: https://github.com/maker-space-experimenta/nfc-checkin-terminal

## Hardware notes
- To flash via USB Serial, remember to unplug the NFC reader from pins 0/1. As it uses hardware serial too, it interfers with the flashing.


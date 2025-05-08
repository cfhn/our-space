#include <Arduino.h>
#include <SPI.h>
#include <Wire.h>

#include "config.h"
#include "led.h"

#include <Ethernet.h>
#include <Dhcp.h>

const String terminalId = CONFIG_TERMINAL_ID;
const int heartbeat_intervall = CONFIG_HEARTBEAT_INTERVAL;
uint8_t macAddr[] = CONFIG_MAC;

EthernetClient client;

uint32_t requestSent = 0;
char responseBuf[512];
uint16_t responseBufIdx = 0;

void sendUidToServer(const char *uid) {
    setAnimation(ANIM_CARD_PROCESSING);
    animationLoop(true);

    requestSent = millis();
    responseBufIdx = 0;
    memset(responseBuf, 0, sizeof(responseBuf));
    
    if (client.connect(CONFIG_BACKEND_HOST, 80)) {
        client.println(F("POST " CONFIG_BACKEND_PATH " HTTP/1.1"));
        client.println(F("Host: " CONFIG_BACKEND_HOST));
        client.println(F("User-Agent: arduino-ethernet"));
        client.println(F("Content-Type: application/json"));
        client.println(F("Connection: close"));
        client.println();
        
        char payload[100] = {0};
        snprintf_P(payload, sizeof(payload), PSTR("{\"uid\": \"%s\", \"terminalId\": \"%s\"}"), uid, CONFIG_TERMINAL_ID);
        client.println(payload);
    }
}

void checkHttpResponse() {
    while (client.available()) {
        int bytesToRead = client.available();
        if (bytesToRead + responseBufIdx < sizeof(responseBuf)) {
            client.readBytes(responseBuf + responseBufIdx, bytesToRead);
        }
    }

    
    if (!client.connected() && requestSent != 0) {  // If client got disconnected and a response is expected
        // handle response

        if (strstr(responseBuf, "added") != NULL) {
            setAnimation(ANIM_CHECK_IN);
        } else if (strstr(responseBuf, "removed") != NULL) {
            setAnimation(ANIM_CHECK_OUT);
        } else if (strstr(responseBuf, "unknown") != NULL) {
            setAnimation(ANIM_UNKNOWN_CARD);
        } else {
            setAnimation(ANIM_ERROR, 3000, true);
        }
        requestSent = 0;    // mark handling done
    }
}

#define DBG Serial
bool ethInit() {
    bool success = true;
    Ethernet.init();
    // DHCP
    if (Ethernet.begin(macAddr, 10000) == 0) {
        DBG.println("Failed to configure Ethernet using DHCP");
        success = false;
    }
    // Static IP
    // Ethernet.begin(macAddr, IPAddress(192,168,13,245));

    if (Ethernet.hardwareStatus() == EthernetNoHardware) {
        DBG.println("Ethernet module was not found.  Sorry, can't run without hardware. :(");
    } else if (Ethernet.linkStatus() == LinkOFF) {
        DBG.println("Ethernet cable is not connected.");
        success = false;
    }
    
    if(success) {
        // print your local IP address:
        DBG.print("My IP address: ");
        DBG.println(Ethernet.localIP());
    }
    return success;

}

void ethLoop() {
    int ret = Ethernet.maintain();
    switch (ret) {
        case DHCP_CHECK_RENEW_FAIL:
        case DHCP_CHECK_REBIND_FAIL:
            setAnimation(ANIM_CONNECTING);
            break;
        case DHCP_CHECK_RENEW_OK:
        case DHCP_CHECK_REBIND_OK:
            setAnimation(ANIM_IDLE);
            break;
    }
}


void setup() {
    Serial.begin(9600);

    initLeds();

    // eth init
    setAnimation(ANIM_CONNECTING);
    animationLoop(true);
    if (ethInit()) {
        setAnimation(ANIM_IDLE);
    }
    else {
        setAnimation(ANIM_ERROR);
        animationLoop(true);
    }
}

uint32_t lastSerialRx = 0;
const int SERIAL_TIMEOUT = 10;
uint8_t rxBufIdx = 0;
char rxBuf[64];

void loop() {
    ethLoop();
    animationLoop();
    checkHttpResponse();

    while (Serial.available()) {
        if (millis() - lastSerialRx > SERIAL_TIMEOUT) {
            rxBufIdx = 0;
            memset(rxBuf, 0, sizeof(rxBuf));
        }

        lastSerialRx = millis();
        
        // read Serial into buffer  
        int bytesToRead = Serial.available();
        if (bytesToRead + rxBufIdx+1 > sizeof(rxBuf)) {
            bytesToRead = sizeof(rxBuf) - rxBufIdx+1;
        }
        rxBufIdx += Serial.readBytes(rxBuf + rxBufIdx, bytesToRead); 

        // full UID received from reader (ascii hex encoded UID received from reader (wrong byte order))
        if (rxBufIdx >= 16) {
            if (rxBuf[14] == '\r' && rxBuf[15] == '\n') {
                char uid[15] = {0};
                for (int i = 0; i < 7; i++) {
                    uid[12 - i*2] = rxBuf[i*2];
                    uid[12 - i*2 + 1] = rxBuf[i*2 + 1];
                } 
                sendUidToServer(uid);
            }
            lastSerialRx = 0;   // force buffer clear (discards spurious data, probably)
        }
    }

    
}
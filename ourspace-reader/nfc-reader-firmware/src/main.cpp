#include <Arduino.h>
#include <SPI.h>
#include <Wire.h>

#include "config.h"
#include "led.h"

#include <Ethernet.h>
#include <Dhcp.h>

const int heartbeat_intervall = CONFIG_HEARTBEAT_INTERVAL;
uint8_t macAddr[] = CONFIG_MAC;

EthernetClient client;

uint32_t requestSent = 0;
char responseBuf[512];
uint16_t responseBufIdx = 0;

// like client.write, but takes pointer to string in program space
void client_write_P(const char *buffer_P, size_t size) {
    char buffer[size];
    memcpy_P(buffer, buffer_P, size);
    client.write(buffer, size);
}

void sendUidToServer(const char *uid) {
    setAnimation(ANIM_CARD_PROCESSING);
    animationLoop(true);

    requestSent = millis();
    responseBufIdx = 0;
    memset(responseBuf, 0, sizeof(responseBuf));
    
    if (client.connect(CONFIG_BACKEND_HOST, CONFIG_BACKEND_PORT)) {
        // Build body
        char body[128] = {0};
        int bodyLen = snprintf_P(body, sizeof(body), PSTR("{\"card_serial\": \"%s\", \"terminalId\": \"%s\"}\r\n"), uid, CONFIG_TERMINAL_ID);
        bodyLen = min(sizeof(body), bodyLen);
        
        // "Build" and send static part of header
        static const char headerStatic[] PROGMEM = 
            "POST " CONFIG_BACKEND_PATH " HTTP/1.1\r\n"
            "Host: " CONFIG_BACKEND_HOST "\r\n"
            "User-Agent: arduino-ethernet\r\n"
            "Content-Type: application/json\r\n"
            "Connection: close\r\n"
            "Content-Length: ";
        client_write_P(headerStatic, sizeof(headerStatic) - 1);
        
        // Build dynamic part of header (Content Length)
        char headerDynamic[16];
        int headerDynamicLen = snprintf(headerDynamic, sizeof(headerDynamic), "%d\r\n\r\n", bodyLen);
        headerDynamicLen = min(sizeof(headerDynamic), headerDynamicLen);

        // Send dynamic header + body
        client.write(headerDynamic, headerDynamicLen);
        client.write(body, bodyLen);
    }
}

void checkHttpResponse() {
    while (client.available()) {
        int bytesToRead = client.available();
        if (bytesToRead + responseBufIdx < sizeof(responseBuf)) {
            responseBufIdx += client.readBytes(responseBuf + responseBufIdx, bytesToRead);
        }
        else {  
            // handle buffer too small
            while(client.available()) {
                client.read();  // discard buffer
            }

        }
    }

    
    if (!client.connected() && requestSent != 0) {  // If client got disconnected and a response is expected
        // handle response
        if (strstr(responseBuf,         "checkin"           ) != NULL)  { setAnimation(ANIM_CHECK_IN); }
        else if (strstr(responseBuf,    "checkout"          ) != NULL)  { setAnimation(ANIM_CHECK_OUT); }
        else if (strstr(responseBuf,    "member-not-found"  ) != NULL)  { setAnimation(ANIM_UNKNOWN_CARD); }
        else if (strstr(responseBuf,    "card-not-found"    ) != NULL)  { setAnimation(ANIM_UNKNOWN_CARD); }
        else {
            setAnimation(ANIM_ERROR, 3000, true);
            // Serial.println("Response:");
            // Serial.write(responseBuf);
            // Serial.println();
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
        DBG.println(F("Failed to configure Ethernet using DHCP"));
        success = false;
    }
    // Static IP
    // Ethernet.begin(macAddr, IPAddress(192,168,13,245));

    if (Ethernet.hardwareStatus() == EthernetNoHardware) {
        DBG.println(F("Ethernet module was not found.  Sorry, can't run without hardware. :("));
    } else if (Ethernet.linkStatus() == LinkOFF) {
        DBG.println(F("Ethernet cable is not connected."));
        success = false;
    }
    
    if(success) {
        // print your local IP address:
        DBG.print(F("My IP address: "));
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
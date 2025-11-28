#if defined(ESP_PLATFORM)
#define HW_ESP
#define HW_NFC_PN532

// WiFi-Credentials
#define CONFIG_WIFI_SSID "<<wlan-ssid>>"
#define CONFIG_WIFI_PASSWORD "<<wlan-password>>"

// backend url
#define CONFIG_BACKEND_URL "https://<<server>>"
#define CONFIG_BACKEND_URL_ALIVE "https://<<alive_url>>"

// finger print for your ssl-secured backend
#define CONFIG_SSL_FINGERPRINT { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 }

// id to idendifiy data source
#define CONFIG_TERMINAL_ID "<<human-readable-terminal-id>>"
#endif

// Ethernet MAC address
#define CONFIG_MAC {0x42, 0x10, 0xEC, 0x3C, 0x9D, 0xB3 }

// Host and path of backend
#define CONFIG_BACKEND_HOST         "<<server>>"
#define CONFIG_BACKEND_PORT         80
#define CONFIG_BACKEND_PATH         "/scan"
// #define CONFIG_BACKEND_PATH_ALIVE   "/heartbeat" // not implemented yet

// id to idendifiy data source
#define CONFIG_TERMINAL_ID "<<human-readable-terminal-id>>"
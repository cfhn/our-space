#pragma once

#include <Arduino.h>
#include <FastLED.h>

#define LED_PIN     0
#define NUM_LEDS    12
#define BRIGHTNESS  64
#define LED_TYPE    WS2811
#define COLOR_ORDER GRB
#define ANIM_SPEED  25 // animation speed in ms
#define ANIM_SCALE  64 // pulsing animation scaling

typedef enum animationStage {
    ANIM_IDLE,
    ANIM_CARD_PROCESSING,
    ANIM_ERROR,
    ANIM_CHECK_IN,
    ANIM_CHECK_OUT,
    ANIM_UNKNOWN_CARD,
    ANIM_BLACK,
    ANIM_CONNECTING,
} animation_t;


extern CRGB leds[NUM_LEDS];

extern uint32_t lastAnimStep;
extern uint8_t animCounter;
extern animation_t currentAnimation;


void initLeds();
void animationLoop(bool forceUpdate = false);

void setAnimation(animation_t anim);
void setAnimation(animation_t anim, uint16_t duration, bool backToPrevious = false);
void setColor(CRGB::HTMLColorCode color);

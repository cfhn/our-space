#pragma once

#include <Arduino.h>
#include <Adafruit_NeoPixel.h>

#define LED_PIN     2
#define NUM_LEDS    24
#define BRIGHTNESS  64
#define LED_TYPE    (NEO_GRBW + NEO_KHZ800)
#define ANIM_SPEED  25 // animation speed in ms
#define ANIM_SCALE  128 // pulsing animation scaling

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


extern uint32_t lastAnimStep;
extern uint8_t animCounter;
extern animation_t currentAnimation;


void initLeds();
void animationLoop(bool forceUpdate = false);

void setAnimation(animation_t anim);
void setAnimation(animation_t anim, uint16_t duration, bool backToPrevious = false);

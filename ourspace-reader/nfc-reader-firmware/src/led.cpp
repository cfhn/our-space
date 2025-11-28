#include "led.h"

// CRGB leds[NUM_LEDS];
uint32_t lastAnimStep = 0, animationTimeout;
uint8_t animCounter = 0;
animation_t currentAnimation = ANIM_IDLE, lastAnimation = ANIM_IDLE;

#define HUE_PURPLE (300 * 65535 / 360)

Adafruit_NeoPixel strip(NUM_LEDS, LED_PIN, NEO_GRBW + NEO_KHZ800);

void initLeds() {
    strip.begin();
    strip.show();
    strip.setBrightness(BRIGHTNESS);

    lastAnimStep = 0;
    animCounter = 0;
}

void setAnimation(animation_t anim) {
    animCounter = 0;
    currentAnimation = anim;
}

void setAnimation(animation_t anim, uint16_t duration, bool backToPrevious) {
    lastAnimation = backToPrevious ? lastAnimation : ANIM_IDLE;
    animationTimeout = millis() + duration;

    setAnimation(anim);
}

void animationLoop(bool forceUpdate) {
    if (millis() - lastAnimStep >= ANIM_SPEED || forceUpdate) {
        lastAnimStep = millis();

        // check for end of animation
        if(animationTimeout != 0 && millis() >= animationTimeout) {
            animationTimeout = 0;
            setAnimation(lastAnimation);
        }


        switch(currentAnimation) {
            case ANIM_IDLE: {           // rainbow pulse & rotate
                uint8_t val = animCounter % ANIM_SCALE;                // limit counter
                val = val < (ANIM_SCALE / 2) ? val : ANIM_SCALE - val; // map to up/down counting
                val = map(val, 0, ANIM_SCALE / 2, 100, 255);           // map to full brightness

                for (uint8_t i = 0; i < NUM_LEDS; i++) {
                    uint8_t hue = (animCounter + (255 / NUM_LEDS) * i) % 256; // hue rainbow
                    strip.setPixelColor(i, strip.gamma32(strip.ColorHSV(hue * 256, 255, val)));
                }
            }
            break;

            case ANIM_CARD_PROCESSING: {// blue light chaser
                bool half = (animCounter % 16) < 8;
                for (uint8_t i = 0; i < NUM_LEDS; i++) {
                    strip.setPixelColor(i, (half ^ (i % 2 == 0)) ? 0x0000FF : 0x000000);
                }
            }
            break;

            case ANIM_ERROR: {      // red blinking
                bool blinkOn = (animCounter % 16) < 8;
                strip.fill(blinkOn ? 0xFF0000 : 0x000000);
            }
            break;

            case ANIM_CHECK_IN: {   // green progress animation, 1s black, then idle
                uint8_t progress = animCounter / 2;
                // if animation done
                if (progress >= NUM_LEDS) {
                    setAnimation(ANIM_BLACK, 500);
                    break;
                }
                for (uint8_t i = 0; i < NUM_LEDS; i++) {
                    strip.setPixelColor(i, (i <= progress) ? 0x00FF00 : 0x000000);
                }
            }
            break;

            case ANIM_CHECK_OUT: {  // orange inverse progress animation, 1s black, then idle
                uint8_t progress = animCounter / 2;
                // if animation done
                if (progress >= NUM_LEDS) {
                    setAnimation(ANIM_BLACK, 500);
                    break;
                }
                for (uint8_t i = 0; i < NUM_LEDS; i++) {
                    strip.setPixelColor(i, (i <= (NUM_LEDS-1 - progress)) ? 0xFF8000 : 0x000000);
                }
            }
            break;

            case ANIM_UNKNOWN_CARD: {// purple single breath, 1s black, then idle
                uint8_t val = animCounter < 10 ? animCounter : 20 - animCounter;
                val *= (256 / 10);
                if (animCounter >= 20) {
                    setAnimation(ANIM_BLACK, 500);
                }
                strip.fill(strip.gamma32(strip.ColorHSV(HUE_PURPLE)));
            }
            break;

            case ANIM_BLACK: {      // black
                strip.fill(0);
            }
            break;

            case ANIM_CONNECTING: { // chasing white dot
                uint8_t ledToLight = (animCounter / 2) % NUM_LEDS;
                for (uint8_t i = 0; i < NUM_LEDS; i++) {
                    strip.setPixelColor(i, (i == ledToLight) ? 0xFFFFFF : 0);
                }
            }
            break;
        }

        strip.show();
        animCounter++;
    }
}
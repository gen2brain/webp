#include <stdlib.h>
#include <string.h>

#include "webp/decode.h"
#include "webp/encode.h"
#include "webp/demux.h"

int decode(uint8_t *webp_in, int webp_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint32_t *animation, uint8_t *delay, uint8_t *out);
uint8_t* encode(uint8_t *rgb_in, int width, int height, size_t *size, int colorspace, int quality, int method, int lossless, int exact);

int decode(uint8_t *webp_in, int webp_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint32_t *animation, uint8_t *delay, uint8_t *out) {

    WebPData data;
    data.bytes = webp_in;
    data.size = webp_in_size;

    WebPDecoderConfig config;
    if(!WebPInitDecoderConfig(&config)) {
        return 0;
    }

    if(WebPGetFeatures(data.bytes, data.size, &config.input) != VP8_STATUS_OK) {
        return 0;
    }

    *width = config.input.width;
    *height = config.input.height;
    *animation = config.input.has_animation;

    if(config_only && !decode_all) {
        *count = 1;

        return 1;
    }

    if(decode_all || config.input.has_animation) {
        WebPAnimDecoderOptions options;
        WebPAnimDecoderOptionsInit(&options);
        options.color_mode = MODE_rgbA;

        WebPAnimDecoder* dec = WebPAnimDecoderNew(&data, &options);

        WebPAnimInfo info;
        WebPAnimDecoderGetInfo(dec, &info);

        *count = info.frame_count;

        if(config_only) {
            WebPAnimDecoderDelete(dec);
            return 1;
        }

        int frame = 0, timestamp = 0, timestampPrev = 0, duration = 0;

        uint8_t* buf;
        int buf_size = info.canvas_width * info.canvas_height * 4;

        while(WebPAnimDecoderHasMoreFrames(dec)) {
            if(!WebPAnimDecoderGetNext(dec, &buf, &timestamp)) {
                WebPAnimDecoderDelete(dec);
                return 0;
            }

            memcpy(out + buf_size*frame, buf, buf_size);

            duration = timestamp - timestampPrev;
            memcpy(delay + sizeof(int)*frame, &duration, sizeof(int));
            timestampPrev = timestamp;

            if(!decode_all) {
                break;
            }

            frame++;
        }

        WebPAnimDecoderDelete(dec);
        return 1;
    }

    int w = *width;
    int h = *height;
    int cw = (w+1)/2;
    int ch = (h+1)/2;

    int i0 = 1*w*h + 0*cw*ch;
    int i1 = 1*w*h + 1*cw*ch;
    int i2 = 1*w*h + 2*cw*ch;
    int i3 = 2*w*h + 2*cw*ch;

    config.output.colorspace = MODE_YUVA;
    config.output.is_external_memory = 1;

    config.output.u.YUVA.y = &out[0];
    config.output.u.YUVA.y_size = i0;
    config.output.u.YUVA.y_stride = w;

    config.output.u.YUVA.u = &out[i0];
    config.output.u.YUVA.u_size = i1;
    config.output.u.YUVA.u_stride = cw;

    config.output.u.YUVA.v = &out[i1];
    config.output.u.YUVA.v_size = i2;
    config.output.u.YUVA.v_stride = cw;

    config.output.u.YUVA.a = &out[i2];
    config.output.u.YUVA.a_size = i3;
    config.output.u.YUVA.a_stride = w;

    if(WebPDecode(data.bytes, data.size, &config) != VP8_STATUS_OK) {
        WebPFreeDecBuffer(&config.output);
        return 0;
    }

    WebPFreeDecBuffer(&config.output);
    return 1;
}

uint8_t* encode(uint8_t *in, int w, int h, size_t *size, int colorspace, int quality, int method, int lossless, int exact) {
    uint8_t *out = NULL;

    WebPConfig config;
    if(!WebPConfigInit(&config)) {
        return out;
    }

    config.quality = quality;
    config.method = method;
    config.lossless = lossless;
    config.exact = exact;

    int cw = (w+1)/2;
    int ch = (h+1)/2;

    int i0 = 1*w*h + 0*cw*ch;
    int i1 = 1*w*h + 1*cw*ch;
    int i2 = 1*w*h + 2*cw*ch;

    WebPPicture picture;
    if(!WebPPictureInit(&picture)) {
        return out;
    }

    picture.width = w;
    picture.height = h;

    if(colorspace == WEBP_YUV420A) {
        picture.use_argb = 0;
        picture.colorspace = colorspace;
        picture.y = &in[0];
        picture.u = &in[i0];
        picture.v = &in[i1];
        picture.a = &in[i2];
        picture.y_stride = w;
        picture.uv_stride = cw;
        picture.a_stride = w;
    } else {
        picture.use_argb = 1;
        picture.argb_stride = w * 4;

        if(!WebPPictureImportRGBA(&picture, in, picture.argb_stride)) {
            WebPPictureFree(&picture);
            return out;
        }
    }

    WebPMemoryWriter writer;
    picture.writer = WebPMemoryWrite;
    picture.custom_ptr = &writer;
    WebPMemoryWriterInit(&writer);

    if(!WebPEncode(&config, &picture)) {
        WebPPictureFree(&picture);
        WebPMemoryWriterClear(&writer);
        return out;
    }

    *size = writer.size;
    out = writer.mem;

    WebPPictureFree(&picture);
    writer.mem = NULL;
    WebPMemoryWriterClear(&writer);

    return out;
}

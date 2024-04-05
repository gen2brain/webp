#include <stdlib.h>
#include <string.h>

#include "webp/decode.h"
#include "webp/encode.h"
#include "webp/demux.h"

void* allocate(size_t size);
void deallocate(void *ptr);

int decode(uint8_t *webp_in, int webp_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint8_t *delay, uint8_t *out);
uint8_t* encode(uint8_t *rgb_in, int width, int height, size_t *size, int colorspace, int quality, int method, int lossless, int exact);

int decode(uint8_t *webp_in, int webp_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint8_t *delay, uint8_t *out) {

    if(!WebPGetInfo(webp_in, webp_in_size, (int *)width, (int *)height)) {
        return 0;
    }

    if(config_only && !decode_all) {
        *count = 1;
        return 1;
    }

    WebPData data;
    data.bytes = webp_in;
    data.size = webp_in_size;

    WebPDemuxer* demux = WebPDemux(&data);
    *count = WebPDemuxGetI(demux, WEBP_FF_FRAME_COUNT);

    if(config_only) {
        WebPDemuxDelete(demux);
        return 1;
    }

    WebPDecoderConfig config;
    if(!WebPInitDecoderConfig(&config)) {
        WebPDemuxDelete(demux);
        return 0;
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

    WebPIterator iter;
    if(WebPDemuxGetFrame(demux, 1, &iter)) {
        do {
            if(WebPDecode(iter.fragment.bytes, iter.fragment.size, &config) != VP8_STATUS_OK) {
                WebPFreeDecBuffer(&config.output);
                WebPDemuxDelete(demux);
                return 0;
            }

            memcpy(delay + sizeof(int)*(iter.frame_num-1), &iter.duration, sizeof(int));

            WebPFreeDecBuffer(&config.output);

            if(!decode_all) {
                break;
            }
        } while(WebPDemuxNextFrame(&iter));

        WebPDemuxReleaseIterator(&iter);
    } else {
        WebPDemuxDelete(demux);
        return 0;
    }

    WebPDemuxDelete(demux);
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

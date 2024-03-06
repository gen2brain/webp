#include <stdlib.h>
#include <string.h>

#include "webp/decode.h"
#include "webp/demux.h"

void *allocate(size_t size);
void deallocate(void *ptr);

int decode(uint8_t *avif_in, int avif_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint8_t *delay, uint8_t *rgb_out);

__attribute__((export_name("allocate")))
void *allocate(size_t size) {
    return malloc(size);
}

__attribute__((export_name("deallocate")))
void deallocate(void *ptr) {
    free(ptr);
}

__attribute__((export_name("decode")))
int decode(uint8_t *avif_in, int avif_in_size, int config_only, int decode_all, uint32_t *width, uint32_t *height, uint32_t *count, uint8_t *delay, uint8_t *rgb_out) {
    if(!WebPGetInfo(avif_in, avif_in_size, NULL, NULL)) {
        return 0;
    }

    WebPData data;
    data.bytes = avif_in;
    data.size = avif_in_size;

    WebPDemuxer* demux = WebPDemux(&data);
    *width = WebPDemuxGetI(demux, WEBP_FF_CANVAS_WIDTH);
    *height = WebPDemuxGetI(demux, WEBP_FF_CANVAS_HEIGHT);
    *count = WebPDemuxGetI(demux, WEBP_FF_FRAME_COUNT);

    if(config_only) {
        WebPDemuxDelete(demux);
        return 1;
    }
            
    int buf_size = *width * *height * 4;

    WebPIterator iter;
    if(WebPDemuxGetFrame(demux, 1, &iter)) {
        do {

            uint8_t* image = WebPDecodeRGBA(iter.fragment.bytes, iter.fragment.size, NULL, NULL);
            memcpy(rgb_out + buf_size*(iter.frame_num-1), image, buf_size);

            memcpy(delay + sizeof(int)*(iter.frame_num-1), &iter.duration, sizeof(int));
            
            WebPFree(image);

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

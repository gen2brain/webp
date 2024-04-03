LIBWEBP_VERSION = v1.3.2

LIBWEBP_SRC = $(PWD)/libwebp
LIBWEBP_BUILD = $(LIBWEBP_SRC)/build

WASI_SDK_PATH ?= /opt/wasi-sdk
export CC = $(WASI_SDK_PATH)/bin/clang --sysroot=$(WASI_SDK_PATH)/share/wasi-sysroot
export CFLAGS = -msimd128

CMAKE_TOOLCHAIN_FILE=$(WASI_SDK_PATH)/share/cmake/wasi-sdk.cmake

BIN := webp.wasm

all: $(BIN)

$(LIBWEBP_SRC):
	git clone -b $(LIBWEBP_VERSION) --depth 1 --recursive --jobs `nproc` https://github.com/webmproject/libwebp
	mkdir -p $(LIBWEBP_BUILD)
	test -d $@

$(LIBWEBP_BUILD)/libwebp.a: $(LIBWEBP_SRC)
	cd $(LIBWEBP_BUILD); \
	cmake $(LIBWEBP_SRC) \
		-DCMAKE_BUILD_TYPE=Release \
		-DBUILD_SHARED_LIBS=0 \
		-DWEBP_ENABLE_SIMD_DEFAULT=1 \
		-DWEBP_BUILD_EXTRAS=0 \
		-DWEBP_USE_THREAD=0 \
		-DWEBP_BUILD_ANIM_UTILS=0 \
		-DWEBP_BUILD_CWEBP=0 \
		-DWEBP_BUILD_DWEBP=0 \
		-DWEBP_BUILD_IMG2WEBP=0 \
		-DWEBP_BUILD_WEBPINFO=0 \
		-DWEBP_BUILD_WEBPMUX=0 \
		-DCMAKE_TOOLCHAIN_FILE=$(CMAKE_TOOLCHAIN_FILE)

	cd $(LIBWEBP_BUILD); \
	$(MAKE) -j$(shell nproc) VERBOSE=1

$(BIN): $(LIBWEBP_BUILD)/libwebp.a
	$(CC) \
		-O3 \
		-Wl,--no-entry \
		-Wl,--export=malloc \
		-Wl,--export=free \
		-Wl,--export=decode \
		-Wl,--export=encode \
		-mexec-model=reactor \
		-mnontrapping-fptoint \
		-I${LIBWEBP_SRC}/src \
		-I${LIBWEBP_BUILD}/src \
		-o $@ \
		-Wall \
		webp.c \
		${LIBWEBP_BUILD}/libwebpdemux.a \
		${LIBWEBP_BUILD}/libwebp.a \
		${LIBWEBP_BUILD}/libsharpyuv.a

.PHONY: clean

clean:
	-rm -rf $(LIBWEBP_SRC)

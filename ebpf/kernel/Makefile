# SPDX-License-Identifier: GPL-2.0

CLANG ?= clang
LLC ?= llc

CFLAGS += -D__clang__ -D__KERNEL__ -Wno-unused-value -Wno-pointer-sign \
	-D__TARGET_ARCH_x86 -D__LITTLE_ENDIAN_BITFIELD -Wno-compare-distinct-pointer-types \
	-Wno-gnu-variable-sized-type-not-at-end \
	-Wno-address-of-packed-member -Wno-tautological-compare \
	-Wno-unknown-warning-option

INCLUDES = -I./include -I /usr/include/x86_64-linux-gnu

OBJ = bpf_cgroup_kern.o

%.o: %.c
	$(CLANG) $(CFLAGS) $(INCLUDES) -target bpf -O2 -emit-llvm -c $< -g -o - | \
	$(LLC) -march=bpf -mattr=dwarfris -filetype=obj -o $@
	pahole -J $@

all:	$(OBJ)

.PHONY: install
install: 
	install -d /usr/local/bin/bpf_cgroup_kern.o

.PHONY: clean
clean:
	rm -f *.o


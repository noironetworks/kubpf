# As of now expectation is that ./build.sh is invoked first to place the artifacts
# in ./out directory
FROM docker.io/library/ubuntu:20.04 as runtime-base

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
    libelf1 && \
    apt-get purge --auto-remove && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

FROM runtime-base as runtime

RUN mkdir -p /bin
COPY out/bpftool /bin/
COPY out/bpf_cgroup_kern.o /bin/
COPY out/statsagent /bin/
CMD /bin/statsagent


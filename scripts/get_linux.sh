curl --fail --show-error --silent --location "https://kernel.googlesource.com/pub/scm/linux/kernel/git/bpf/bpf-next/+archive/9ff9b0d392ea08090cd1780fb196f36dbb586529.tar.gz" --output /tmp/linux.tgz
mkdir -p /src/linux
tar -xf /tmp/linux.tgz -C /src/linux
rm -f /tmp/linux.tgz

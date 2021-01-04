mkdir -p /src/
cd /src/
git clone --depth 1 --branch v1.19 https://git.kernel.org/pub/scm/devel/pahole/pahole.git
cd /src/pahole
mkdir build
cd build
cmake -D__LIB=lib ..
make install
ldconfig

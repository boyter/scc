#!/bin/bash

# Assumes we need to install everything from scratch on a box for benchmarking
# Also assumes we have copied the ./examples/performance_tests/create_performance_test.py file to where this is run

apt update
apt install --assume-yes build-essential unzip tmux htop

rm *.zip
wget https://github.com/boyter/scc/releases/download/v1.0.0/scc-1.0.0-x86_64-unknown-linux.zip
unzip scc-1.0.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.0.0

wget https://github.com/boyter/scc/releases/download/v1.1.0/scc-1.1.0-x86_64-unknown-linux.zip
unzip scc-1.1.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.1.0

wget https://github.com/boyter/scc/releases/download/v1.2.0/scc-1.2.0-x86_64-unknown-linux.zip
unzip scc-1.2.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.2.0

wget https://github.com/boyter/scc/releases/download/untagged-928286b8064e2cf6dd35/scc-1.3.0-x86_64-unknown-linux.zip
unzip scc-1.3.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.3.0

wget https://github.com/boyter/scc/releases/download/v1.4.0/scc-1.4.0-x86_64-unknown-linux.zip
unzip scc-1.4.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.4.0

wget https://github.com/boyter/scc/releases/download/v1.5.0/scc-1.5.0-x86_64-unknown-linux.zip
unzip scc-1.5.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.5.0

wget https://github.com/boyter/scc/releases/download/v1.6.0/scc-1.6.0-x86_64-unknown-linux.zip
unzip scc-1.6.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.6.0

wget https://github.com/boyter/scc/releases/download/v1.7.0/scc-1.7.0-x86_64-unknown-linux.zip
unzip scc-1.7.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.7.0

wget https://github.com/boyter/scc/releases/download/v1.8.0/scc-1.8.0-x86_64-unknown-linux.zip
unzip scc-1.8.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.8.0

wget https://github.com/boyter/scc/releases/download/v1.9.0/scc-1.9.0-x86_64-unknown-linux.zip
unzip scc-1.9.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.9.0

# Whoops... the name was not set correctly...
rm scc-1.0.0-x86_64-unknown-linux.zip
wget https://github.com/boyter/scc/releases/download/v1.10.0/scc-1.0.0-x86_64-unknown-linux.zip
unzip scc-1.0.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.10.0

wget https://github.com/boyter/scc/releases/download/v1.11.0/scc-1.11.0-x86_64-unknown-linux.zip
unzip scc-1.11.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.11.0

wget https://github.com/boyter/scc/releases/download/v1.12.0/scc-1.12.0-x86_64-unknown-linux.zip
unzip scc-1.12.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.12.0

wget https://github.com/boyter/scc/releases/download/v1.12.1/scc-1.12.1-x86_64-unknown-linux.zip
unzip scc-1.12.1-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc1.12.1

wget https://github.com/boyter/scc/releases/download/v2.0.0/scc-2.0.0-x86_64-unknown-linux.zip
unzip scc-2.0.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.0.0

wget https://github.com/boyter/scc/releases/download/v2.1.0/scc-2.1.0-x86_64-unknown-linux.zip
unzip scc-2.1.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.1.0

wget https://github.com/boyter/scc/releases/download/v2.2.0/scc-2.2.0-x86_64-unknown-linux.zip
unzip scc-2.2.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.2.0

wget https://github.com/boyter/scc/releases/download/v2.3.0/scc-2.3.0-x86_64-unknown-linux.zip
unzip scc-2.3.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.3.0

wget https://github.com/boyter/scc/releases/download/v2.4.0/scc-2.4.0-x86_64-unknown-linux.zip
unzip scc-2.4.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.4.0

wget https://github.com/boyter/scc/releases/download/v2.5.0/scc-2.5.0-x86_64-unknown-linux.zip
unzip scc-2.5.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.5.0

wget https://github.com/boyter/scc/releases/download/v2.6.0/scc-2.6.0-x86_64-unknown-linux.zip
unzip scc-2.6.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.6.0

wget https://github.com/boyter/scc/releases/download/v2.7.0/scc-2.7.0-x86_64-unknown-linux.zip
unzip scc-2.7.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.7.0

wget https://github.com/boyter/scc/releases/download/v2.8.0/scc-2.8.0-x86_64-unknown-linux.zip
unzip scc-2.8.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.8.0

wget https://github.com/boyter/scc/releases/download/v2.9.0/scc-2.9.0-x86_64-unknown-linux.zip
unzip scc-2.9.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.9.0

wget https://github.com/boyter/scc/releases/download/v2.9.1/scc-2.9.1-x86_64-unknown-linux.zip
unzip scc-2.9.1-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.9.1

wget https://github.com/boyter/scc/releases/download/v2.10.0/scc-2.10.0-x86_64-unknown-linux.zip
unzip scc-2.10.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.10.0

wget https://github.com/boyter/scc/releases/download/v2.10.0/scc-2.11.0-x86_64-unknown-linux.zip
unzip scc-2.11.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.11.0

wget https://github.com/boyter/scc/releases/download/v2.10.0/scc-2.12.0-x86_64-unknown-linux.zip
unzip scc-2.12.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.12.0

wget https://github.com/boyter/scc/releases/download/v2.10.0/scc-2.13.0-x86_64-unknown-linux.zip
unzip scc-2.13.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc2.13.0

wget https://github.com/boyter/scc/releases/download/v3.0.0/scc-3.0.0-x86_64-unknown-linux.zip
unzip scc-3.0.0-x86_64-unknown-linux.zip
mv scc /usr/local/bin/scc3.0.0

wget https://github.com/boyter/scc/releases/download/v3.1.0/scc_3.1.0_Linux_x86_64.tar.gz
tar zxvf scc_3.1.0_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.1.0

wget https://github.com/boyter/scc/releases/download/v3.2.0/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.2.0
rm scc_Linux_x86_64.tar.gz

wget https://github.com/boyter/scc/releases/download/v3.3.0/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.3.0
rm scc_Linux_x86_64.tar.gz

wget https://github.com/boyter/scc/releases/download/v3.3.2/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.3.2
rm scc_Linux_x86_64.tar.gz

wget https://github.com/boyter/scc/releases/download/v3.3.3/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.3.3
rm scc_Linux_x86_64.tar.gz

wget https://github.com/boyter/scc/releases/download/v3.3.4/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc3.3.4
rm scc_Linux_x86_64.tar.gz

# Now setup the most recent as the default
wget https://github.com/boyter/scc/releases/download/v3.3.4/scc_Linux_x86_64.tar.gz
tar zxvf scc_Linux_x86_64.tar.gz
mv scc /usr/local/bin/scc

#echo "Setting up rust toolchain"
#curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
#source $HOME/.cargo/env
#
#cargo install hyperfine tokei loc

wget https://github.com/sharkdp/hyperfine/releases/download/v1.18.0/hyperfine-v1.18.0-x86_64-unknown-linux-musl.tar.gz
tar zxvf hyperfine-v1.18.0-x86_64-unknown-linux-musl.tar.gz
mv hyperfine-v1.18.0-x86_64-unknown-linux-musl/hyperfine /usr/local/bin/hyperfine
chmod +x /usr/local/bin/hyperfine

wget https://github.com/XAMPPRocky/tokei/releases/download/v12.1.2/tokei-x86_64-unknown-linux-musl.tar.gz
tar zxvf tokei-x86_64-unknown-linux-musl.tar.gz
chmod +x ./tokei
mv ./tokei /usr/local/bin/

wget https://github.com/vmchale/polyglot/releases/download/0.5.29/poly-x86_64-unknown-linux-gcc-9
mv poly-x86_64-unknown-linux-gcc-9 /usr/local/bin/polyglot
chmod +x /usr/local/bin/polyglot

# Now setup all of the benchmarks

# Clone the stuff we want to test
rm -rf valkey
rm -rf cpython
rm -rf linux
rm -rf sourcegraph

git clone --depth=1 https://github.com/valkey-io/valkey.git
git clone --depth=1 https://github.com/python/cpython.git
git clone --depth=1 https://github.com/torvalds/linux.git
git clone --depth=1 https://github.com/sourcegraph/sourcegraph.git

# Regression test all versions of scc
echo "Running regression benchmark"
hyperfine 'scc1.0.0 linux' 'scc1.1.0 linux' 'scc1.2.0 linux' 'scc1.3.0 linux' 'scc1.4.0 linux' 'scc1.5.0 linux' 'scc1.6.0 linux' 'scc1.7.0 linux' 'scc1.8.0 linux' 'scc1.9.0 linux' 'scc1.10.0 linux' 'scc1.11.0 linux' 'scc1.12.0 linux' 'scc1.12.1 linux' 'scc2.0.0 linux' 'scc2.1.0 linux' 'scc2.2.0 linux' 'scc2.3.0 linux' 'scc2.4.0 linux' 'scc2.5.0 linux' 'scc2.6.0 linux' 'scc2.7.0 linux' 'scc2.8.0 linux' 'scc2.9.0 linux' 'scc2.9.1 linux' 'scc2.10.0 linux' 'scc2.11.0 linux' 'scc2.12.0 linux' 'scc2.13.0 linux' 'scc3.0.0 linux' 'scc3.1.0 linux' 'scc3.2.0 linux' 'scc3.3.0 linux' 'scc3.3.2 linux' 'scc3.3.3 linux' 'scc3.3.4 linux' > benchmark_regression.txt

# Benchmark against everything
echo "Running valkey benchmark"
hyperfine 'scc valkey' 'scc -c valkey' 'tokei valkey' 'polyglot valkey' > benchmark_valkey.txt

echo "Running cpython benchmark"
hyperfine 'scc cpython' 'scc -c cpython' 'tokei cpython' 'polyglot cpython' > benchmark_cpython.txt

echo "Running sourcegraph benchmark"
hyperfine 'scc sourcegraph' 'scc -c sourcegraph' 'tokei sourcegraph' 'polyglot sourcegraph' > benchmark_sourcegraph.txt

echo "Running linux benchmark"
hyperfine 'scc linux' 'scc -c linux' 'tokei linux' 'polyglot linux' > benchmark_linux.txt

echo "All done!"
#!/bin/bash

# Assumes we need to install everything from scratch on a box for benchmarking
# Also assumes we have copied the ./examples/performance_tests/create_performance_test.py file to where this is run

apt-get update
apt-get install --assume-yes build-essential unzip cloc sloccount tmux python htop

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
cp scc /usr/local/bin/scc2.10.0
mv scc /usr/local/bin/scc

echo "Setting up rust toolchain"
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env

cargo install hyperfine tokei loc

wget https://github.com/vmchale/polyglot/releases/download/0.5.24/poly-x86_64-unknown-linux-icc
mv poly-x86_64-unknown-linux-icc /usr/local/bin/polyglot
chmod +x /usr/local/bin/polyglot

wget https://github.com/hhatto/gocloc/releases/download/v0.3.0/gocloc_0.3.0_Linux_x86_64.tar.gz
tar zxvf gocloc_0.3.0_Linux_x86_64.tar.gz
mv gocloc /usr/local/bin/

# TODO want to get a build for loccount here at some point

# Now setup all of the benchmarks

rm -rf artifical
mkdir artifical
cp create_performance_test.py artifical
cd artifical
python create_performance_test.py
cd ..
rm artifical/create_performance_test.py

# Clone the stuff we want to test
rm -rf redis
rm -rf cpython
rm -rf linux

git clone --depth=1 https://github.com/antirez/redis.git
git clone --depth=1 https://github.com/python/cpython.git
git clone --depth=1 https://github.com/torvalds/linux.git

# Setup torture test

echo "Copying 10 linuxes"
mkdir -p linux10
cp -R linux linux10/0
cp -R linux linux10/1
cp -R linux linux10/2
cp -R linux linux10/3
cp -R linux linux10/4
cp -R linux linux10/5
cp -R linux linux10/6
cp -R linux linux10/7
cp -R linux linux10/8
cp -R linux linux10/9

# Setup uber torture test

echo "Copying 50 linuxes"
mkdir -p linux50
cp -R linux10 linux50/linux0
cp -R linux10 linux50/linux1
cp -R linux10 linux50/linux2
cp -R linux10 linux50/linux3
cp -R linux10 linux50/linux4

# Regression test all versions of scc
echo "Running regression benchmark"
hyperfine 'scc1.0.0 linux' 'scc1.1.0 linux' 'scc1.2.0 linux' 'scc1.3.0 linux' 'scc1.4.0 linux' 'scc1.5.0 linux' 'scc1.6.0 linux' 'scc1.7.0 linux' 'scc1.8.0 linux' 'scc1.9.0 linux' 'scc1.10.0 linux' 'scc1.11.0 linux' 'scc1.12.0 linux' 'scc1.12.1 linux' 'scc2.0.0 linux' 'scc2.1.0 linux' 'scc2.2.0 linux' 'scc2.3.0 linux' 'scc2.4.0 linux' 'scc2.5.0 linux' 'scc2.6.0 linux' 'scc2.7.0 linux' 'scc2.8.0 linux' 'scc2.9.0 linux' 'scc2.9.1 linux' 'scc2.10.0 linux' > benchmark_regression.txt

# Benchmark against everything
echo "Running artifical benchmark"
hyperfine 'scc artifical' 'scc -c artifical' 'tokei artifical' 'loc artifical' 'polyglot artifical' 'gocloc artifical' > benchmark_artifical.txt

echo "Running redis benchmark"
hyperfine 'scc redis' 'scc -c redis' 'tokei redis' 'loc redis' 'polyglot redis' 'gocloc redis' 'cloc redis' 'sloccount redis' > benchmark_redis.txt

echo "Running cpython benchmark"
hyperfine 'scc cpython' 'scc -c cpython' 'tokei cpython' 'loc cpython' 'polyglot cpython' 'gocloc cpython' > benchmark_cpython.txt

echo "Running linux benchmark"
hyperfine 'scc linux' 'scc -c linux' 'tokei linux' 'loc linux' 'polyglot linux' 'gocloc linux' > benchmark_linux.txt

echo "Running linux10 benchmark"
hyperfine 'scc linux10' 'scc -c linux10' 'tokei linux10' 'loc linux10' 'polyglot linux10' > benchmark_linux10.txt

echo "Running linux50 benchmark"
hyperfine 'scc linux50' 'scc -c linux50' 'tokei linux50' 'loc linux50' 'polyglot linux50' > benchmark_linux50.txt

echo "All done!"
#!/bin/bash

# Assumes we need to install everything from scratch on a box for benchmarking
# Also assumes we have copied the ./examples/performance_tests/create_performance_test.py file to where this is run

apt update
apt install --assume-yes build-essential unzip tmux htop

rm ./*.zip
rm ./*.gz

install_scc_zip() {
    local version=$1
    local url=$2
    wget "$url"
    local file=$(basename "$url")
    unzip "$file"
    mv scc /usr/local/bin/scc${version}
    rm "$file"
}

install_scc_tar() {
    local version=$1
    local url=$2
    wget "$url"
    local file=$(basename "$url")
    tar zxvf "$file"
    mv scc /usr/local/bin/scc${version}
    rm "$file"
}

# v1.x zip releases
ZIP_VERSIONS=(
    "1.0.0 https://github.com/boyter/scc/releases/download/v1.0.0/scc-1.0.0-x86_64-unknown-linux.zip"
    "1.1.0 https://github.com/boyter/scc/releases/download/v1.1.0/scc-1.1.0-x86_64-unknown-linux.zip"
    "1.2.0 https://github.com/boyter/scc/releases/download/v1.2.0/scc-1.2.0-x86_64-unknown-linux.zip"
    "1.3.0 https://github.com/boyter/scc/releases/download/untagged-928286b8064e2cf6dd35/scc-1.3.0-x86_64-unknown-linux.zip"
    "1.4.0 https://github.com/boyter/scc/releases/download/v1.4.0/scc-1.4.0-x86_64-unknown-linux.zip"
    "1.5.0 https://github.com/boyter/scc/releases/download/v1.5.0/scc-1.5.0-x86_64-unknown-linux.zip"
    "1.6.0 https://github.com/boyter/scc/releases/download/v1.6.0/scc-1.6.0-x86_64-unknown-linux.zip"
    "1.7.0 https://github.com/boyter/scc/releases/download/v1.7.0/scc-1.7.0-x86_64-unknown-linux.zip"
    "1.8.0 https://github.com/boyter/scc/releases/download/v1.8.0/scc-1.8.0-x86_64-unknown-linux.zip"
    "1.9.0 https://github.com/boyter/scc/releases/download/v1.9.0/scc-1.9.0-x86_64-unknown-linux.zip"
    "1.10.0 https://github.com/boyter/scc/releases/download/v1.10.0/scc-1.0.0-x86_64-unknown-linux.zip"
    "1.11.0 https://github.com/boyter/scc/releases/download/v1.11.0/scc-1.11.0-x86_64-unknown-linux.zip"
    "1.12.0 https://github.com/boyter/scc/releases/download/v1.12.0/scc-1.12.0-x86_64-unknown-linux.zip"
    "1.12.1 https://github.com/boyter/scc/releases/download/v1.12.1/scc-1.12.1-x86_64-unknown-linux.zip"
)

# v2.x zip releases
ZIP_VERSIONS+=(
    "2.0.0 https://github.com/boyter/scc/releases/download/v2.0.0/scc-2.0.0-x86_64-unknown-linux.zip"
    "2.1.0 https://github.com/boyter/scc/releases/download/v2.1.0/scc-2.1.0-x86_64-unknown-linux.zip"
    "2.2.0 https://github.com/boyter/scc/releases/download/v2.2.0/scc-2.2.0-x86_64-unknown-linux.zip"
    "2.3.0 https://github.com/boyter/scc/releases/download/v2.3.0/scc-2.3.0-x86_64-unknown-linux.zip"
    "2.4.0 https://github.com/boyter/scc/releases/download/v2.4.0/scc-2.4.0-x86_64-unknown-linux.zip"
    "2.5.0 https://github.com/boyter/scc/releases/download/v2.5.0/scc-2.5.0-x86_64-unknown-linux.zip"
    "2.6.0 https://github.com/boyter/scc/releases/download/v2.6.0/scc-2.6.0-x86_64-unknown-linux.zip"
    "2.7.0 https://github.com/boyter/scc/releases/download/v2.7.0/scc-2.7.0-x86_64-unknown-linux.zip"
    "2.8.0 https://github.com/boyter/scc/releases/download/v2.8.0/scc-2.8.0-x86_64-unknown-linux.zip"
    "2.9.0 https://github.com/boyter/scc/releases/download/v2.9.0/scc-2.9.0-x86_64-unknown-linux.zip"
    "2.9.1 https://github.com/boyter/scc/releases/download/v2.9.1/scc-2.9.1-x86_64-unknown-linux.zip"
    "2.10.0 https://github.com/boyter/scc/releases/download/v2.10.0/scc-2.10.0-x86_64-unknown-linux.zip"
    "2.11.0 https://github.com/boyter/scc/releases/download/v2.11.0/scc-2.11.0-x86_64-unknown-linux.zip"
    "2.12.0 https://github.com/boyter/scc/releases/download/v2.12.0/scc-2.12.0-x86_64-unknown-linux.zip"
    "2.13.0 https://github.com/boyter/scc/releases/download/v2.13.0/scc-2.13.0-x86_64-unknown-linux.zip"
)

# v3.0.0 is a zip
ZIP_VERSIONS+=(
    "3.0.0 https://github.com/boyter/scc/releases/download/v3.0.0/scc-3.0.0-x86_64-unknown-linux.zip"
)

# v3.1.0 has a versioned tar name
TAR_VERSIONS=(
    "3.1.0 https://github.com/boyter/scc/releases/download/v3.1.0/scc_3.1.0_Linux_x86_64.tar.gz"
)

# v3.2.0+ use the generic tar name
GENERIC_TAR_VERSIONS=(3.2.0 3.3.0 3.3.2 3.3.3 3.3.4 3.4.0 3.5.0 3.6.0 3.7.0)

# Install all zip versions
for entry in "${ZIP_VERSIONS[@]}"; do
    version=${entry%% *}
    url=${entry#* }
    install_scc_zip "$version" "$url"
done

# Install versioned tar
for entry in "${TAR_VERSIONS[@]}"; do
    version=${entry%% *}
    url=${entry#* }
    install_scc_tar "$version" "$url"
done

# Install generic-named tar versions
for version in "${GENERIC_TAR_VERSIONS[@]}"; do
    install_scc_tar "$version" "https://github.com/boyter/scc/releases/download/v${version}/scc_Linux_x86_64.tar.gz"
done

# Make the latest version available as just 'scc'
cp /usr/local/bin/scc${GENERIC_TAR_VERSIONS[-1]} /usr/local/bin/scc

# Collect all scc version names for hyperfine
ALL_VERSIONS=()
for entry in "${ZIP_VERSIONS[@]}"; do
    ALL_VERSIONS+=("${entry%% *}")
done
for entry in "${TAR_VERSIONS[@]}"; do
    ALL_VERSIONS+=("${entry%% *}")
done
for v in "${GENERIC_TAR_VERSIONS[@]}"; do
    ALL_VERSIONS+=("$v")
done

build_hyperfine_args() {
    local target=$1
    local args=()
    for v in "${ALL_VERSIONS[@]}"; do
        args+=("scc${v} ${target}")
    done
    printf "'%s' " "${args[@]}"
}

# Now setup comparison applications starting with hyperfine which we compare against

wget https://github.com/sharkdp/hyperfine/releases/download/v1.18.0/hyperfine-v1.18.0-x86_64-unknown-linux-musl.tar.gz
tar zxvf hyperfine-v1.18.0-x86_64-unknown-linux-musl.tar.gz
mv hyperfine-v1.18.0-x86_64-unknown-linux-musl/hyperfine /usr/local/bin/hyperfine
chmod +x /usr/local/bin/hyperfine

# Now the comparison applications

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
git clone --depth=1 https://github.com/SINTEF/sourcegraph.git

echo "Running regression benchmark"
eval hyperfine --export-json benchmark_regression.json $(build_hyperfine_args linux) > benchmark_regression.txt

echo "Generating chart"
python3 benchmark_to_chart.py benchmark_regression.json "scc performance linux kernel" > benchmark_chart.html

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

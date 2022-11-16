#!/bin/bash
set -euo pipefail

NEEDED_BINARIES=(
  curl
  jq
  lsb_release
)
MISSING_BINARIES=()

SUDO=
SUDOPKG=
APTINSTALL='apt install --no-install-recommends'
if [[ -z ${SKIPROOTCHECK:-} ]]; then
  NEEDED_BINARIES+=(sudo)
  SUDO=sudo
  SUDOPKG=sudo
  APTINSTALL="sudo $APTINSTALL"
  if [[ $(id -u) -eq 0 ]]; then
    echo 'Do not run this script as root'
    echo 'To skip this check, set the environment variable SKIPROOTCHECK=1'
    exit 1
  fi
fi

for B in "${NEEDED_BINARIES[@]}"; do
  if ! command -v "$B" >/dev/null ; then
    MISSING_BINARIES+=("$B")
  fi
done
if [[ ${#MISSING_BINARIES[@]} -gt 0 ]] ; then
  echo "Missing commands (${MISSING_BINARIES[*]}) to start this script. To install, run:"
  echo "  ${APTINSTALL} ca-certificates curl jq lsb-release ${SUDOPKG}"
  exit 1
fi

DFLT_CODEROOT=$HOME/code
DFLT_NODEVER=16.x
DFLT_GOVER=latest
DFLT_UBPFVER=57c842f31259bcb4e7d6c1ff7e6e15ac43cf862f
DFLT_XDPTOOLSVER=v1.2.8
DFLT_LIBBPFVER=v0.8.1
DFLT_URINGVER=liburing-2.3
DFLT_DPDKVER=v22.07
DFLT_DPDKPATCH=25787
DFLT_DPDKOPTS={}
DFLT_SPDKVER=v22.09
DFLT_NJOBS=$(nproc)
DFLT_TARGETARCH=native

HELP=0
CONFIRM=0
CODEROOT=$DFLT_CODEROOT
NODEVER=$DFLT_NODEVER
GOVER=$DFLT_GOVER
UBPFVER=$DFLT_UBPFVER
LIBBPFVER=$DFLT_LIBBPFVER
XDPTOOLSVER=$DFLT_XDPTOOLSVER
URINGVER=$DFLT_URINGVER
DPDKVER=$DFLT_DPDKVER
DPDKPATCH=$DFLT_DPDKPATCH
DPDKOPTS=$DFLT_DPDKOPTS
SPDKVER=$DFLT_SPDKVER
NJOBS=$DFLT_NJOBS
TARGETARCH=$DFLT_TARGETARCH

ARGS=$(getopt -o 'hy' -l 'dir:,node:,go:,ubpf:,libbpf:,xdp:,dpdk:,dpdk-patch:,dpdk-opts:,spdk:,uring:,jobs:,arch:' -- "$@")
eval set -- "$ARGS"
while true; do
  case $1 in
    -h) HELP=1; shift;;
    -y) CONFIRM=1; shift;;
    --dir) CODEROOT=$2; shift 2;;
    --node) NODEVER=$2; shift 2;;
    --go) GOVER=$2; shift 2;;
    --ubpf) UBPFVER=$2; shift 2;;
    --libbpf) LIBBPFVER=$2; shift 2;;
    --xdp) XDPTOOLSVER=$2; shift 2;;
    --uring) URINGVER=$2; shift 2;;
    --dpdk) DPDKVER=$2; DPDKPATCH=''; shift 2;;
    --dpdk-patch) DPDKPATCH=$2; shift 2;;
    --dpdk-opts) DPDKOPTS=$2; shift 2;;
    --spdk) SPDKVER=$2; shift 2;;
    --jobs) NJOBS=$2; shift 2;;
    --arch) TARGETARCH=$2; shift 2;;
    --) shift; break;;
    *) exit 1;;
  esac
done

if [[ $HELP -eq 1 ]]; then
  cat <<EOT
ndndpdk-depends.sh [OPTION]...
  -h  Display help and exit.
  -y  Skip confirmation.
  --dir=${DFLT_CODEROOT}
      Set where to download and compile the code.
  --node=${DFLT_NODEVER}
      Set Node.js version. '0' to skip.
  --go=${DFLT_GOVER}
      Set Go version. 'latest' for the latest 1.x version. '0' to skip.
  --ubpf=${DFLT_UBPFVER}
      Set uBPF branch or commit SHA. '0' to skip.
  --libbpf=${DFLT_LIBBPFVER}
      Set libbpf branch or commit SHA. '0' to skip.
  --xdp=${DFLT_XDPTOOLSVER}
      Set xdp-tools branch or commit SHA. '0' to skip.
  --uring=${DFLT_URINGVER}
      Set liburing version. '0' to skip.
  --dpdk=${DFLT_DPDKVER}
      Set DPDK version. '0' to skip.
  --dpdk-patch=${DFLT_DPDKPATCH}
      Add DPDK patch series (comma separated). '0' to skip.
  --dpdk-opts=${DFLT_DPDKOPTS}
      Set/override DPDK Meson options (JSON object).
  --spdk=${DFLT_SPDKVER}
      Set SPDK version. '0' to skip.
  --jobs=${DFLT_NJOBS}
      Set number of parallel jobs.
  --arch=${DFLT_TARGETARCH}
      Set target architecture.
EOT
  exit 0
fi

: "${NDNDPDK_DL_GITHUB:=https://github.com}"
: "${NDNDPDK_DL_LLVM_APT:=https://apt.llvm.org}"
: "${NDNDPDK_DL_NODESOURCE_DEB:=https://deb.nodesource.com}"
: "${NDNDPDK_DL_PYPA_BOOTSTRAP:=https://bootstrap.pypa.io}"
: "${NDNDPDK_DL_GODEV:=https://go.dev}"
: "${NDNDPDK_DL_DPDK_PATCHES:=https://patches.dpdk.org}"
# you can also set the GOPROXY environment variable, which will be persisted

curl_test() {
  local SITE=${!1}
  if ! curl -fsILS "$SITE${2:-}" >/dev/null; then
    echo "Cannot reach ${SITE}"
    echo "You can specify a mirror site by setting the $1 environment variable"
    echo "Example: $1=${SITE}"
    exit 1
  fi
}
curl_test NDNDPDK_DL_GITHUB /robots.txt
curl_test NDNDPDK_DL_LLVM_APT
curl_test NDNDPDK_DL_NODESOURCE_DEB
curl_test NDNDPDK_DL_PYPA_BOOTSTRAP
curl_test NDNDPDK_DL_GODEV /VERSION
curl_test NDNDPDK_DL_DPDK_PATCHES

github_download() {
  local REPO=$1
  local VER=$2
  local DIR="${REPO#*/}-${VER#v}"
  cd "$CODEROOT"
  rm -rf "$DIR"
  curl -fsLS "${NDNDPDK_DL_GITHUB}/${REPO}/archive/${VER}.tar.gz" | tar -xz
  readlink -f "$DIR"
}

DISTRO=$(lsb_release -sc)
case $DISTRO in
  bionic) ;;
  focal) ;;
  jammy) ;;
  bullseye) ;;
  *)
    echo "Distro ${DISTRO} is not supported by this script."
    if [[ -z ${SKIPDISTROCHECK:-} ]]; then
      echo 'To skip this check, set the environment variable SKIPDISTROCHECK=1'
      exit 1
    fi
    ;;
esac

if [[ $(uname -r | awk -F. '{ print ($1*1000+$2>=5004) }') -ne 1 ]] &&
   [[ -z ${SKIPKERNELCHECK:-} ]] && ! [[ -f /.dockerenv ]]; then
  echo 'Linux kernel 5.4 or newer is required'
  if [[ $DISTRO == bionic ]]; then
    echo 'To upgrade kernel, run this command and reboot:'
    echo "  ${APTINSTALL} linux-generic-hwe-18.04"
  fi
  echo 'To skip this check, set the environment variable SKIPKERNELCHECK=1'
  exit 1
fi

APT_PKGS=(
  build-essential
  clang-11
  clang-format-11
  doxygen
  file
  gcc-multilib
  git
  lcov
  llvm-11
  libaio-dev
  libc6-dev-i386
  libelf-dev
  libnuma-dev
  libpcap-dev
  libssl-dev
  liburcu-dev
  m4
  ninja-build
  pkg-config
  python3-distutils
  uuid-dev
  yamllint
)

if [[ $DISTRO == bionic ]]; then
  APT_PKGS+=(python3.8)
else
  APT_PKGS+=(python-is-python3)
fi

echo "Will download to ${CODEROOT}"
echo 'Will install C compiler and build tools'

if [[ $NODEVER != 0 ]]; then
  echo "Will install Node ${NODEVER}"
elif ! command -v node >/dev/null; then
  echo '--node=0 specified but the `node` command was not found, which may cause build errors'
fi

if [[ $GOVER != 0 ]]; then
  if [[ $GOVER == latest ]]; then
    GOVER=$(curl -fsLS "${NDNDPDK_DL_GODEV}/VERSION?m=text")
  fi
  echo "Will install Go ${GOVER}"
elif ! command -v go >/dev/null; then
  echo '--go=0 specified but the `go` command was not found, which may cause build errors'
fi
echo 'Will install Go linters and tools'

if [[ $UBPFVER != 0 ]]; then
  echo "Will install uBPF ${UBPFVER}"
elif ! [[ -f /usr/local/include/ubpf.h ]]; then
  echo '--ubpf=0 specified but uBPF was not found, which may cause build errors'
fi

if [[ $LIBBPFVER != 0 ]]; then
  echo "Will install libbpf ${LIBBPFVER}"
fi

if [[ $XDPTOOLSVER != 0 ]]; then
  echo "Will install libxdp ${XDPTOOLSVER}"
fi

if [[ $URINGVER != 0 ]]; then
  echo "Will install liburing ${URINGVER}"
elif ! pkg-config liburing; then
  echo '--uring=0 specified but liburing was not found, which may cause build errors'
fi

if [[ $DPDKVER != 0 ]]; then
  echo "Will install DPDK ${DPDKVER} for ${TARGETARCH} architecture"
  echo -n "$DPDKPATCH" | xargs -d, --no-run-if-empty -I{} echo "Will patch DPDK with ${NDNDPDK_DL_DPDK_PATCHES}/series/{}/mbox/"
elif ! pkg-config libdpdk; then
  echo '--dpdk=0 specified but DPDK was not found, which may cause build errors'
fi

if [[ $SPDKVER != 0 ]]; then
  echo "Will install SPDK ${SPDKVER} for ${TARGETARCH} architecture"
elif ! pkg-config spdk_thread; then
  echo '--spdk=0 specified but SPDK was not found, which may cause build errors'
fi

echo "Will compile with ${NJOBS} parallel jobs"
echo 'Will delete conflicting versions if present'
if [[ $CONFIRM -ne 1 ]]; then
  read -p 'Press ENTER to continue or CTRL+C to abort '
fi

$SUDO mkdir -p "$CODEROOT"
$SUDO chown -R "$(id -u):$(id -g)" "$CODEROOT"

echo 'Dpkg::Options {
   "--force-confdef";
   "--force-confold";
}
APT::Install-Recommends "no";
APT::Install-Suggests "no";' | $SUDO tee /etc/apt/apt.conf.d/80custom >/dev/null
if [[ $DISTRO == bionic ]] && ! [[ -f /etc/apt/sources.list.d/llvm-11.list ]]; then
  curl -fsLS "${NDNDPDK_DL_LLVM_APT}/llvm-snapshot.gpg.key" | $SUDO apt-key add -
  echo "deb ${NDNDPDK_DL_LLVM_APT}/bionic/ llvm-toolchain-bionic-11 main" | $SUDO tee /etc/apt/sources.list.d/llvm-11.list
fi
$SUDO apt-get -qq update
$SUDO env DEBIAN_FRONTEND=noninteractive apt-get -qq dist-upgrade

if [[ $NODEVER != 0 ]]; then
  curl -fsLS "${NDNDPDK_DL_NODESOURCE_DEB}/setup_${NODEVER}" | $SUDO bash -
  APT_PKGS+=(nodejs)
fi

$SUDO env DEBIAN_FRONTEND=noninteractive apt-get -qq install "${APT_PKGS[@]}"

if [[ $DISTRO == bionic ]]; then
  $SUDO update-alternatives --remove-all python || true
  $SUDO update-alternatives --install /usr/bin/python python /usr/bin/python3.8 1
fi
curl -fsLS "${NDNDPDK_DL_PYPA_BOOTSTRAP}/get-pip.py" | $SUDO python
$SUDO pip install -U meson pyelftools

if [[ $GOVER != 0 ]]; then
  $SUDO rm -rf /usr/local/go
  curl -fsLS "${NDNDPDK_DL_GODEV}/dl/${GOVER}.linux-amd64.tar.gz" | $SUDO tar -C /usr/local -xz
  if ! grep -q go/bin ~/.bashrc; then
    echo 'export PATH=${HOME}/go/bin${PATH:+:}${PATH}' >>~/.bashrc
  fi
  if [[ -n ${GOPROXY:-} ]]; then
    go env -w GOPROXY="$GOPROXY"
  fi
  $SUDO update-alternatives --remove-all go || true
  $SUDO update-alternatives --install /usr/bin/go go /usr/local/go/bin/go 1
  $SUDO update-alternatives --remove-all gofmt || true
  $SUDO update-alternatives --install /usr/bin/gofmt gofmt /usr/local/go/bin/gofmt 1
fi

if [[ $UBPFVER != 0 ]]; then
  cd "$CODEROOT"
  rm -rf "ubpf-${UBPFVER}"
  curl -fsLS "${NDNDPDK_DL_GITHUB}/iovisor/ubpf/archive/${UBPFVER}.tar.gz" | tar -xz
  cd "ubpf-${UBPFVER}/vm"
  make -j${NJOBS}
  $SUDO make install
  $SUDO rm -f /usr/local/lib/libubpf.a
fi

if [[ $LIBBPFVER != 0 ]]; then
  cd "$(github_download libbpf/libbpf $LIBBPFVER)/src"
  sh -c "umask 0000 && make -j${NJOBS}"
  $SUDO find /usr/local/lib -name 'libbpf.*' -delete
  $SUDO sh -c "umask 0000 && make install PREFIX=/usr/local LIBDIR=/usr/local/lib"
  $SUDO install -d -m0755 /usr/local/include/linux
  $SUDO install -m0644 ../include/uapi/linux/* /usr/local/include/linux
  $SUDO ldconfig
fi

if [[ $XDPTOOLSVER != 0 ]]; then
  cd "$(github_download xdp-project/xdp-tools $XDPTOOLSVER)"
  CLANG=clang-11 LLC=llc-11 ./configure
  sh -c "umask 0000 && make -j${NJOBS}"
  $SUDO find /usr/local/lib '(' -name 'libxdp.*' -or -name 'xdp*.o' ')' -delete
  $SUDO sh -c "umask 0000 && make install PREFIX=/usr/local LIBDIR=/usr/local/lib"
  $SUDO ldconfig
fi

if [[ $URINGVER != 0 ]]; then
  cd "$(github_download axboe/liburing $URINGVER)"
  ./configure --prefix=/usr/local
  make -C src -j${NJOBS}
  $SUDO find /usr/local/lib -name 'liburing.*' -delete
  $SUDO find /usr/local/share/man -name 'io_uring*' -delete
  $SUDO rm -rf /usr/local/include/liburing /usr/local/include/liburing.h
  $SUDO make install
  $SUDO ldconfig
fi

if [[ $DPDKVER != 0 ]]; then
  cd "$(github_download DPDK/dpdk $DPDKVER)"
  echo -n "$DPDKPATCH" | xargs -d, --no-run-if-empty -I{} \
    sh -c "curl -fsLS ${NDNDPDK_DL_DPDK_PATCHES}/series/{}/mbox/ | patch -p1"
  meson $(echo "$DPDKOPTS" | jq -r --arg arch "$TARGETARCH" \
          '{ debug:true, cpu_instruction_set:$arch, optimization:3, tests:false } + .
           | to_entries[] | "-D"+.key+"="+(.value|tostring)') --libdir=lib build
  cd build
  ninja -j${NJOBS}
  $SUDO find /usr/local/lib -name 'librte_*' -delete
  $SUDO ninja install
  $SUDO find /usr/local/lib -name 'librte_*.a' -delete
  $SUDO ldconfig
fi

if [[ $SPDKVER != 0 ]]; then
  cd "$(github_download spdk/spdk $SPDKVER)"
  sed -i '/^\s*if .*isa-l\/autogen.sh/,/^\s*fi$/ s/.*/CONFIG[ISAL]=n/' configure
  ./configure --target-arch=${TARGETARCH} --with-shared \
    --disable-tests --disable-unit-tests --disable-examples --disable-apps \
    --with-dpdk --with-uring \
    --without-idxd --without-crypto --without-fio --without-xnvme --without-vhost \
    --without-virtio --without-vfio-user --without-pmdk --without-reduce --without-rbd \
    --without-rdma --without-fc --without-daos --without-iscsi-initiator --without-vtune \
    --without-ocf --without-fuse --without-nvme-cuse --without-raid5f --without-wpdk \
    --without-usdt --without-sma
  make -j${NJOBS}
  $SUDO find /usr/local/lib -name 'libspdk_*' -delete
  $SUDO make install
  $SUDO find /usr/local/lib -name 'libspdk_*.a' -delete
  $SUDO ldconfig
fi

(
  cd /tmp
  go install golang.org/x/tools/cmd/godoc@latest
  go install honnef.co/go/tools/cmd/staticcheck@latest
)

echo "NDN-DPDK dependencies installation completed"

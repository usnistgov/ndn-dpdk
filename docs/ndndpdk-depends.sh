#!/bin/bash
set -eo pipefail

SUDO=sudo
if [[ $(id -u) -eq 0 ]]; then
  SUDO=
elif ! which sudo >/dev/null; then
  echo 'sudo is required to start this script; to install:'
  echo '  apt install sudo'
  exit 1
fi

if ! which curl >/dev/null; then
  echo 'curl is required to start this script; to install:'
  echo '  sudo apt install curl'
  exit 1
fi

DFLT_CODEROOT=$HOME/code
DFLT_NODEVER=14.x
DFLT_GOVER=latest
DFLT_UBPFVER=HEAD
DFLT_LIBBPFVER=HEAD
DFLT_DPDKVER=20.11
DFLT_KMODSVER=HEAD
DFLT_SPDKVER=20.10
DFLT_NJOBS=$(nproc)
DFLT_TARGETARCH=native

IS_DEBIAN=0
if [[ $(awk '$1=="deb" && $3=="buster"' /etc/apt/sources.list | wc -l) -gt 0 ]]; then
  IS_DEBIAN=1
fi

KERNELVER=$(uname -r)
HAS_KERNEL_HEADERS=0
if [[ -d /usr/src/linux-headers-${KERNELVER} ]]; then
  HAS_KERNEL_HEADERS=1
else
  DFLT_LIBBPFVER=0
  DFLT_KMODSVER=0
fi
if [[ $KERNELVER == 5.10.* ]]; then
  # kmods build is broken on kernel 5.10, https://bugs.debian.org/975571
  # TODO delete this when Debian and Ubuntu fix this bug
  DFLT_KMODSVER=0
fi
if [[ $(echo $KERNELVER | awk -F. '{ print ($1*1000+$2>=4018) }') -eq 0 ]]; then
  DFLT_LIBBPFVER=0
fi

CODEROOT=$DFLT_CODEROOT
NODEVER=$DFLT_NODEVER
GOVER=$DFLT_GOVER
UBPFVER=$DFLT_UBPFVER
LIBBPFVER=$DFLT_LIBBPFVER
DPDKVER=$DFLT_DPDKVER
KMODSVER=$DFLT_KMODSVER
SPDKVER=$DFLT_SPDKVER
NJOBS=$DFLT_NJOBS
TARGETARCH=$DFLT_TARGETARCH

ARGS=$(getopt -o 'hy' --long 'dir:,node:,go:,libbpf:,dpdk:,kmods:,spdk:,ubpf:,jobs:,arch:,skiprootcheck' -- "$@")
eval "set -- $ARGS"
while true; do
  case $1 in
    (-h) HELP=1; shift;;
    (-y) CONFIRM=1; shift;;
    (--dir) CODEROOT=$2; shift 2;;
    (--node) NODEVER=$2; shift 2;;
    (--go) GOVER=$2; shift 2;;
    (--ubpf) UBPFVER=$2; shift 2;;
    (--libbpf) LIBBPFVER=$2; shift 2;;
    (--dpdk) DPDKVER=$2; shift 2;;
    (--kmods) KMODSVER=$2; shift 2;;
    (--spdk) SPDKVER=$2; shift 2;;
    (--jobs) NJOBS=$2; shift 2;;
    (--arch) TARGETARCH=$2; shift 2;;
    (--skiprootcheck) SKIPROOTCHECK=1; shift;;
    (--) shift; break;;
    (*) exit 1;;
  esac
done

if [[ $HELP -eq 1 ]]; then
  cat <<EOT
ndndpdk-depends.sh ...ARGS
  -h  Display help and exit.
  -y  Skip confirmation.
  --dir=${DFLT_CODEROOT}
      Set where to download and compile code.
  --node=${DFLT_NODEVER}
      Set Node.js version. '0' to skip.
  --go=${DFLT_GOVER}
      Set Go version. 'latest' for latest 1.x version. '0' to skip.
  --ubpf=${DFLT_UBPFVER}
      Set uBPF branch or commit SHA. '0' to skip.
  --libbpf=${DFLT_LIBBPFVER}
      Set libbpf branch or commit SHA. '0' to skip.
  --dpdk=${DFLT_DPDKVER}
      Set DPDK version. '0' to skip.
  --kmods=${DFLT_KMODSVER}
      Set DPDK kernel modules branch or commit SHA. '0' to skip.
  --spdk=${DFLT_SPDKVER}
      Set SPDK version. '0' to skip.
  --jobs=${DFLT_NJOBS}
      Set number of parallel jobs.
  --arch=${DFLT_TARGETARCH}
      Set target architecture.
EOT
  exit 0
fi

if [[ $(id -u) -eq 0 ]] && [[ $SKIPROOTCHECK -ne 1 ]]; then
  echo 'Do not run this script as root'
  exit 1
fi

DISPLAYARCH=$TARGETARCH
if [[ $TARGETARCH == native ]] && which gcc >/dev/null; then
  DISPLAYARCH=$DISPLAYARCH' ('$(gcc -march=native -Q --help=target | awk '$1=="-march=" { print $2 }')')'
fi

APT_PKGS=(
  build-essential
  clang-8
  clang-format-8
  doxygen
  git
  jq
  libc6-dev-i386
  libelf-dev
  libnuma-dev
  libssl-dev
  liburcu-dev
  pkg-config
  python3-distutils
  yamllint
)

echo "Will download to ${CODEROOT}"
echo 'Will install C compiler and build tools'

if [[ $HAS_KERNEL_HEADERS == '0' ]]; then
  echo "Will skip certain features due to missing kernel headers; to install:"
  if [[ $IS_DEBIAN == '1' ]]; then
    echo "  sudo apt install linux-headers-amd64 linux-headers-${KERNELVER}-amd64"
  else
    echo "  sudo apt install linux-generic linux-headers-${KERNELVER}"
  fi
fi

if [[ $NODEVER == '0' ]]; then
  if ! which node >/dev/null; then
    echo '--node=0 specified but `node` command is unavailable'
    exit 1
  fi
else
  echo "Will install Node ${NODEVER}"
fi

if [[ $GOVER == '0' ]]; then
  if ! which go >/dev/null; then
    echo '--go=0 specified but `go` command is unavailable'
    exit 1
  fi
else
  if [[ $GOVER == 'latest' ]]; then
    GOVER=$(curl -sfL https://golang.org/VERSION?m=text)
  fi
  echo "Will install Go ${GOVER}"
fi
echo 'Will install Go linters and tools'

if [[ $UBPFVER == '0' ]]; then
  if ! [[ -f /usr/local/include/ubpf.h ]]; then
    echo '--ubpf=0 specified but uBPF headers are absent'
    exit 1
  fi
else
  if [[ ${#UBPFVER} -ne 40 ]] && which jq >/dev/null; then
    UBPFVER=$(curl -sfL https://api.github.com/repos/iovisor/ubpf/commits/${UBPFVER} | jq -r '.sha')
  fi
  echo "Will install uBPF ${UBPFVER}"
fi

if [[ $LIBBPFVER != '0' ]]; then
  if [[ ${#LIBBPFVER} -ne 40 ]] && which jq >/dev/null; then
    LIBBPFVER=$(curl -sfL https://api.github.com/repos/libbpf/libbpf/commits/${LIBBPFVER} | jq -r '.sha')
  fi
  echo "Will install libbpf ${LIBBPFVER}"
fi

if [[ $DPDKVER == '0' ]]; then
  if ! [[ -f /usr/local/include/rte_common.h ]]; then
    echo '--dpdk=0 specified but DPDK headers are absent'
    exit 1
  fi
else
  echo "Will install DPDK ${DPDKVER} for ${DISPLAYARCH} architecture"
fi

if [[ $KMODSVER != '0' ]]; then
  echo "Will install DPDK kernel modules ${KMODSVER}"
  APT_PKGS+=(kmod)
fi

if [[ $SPDKVER == '0' ]]; then
  if ! [[ -f /usr/local/include/spdk/version.h ]]; then
    echo '--spdk=0 specified but SPDK headers are absent'
    exit 1
  fi
else
  echo "Will install SPDK ${SPDKVER} for ${DISPLAYARCH} architecture"
fi

echo "Will compile with ${NJOBS} parallel jobs"
echo 'Will delete conflicting versions if present'
if [[ $CONFIRM -ne 1 ]]; then
  read -p 'Press ENTER to continue or CTRL+C to abort '
fi

${SUDO} mkdir -p $CODEROOT
${SUDO} chown -R $(id -u):$(id -g) $CODEROOT

echo 'Dpkg::Options {
   "--force-confdef";
   "--force-confold";
}
APT::Install-Recommends "no";
APT::Install-Suggests "no";' | ${SUDO} tee /etc/apt/apt.conf.d/80custom >/dev/null
if [[ ${IS_DEBIAN} == '1' ]]; then
  echo 'deb http://deb.debian.org/debian/ buster-backports main' | ${SUDO} tee /etc/apt/sources.list.d/buster-backports.list >/dev/null
  ${SUDO} apt-get -y -qq update
fi
${SUDO} sh -c 'DEBIAN_FRONTEND=noninteractive apt-get -y -qq dist-upgrade'

if [[ $NODEVER != '0' ]]; then
  curl -sfL https://deb.nodesource.com/setup_${NODEVER} | ${SUDO} bash -
  APT_PKGS+=(nodejs)
fi

APT_PKG_LIST="${APT_PKGS[@]}"
${SUDO} sh -c "DEBIAN_FRONTEND=noninteractive apt-get -y -qq install ${APT_PKG_LIST}"
which gq || ${SUDO} npm install -g graphqurl

curl -sfL https://bootstrap.pypa.io/get-pip.py | ${SUDO} python3
${SUDO} pip install -U meson ninja

if [[ $GOVER != '0' ]]; then
  ${SUDO} rm -rf /usr/local/go
  curl -sfL https://dl.google.com/go/${GOVER}.linux-amd64.tar.gz | ${SUDO} tar -C /usr/local -xz
  export PATH=$HOME/go/bin:/usr/local/go/bin:$PATH
  if ! grep /usr/local/go/bin ~/.bashrc >/dev/null; then
    echo 'export PATH=$HOME/go/bin:/usr/local/go/bin:$PATH' >>~/.bashrc
  fi
fi

if [[ $UBPFVER != '0' ]]; then
  if [[ ${#UBPFVER} -ne 40 ]]; then
    UBPFVER=$(curl -sfL https://api.github.com/repos/iovisor/ubpf/commits/${UBPFVER} | jq -r '.sha')
  fi
  cd $CODEROOT
  rm -rf ubpf-${UBPFVER}
  curl -sfL https://github.com/iovisor/ubpf/archive/${UBPFVER}.tar.gz | tar -xz
  cd ubpf-${UBPFVER}/vm
  make -j${NJOBS}
  ${SUDO} make install
fi

if [[ $LIBBPFVER != '0' ]]; then
  if [[ ${#LIBBPFVER} -ne 40 ]]; then
    LIBBPFVER=$(curl -sfL https://api.github.com/repos/libbpf/libbpf/commits/${LIBBPFVER} | jq -r '.sha')
  fi
  cd $CODEROOT
  rm -rf libbpf-${LIBBPFVER}
  curl -sfL https://github.com/libbpf/libbpf/archive/${LIBBPFVER}.tar.gz | tar -xz
  cd libbpf-${LIBBPFVER}/src
  sh -c "umask 0000 && make -j${NJOBS}"
  ${SUDO} find /usr/local/lib -name 'libbpf.*' -delete
  ${SUDO} sh -c "umask 0000 && make install PREFIX=/usr/local LIBDIR=/usr/local/lib"
  ${SUDO} install -d -m0755 /usr/local/include/linux
  ${SUDO} install -m0644 ../include/uapi/linux/* /usr/local/include/linux
  ${SUDO} ldconfig
fi

if [[ $DPDKVER != '0' ]]; then
  cd $CODEROOT
  rm -rf dpdk-${DPDKVER}
  curl -sfL https://static.dpdk.org/rel/dpdk-${DPDKVER}.tar.xz | tar -xJ
  cd dpdk-${DPDKVER}
  meson -Ddebug=true -Doptimization=3 -Dmachine=${TARGETARCH} -Dtests=false --libdir=lib build
  cd build
  ninja -j${NJOBS}
  ${SUDO} find /usr/local/lib -name 'librte_*' -delete
  ${SUDO} ninja install
  ${SUDO} find /usr/local/lib -name 'librte_*.a' -delete
  ${SUDO} ldconfig
fi

if [[ $KMODSVER != '0' ]]; then
  cd $CODEROOT
  rm -rf dpdk-kmods
  git clone https://dpdk.org/git/dpdk-kmods
  cd dpdk-kmods
  git -c advice.detachedHead=false checkout $KMODSVER
  cd linux/igb_uio
  make
  UIODIR=/lib/modules/${KERNELVER}/kernel/drivers/uio
  ${SUDO} install -d -m0755 $UIODIR
  ${SUDO} install -m0644 igb_uio.ko $UIODIR
  ${SUDO} depmod
fi

if [[ $SPDKVER != '0' ]]; then
  cd $CODEROOT
  rm -rf spdk-${SPDKVER}
  curl -sfL https://github.com/spdk/spdk/archive/v${SPDKVER}.tar.gz | tar -xz
  cd spdk-${SPDKVER}
  ${SUDO} ./scripts/pkgdep.sh
  ./configure --target-arch=${TARGETARCH} --enable-debug --disable-tests --with-shared \
    --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse
  make -j${NJOBS}
  ${SUDO} find /usr/local/lib -name 'libspdk_*' -delete
  ${SUDO} make install
  ${SUDO} find /usr/local/lib -name 'libspdk_*.a' -delete
  ${SUDO} ldconfig
fi

${SUDO} update-alternatives --remove-all python || true
${SUDO} update-alternatives --install /usr/bin/python python /usr/bin/python3 1

(
  cd /tmp
  which staticcheck || go get honnef.co/go/tools/cmd/staticcheck
)

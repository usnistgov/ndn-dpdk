#!/bin/bash
set -eo pipefail

if ! which sudo >/dev/null || ! which curl >/dev/null >/dev/null; then
  echo 'sudo and curl are required to start this script'
  exit 1
fi

DFLT_CODEROOT=$HOME/code
DFLT_NODEVER=14.x
DFLT_GOVER=latest
DFLT_UBPFVER=HEAD
DFLT_LIBBPFVER=0.2-1
DFLT_DPDKVER=20.11
DFLT_KMODSVER=HEAD
DFLT_SPDKVER=20.10
DFLT_TARGETARCH=native

KERNEL_HEADERS_PKG=linux-headers-$(uname -r)
if ! apt-cache show ${KERNEL_HEADERS_PKG} &>/dev/null; then
  sudo apt-get -y -qq update
fi
if ! apt-cache show ${KERNEL_HEADERS_PKG} &>/dev/null; then
  DFLT_LIBBPFVER=0
  DFLT_KMODSVER=0
  KERNEL_HEADERS_PKG=
fi
if [[ $(uname -r) == 5.10.* ]]; then
  # kmods build is broken on kernel 5.10, https://bugs.debian.org/975571
  # TODO delete this when Debian and Ubuntu fix this bug
  DFLT_KMODSVER=0
fi
if [[ $(uname -r | awk -F. '{ print ($1*1000+$2>=4018) }') -eq 0 ]]; then
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
TARGETARCH=$DFLT_TARGETARCH

ARGS=$(getopt -o 'hy' --long 'dir:,node:,go:,libbpf:,dpdk:,kmods:,spdk:,ubpf:,arch:,skiprootcheck' -- "$@")
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
      Set libbpf version. '0' to skip.
  --dpdk=${DFLT_DPDKVER}
      Set DPDK version. '0' to skip.
  --kmods=${DFLT_KMODSVER}
      Set DPDK kernel modules branch or commit SHA. '0' to skip.
  --spdk=${DFLT_SPDKVER}
      Set SPDK version. '0' to skip.
  --arch=${DFLT_TARGETARCH}
      Set target architecture.
EOT
  exit 0
fi

if [[ $(id -u) -eq 0 ]] && [[ $SKIPROOTCHECK -ne 1 ]]; then
  echo 'Do not run this script as root'
  exit 1
fi

echo "Will download to ${CODEROOT}"

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
  echo "Will install libbpf ${LIBBPFVER}"
fi

if [[ $DPDKVER == '0' ]]; then
  if ! [[ -f /usr/local/include/rte_common.h ]]; then
    echo '--dpdk=0 specified but DPDK headers are absent'
    exit 1
  fi
else
  echo "Will install DPDK ${DPDKVER} for ${TARGETARCH} architecture"
fi

if [[ $KMODSVER != '0' ]]; then
  echo "Will install DPDK kernel modules ${KMODSVER}"
fi

if [[ $SPDKVER == '0' ]]; then
  if ! [[ -f /usr/local/include/spdk/version.h ]]; then
    echo '--spdk=0 specified but SPDK headers are absent'
    exit 1
  fi
else
  echo "Will install SPDK ${SPDKVER} for ${TARGETARCH} architecture"
fi

echo 'Will install other apt and pip packages'
echo 'Will delete conflicting versions if present'
if [[ $CONFIRM -ne 1 ]]; then
  read -p 'Press ENTER to continue or CTRL+C to abort '
fi

sudo mkdir -p $CODEROOT
sudo chown -R $(id -u):$(id -g) $CODEROOT

echo 'Dpkg::Options {
   "--force-confdef";
   "--force-confold";
}
APT::Install-Recommends "no";
APT::Install-Suggests "no";' | sudo tee /etc/apt/apt.conf.d/80custom >/dev/null
if [[ $(awk '$1=="deb" && $3=="buster"' /etc/apt/sources.list | wc -l) -gt 0 ]]; then
  echo 'deb http://deb.debian.org/debian/ buster-backports main' | sudo tee /etc/apt/sources.list.d/buster-backports.list
  sudo apt-get -y -qq update
fi
sudo DEBIAN_FRONTEND=noninteractive apt-get -y -qq dist-upgrade
sudo DEBIAN_FRONTEND=noninteractive apt-get -y -qq install \
  build-essential \
  clang-8 \
  clang-format-8 \
  doxygen \
  git \
  jq \
  kmod \
  libc6-dev-i386 \
  libelf-dev \
  libnuma-dev \
  libssl-dev \
  liburcu-dev \
  pkg-config \
  python3-distutils \
  yamllint \
  $KERNEL_HEADERS_PKG

if [[ $NODEVER != '0' ]]; then
  curl -sfL https://deb.nodesource.com/setup_${NODEVER} | sudo bash -
  sudo DEBIAN_FRONTEND=noninteractive apt-get -y -qq install nodejs
fi

curl -sfL https://bootstrap.pypa.io/get-pip.py | sudo python3
sudo pip install -U meson ninja

if [[ $GOVER != '0' ]]; then
  sudo rm -rf /usr/local/go
  curl -sfL https://dl.google.com/go/${GOVER}.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
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
  curl -sfL https://github.com/iovisor/ubpf/archive/${UBPFVER}.tar.gz | tar -xz
  cd ubpf-${UBPFVER}/vm
  make
  sudo make install
fi

if [[ $LIBBPFVER != '0' ]]; then
  cd $CODEROOT
  rm -rf libbpf-${LIBBPFVER}
  mkdir libbpf-${LIBBPFVER}
  cd libbpf-${LIBBPFVER}
  curl -sfL -o libbpf0_${LIBBPFVER}_amd64.deb \
    https://mirrors.kernel.org/ubuntu/pool/universe/libb/libbpf/libbpf0_${LIBBPFVER}_amd64.deb
  curl -sfL -o libbpf-dev_${LIBBPFVER}_amd64.deb \
    https://mirrors.kernel.org/ubuntu/pool/universe/libb/libbpf/libbpf-dev_${LIBBPFVER}_amd64.deb
  sudo dpkg -i libbpf0_${LIBBPFVER}_amd64.deb libbpf-dev_${LIBBPFVER}_amd64.deb
  sudo mkdir -p /usr/local/include/linux
  if ! [[ -f /usr/local/include/linux/if_xdp.h ]]; then
    sudo ln -s /usr/src/linux-headers-$(uname -r)/include/uapi/linux/if_xdp.h /usr/local/include/linux/if_xdp.h
  fi
fi

if [[ $DPDKVER != '0' ]]; then
  cd $CODEROOT
  rm -rf dpdk-${DPDKVER}
  curl -sfL https://static.dpdk.org/rel/dpdk-${DPDKVER}.tar.xz | tar -xJ
  cd dpdk-${DPDKVER}
  meson -Ddebug=true -Doptimization=3 -Dmachine=${TARGETARCH} -Dtests=false --libdir=lib build
  cd build
  ninja
  sudo find /usr/local/lib -name 'librte_*' -delete
  sudo ninja install
  sudo find /usr/local/lib -name 'librte_*.a' -delete
  sudo ldconfig
fi

if [[ $KMODSVER != '0' ]]; then
  cd $CODEROOT
  rm -rf dpdk-kmods
  git clone https://dpdk.org/git/dpdk-kmods
  cd dpdk-kmods
  git -c advice.detachedHead=false checkout $KMODSVER
  cd linux/igb_uio
  make
  UIODIR=/lib/modules/$(uname -r)/kernel/drivers/uio
  sudo install -d -m0755 $UIODIR
  sudo install -m0644 igb_uio.ko $UIODIR
  sudo depmod
fi

if [[ $SPDKVER != '0' ]]; then
  cd $CODEROOT
  rm -rf spdk-${SPDKVER}
  curl -sfL https://github.com/spdk/spdk/archive/v${SPDKVER}.tar.gz | tar -xz
  cd spdk-${SPDKVER}
  sudo ./scripts/pkgdep.sh
  ./configure --target-arch=${TARGETARCH} --enable-debug --disable-tests --with-shared \
    --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse
  make -j$(nproc)
  sudo find /usr/local/lib -name 'libspdk_*' -delete
  sudo make install
  sudo find /usr/local/lib -name 'libspdk_*.a' -delete
  sudo ldconfig
fi

sudo update-alternatives --remove-all python || true
sudo update-alternatives --install /usr/bin/python python /usr/bin/python3 1

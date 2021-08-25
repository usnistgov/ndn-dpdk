project('NDN-DPDK', 'c',
  default_options: [
    'b_pie=true',
    'debug=true',
    'optimization=3',
    'warning_level=2',
    'werror=true'
  ],
  meson_version: '>=0.55.0'
)

subdir('csrc')
subdir('mk')

cc = meson.get_compiler('c')
dpdk = dependency('libdpdk')
spdk = dependency('spdk_init')
uring = dependency('liburing')

static_library('ndn-dpdk-c', csrc,
  dependencies: [dpdk, spdk, uring],
  pic: true
)

run_target('cgoflags',
  command: [find_program('mk/cgoflags.sh'), cgoflags_dirs]
)
run_target('cgostruct',
  command: [find_program('mk/cgostruct.sh'), cgostruct_dirs]
)
run_target('cgotest',
  command: [find_program('mk/cgotest.sh'), cgotest_dirs]
)
run_target('schema',
  command: [find_program('mk/schema/build.sh')]
)
run_target('bpf',
  command: [find_program('mk/bpf.sh'), (cc.has_header('bpf/bpf_helpers.h') ? 'xdp=1' : 'xdp=0')]
)

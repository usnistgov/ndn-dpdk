project('NDN-DPDK', 'c',
  default_options: [
    'b_pie=true',
    'debug=true',
    'optimization=3',
    'warning_level=2',
    'werror=true'
  ])

add_global_arguments(
  '-Wno-error=sign-compare',
  '-Wno-unused-function',
  '-Wno-unused-parameter',
  language: 'c'
)
if meson.get_compiler('c').get_id() == 'gcc'
  add_global_arguments(
    '-Wno-error=implicit-fallthrough',
    language: 'c'
  )
endif
if meson.get_compiler('c').get_id() == 'clang'
  add_global_arguments(
    '-Wno-error=address-of-packed-member',
    language: 'c'
  )
endif

dpdk = dependency('libdpdk')

subdir('csrc')
static_library('ndn-dpdk-c', csrc,
  dependencies: [dpdk],
  pic: true)

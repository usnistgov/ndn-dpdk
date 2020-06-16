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

subdir('csrc')
subdir('mk')

dpdk = dependency('libdpdk')

static_library('ndn-dpdk-c', csrc,
  dependencies: [dpdk],
  pic: true
)

alias_target('cgoflags', custom_target('cgoflags',
  build_always_stale: true,
  command: [find_program('mk/cgoflags.sh'), cgoflags_dirs],
  output: 'cgoflags.done',
  input: []
))

alias_target('cgostruct', custom_target('cgostruct',
  build_always_stale: true,
  command: [find_program('mk/cgostruct.sh'), cgostruct_dirs],
  output: 'cgostruct.done',
  input: []
))

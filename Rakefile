BPFCC = ENV["BPFCC"] || "clang-8"
BPFFLAGS = "-O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu"

desc "Build forwarding strategies"
task "strategies" => "strategy/strategy_elf/bindata.go"
SgBpfPath = "build/strategy-bpf"
directory SgBpfPath
file "strategy/strategy_elf/bindata.go" do |t|
  sh "go-bindata -nomemcopy -nometadata -pkg strategy_elf -prefix #{SgBpfPath} -o /dev/stdout #{SgBpfPath} | gofmt -s > #{t.name}"
end
SgDeps = [SgBpfPath, "build/libndn-dpdk-c.a"]
SgSrc = Rake::FileList["strategy/*.c"]
SgSrc.exclude("strategy/api*")
SgSrc.each do |f|
  name = f.pathmap("build/strategy-bpf/%n.o")
  file name => [f] + SgDeps do |t|
    sh "#{BPFCC} #{BPFFLAGS} -c #{t.source} -o #{t.name}"
  end
  task "strategy/strategy_elf/bindata.go" => name
end

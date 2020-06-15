BPFCC = ENV["BPFCC"] || "clang-8"
BPFFLAGS = "-O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu"

desc "Generate **/cgostruct.go"
task "cgostruct"
Rake::FileList["**/cgostruct.in.go"].each do |f|
  name = f.pathmap("%d/cgostruct.go")
  file name => f do |t|
    sh "mk/godef.sh #{name}"
  end
  task name => f.pathmap("%d/cgoflags.go")
  task "cgostruct" => name
end

CDeps = {}
CDeps["app/fwdp"] = ["app/inputdemux", "container/fib", "container/pcct"]
CDeps["app/fetch"] = ["container/mintmr", "iface"]
CDeps["app/inputdemux"] = ["container/ndt", "container/pktqueue", "iface"]
CDeps["app/ping"] = ["app/inputdemux", "app/pingclient", "app/pingserver"]
CDeps["app/pingclient"] = ["iface"]
CDeps["app/pingserver"] = ["iface"]
CDeps["appinit"] = ["dpdk/pktmbuf"]
CDeps["container/cs"] = ["container/pcct"]
CDeps["container/diskstore"] = ["spdk", "ndn"]
CDeps["container/fib"] = ["container/strategycode", "core/urcu", "dpdk/mempool", "ndn"]
CDeps["container/mintmr"] = ["dpdk/eal"]
CDeps["container/mintmr/mintmrtest"] = ["container/mintmr"]
CDeps["container/ndt"] = ["ndn"]
CDeps["container/pcct"] = ["container/fib", "container/mintmr"]
CDeps["container/pit"] = ["container/pcct"]
CDeps["container/pktqueue"] = ["dpdk/pktmbuf", "dpdk/ringbuffer"]
CDeps["container/strategycode"] = ["core"]
CDeps["core"] = []
CDeps["core/coretest"] = ["core"]
CDeps["core/runningstat"] = []
CDeps["core/urcu"] = []
CDeps["dpdk/cryptodev"] = ["dpdk/mempool"]
CDeps["dpdk/eal"] = ["core"]
CDeps["dpdk/eal/ealtest"] = ["dpdk/eal"]
CDeps["dpdk/ethdev"] = ["dpdk/pktmbuf"]
CDeps["dpdk/mempool"] = ["dpdk/eal"]
CDeps["dpdk/pktmbuf"] = ["dpdk/mempool"]
CDeps["dpdk/pktmbuf/mbuftestenv"] = ["dpdk/pktmbuf"]
CDeps["dpdk/ringbuffer"] = ["dpdk/eal"]
CDeps["iface"] = ["mgmt/hrlog", "ndn"]
CDeps["iface/ethface"] = ["iface"]
CDeps["iface/ifacetest"] = ["iface"]
CDeps["iface/mockface"] = ["iface"]
CDeps["iface/socketface"] = ["iface"]
CDeps["mgmt/hrlog"] = ["dpdk/pktmbuf", "dpdk/ringbuffer"]
CDeps["ndn"] = ["dpdk/cryptodev", "dpdk/pktmbuf"]
CDeps["spdk"] = ["dpdk/pktmbuf"]
CDeps["strategy"] = ["container/fib", "container/pcct", "ndn"]

desc "Generate **/cgoflags.go"
task "cgoflags"
CgoflagsPathmap = "%p/cgoflags.go"
CDeps.each do |key,value|
  name = key.pathmap(CgoflagsPathmap)
  file name => value.map{|v| v.pathmap(CgoflagsPathmap)} do |t|
    sh "mk/cgoflags.sh #{key} #{value.join(" ")}"
  end
  task "cgoflags" => name
end
Rake::Task["strategy".pathmap(CgoflagsPathmap)].clear

file "ndn/error.go" => "ndn/error.tsv" do
  sh "ndn/make-error.sh"
end
file "ndn/tlv-type.go" => "ndn/tlv-type.tsv" do
  sh "ndn/make-tlv-type.sh"
end
task "ndn/cgostruct.go" => ["ndn/error.go", "ndn/tlv-type.go"]

desc "Build forwarding strategies"
task "strategies" => "strategy/strategy_elf/bindata.go"
SgBpfPath = "build/strategy-bpf"
directory SgBpfPath
file "strategy/strategy_elf/bindata.go" do |t|
  sh "go-bindata -nomemcopy -pkg strategy_elf -prefix #{SgBpfPath} -o /dev/stdout #{SgBpfPath} | gofmt -s > #{t.name}"
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

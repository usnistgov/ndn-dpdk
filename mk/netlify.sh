#!/bin/bash
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"/..

rm -rf build/netlify
mkdir -p build/netlify

cp -r build/docs/html build/netlify/doxygen
find build/netlify/doxygen -name '*.html' | xargs sed -i '/<\/head>/ i\<script async src="https://www.googletagmanager.com/gtag/js?id=G-YSW3MP43Z4"></script><script>window.dataLayer=[];function gtag(){dataLayer.push(arguments);}if(location.hostname.endsWith(".ndn.today")){gtag("js",new Date());gtag("config","G-YSW3MP43Z4");}</script>'

mkdir -p build/netlify/schema
find build/share/ndn-dpdk -name '*.json' | xargs cp -t build/netlify/schema/

build/bin/ndndpdk-svc &
NDNDPDK_SVC_PID=$!
corepack pnpm -s dlx graphqurl http://127.0.0.1:3030 --introspect > build/netlify/schema/ndndpdk-svc.graphql
kill $NDNDPDK_SVC_PID

cp docs/favicon.ico build/netlify/

cat > build/netlify/_redirects <<EOT
https://ndn-dpdk.netlify.app/* https://ndn-dpdk.ndn.today/:splat 301!
https://ndn-dpdk.ndn.today/ https://github.com/usnistgov/ndn-dpdk 301!
EOT

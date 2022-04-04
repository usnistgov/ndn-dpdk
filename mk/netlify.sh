#!/bin/bash
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"/..

rm -rf build/netlify
mkdir -p build/netlify

cp -r build/docs/html build/netlify/doxygen
find build/netlify/doxygen -name '*.html' | xargs sed -i '/<\/head>/ i\<script>(function(e,t,n,i,s,a,c){e[n]=e[n]||function(){(e[n].q=e[n].q||[]).push(arguments)};a=t.createElement(i);c=t.getElementsByTagName(i)[0];a.async=true;a.src=s;c.parentNode.insertBefore(a,c)})(window,document,"galite","script","https://cdn.jsdelivr.net/npm/ga-lite@2/dist/ga-lite.min.js"); if (location.hostname.endsWith(".ndn.today")) { galite("create", "UA-935676-11", "auto"); galite("send", "pageview"); }</script>'

mkdir -p build/netlify/schema
find build/share/ndn-dpdk -name '*.json' | xargs cp -t build/netlify/schema/

build/bin/ndndpdk-svc &
NDNDPDK_SVC_PID=$!
npx -y graphqurl http://127.0.0.1:3030 --introspect > build/netlify/schema/ndndpdk-svc.graphql
kill $NDNDPDK_SVC_PID

cp docs/favicon.ico build/netlify/

cat > build/netlify/_redirects <<EOT
https://ndn-dpdk.netlify.app/* https://ndn-dpdk.ndn.today/:splat 301!
https://ndn-dpdk.ndn.today/ https://github.com/usnistgov/ndn-dpdk 301!
EOT

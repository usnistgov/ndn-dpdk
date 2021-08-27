#!/bin/bash
set -eo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"/..

rm -rf build/docs

make doxygen
find build/docs/html -name '*.html' | xargs sed -i '/<\/head>/ i\ <script>(function(e,t,n,i,s,a,c){e[n]=e[n]||function(){(e[n].q=e[n].q||[]).push(arguments)};a=t.createElement(i);c=t.getElementsByTagName(i)[0];a.async=true;a.src=s;c.parentNode.insertBefore(a,c)})(window,document,"galite","script","https://cdn.jsdelivr.net/npm/ga-lite@2/dist/ga-lite.min.js"); if (location.hostname.endsWith(".ndn.today")) { galite("create", "UA-935676-11", "auto"); galite("send", "pageview"); }</script>'
mv build/docs/html build/docs/doxygen

mkdir -p build/docs/schema
find build/share/ndn-dpdk -name '*.json' | xargs cp -t build/docs/schema/

cp docs/favicon.ico build/docs/

cat > build/docs/_redirects <<EOT
https://ndn-dpdk.netlify.app/* https://ndn-dpdk.ndn.today/:splat 301!
https://ndn-dpdk.ndn.today/ https://github.com/usnistgov/ndn-dpdk 301!
EOT

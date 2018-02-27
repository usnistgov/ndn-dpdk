#!/bin/bash
# write godoc as HTML
OUTDIR=docs/godoc
mkdir -p $OUTDIR

(
  echo '<!DOCTYPE html>'
  echo '<title>ndn-dpdk godoc</title>'
  echo '<h1>ndn-dpdk godoc</h1>'
  echo '<ul>'
  for PKG in $(find -path ./integ -prune -o -name '*.go' -printf '%h\n' | sort -u); do
    PKGNAME=$(echo $PKG | sed 's|^\./|ndn-dpdk/|')
    HTMLFILE=$(echo $PKG | sed -e 's|^\./||' -e 's|/|\.|g').html
    echo '<li><a href="'$HTMLFILE'">'$PKGNAME'</a></li>'
    godoc -url '/pkg/'$PKGNAME | sed 's|/lib/|https://golang.org/lib/|' > $OUTDIR/$HTMLFILE
  done
  echo '</ul>'
) > $OUTDIR/index.html

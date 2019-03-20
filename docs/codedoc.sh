#!/bin/bash
# write godoc and Markdown as HTML
OUTDIR=docs/codedoc
mkdir -p $OUTDIR
rm $OUTDIR/*

(
  echo '<!DOCTYPE html>'
  echo '<title>ndn-dpdk codedoc</title>'
  echo '<h1>ndn-dpdk codedoc</h1>'
  echo '<ul>'

  for PKG in $(find -mindepth 2 -type f \( -name '*.go' -o -name '*.md' \) -printf '%h\n' | grep -v node_modules | sort -u); do
    PKGNAME=$(echo $PKG | sed 's|^\./|ndn-dpdk/|')
    PREFIX=$(echo $PKG | sed -e 's|^\./||' -e 's|/|\.|g')
    echo '<li><b>'$PKGNAME'</b>'

    HTML=$PREFIX.godoc.html
    godoc -url '/pkg/'$PKGNAME | sed 's|/lib/|https://golang.org/lib/|' > $OUTDIR/$HTML
    echo ' <a href="'$HTML'">godoc</a>'

    for MD in $(find $PKG -maxdepth 1 -name '*.md'); do
      TITLE=$(basename $MD .md)
      HTML=$PREFIX.$TITLE.html
      pandoc -s -o $OUTDIR/$HTML $MD
      echo ' <a href="'$HTML'">'$TITLE'</a>'
    done

    echo '</li>'
  done

  echo '</ul>'
  echo '<p>Generated on '$(date -u)'</p>'
) > $OUTDIR/index.html

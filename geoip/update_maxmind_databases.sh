#!/bin/bash

# Get license key from ENV or first arg.
LICENSE_KEY=${1:-$MAXMIND_LICENSE_KEY}

if [ "$LICENSE_KEY" == "" ];
then
    echo "No Maxmind license key. Add to environment as MAXMIND_LICENSE_KEY or pass as first arg to this script" 1>&2
    exit 1
fi

download_and_extract() {
    local url="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-${1}&license_key=${LICENSE_KEY}&suffix=tar.gz"
    local target=$2
    echo "Downloading from: $url"
    echo "              to: $target"
    curl --silent "$url" | tar -zxOf- --wildcards "*.mmdb" > $target
}


#download_and_extract ASN ./maxmind-geolite2-asn.mmdb
download_and_extract City ./maxmind-geolite2-city.mmdb
#download_and_extract Country ./maxmind-geolite2-country.mmdb

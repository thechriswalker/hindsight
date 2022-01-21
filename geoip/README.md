# Geolocation

MaxMind offer a free dataset for IP Geolocation, however you must register
and obtain a license key to download the datasets. Looking around on github
there are definitely projects that use and embed the datasets within the
code, so I don't feel bad putting these large files here.

But there is way more data than we need. I would rather process the files to
strip out the data we don't need, but that is a whole library of it's own.
Maybe in the future.

To reduce the amount of data to embed, I have chosen to only lookup the country
code and the ASN. The DBs needed for this are only ~13MB.

I should probably also build in some code to periodically update the databases.

Let's call that phase 2...

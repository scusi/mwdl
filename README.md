# mwdl - malware download tool

## Requirements

### Install tor

You need to have tor installed local on your machine or you need to have a tor-node with a SOCKS5 port open which you can use.
In case you have a local tor with an open SOCKS5 listener on 127.0.0.1 port 9050 you do not need to change anything.

If you do not, you need to make sure your _torrc_ file containes the following line. 
_torrc_ file usually resides at _/etc/tor/torrc_ on a unix machine.

```SOCKSPort localhost:9050```

If you want to use a tor-socks5 port on some other node, please use the 't' switch like in the following example:

```
mwdl -t tor.ccc.de:9050 -u https://github.com/scusi/mwdl/archive/master.zip
```

## Install Release

## Install from Source

In order to install from source you need to have a go workspace setup and the sources.

### Install a golang workspace

If you do not have a go workspace installed on your machine you need to do this first.
Please follow the official _Getting started_ guide from: https://golang.org/doc/install.

### Get the Sourcecode

Once you have a go workspace you can get the sources and install with ```go get```

```
go get github.com/scusi/mwdl
```

## Usage of mwdl

show help

```
mwdl -h 
```

download the zipfile of this project with mwdl

```
mwdl -u https://github.com/scusi/mwdl/archive/master.zip
```

You can also pass a file with urls (one per line) to be downloaded.

```
mwdl -f fileWithUrls.txt
```

### Outputformat

Files a written to the local directory by default.
The naming convention is:

 ```<md5sum(url)>_<filename>```

where <md5sum(url)> is replaced by the md5 sum of the url requested.
<filename> will be taken from the path or (if there is not filename in the path) use ```outfile```.




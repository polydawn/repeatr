# repeatr [![Build status](https://img.shields.io/travis/polydawn/repeatr/master.svg?style=flat-square)](https://travis-ci.org/polydawn/repeatr)

Run it. Run it again.

## What's all this then?

Repeatr is a process reasoning framework. You have things to do, and repeatr helps you do them!

Repeatr is very much a work in progress. Some assembly may be required. Not tested on animals.

## Try it out

First, clone our repository and its dependencies, then download some testing assets:

```bash
# Clone
git clone https://github.com/polydawn/repeatr.git && cd repeatr

# Install dependencies
./goad init

# Download a small Ubuntu tarball and required binary
./lib/integration/assets.sh
```

Build repeatr:

```bash
# Build
./goad install

# See usage
./goad exec

# Try an example!
sudo ./goad exec run -i lib/integration/basic.json

# See the forumla repeatr just ran
cat lib/integration/basic.json

# See the /var/log output repeatr has generated!
tar -tf basic.tar
```

You can run our test suite (several of which will will be skipped without root):

```
sudo ./goad test
```

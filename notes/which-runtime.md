# Runtime notes

Repeatr desires a container runtime which is well-adapted to:

* Sane process model
* Low-friction path to guaranteed idempotency
* Trivial usage of arbitrary rootFS somewhere on the system
* Basic networking

Nice-to-haves:

* Host mounts
* Trivial compatibility with `docker export` and rootFS tars
* No persistent artifacts to clean & GC

Deciding which to use for the above properties is the purpose of this document.

This is NOT an exhaustive list of every runtime or even an [objective feature comparison](https://github.com/containers/support-matrix#support-matrix), but instead an informal fitness evaluation for our specific purposes.

By design, repeatr does not care much about runtimes: they are just one of arbitrarily-many job executors. Thus the relative velocity of these projects matters less than the current position; runtimes could later be trivially swapped for any reason.


## [Libcontainer](https://github.com/docker/libcontainer) / nsinit

The library that powers Docker (same authors), and a CLI using said library.

Executive summary: provides all desired features, slightly prickly and unfriendly.
Acceptable in light of convenience.

### Building

From [docs](https://github.com/docker/libcontainer/blob/master/CONTRIBUTING.md#building-libcontainer-directly). Establish a clean gopath, and:

```bash
go get -v github.com/docker/libcontainer
cd $GOPATH/src/github.com/docker/libcontainer

# They has some pinned deps. Script branch refs rather than submodules :(
export GOPATH=$PWD/vendor:$GOPATH
./update-vendor.sh
go get -v -d ./...

# Builds (shell and git only; finds package names)
make direct-build direct-install

# Run the tests (some tests fail without root)
make direct-test-short | egrep --color 'FAIL|$'

# Run all the test (more tests fail without root)
# Only failures right now are some 'not a symlink' errors
make direct-test | egrep --color 'FAIL|$'
```

This accomplishes a build with only go + shell.

Opened ticket regarding failing tests: https://github.com/docker/libcontainer/issues/398

Problem: Identifying a stable point to build libcontainer from, if in fact such a concept exists.
Solution: Use hard library references. Should be trivial to untangle this into golink, for example.

### Execution

Default operation is to reference an incredibly long `container.json`. While entirely reasonable, this file is largely boilerplate (insofar as you will probably always want device mounts, etc).
Luckily, the `--create` flag will implicitly use a default & sane config, extended by runtime flags.

The result is not far removed from `docker run`, with several advantages.
For example:

```bash
# Disregard layers, acquire containers
docker run --name "export-me" ubuntu:14.04 /bin/true
docker export export-me > ubuntu.tar

# Needs root for device files; Fear and Mknod in Las Vegas
mkdir myroot
tar -xf ubuntu.tar -C myroot

# Nsinit forks itself; put it on your path before continuing
which nsinit

# The rootfs flag requires an absolute path
nsinit exec --create --tty --rootfs $PWD/myroot /bin/bash
```

You now have a shell. The rootfs is obviously persistant.

### Dreams of the internet

Running `nsinit config` results in much json.
I think this output is the default for `--create`.

Ref libcontainer's [sample_configs folder](https://github.com/docker/libcontainer/tree/master/sample_configs) for several examples. Verbose as they are, diffing against the `minimal.json` can be instructive.

You'll note your initial invokation has no internet access:

```
$ ifconfig
lo        Link encap:Local Loopback
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)
```

We can discover why by having a look at the abbridged output of `nsinit config | grep networks -C 30`:

	"networks": [
		{
			"type": "loopback",
			"name": "",
			"bridge": "",
			"mac_address": "",
			"address": "127.0.0.1/0",
			"gateway": "localhost",
			"ipv6_address": "",
			"ipv6_gateway": "",
			"mtu": 0,
			"txqueuelen": 0,
			"host_interface_name": ""
		}
	],

So, we're simply not asking nsinit for a relevant network interface.
From a sample configuration (also abridged):

```
$ diff minimal.json attach_to_bridge.json
>         {
> 			"type": "veth",
> 			"name": "eth0",
> 			"bridge": "docker0",
> 			"mac_address": "",
> 			"address": "172.17.0.101/16",
> 			"gateway": "172.17.42.1",
> 			"ipv6_address": "",
> 			"ipv6_gateway": "",
> 			"mtu": 1500,
> 			"txqueuelen": 0,
> 			"host_interface_name": "vethnsinit"
< }
```

Adding some relevant flags to our invocation will generate this configuration:

	--veth-bridge 				veth bridge
	--veth-address 				veth ip address
	--veth-gateway 				veth gateway address
	--veth-mtu '0'				veth mtu

To wit: `--veth-bridge docker0 --veth-address "172.17.0.101/16" --veth-gateway "172.17.42.1" --veth-mtu 1500`


### Interlude & commentary

If you omit the MTU flag (defaults to 0; example shown of 1500) it will crash.

In general this binary is a tad fragile & irritable; my assessment is that you'll want to already have resolved a sane environment and configuration *before* invoking. Know that your invokation will succeed, versus attempting to reverse-engineer that an "out of bounds" error means you forgot to specify a command (for example). Possible target for future libcontainer PRs!

### Networking 2: Network Harder

A network bridge is something Docker provides automatically, and is a feature not explicitly provided here. Anyone who has invoked the Docker daemon already has a `docker0` bridge.

From the [contribution docs](https://github.com/docker/libcontainer/blob/master/CONTRIBUTING.md#testing-changes-with-nsinit-directly):

```bash
# Optional, add a docker0 bridge
ip link add docker0 type bridge
ifconfig docker0 172.17.0.1/16 up
```

For now, add that bridge either via those commands or running a Docker daemon briefly.
We'll eventually want to use a different bridge name and subnet, for obvious reasons.

Result (and added a PATH for good measure):

`nsinit exec --create --tty --veth-bridge docker0 --veth-address "172.17.0.101/16" --veth-gateway "172.17.42.1" --veth-mtu 1500 --rootfs $PWD/myroot --env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin' bash`

### DNS

You now have a network address  access, but not DNS.
Inside the container, here is my immediate way to fix that:

`echo "nameserver 8.8.8.8" >  /etc/resolv.conf`

Checking if the rootFS `resolve.conf` (or distribution-specific? ugh) file is valid and defaulting to correcting it might be sane. Unless there's a better way to do this, which wouldn't surprise me.

### Subnets and collisions

Every container ends up with an IPv4 address. While not necessarily a hard requirement, it makes things a lot simpler for the (frankly unready for IPv6) world we live in. Thus, avoiding multiple IP assignments to running invocations is desirable.

Having a server / client daemon architecture (as per the Docker daemon) is one solution to this problem, but violates one of our design goals. Additional approaches include, but are not limited to:

* File-locked index (eg, BoltDB, sqlite)
* POSIX convention (pid-IP pairs, store in file/etc?)
* Sysadmin-experienced colleague claims linux networking solution (ARP? I forget)
* Use a random local IPv6 address

One of the first three would probably be optimal.


## Rocket

Runtime by the CoreOS folks.

Executive summary: has some great features; defaulting to PGP is very cool.
Currently has strong opinions about CAS and ACI + GC (!) stuff that conflict with our goals.
Nascent run options are worrying relative to competitors; will likely recede with time.
A possible integration in the future.

### Trust and transport

Unlike libcontainer, rocket is aware of a container transport (URLs).
Defaults to using PGP keys:

`rkt trust --prefix coreos.com/etcd`
Results in `/etc/rkt/trustedkeys/prefix.d/coreos.com/etcd/8b86de...`

Transport seems to match ACI domain to key domain and assumes URL has .aci and .sig suffixes.
You can pass two URLs to manually specify where they are.

### Storage

My current understanding is that Rocket requires use of its CAS for storage and invocation.

`rkt fetch coreos.com/etcd:v2.0.0`

In /var/lib/rkt/cas/remote/sha512/ae/sha512-aebda6...

	{ "ACIURL" : "https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.aci",
	  "BlobKey" : "sha512-fa1cb92dc276b0f9bedf87981e61ecde93cc16432d2441f23aa006a42bb873df",
	  "ETag" : "",
	  "SigURL" : "https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.sig"
	}

Meanwhile, the ACI landed in `/var/lib/rkt/cas/blob/sha512/fa/sha512-fa1cb92d...`

### ACI format

An ACI is a renamed tar.gz

Contents:
	/rootfs
	/manifest

Example manifest:

	{ "acKind" : "ImageManifest",
	  "acVersion" : "0.1.1",
	  "app" : { "exec" : [ "/etcd" ],
	      "group" : "0",
	      "user" : "0"
	    },
	  "labels" : [ { "name" : "os",
	        "value" : "linux"
	      },
	      { "name" : "arch",
	        "value" : "amd64"
	      },
	      { "name" : "version",
	        "value" : "v2.0.0"
	      }
	    ],
	  "name" : "coreos.com/etcd"
	}

So notably, NOT compatible with Docker export out of the box, you WILL re-process for compat.

Their example (13 MB) rootfs is amusing; basically a git clone of etcd.
No normal linux FS layout; just the etcd binary and some markdown docs.
Go go golang power. Hooray for static binaries!

No particular reason that couldn't be done with other

### Attempt to import with Docker export

As a matter of practicality, (in my experience) developers will most strongly identify a "container" as something that can be trivially ran with Docker. In other words - a tarball of a rootFS.

This is also a convenient reason to keep runtime-specific configuration outside the tarball. You should be able to trvially throw these around between runtimes.

As shown above, ACIs do not quite do this, adding an extra layer. There are a few ways to convert a 'docker-compatible' rootFS to an ACI.


#### ACtool

Tool to build from a filesystem + manifest.

```bash
go build -o $GOPATH/bin/actool github.com/appc/spec/actool

# Assuming container exists from before (libcontainer tutorial)
docker export export-me > ubuntu.tar

mkdir -p wat/rootfs
tar -xf ubuntu.tar -C wat/rootfs/

# Create a manifest from below docker2aci section
nano wat/manifest

time ./bin/actool build ./wat watbuntu.aci

# real	0m18.098s
```

It's not actually clear to me if there's a reason this tool exists beyond convenience.
Was this all literally just to tar-gz it? Was there vodoo involved?
I'd need to double-check the ACI format and ACtool functionality.

Note the execution time. This is basically a build artifact needed to run a Rocket container.
It might be possible to expand a rootFS into the CAS by lying about its (unknown) hash?

#### docker2aci

Go binary that does conversion from a docker registry.

```bash
go get github.com/appc/docker2aci

time ./bin/docker2aci ubuntu:14.04

Downloading layer: 511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158
Downloading layer: fa4fd76b09ce9b87bfdc96515f9a5dd5121c01cc996cf5379050d8e13d4a864b
Downloading layer: 1c8294cc516082dfbb731f062806b76b82679ce38864dd87635f08869c993e45
Downloading layer: 117ee323aaa9d1b136ea55e4421f4ce413dfc6c0cc6b2186dea6c88d93e1ad7c
Downloading layer: 2d24f826cb16146e2016ff349a8a33ed5830f3b938d45c0f82943f4ab8c097e7

Generated ACI(s):
ubuntu-14.04.aci

# real	0m32.135s
```

Results in 189 MB file, same as a docker export of same.
Squashes docker layers by default.

Resultant generated manifest:

	{ "acKind" : "ImageManifest",
	  "acVersion" : "0.1.1",
	  "app" : { "environment" : [ { "name" : "PATH",
	            "value" : "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	          } ],
	      "exec" : [ "/bin/bash" ],
	      "group" : "0",
	      "user" : "0"
	    },
	  "labels" : [ { "name" : "version",
	        "value" : "14.04"
	      },
	      { "name" : "os",
	        "value" : "linux"
	      }
	    ],
	  "name" : "index.docker.io/ubuntu"
	}

Attempting to run this (note it is a bash entry point), I got:

```bash
$ rkt run ./ubuntu-14.04.aci
/etc/localtime is not a symlink, not updating container timezone.
Sending SIGTERM to remaining processes...
Sending SIGKILL to remaining processes...
Unmounting file systems.
Unmounting /proc/sys/kernel/random/boot_id.
All filesystems unmounted.
Halting system.
```

I actually did not get far enough to discover if TTYs are a feature.
Run flags are noticeably thin:

```
$ rkt run --help
Usage:
  -private-net=false: give container a private network
  -spawn-metadata-svc=false: launch metadata svc if not running
  -stage1-image="/pd/app/rocket/rocket-v0.3.1/stage1.aci": image to use as stage1, local paths and http/https urls are supported
  -volume=: volumes to mount into the shared container environment
```


#### The garbage man

Running any of these will end up with copies of the FS and ACI in the CAS /var/lib/rkt.
You now have a docker-esque GC problem of containers + images. Enjoy!

To be fair there is a `rkt gc` command. Exact behavior untested. But for fundamentally disposable & transient invocations, I can't ever see myself gratified for mandated use of a polluted CAS folder.


### Host mounts

Documentation notes [here](https://github.com/coreos/rocket/blob/master/Documentation/commands.md#mount-volumes-into-a-container).

It would appear that you can specify mounts in the ACI manifest or runtime flags.
Not sure if you need to specify first in ACI and *then* in flags! Unverified.


## [Novm](https://github.com/google/novm)

Google-built (unofficial) type 2 hypervisor in Go.

In general, guest-host-same-kernel is a blocker for ~50-year forward compatibility desires.
This category of solution is thus in need of research.

As per the commentary at the top of this file, adapting future executors would not be unduly complex.


## More

There's a few hundred more choices that could belong in this file.

Not all ended up with extensive enough notes to warrant inclusion.
Discussion PRs to this file accepted.

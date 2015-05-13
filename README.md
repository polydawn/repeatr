# repeatr [![Build status](https://img.shields.io/travis/polydawn/repeatr/master.svg?style=flat-square)](https://travis-ci.org/polydawn/repeatr)

Repeatr is a job execution framework that makes any task repeatable.
If you can run it with repeatr, you can run it again, you can still run it next tuesday, and you can still run it in 50 years.

Gone are the days of mysteriously "updating" dependencies that break your software;
gone are the days when losing access to a researcher's laptop means publication data is permanently unreproducible.

Welcome to repeatr.  Run it.  Run it again.



What's all this then?
---------------------

Repeatability is the cornerstone of science and engineering.
In any given process, if you start with the same inputs, and do the same process to them, you should get the same outputs, right?
Repeatr is about making that reality.

Repeatr combines sandboxing (so your know your application is working independently from the rest of the system)
with data integrity (meaning you can always start with a known system state).
It's a match that means real durability, and this solid base lets you drive the quality of your engineering even higher.


### Executive summary

Repeatr is a sandboxing execution framework, excellent at ETL jobs and working with "heavy" data.

What does that mean, buzzword by buzzword?

- **sandboxing**          -- if you can run it once, you can run it again.  we guarantee isolation from the environment, making system design and maintenance easier, reliable, and something you can do with confidence.
- **execution framework** -- this'll monitor jobs, gather logs, and make reporting accessible.
- **ETL**                 -- perfectly suited for extract-transform-load or other parallelizable tasks.
- **heavy data**          -- data that's hundreds of megs or dozens of gigs can managed and moved around quickly and reliably.  integrity guaranteed.

Our sandboxing framework provides freedom from vendor lock-in --
 frankly, a *really* big deal in the current ecosystem.
 (Quick, who's winning?  LXC, LXD, Docker, Rocket, systemd, sandstorm, or someone else out of left field?
 With repeatr's pluggable sandboxing backends, we're free to use any of them.)
The freedom to choose sandboxing backend means working within repeatr gives you confidence that your
 work is future-proofed, even when the landscape of sandboxing tools has changed.
No matter which execution system you want to use, our data integrity can be relied on to work the same way.

By painting our sandboxing API with broad strokes, we can even provide
 seamless transition from linux-lightweight-containers to full virtualization,
 meaning you can have the fastest systems when you want them, and full kernel
 isolation the instant you need it.

Repeatr is especially excellent at working with ETL-style data, but
any situation where you need to line up the data and then get down to work is a good fit for repeatr.


### Techno-mumbojumbo

- Containers!
- Content-addressable storage!
- Decentralized! Host your own!


### What is this good for?

Repeatability is critical in many fields.  Consider the following:

- Sandboxing to make sure your builds work -- like a Continuous Integration system, but also ready to help you debug reproducibility itself.
- Immutable deployments -- and we mean it.  Once the formula is committed, everything is pinned down and guaranteed to deliver.  Ops teams delivering IT infrastructure can rest easy.
- Data warehousing -- where file corruption can mean millions of dollars in damage from inaccurate data or expensive calculations that need to be re-run, repeatr detects issues before they cause problems.
- Software quality -- know with confidence that your software is race-condition free and acts the same every time.
- Roll it back -- formulas used in the past continue to work.  Forever.  If you change your data, and later discover you want to run with an older configuration again, that's *always* possible.

In any situation where quality is critical, repeatr can help you raise the bar.
Think of it like source control, but for your entire environment.



Technical Overview
------------------

Repeatr thinks of the world in formulas.
Just like the ones in grade school: 1 + 3 = 4.

The important thing about `1 + 3 = 4` is that it's the same -- Every time.
The inputs are 1 and 3; the process is "add"; and if the output isn't 4, we've got a problem with our addition.

Repeatr works similarly, except with repeatr, the formula goes like this:
`(inputs) + (computation) = (outputs)`

Given the same inputs, and the same computation, the outputs should naturally be identical.  (And when they're not, repeatr can tell you!)


### What's in a Formula?

A formula starts with an "input":

```
"Inputs": [{
	"Type": "tar",
	"Mountpoint": "/",
	"Hash": "b6nXWuXamKB3TfjdzUSL82Gg1avuvTk0mWQP4wgegscZ_ZzG9GfHDwKXQ9BfCx6v",
	"SiloURI": "s3+ca://mybucket/prefix/" // content-addressable!
}]
```

Inputs come in different types -- these are like plugins; repeatr can support local storage systems, S3, git, and a ton of other systems for data storage.

Inputs have a SiloURI -- this describes *where* the data is stored -- and a hash -- which describes the *identity* of the data.
(Repeatr treats most data storage as content-addressable.  This means good things for deduplication when you store large amounts of data, and it also means your data is always integrity-guaranteed.)

Finally, inputs have a Mountpoint.  In this example, it's `"/"` -- the root of a linux filesystem.  That's because in this example, we're creating a container's rootfs from that input data.

When you want to run a process on this data, that looks about like you'd expect.
Just put a snippet like this after your input definition:

```
"Execution": {
	"Cmd": [ "echo", "Hello from repeatr!" ]
}
```

Jobs usually produce some data that you want to keep.
For this, you can configure an output.
These will collect files after your job execution is complete, create the integrity check, and upload them to your storage.

Outputs look pretty similar to inputs, except instead of specifying the hash, it'll be given to you when the job runs:

```
"Outputs": [{
	"Type": "tar",
	"Mountpoint": "/var/log",
	"SiloURI": "file://assets/ubuntu.tar.gz" // just keep this output locally
}]
```

Now, here's where things really get interesting: you can have *lots* of inputs!

```
"Inputs": [
	{
		"Type": "s3",
		"Mountpoint": "/",
		"Hash": "b6nXWuXamKB3TfjdzUSL82Gg1avuvTk0mWQP4wgegscZ_ZzG9GfHDwKXQ9BfCx6v",
		"SiloURI": "s3+ca://mybucket/prefix/" // content-addressable!
	},
	{
		"Type": "dir",
		"Mountpoint": "/mnt/addtnl-data",
		"Hash": "9GYDihlrhHQRNPV10lms35kogosBekjqJVYzTj0O5H-QJYTU7vf0YAgh3XBWKKBC",
		"SiloURI": "file://mybucket/prefix/" // use local resources
	},
	{
		"Type": "ipfs",
		"Mountpoint": "/opt/app/python27",
		"Hash": "ipfs-sha1-welkjsoivweuhiuhsdf",
		"SiloURI": "ipv4://ipfs.cluster.mynet.org"
	},
	{
		"Type": "git",
		"Mountpoint": "/opt/algorithm",
		"Hash": "1c43gdf9j4",
		"SiloURI": [
			"ssh://git@mycorp.com/image-processor.git",     // use the in-house copy if possible
			"http://github.com/mycorp/image-processor.git", // public mirror has the data too
		]
	}
]
```

Multiple inputs, *all using the same integrity-guaranteed transport systems*, give you a ton of power as well as a ton of safety:

- Plug in applications, source code, and whole root filesystem images... with the same tool.
  - *Swap* in applications and source code, while *leaving the root filesystem alone*.
- Everything caches.  (Caching with content-addressable data storage is trivial!)
  - Suddenly being able to plug in applications and source code without touching the rootfs means layering functionality is in *your control*, and you can cache different pieces independently.
- Treat any of your data warehousing options interchangeably.  Don't worry about helper programs; Repeatr's already connected the dots.

Now combine all three:

```
{
	"Inputs": [...]
	"Execution": [...]
	"Outputs": [...]
}
```

and there you have it: a formula.



Try it out
----------

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
.gopath/bin/repeatr

# Try an example!
sudo .gopath/bin/repeatr run -i lib/integration/basic.json

# See the forumla repeatr just ran
cat lib/integration/basic.json

# See the /var/log output repeatr has generated!
tar -tf basic.tar
```

You can run our test suite (several of which will will be skipped without root):

```
sudo ./goad test
```

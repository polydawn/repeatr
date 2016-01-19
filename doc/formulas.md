Formulas
========

Repeatr thinks of the world in formulas.
Just like the ones in grade school: 1 + 3 = 4.

`1 + 3 = 4` is the same, every time.
The inputs are 1 and 3; the process is "add"; and if the output isn't 4, we've got a problem.

Repeatr formulas are the same: we list inputs, describe a process, and expect outputs.

Given the same inputs, and the same process, the outputs should naturally be identical.
Writing a Repeatr formula is a way to capture all these, and make the process both easy to run -- not just once, but over and over.
This makes it easy to check that processes are stable (and when they're not, Repeatr can tell you!).


What's in a Formula?
--------------------

Formulas come in three parts (and are often expressed in yaml):

```
inputs:
	- [... list of data ...]
action:
 	- [... some kind of script ...]
outputs:
	- [... list of directories we should save ...]
```

### Inputs

We need to get some data into place to get work done.  Inputs to the rescue!

```yaml
inputs:
	"/":
		type: "tar"
		hash: "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL"
		silo: "s3+ca://mybucket/my/data/"
```

Inputs need to specify
- where they should appear in the container's filesystem,
- what kind of storage system they're based on,
- the data's identifier,
- and where to get it from.

Inputs come in different types -- these are like plugins; Repeatr can support local storage systems, S3, git, and a ton of other systems for data storage.
This gives us a lot of flexibility: need some big files stored in a cloud service?  Got it.
Want to drag in the source from some project that tracks changes in git?  No problem.
Just wanna work offline on the local disk?  Yes ma'am or sir!
Treat any of your data warehousing options interchangeably.  Don't worry about helper programs; Repeatr's already connected the dots.

Inputs have a "silo" URL -- this describes where the data is stored -- and a hash -- which describes the identity of the data.  *Where* to get data versus *what* the identity of data to get are treated as totally separate concepts.  Why?  Because it's empowering:
- Stuff moves, frankly.  Network locations can change.
  - Is the identity of the data you want changed when it moves?  No!  Of course not.
  - Worldwide mirroring and offline caching Just Works this way.
  - You can list multiple network locations.  If one's down, Repeatr can just try the next (or use them both in parallel for speed!)
  - Need a formula to keep working for years and years?  We can configure network redirects; what the formula does is guaranteed not to be affected as a result.
- The data identifier can be derived from the data itself.  Data integrity?  Guaranteed.
  - Got a flaky network?  No big deal.  Partial downloads, even maliciously manipulated data, all of it sticks out like a sore thumb.
  - Storage can de-duplicate: Two pieces of identical data end up with identical names.  It's easy to see this, and just store one copy.
  - You can use a shared storage service without needing to trust it -- same story as network; nobody can change *what* the data is out from under your feet.  Intern fat-finger to full scale industrial espionage-powered malice, it just can't touch this.
  - Some advanced input plugins can even use this to do "swarm"-like network operations, making downloads incredibly fast (think bit-torrent).
- It's really convenient.  We'll see more about this when we get to talking about outputs... it means you can configure one "bucket" of storage, and have it be used for tons of stuff, pass it around between multiple steps of processing with their own formulas, and the whole time you never need to futz about giving data individual names.

If this sounds familiar, it probably is: in a nutshell, Repeatr treats most data storage as "content-addressable".

You'll probably need several inputs for most tasks.
Formulas almost always start with some kind of "operating system" input mounted at `/`, since Repeatr's promise of isolation means you start with *no* data from your host.
After that, it's up to you!  Do what fits.

It's common to see at least one other application mounted as an input and some kind of data for processing (source code?  raw experimental data?  your website and server config?) as more input(s), but anything is possible, and there is no limit to the number of inputs you can configure.

Some applications are difficult to install without taking a snapshot of the entire filesystem; that's totally okay.  However, multiple inputs open the door to going further!  Plug in applications, source code, and whole root filesystem images.  *Swap* in applications and source code, while *leaving the root filesystem alone*.  If you've previously worked within a system that can only do full filesystem snapshots, you can easily see how multiple inputs gives you a lot more power, flexibility, and efficiency.

### Actions

What do you want to get done?  That's your action.

```yaml
action:
	command: [ "echo", "Hello from repeatr!" ]
```

Actions are pretty much a shell script.  You can use bash, or python, or anything you can `exec`.

When your action is executed, it gets the full filesystem of all of the assembled inputs, so you can refer to any commands and tools you set up there.

When your action is finished executing, we move on to outputs...

### Outputs

So you produced some new files of data you want to keep?  Excellent.  That's an Output.

```yaml
outputs:
	"compiled":
		type: "tar"
		mount: "/task/build/bin"
		silo:
			- "file+ca:///assets/"       # keep a snapshot of this output locally
			- "s3+ca://mybucket/assets/" # upload another snapshot to the cloud!
```

Outputs will collect files after your job execution is complete, compute the data identifier, and upload them to any storage warehouse locations you specify.

Outputs look pretty similar to inputs.  The only difference is instead of providing the hash, Repeatr will give you the hash after your job is done.  (We're also flashing some different features here: you can name inputs and outputs and specify their mount path in the container separately, and we're also showing use of multiple storage locations here.  Both these things can be done with Inputs, too.)

Outputs and Inputs are symmetric with each other!  When an output produces a data ID, you can turn around and use that as an input in another formula.

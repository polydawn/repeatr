Containers
==========

Repeatr offers several different types of isolation systems.
We often refer to these as "containers".

We support several different kinds of containers and many different implementations.
Some may be more efficient than others;
some are certainly more secure than others if you need to run untrusted code;
some are all around more powerful but just a bigger PITA to set up.
So, Repeatr gives you choices.
As long as a system can give us some basic isolation from your host, we can make containers with it and run your jobs.

Repeatr will generally try to pick the most convenient and performant thing it can do on your host, and do that by default.
If that's good enough for you, you can skip this section of the docs.
If you care about:
- squeezing out more performance
- secure containers for untrusted processes
- advanced wizardry and just good ol' curiosity

... then read on!  But fair warning, there's no way to talk about this without getting deeply technical.


Standard vs not
---------------

Some features are minimum viable standards you can count on from all Repeatr containers.
Others are available from only some containment implementations; you're free to use these,
as long as you understand it may limit the portability of your work.

Here's a breakdown:

- Basic POSIX-like filesystems: standard ✔
- Basic POSIX-like execution model: standard ✔
- Environment variables: standard ✔
- Working directories ("cwd"): standard ✔
- Exit codes: standard ✔
- Hostname: ┐(￣ー￣)┌
- PID isolation: ┐(￣ー￣)┌
- Allow/Deny all networking: ┐(￣ー￣)┌
- Anything fancy networking: ヽ(´ー｀)ノ
- Resource limits: ┐(￣ー￣)┌

Generally, Repeatr's contract is to provide enough features to make guaranteed repeatable processes possible, and *that's it*.

Repeatr will also try to make it possible to do as much fancy footwork as you want.
If you want to use content-adddressible provisioning goodness to ship services to production, Repeatr is absolutely here to help!
However, if you want resource budgeting and advanced network topologies as part of that,
you'll need to make sure you're okay requiring your machines support the kind of container engines that make those features possible.


Supported Containers
--------------------

Currently supported container engines:

- chroot (linux)
  - isolation: bare minimum (filesystem isolated; almost everything else leaks)
  - convenience: extremely; almost universally available
  - performance: good (part of the "lightweight" container spectrum)
  - security: effectively zero
- runc (linux)
  - isolation: good (env vars, hostname, etc, prevented from leaking by default)
  - convenience: fairly easy to automatically deploy on most modern linuxes (but unfortunately not obvious what's going on when it breaks, which is currently the main reason it's not used by default.)
  - performance: good (part of the "lightweight" container spectrum)
  - security: arguably secure against some threats, but still not recommended to put your life on it
- nsinit (linux; legacy: runc is the up-to-date replacement for this.)

Planned:

- bhyve (or something for mac)
- VMs (kvm, etc.  can maintain same basic API contracts as "lightweight" containers; just slower and more secure.)

PRs and proposals for additional containment engines are welcomed!

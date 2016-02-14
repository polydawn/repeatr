package cradle

import (
	"github.com/spacemonkeygo/errors"
)

type Policy string

const (
	/*
		Operate with a low uid, as if you were a regular user on a
		regular system.  No special permissions will be granted
		(nor will they be available even if processes do manage to
		change uid, e.g. through suid binaries; most capabilities
		are dropped).

		The default uid to switch to is uid=1000,gid=1000.
		These are same as the default filters applied to file permissions,
		which should usually result in seamless "do the right thing".

		This is the safest mode to run as.  And, naturally, the default.
	*/
	Someguy = Policy("someguy")

	/*
		Operate with uid=0, but drop all interesting capabilities.
		This means things root would normally be able to do (like chown
		any file) will result in permission denied.

		Usually if you can use 'fakeroot', you can go all the way down to
		'someguy' mode (and you should, it's the default after all -- one
		less thing to configure!).  Sometimes however fakeroot can be useful
		for tricking programs that *think* they need to be uid=0, but really
		don't actually require very special priviledges during the work
		you need them to do (`apt` tools are frequently an example of this).
	*/
	Fakeroot = Policy("fakeroot")

	/*
		Operate with uid=0, with some of the most dangers capabilities
		(e.g. "muck with devices") dropped, but most of root's powers
		(like chown any file) still available.

		This may be slightly safer than enabling full 'sysad' mode,
		but you should still prefer to use any of the lower power levels
		if possible.

		This mode is the most similar to what you would experience with
		docker defaults.
	*/
	Dacroot = Policy("dacroot")

	/*
		Operate with uid=0 and *ALL CAPABILITIES*.

		This is absolutly not secure against untrusted code -- it is
		completely equivalent in power to root on your host.  Please
		try to use any of the lower power levels first.

		Among the things a system administrator may do is rebooting
		the machine and updating the kernel.  Seriously, *only* use
		with trusted code.
	*/
	Sysad = Policy("sysad")
)

func Caps(m Policy) []string {
	switch m {
	case Someguy:
		return []string{
			"CAP_AUDIT_WRITE",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
		}
	case Fakeroot:
		return []string{
			"CAP_AUDIT_WRITE",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
		}
	case Dacroot:
		return []string{
			"CAP_AUDIT_WRITE",
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_FSETID",
			"CAP_FOWNER",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETUID", // fairly terrifying
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_SYS_CHROOT",
		}
	case Sysad:
		return []string{
			"CAP_AUDIT_CONTROL",
			"CAP_AUDIT_READ",
			"CAP_AUDIT_WRITE",
			"CAP_BLOCK_SUSPEND",
			"CAP_CHOWN",
			"CAP_DAC_OVERRIDE",
			"CAP_DAC_READ_SEARCH",
			"CAP_FOWNER",
			"CAP_FSETID",
			"CAP_IPC_LOCK",
			"CAP_IPC_OWNER",
			"CAP_KILL",
			"CAP_LEASE",
			"CAP_LINUX_IMMUTABLE",
			"CAP_MAC_ADMIN",
			"CAP_MAC_OVERRIDE",
			"CAP_MKNOD",
			"CAP_NET_ADMIN",
			"CAP_NET_BIND_SERVICE",
			"CAP_NET_BROADCAST",
			"CAP_NET_RAW",
			"CAP_SETGID",
			"CAP_SETFCAP",
			"CAP_SETPCAP",
			"CAP_SETUID",
			"CAP_SYS_ADMIN",
			"CAP_SYS_BOOT",
			"CAP_SYS_CHROOT",
			"CAP_SYS_MODULE",
			"CAP_SYS_NICE",
			"CAP_SYS_PACCT",
			"CAP_SYS_PTRACE",
			"CAP_SYS_RAWIO",
			"CAP_SYS_RESOURCE",
			"CAP_SYS_TIME",
			"CAP_SYS_TTY_CONFIG",
			"CAP_SYSLOG",
			"CAP_WAKE_ALARM",
		}
	default:
		panic(errors.ProgrammerError.New("missing switch case"))
	}
}

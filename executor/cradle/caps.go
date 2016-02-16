package cradle

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/def"
)

/*
	Map of policies to the capabilities we give that policy by default.

	May not be implemented/enforced by all executors (e.g. chroot is
	simply not capable of these).
*/
func CapsForPolicy(m def.Policy) []string {
	switch m {
	case def.PolicyRoutine:
		return []string{
			"CAP_AUDIT_WRITE",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
		}
	case def.PolicyUidZero:
		return []string{
			"CAP_AUDIT_WRITE",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE",
		}
	case def.PolicyGovernor:
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
	case def.PolicySysad:
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

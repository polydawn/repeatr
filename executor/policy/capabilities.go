package policy

import (
	. "github.com/polydawn/go-errcat"
	"github.com/syndtr/gocapability/capability"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/rio"
)

func GetCapsForPolicy(policy api.Policy) ([]capability.Cap, error) {
	switch policy {
	case api.Policy_Routine:
		return []capability.Cap{
			capability.CAP_AUDIT_WRITE,
			capability.CAP_KILL,
			capability.CAP_NET_BIND_SERVICE,
		}, nil
	case api.Policy_Governor:
		return []capability.Cap{
			capability.CAP_AUDIT_WRITE,
			capability.CAP_CHOWN,
			capability.CAP_DAC_OVERRIDE,
			capability.CAP_FSETID,
			capability.CAP_FOWNER,
			capability.CAP_KILL,
			capability.CAP_NET_BIND_SERVICE,
			capability.CAP_NET_RAW,
			capability.CAP_SETGID,
			capability.CAP_SETUID,
			capability.CAP_SETFCAP,
			capability.CAP_SETPCAP,
			capability.CAP_SYS_CHROOT,
		}, nil
	case api.Policy_Sysad:
		return []capability.Cap{
			capability.CAP_AUDIT_CONTROL,
			capability.CAP_AUDIT_READ,
			capability.CAP_AUDIT_WRITE,
			capability.CAP_BLOCK_SUSPEND,
			capability.CAP_CHOWN,
			capability.CAP_DAC_OVERRIDE,
			capability.CAP_DAC_READ_SEARCH,
			capability.CAP_FOWNER,
			capability.CAP_FSETID,
			capability.CAP_IPC_LOCK,
			capability.CAP_IPC_OWNER,
			capability.CAP_KILL,
			capability.CAP_LEASE,
			capability.CAP_LINUX_IMMUTABLE,
			capability.CAP_MAC_ADMIN,
			capability.CAP_MAC_OVERRIDE,
			capability.CAP_MKNOD,
			capability.CAP_NET_ADMIN,
			capability.CAP_NET_BIND_SERVICE,
			capability.CAP_NET_BROADCAST,
			capability.CAP_NET_RAW,
			capability.CAP_SETGID,
			capability.CAP_SETFCAP,
			capability.CAP_SETPCAP,
			capability.CAP_SETUID,
			capability.CAP_SYS_ADMIN,
			capability.CAP_SYS_BOOT,
			capability.CAP_SYS_CHROOT,
			capability.CAP_SYS_MODULE,
			capability.CAP_SYS_NICE,
			capability.CAP_SYS_PACCT,
			capability.CAP_SYS_PTRACE,
			capability.CAP_SYS_RAWIO,
			capability.CAP_SYS_RESOURCE,
			capability.CAP_SYS_TIME,
			capability.CAP_SYS_TTY_CONFIG,
			capability.CAP_SYSLOG,
			capability.CAP_WAKE_ALARM,
		}, nil
	default:
		return nil, Errorf(rio.ErrUsage, "invalid policy %q", policy)
	}
}

package runc

var confConfigTmpl = `{
	"version": "0.1.0",
	"platform": {
		"os": "linux",
		"arch": "amd64"
	},
	"process": {
		"terminal": true,
		"user": {
			"uid": 0,
			"gid": 0,
			"additionalGids": null
		},
		"args": [
			"sh"
		],
		"env": [
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"TERM=xterm"
		],
		"cwd": ""
	},
	"root": {
		"path": "rootfs",
		"readonly": true
	},
	"hostname": "shell",
	"mounts": [
		{
			"name": "proc",
			"path": "/proc"
		},
		{
			"name": "dev",
			"path": "/dev"
		},
		{
			"name": "devpts",
			"path": "/dev/pts"
		},
		{
			"name": "shm",
			"path": "/dev/shm"
		},
		{
			"name": "mqueue",
			"path": "/dev/mqueue"
		},
		{
			"name": "sysfs",
			"path": "/sys"
		},
		{
			"name": "cgroup",
			"path": "/sys/fs/cgroup"
		}
	],
	"linux": {
		"capabilities": [
			"CAP_AUDIT_WRITE",
			"CAP_KILL",
			"CAP_NET_BIND_SERVICE"
		]
	}
}`

var confRuntimeTmpl = `{
	"mounts": {
		"cgroup": {
			"type": "cgroup",
			"source": "cgroup",
			"options": [
				"nosuid",
				"noexec",
				"nodev",
				"relatime",
				"ro"
			]
		},
		"dev": {
			"type": "tmpfs",
			"source": "tmpfs",
			"options": [
				"nosuid",
				"strictatime",
				"mode=755",
				"size=65536k"
			]
		},
		"devpts": {
			"type": "devpts",
			"source": "devpts",
			"options": [
				"nosuid",
				"noexec",
				"newinstance",1
				"ptmxmode=0666",
				"mode=0620",
				"gid=5"
			]
		},
		"mqueue": {
			"type": "mqueue",
			"source": "mqueue",
			"options": [
				"nosuid",
				"noexec",
				"nodev"
			]
		},
		"proc": {
			"type": "proc",
			"source": "proc",
			"options": null
		},
		"shm": {
			"type": "tmpfs",
			"source": "shm",
			"options": [
				"nosuid",
				"noexec",
				"nodev",
				"mode=1777",
				"size=65536k"
			]
		},
		"sysfs": {
			"type": "sysfs",
			"source": "sysfs",
			"options": [
				"nosuid",
				"noexec",
				"nodev"
			]
		}
	},
	"hooks": {
		"prestart": null,
		"poststop": null
	},
	"linux": {
		"uidMappings": null,
		"gidMappings": null,
		"rlimits": [
			{
				"type": "RLIMIT_NOFILE",
				"hard": 1024,
				"soft": 1024
			}
		],
		"sysctl": null,
		"resources": {
			"disableOOMKiller": false,
			"memory": {
				"limit": 0,
				"reservation": 0,
				"swap": 0,
				"kernel": 0,
				"swappiness": -1
			},
			"cpu": {
				"shares": 0,
				"quota": 0,
				"period": 0,
				"realtimeRuntime": 0,
				"realtimePeriod": 0,
				"cpus": "",
				"mems": ""
			},
			"pids": {
				"limit": 0
			},
			"blockIO": {
				"blkioWeight": 0,
				"blkioLeafWeight": 0,
				"blkioWeightDevice": null,
				"blkioThrottleReadBpsDevice": null,
				"blkioThrottleWriteBpsDevice": null,
				"blkioThrottleReadIOPSDevice": null,
				"blkioThrottleWriteIOPSDevice": null
			},
			"hugepageLimits": null,
			"network": {
				"classId": "",
				"priorities": null
			}
		},
		"cgroupsPath": "",
		"namespaces": [
			{
				"type": "pid",
				"path": ""
			},
			{
				"type": "network",
				"path": ""
			},
			{
				"type": "ipc",
				"path": ""
			},
			{
				"type": "uts",
				"path": ""
			},
			{
				"type": "mount",
				"path": ""
			}
		],
		"devices": [
			{
				"path": "/dev/null",
				"type": 99,
				"major": 1,
				"minor": 3,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			},
			{
				"path": "/dev/random",
				"type": 99,
				"major": 1,
				"minor": 8,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			},
			{
				"path": "/dev/full",
				"type": 99,
				"major": 1,
				"minor": 7,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			},
			{
				"path": "/dev/tty",
				"type": 99,
				"major": 5,
				"minor": 0,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			},
			{
				"path": "/dev/zero",
				"type": 99,
				"major": 1,
				"minor": 5,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			},
			{
				"path": "/dev/urandom",
				"type": 99,
				"major": 1,
				"minor": 9,
				"permissions": "rwm",
				"fileMode": 438,
				"uid": 0,
				"gid": 0
			}
		],
		"apparmorProfile": "",
		"selinuxProcessLabel": "",
		"seccomp": {
			"defaultAction": "SCMP_ACT_ALLOW",
			"syscalls": []
		},
		"rootfsPropagation": ""
	}
}`

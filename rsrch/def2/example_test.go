package def

var frm = &Formula{
	Inputs: Inputs{
		"hostname":   {Hostname: "custom-host-name"},
		"gopath env": {EnvValue: EnvValue{"GOPATH", "/home/luser/go"}},
		"arch":       {EnvValue: EnvValue{"GOARCH", "amd64"}},
		"rootfs":     {Filesystem: Filesystem{"/", ParseWare("tar:9g0joixj9js2")}},
	},
	Action: Action{
		Script: []string{
			"cd /whee",
			"echo $ARCH > arch",
			"export HEH=whynot",
		},
	},
	SaveSlots: SaveSlots{
		"avar": {EnvValue: "HEH"},
		"whee": {Filesystem: FilesystemSlot{"/whee/", "tar"}},
	},
	Warehousing: &Warehousing{
	// - prev outputs have an opinion
	//   - but probably not a useful one, big-picture:
	//     - the local fs path is great for your localhost work
	//     - but generally there's mirroring afterward for published artifacts
	//     - so the local fs path is the exception, not the rule (and you can
	//       fill that in from the 'workspace' ceoncept in some other layer like reppl).
	// - intuitively: we ought to decouple these from the input names,
	//   because those are per formula,
	//   and the directory of warehouses we can use/auth to is per workspace.
	//   - furthermore, which warehouse you can fulfill a hash from is moot
	//     - sometimes it matters which order we check them in, e.g. git,
	//       but this is the exception, not the rule.
	// - so the interesting insight here is mostly that...
	//   - we should be narrowing things *down* in the list here, mostly;
	//     sometimes there is no default (ahem, git), but the normal use case
	//     is that checking the workspace's configured set of public warehouses
	//     is all fine (we're not very secret about what wares we want);
	//   - save slots *don't need* a warehouse config by default;
	//     maybe you want to configure an s3 bucket to be your working storage
	//     for a workspace, but the default is $workspaceDir/wares/ and that's fine;
	//     anything further is a question for your mirroring config and comes up
	//     at publish time, which is a full step separated from eval'ing something.
	// - to hammer in on that a bit more:
	//   - asking a CAS warehouse if it has a ware is dirt cheap.
	//   - the only reason *not* to ask a warehouse about something is if it's
	//     - A) a badly behaved warehouse and there are performance implications
	//       - git
	//       - arguably, non-cas http (but we can handle that by hiding them
	//         behind a 'directory' abstraction that makes them behave better).
	//     - B) the subject is one you don't want to admit outside certain circles
	//       - which happens: if the ware is a corporate IP artifact, it's not
	//         exactly a "breach" per se if I accidentally ask publichub.comnet
	//         if they have seen that hash, it's still not desirable.
	//         Like, I don't want to explain to my CISO why publichub.comnet's
	//         traffic logs can chronical how frequently our CI server built
	//         the top-secret-self-driving-cars-lol project.  publichub.comnet
	//         wouldn't *know* that's what they're seeing requests for, but still.
	},
}

var later = &Formula{
	Results: &Results{
		RunErr:   nil,
		ExitCode: 0,
		Saved: map[string]interface{}{
			"avar": EnvValue{Value: "whynot"},
			"whee": Ware{"tar", "2tu4gjjr23vkmsk4z9js2kl3zkln"},
		},
	},
}

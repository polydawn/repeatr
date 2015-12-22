/*
	Data warehousing backed by Google Cloud Storage.

	This transport uses tars as the metaphor for mapping filesystems onto gs.
	Using tarballs gives us a place to preserve file metadata in a widely-understood way.
	(Note that this is not the only possible approach)

	Silo URIs must parse as URIs and have one of two schemes: "gs://" or "gs+ca://".

	The "gs+ca://" scheme will store data in content-addressable names
	where the gs bucket is the host component of the URI, and the path component
	of the URI is used as a prefix to the full data path.
	In this configuration, the transport is compliant with the usual expectation
	that the same location string can be used for as a prefix for as much data
	as you want, and it will Do The Right Thing, and also automatically deduplicate.

	The "gs://" scheme store data at URI location literally and without
	content-addressible properties; be advised that in this configuration, the
	transport will *NOT* be compliant with the usual expectation that more
	than one piece of data may be stored at the same silo URI.

	Login secrets configuration is handled through environment variables:
	'GS_ACCESS_TOKEN' or 'GS_SERVICE_ACCOUNT_FILE'

	Setup and management of Google Cloud Storage permissions models are *not* handled by this system --
	Those are arcana that should be managed by your organization, not repeatr.

	This IO system happens to share the same hash-space as the Tar IO system,
	and may thus safely share a cache with Tar IO systems.

	More information on Google Cloud Storage can be found at https://cloud.google.com/storage/docs
*/
package gs

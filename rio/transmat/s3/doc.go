/*
	Data warehousing backed by Amazon S3.

	This transport uses tars as the metaphor for mapping filesystems onto S3.
	Amazon S3 is a key-value store at heart, and (despite what some of the web
	UI may seem to suggest) requires more bits for file attributes, etc to manifest as a filesystem.
	Using tarballs gives us a place to preserve file metadata in a widely-understood way.
	(Note that this is not the only possible approach; other transports may
	use S3 as a data warehouse, but map it to filesystems differently.)

	Silo URIs must parse as URIs and have one of two schemes: "s3://" or "s3+ca://".
	("s3+splay://" is the same as "s3+ca://" and supported for backwards-compatibility,
	but is deprecated and may be removed without notice.)

	The "s3+ca://" scheme will store data in content-addressable names
	where the S3 bucket is the host component of the URI, and the path component
	of the URI is used as a prefix to the full data path.
	In this configuration, the transport is compliant with the usual expectation
	that the same location string can be used for as a prefix for as much data
	as you want, and it will Do The Right Thing, and also automatically deduplicate.

	The "s3://" scheme store data at URI location literally and without
	content-addressible properties; be advised that in this configuration, the
	transport will *NOT* be compliant with the usual expectation that more
	than one piece of data may be stored at the same silo URI.

	Login secrets configuration is handled through environment variables:
	`AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY`.

	Setup and management of S3/AWS permissions models are *not* handled by this system --
	Those are arcana that should be managed by your organization, not repeatr.

	Note that S3/AWS permissions are complicated; and be advised that in many configurations,
	accounts may be able to write data which they then instantly lose permission to
	manage or even read back; this will result in permission denied errors from this
	system, and may result in dirty state left behind (because we lack permission
	to clean it up, and cannot discover this until it's too late!).
	PRs for improving this behavior deeply welcome; this author is not an S3/AWS/IAM professional.

	If you're just getting started with S3/AWS permissions, try the docs here:
	https://docs.aws.amazon.com/AmazonS3/latest/dev/example-bucket-policies.html
	These describe attaching policies to a bucket that can grant permissions to a user account.

	This IO system happens to share the same hash-space as the Tar IO system,
	and may thus safely share a cache with Tar IO systems.
*/
package s3

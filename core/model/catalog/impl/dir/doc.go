/*
	A directory catalog is a simple glue piece that allows you to construct
	prototypes for work based on automatic scans of the current contents of
	a directory.

	Dircat only emits one SKU: whatever the current scan of the directory is.
	It does not retain any historical info, since directories don't retain
	any historical forms of their contents either.

	Dircat is useful for prototyping, or possibly for integrating with
	other systems that are manipulating files, but be wary of using it
	heavily or relying on it: because plain directories don't store
	versions of their contents over time, you're likely to end up with
	formulas refering to input data you *don't have* anymore -- caveat emptor.
*/
package dircat

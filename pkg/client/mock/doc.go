package mock

// We use the mockgen binary which is installed via make mockgen.
// This helps us to get consistent builds and also make the release process work
// without chaning PATH variables etc.

//go:generate ../../../bin/mockgen -package mock -destination=enterprise.go -source=../enterprise.go Enterprise
//go:generate ../../../bin/mockgen -package mock -destination=organization.go -source=../organization.go Organization
//go:generate ../../../bin/mockgen -package mock -destination=pool.go -source=../pool.go Pool

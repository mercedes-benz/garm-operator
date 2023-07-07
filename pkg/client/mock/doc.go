// SPDX-License-Identifier: MIT

package mock

// We use the mockgen binary which is installed via make mockgen.
// This helps us to get consistent builds and also make the release process work
// without chaning PATH variables etc.

//go:generate ../../../bin/mockgen -package mock -destination=enterprise.go -source=../enterprise.go Enterprise
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt enterprise.go > _enterprise.go && mv _enterprise.go enterprise.go"
//go:generate ../../../bin/mockgen -package mock -destination=organization.go -source=../organization.go Organization
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt organization.go > _organization.go && mv _organization.go organization.go"
//go:generate ../../../bin/mockgen -package mock -destination=pool.go -source=../pool.go Pool
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt pool.go > _pool.go && mv _pool.go pool.go"
//go:generate ../../../bin/mockgen -package mock -destination=instance.go -source=../instance.go Instance
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt instance.go > _instance.go && mv _instance.go instance.go"
//go:generate ../../../bin/mockgen -package mock -destination=repository.go -source=../repository.go Repository
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt repository.go > _repository.go && mv _repository.go repository.go"

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
//go:generate ../../../bin/mockgen -package mock -destination=client.go -source=../client.go GarmClient
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt client.go > _client.go && mv _client.go client.go"
//go:generate ../../../bin/mockgen -package mock -destination=endpoint.go -source=../endpoint.go Endpoint
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt endpoint.go > _endpoint.go && mv _endpoint.go endpoint.go"
//go:generate ../../../bin/mockgen -package mock -destination=credentials.go -source=../credentials.go GithubCredentials
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt credentials.go > _credentials.go && mv _credentials.go credentials.go"
//go:generate ../../../bin/mockgen -package mock -destination=controller.go -source=../controller.go GarmServerConfig
//go:generate /usr/bin/env bash -c "cat ../../../hack/boilerplate.go.txt controller.go > _controller.go && mv _controller.go controller.go"

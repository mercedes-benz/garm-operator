package mock

//go:generate mockgen -package mock -destination=enterprise.go -source=../enterprise.go Enterprise
//go:generate mockgen -package mock -destination=organization.go -source=../organization.go Organization

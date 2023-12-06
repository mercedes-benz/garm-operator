package main

import "github.com/mercedes-benz/garm-operator/pkg/command"

func main() {
	if err := command.RootCommand.Execute(); err != nil {
		panic(err)
	}
}

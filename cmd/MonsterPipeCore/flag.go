package main

import "flag"

var (
	managerListenAddr string = *flag.String("managerListenAddr", "", "manager listen address")
)

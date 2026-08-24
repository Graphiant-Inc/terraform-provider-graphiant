// Package main starts the Graphiant Terraform provider server.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider"
)

//go:generate api/generate.sh
//go:generate go tool tfplugindocs generate

// version is set via -ldflags "-X main.version=..." during release builds (see .goreleaser.yml).
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/Graphiant-Inc/graphiant",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}

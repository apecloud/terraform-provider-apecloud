package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/apecloud/terraform-provider-kbcloud/internal/provider"
)

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/apecloud/kbcloud",
		Debug:   debugMode,
	})
	if err != nil {
		log.Fatal(err)
	}
}

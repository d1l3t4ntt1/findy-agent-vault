package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/findy-network/findy-agent-vault/resolver"
	"github.com/findy-network/findy-agent-vault/utils"

	"github.com/golang/glog"

	"github.com/findy-network/findy-agent-vault/server"
)

func main() {
	utils.SetLogDefaults()
	config := utils.LoadConfig()

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Vault version %s\n", config.Version)
		return
	}

	gqlResolver := resolver.InitResolver(config)

	srv := server.NewServer(gqlResolver, config.JWTKey)
	http.Handle("/query", srv.Handle())
	if config.UsePlayground {
		http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if utils.LogTrace() {
			glog.Infof("health check %s %s", r.URL.Path, config.Version)
		}
		_, _ = w.Write([]byte(config.Version))
	})

	glog.Fatal(http.ListenAndServe(config.Address, nil))
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/lithammer/dedent"
	"github.com/marxus/k8s-mca/pkg/inject"
	"github.com/marxus/k8s-mca/pkg/serve"
)

func main() {
	var (
		injectFlag = flag.Bool("inject", false, "Inject MCA sidecar into Pod manifest")
		serveFlag  = flag.Bool("serve", false, "Start MCA proxy server")
	)
	flag.Parse()

	switch {
	case *injectFlag:
		if err := runInject(); err != nil {
			log.Fatalf("Injection failed: %v", err)
		}
	case *serveFlag:
		if err := runServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	default:
		fmt.Fprint(os.Stderr, dedent.Dedent(fmt.Sprintf(`
			Usage: %s [--inject|--serve]
			  --inject  Inject MCA sidecar into Pod manifest (stdin/stdout)
			  --serve   Start MCA proxy server
		`, os.Args[0])))
		os.Exit(1)
	}
}

func runInject() error {
	input, err := os.ReadFile("/dev/stdin")
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	output, err := inject.InjectMCA(input)
	if err != nil {
		return fmt.Errorf("failed to inject MCA: %w", err)
	}

	if _, err := os.Stdout.Write(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func runServe() error {
	return serve.Start()
}

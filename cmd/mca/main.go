package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/marxus/k8s-mca/pkg/inject"
	"github.com/marxus/k8s-mca/pkg/serve"
)

var cliUsage = `
Usage: %s [--inject|--proxy|--webhook]
  --inject   Inject MCA sidecar into Pod manifest (stdin/stdout)
  --proxy    Start MCA proxy server
  --webhook  Start MCA webhook server
`

func main() {
	var (
		injectFlag  = flag.Bool("inject", false, "Inject MCA sidecar into Pod manifest")
		proxyFlag   = flag.Bool("proxy", false, "Start MCA proxy server")
		webhookFlag = flag.Bool("webhook", false, "Start MCA webhook server")
	)
	flag.Parse()

	switch {
	case *injectFlag:
		if err := runInject(); err != nil {
			log.Fatalf("Injection failed: %v", err)
		}
	case *proxyFlag:
		if err := runProxy(); err != nil {
			log.Fatalf("Proxy server failed: %v", err)
		}
	case *webhookFlag:
		if err := runWebhook(); err != nil {
			log.Fatalf("Webhook server failed: %v", err)
		}
	default:
		fmt.Fprint(os.Stderr, fmt.Sprintf(cliUsage, os.Args[0]))
		os.Exit(1)
	}
}

func runInject() error {
	input, err := os.ReadFile("/dev/stdin")
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	output, err := inject.InjectViaCLI(input)
	if err != nil {
		return fmt.Errorf("failed to inject MCA: %w", err)
	}

	if _, err := os.Stdout.Write(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func runProxy() error {
	return serve.StartProxy()
}

func runWebhook() error {
	return serve.StartWebhook()
}

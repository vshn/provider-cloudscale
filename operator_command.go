package main

import (
	"context"
	"crypto/tls"
	"time"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/go-logr/logr"
	"github.com/vshn/provider-cloudscale/apis"
	"github.com/vshn/provider-cloudscale/operator"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/urfave/cli/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type operatorCommand struct {
	LeaderElectionEnabled bool
	WebhookCertDir        string

	manager    manager.Manager
	kubeconfig *rest.Config
}

func newOperatorCommand() *cli.Command {
	command := &operatorCommand{}
	return &cli.Command{
		Name:   "operator",
		Usage:  "Start provider in operator mode",
		Action: command.execute,
		Flags: []cli.Flag{
			newLeaderElectionEnabledFlag(&command.LeaderElectionEnabled),
			newWebhookTLSCertDirFlag(&command.WebhookCertDir),
		},
	}
}

func (c *operatorCommand) execute(ctx *cli.Context) error {
	_ = LogMetadata(ctx)
	log := logr.FromContextOrDiscard(ctx.Context).WithName(ctx.Command.Name)
	log.Info("Setting up controllers", "config", c)
	ctrl.SetLogger(log)

	p := pipeline.NewPipeline[context.Context]()
	p.WithBeforeHooks(
		func(step pipeline.Step[context.Context]) {
			log.V(1).Info(step.Name)
		},
	)
	p.AddStepFromFunc("get config", func(ctx context.Context) error {
		cfg, err := ctrl.GetConfig()
		c.kubeconfig = cfg
		return err
	})
	p.AddStepFromFunc("create manager", func(ctx context.Context) error {
		// configure client-side throttling
		c.kubeconfig.QPS = 100
		c.kubeconfig.Burst = 150 // more Openshift friendly

		webhookOpts := webhook.Options{
			Port: 9443,
			TLSOpts: []func(*tls.Config){
				func(tlsConfig *tls.Config) {
					tlsConfig.MinVersion = tls.VersionTLS13
				},
			},
			CertDir: c.WebhookCertDir,
		}

		mgr, err := ctrl.NewManager(c.kubeconfig, ctrl.Options{
			// controller-runtime uses both ConfigMaps and Leases for leader election by default.
			// Leases expire after 15 seconds, with a 10-second renewal deadline.
			// We've observed leader loss due to renewal deadlines being exceeded when under high load - i.e.
			//  hundreds of reconciles per second and ~200rps to the API server.
			// Switching to Leases only and longer leases appears to alleviate this.
			LeaderElection:             c.LeaderElectionEnabled,
			LeaderElectionID:           "leader-election-provider-cloudscale",
			LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
			LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
			RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
			WebhookServer:              webhook.NewServer(webhookOpts),
		})
		c.manager = mgr
		return err
	})
	p.AddStep(p.WithNestedSteps("register schemes", nil,
		p.NewStep("register API schemes", func(ctx context.Context) error {
			return apis.AddToScheme(c.manager.GetScheme())
		}),
	))
	p.AddStepFromFunc("setup controllers", func(ctx context.Context) error {
		return operator.SetupControllers(c.manager)
	})
	p.AddStepFromFunc("setup webhooks", func(ctx context.Context) error {
		if c.WebhookCertDir != "" {
			return operator.SetupWebhooks(c.manager)
		}
		return nil
	})
	p.AddStepFromFunc("run manager", func(ctx context.Context) error {
		log.Info("Starting manager")
		return c.manager.Start(ctx)
	})
	return p.RunWithContext(ctx.Context)
}

package main

import (
	"context"
	"fmt"
	"os"

	rolloutclientset "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-rollouts/pkg/kubectl-argo-rollouts/options"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-rollouts/server"
	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	// CLIName is the name of the CLI
	cliName = "argo-rollouts-server"
)

func AddKubectlFlagsToCmd(cmd *cobra.Command) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func NewCommand(o *options.ArgoRolloutsOptions) *cobra.Command {
	var (
		listenPort   int
		clientConfig clientcmd.ClientConfig
	)

	var command = &cobra.Command{
		Use:   cliName,
		Short: "argo-rollouts-server is an API server that provides UI assets and Rollout data",
		Run: func(c *cobra.Command, args []string) {
			config, err := clientConfig.ClientConfig()
			errors.CheckError(err)

			namespace, _, err := clientConfig.Namespace()
			errors.CheckError(err)

			kubeclientset := kubernetes.NewForConfigOrDie(config)

			rolloutclientsetConfig, err := clientConfig.ClientConfig()
			errors.CheckError(err)

			rolloutclientset := rolloutclientset.NewForConfigOrDie(rolloutclientsetConfig)

			opts := server.ServerOptions{
				Namespace:         namespace,
				KubeClientset:     kubeclientset,
				RolloutsClientset: rolloutclientset,
				DynamicClientset:  o.DynamicClientset(),
			}
			for {
				ctx := context.Background()
				ctx, cancel := context.WithCancel(ctx)
				argorollouts := server.NewServer(opts)
				argorollouts.Run(ctx, listenPort, false)
				cancel()
			}
		},
	}

	clientConfig = AddKubectlFlagsToCmd(command)
	command.Flags().IntVar(&listenPort, "port", 3100, "Listen on given port")
	return command
}

func main() {
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	o := options.NewArgoRolloutsOptions(streams)

	if err := NewCommand(o).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
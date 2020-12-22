package main

import (
	"context"
	"fmt"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/skupperproject/skupper/api/types"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func NewCmdLink() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link create <input-token-file> [--name <name>] or link delete ...",
		Short: "Manage skupper links definitions",
	}
	return cmd
}

var connectorCreateOpts types.ConnectorCreateOptions

func NewCmdLinkCreate(newClient cobraFunc, flag string) *cobra.Command {

	if flag == "" { //hack for backwards compatibility
		flag = "name"
	}

	cmd := &cobra.Command{
		Use:    "create <input-token-file>",
		Short:  "Links this skupper installation to that which issued the specified connectionToken",
		Args:   cobra.ExactArgs(1),
		PreRun: newClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceCobra(cmd)
			siteConfig, err := cli.SiteConfigInspect(context.Background(), nil)
			if err != nil {
				return fmt.Errorf("Unable to retrieve site config: %w", err.Error())
			}
			header := ""
			var secret *corev1.Secret
			if siteConfig == nil || !siteConfig.Spec.SiteControlled {
				connectorCreateOpts.SkupperNamespace = cli.GetNamespace()
				header = "Skupper"
				secret, err = cli.ConnectorCreateFromFile(context.Background(), args[0], connectorCreateOpts)
			} else {
				// create the secret, site-controller will do the rest
				header = "Skupper site-controller"
				secret, err = cli.ConnectorCreateSecretFromFile(context.Background(), args[0], connectorCreateOpts)
			}

			if err != nil {
				return fmt.Errorf("Failed to create connection: %w", err)
			}

			if siteConfig.Spec.IsEdge {
				fmt.Printf("%s site-controller configured to connect to %s:%s (name=%s)\n",
					header,
					secret.ObjectMeta.Annotations["edge-host"],
					secret.ObjectMeta.Annotations["edge-port"],
					secret.ObjectMeta.Name)
			} else {
				fmt.Printf("%s site-controller configured to connect to %s:%s (name=%s)\n",
					header,
					secret.ObjectMeta.Annotations["inter-router-host"],
					secret.ObjectMeta.Annotations["inter-router-port"],
					secret.ObjectMeta.Name)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&connectorCreateOpts.Name, flag, "", "", "Provide a specific name for the connection (used when removing it with disconnect)")
	cmd.Flags().Int32VarP(&connectorCreateOpts.Cost, "cost", "", 1, "Specify a cost for this connection.")

	return cmd
}

var connectorRemoveOpts types.ConnectorRemoveOptions

func NewCmdLinkDelete(newClient cobraFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "delete <name>",
		Short:  "Remove specified link",
		Args:   cobra.ExactArgs(1),
		PreRun: newClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceCobra(cmd)
			connectorRemoveOpts.Name = args[0]
			connectorRemoveOpts.SkupperNamespace = cli.GetNamespace()
			connectorRemoveOpts.ForceCurrent = false
			err := cli.ConnectorRemove(context.Background(), connectorRemoveOpts)
			if err == nil {
				fmt.Println("Link '" + args[0] + "' has been removed")
			} else {
				return fmt.Errorf("Failed to remove link: %w", err)
			}
			return nil
		},
	}

	return cmd
}

var waitFor int

func NewCmdLinkStatus(newClient cobraFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "status all|<connection-name>",
		Short:  "Check whether a link to another Skupper site is active",
		Args:   cobra.ExactArgs(1),
		PreRun: newClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			silenceCobra(cmd)

			var connectors []*types.ConnectorInspectResponse
			connected := 0

			if args[0] == "all" {
				vcis, err := cli.ConnectorList(context.Background())
				if err == nil {
					for _, vci := range vcis {
						connectors = append(connectors, &types.ConnectorInspectResponse{
							Connector: vci,
							Connected: false,
						})
					}
				}
			} else {
				vci, err := cli.ConnectorInspect(context.Background(), args[0])
				if err == nil {
					connectors = append(connectors, vci)
					if vci.Connected {
						connected++
					}
				}
			}

			for i := 0; connected < len(connectors) && i < waitFor; i++ {
				for _, c := range connectors {
					vci, err := cli.ConnectorInspect(context.Background(), c.Connector.Name)
					if err == nil && vci.Connected && c.Connected == false {
						c.Connected = true
						connected++
					}
				}
				time.Sleep(time.Second)
			}

			if len(connectors) == 0 {
				fmt.Println("There are no connectors configured or active")
			} else {
				for _, c := range connectors {
					if c.Connected {
						fmt.Printf("Connection for %s is active", c.Connector.Name)
						fmt.Println()
					} else {
						fmt.Printf("Connection for %s not active", c.Connector.Name)
						fmt.Println()
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&waitFor, "wait", 1, "The number of seconds to wait for connections to become active")

	return cmd
}

package inet256cmd

import (
	"fmt"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newStatusCmd(newClient func() (mesh256.Service, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "prints status of the main node",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var localAddr inet256.Addr
			var transportAddrs []p2p.Addr
			var peerStatuses []mesh256.PeerStatus
			eg := errgroup.Group{}
			eg.Go(func() error {
				var err error
				localAddr, err = c.MainAddr()
				return err
			})
			eg.Go(func() error {
				var err error
				transportAddrs, err = c.TransportAddrs()
				return err
			})
			eg.Go(func() error {
				var err error
				peerStatuses, err = c.PeerStatus()
				return err
			})
			if err := eg.Wait(); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "LOCAL ADDR: %v\n", localAddr)
			fmt.Fprintf(w, "TRANSPORTS:\n")
			for _, addr := range transportAddrs {
				fmt.Fprintf(w, "\t%s\n", addr)
			}
			fmt.Fprintf(w, "PEERS:\n")
			for _, status := range peerStatuses {
				fmt.Fprintf(w, "\t%s\n", status.Addr)
				for addr, lastSeen := range status.LastSeen {
					fmt.Fprintf(w, "\t\t%s\t%v\n", addr, lastSeen)
				}
			}
			return nil
		},
	}
}

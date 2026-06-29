package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/nitrocao/aliyun-workbench-cli/internal/workbench"
)

const (
	appVersion     = "v0.1.0"
	envLoginTicket = "LOGIN_ALIYUNID_TICKET"
)

type app struct {
	debug  bool
	stdout io.Writer
}

// Execute runs the aliyun-workbench command tree.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := &app{
		stdout: os.Stdout,
	}
	return a.newRootCommand().ExecuteContext(ctx)
}

func (a *app) newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "aliyun-workbench",
		Short:         "Connect to Alibaba Cloud ECS Workbench from a local terminal",
		Version:       appVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			configureLogger(a.debug)
		},
	}
	cmd.PersistentFlags().BoolVar(&a.debug, "debug", false, "enable debug logs")
	cmd.AddCommand(a.newListCommand(), a.newLoginCommand())
	return cmd
}

func (a *app) newListCommand() *cobra.Command {
	var opts listOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ECS instances that can be opened in Workbench",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runList(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.region, "region", "", "ECS region ID, for example cn-beijing")
	cmd.Flags().IntVar(&opts.pageSize, "page-size", 100, "page size")
	_ = cmd.MarkFlagRequired("region")
	return cmd
}

func (a *app) newLoginCommand() *cobra.Command {
	var opts loginOptions
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Open an interactive SSH terminal through ECS Workbench",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runLogin(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.region, "region", "", "ECS region ID, for example cn-beijing")
	cmd.Flags().StringVar(&opts.instanceID, "instance-id", "", "ECS instance ID")
	cmd.Flags().StringVar(&opts.username, "username", "root", "remote OS username")
	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("instance-id")
	return cmd
}

type listOptions struct {
	region   string
	pageSize int
}

func (a *app) runList(ctx context.Context, opts listOptions) error {
	client, err := workbench.NewClient(loginTicket())
	if err != nil {
		return err
	}
	instances, err := client.ListECSInstances(ctx, opts.region, 1, opts.pageSize)
	if err != nil {
		return err
	}

	return printInstancesTable(a.stdout, instances)
}

func printInstancesTable(w io.Writer, instances []workbench.ECSInstance) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "INSTANCE_ID\tNAME\tSTATUS\tPRIVATE_IP\tPUBLIC_IP\tOS"); err != nil {
		return err
	}
	for _, inst := range instances {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			inst.InstanceID,
			inst.InstanceName,
			inst.Status,
			inst.PrivateIPAddress(),
			inst.PublicIPAddress(),
			inst.OSName,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

type loginOptions struct {
	region     string
	instanceID string
	username   string
}

func (a *app) runLogin(ctx context.Context, opts loginOptions) error {
	client, err := workbench.NewClient(loginTicket())
	if err != nil {
		return err
	}

	logrus.Info("loading instance resource")
	resources, err := client.ResourceList(ctx, opts.region, opts.instanceID)
	if err != nil {
		return err
	}
	if len(resources) == 0 {
		return fmt.Errorf("instance %s not found in workbench resources", opts.instanceID)
	}
	resource := resources[0]

	logrus.Info("acquiring request token")
	requestToken, err := client.AcquireRequestToken(ctx, opts.region, opts.instanceID)
	if err != nil {
		return err
	}

	logrus.Info("creating workbench login token")
	login, err := client.LoginInstance(ctx, opts.region, opts.instanceID, opts.username, resource, requestToken)
	if err != nil {
		return err
	}

	logrus.Info("checking workbench login token")
	if err := client.CheckInstanceLogin(ctx, opts.region, opts.instanceID, login.Info.InstanceLoginToken); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"host":     login.Info.Host,
		"username": login.Info.Username,
	}).Info("opening terminal")
	return workbench.RunInteractiveSession(ctx, loginTicket(), opts.region, opts.instanceID, login)
}

func loginTicket() string {
	return strings.TrimSpace(os.Getenv(envLoginTicket))
}

func configureLogger(debug bool) {
	logrus.SetOutput(os.Stderr)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		return
	}
	logrus.SetLevel(logrus.InfoLevel)
}

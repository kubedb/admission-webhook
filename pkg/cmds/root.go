package cmds

import (
	"flag"
	"log"
	"os"
	"strings"

	v "github.com/appscode/go/version"
	"github.com/appscode/kutil/tools/analytics"
	"github.com/jpillora/go-ogle-analytics"
	"github.com/kubedb/apimachinery/client/clientset/versioned/scheme"
	"github.com/kubedb/apimachinery/pkg/admission/dormantdatabase"
	"github.com/kubedb/apimachinery/pkg/admission/snapshot"
	"github.com/kubedb/kubedb-server/pkg/admission/plugin/elasticsearch"
	"github.com/kubedb/kubedb-server/pkg/admission/plugin/memcached"
	"github.com/kubedb/kubedb-server/pkg/admission/plugin/mysql"
	"github.com/kubedb/kubedb-server/pkg/admission/plugin/postgres"
	"github.com/kubedb/kubedb-server/pkg/admission/plugin/redis"
	"github.com/kubedb/kubedb-server/pkg/cmds/server"
	mgAdmsn "github.com/kubedb/mongodb/pkg/admission"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	gaTrackingCode = "UA-62096468-20"
)

var (
	analyticsClientID = analytics.ClientID()
	enableAnalytics   = true
)

func NewRootCmd(version string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               "server",
		DisableAutoGenTag: true,
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if enableAnalytics && gaTrackingCode != "" {
				if client, err := ga.NewClient(gaTrackingCode); err == nil {
					client.ClientID(analyticsClientID)
					parts := strings.Split(c.CommandPath(), " ")
					client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
				}
			}
			scheme.AddToScheme(clientsetscheme.Scheme)
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// ref: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Analytics")

	rootCmd.AddCommand(v.NewCmdVersion())

	stopCh := genericapiserver.SetupSignalHandler()
	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopCh,
		&elasticsearch.ElasticsearchValidator{},
		&memcached.MemcachedValidator{},
		&mgAdmsn.MongoDBValidator{},
		&mgAdmsn.MongoDBMutator{},
		&mysql.MySQLValidator{},
		&postgres.PostgresValidator{},
		&redis.RedisValidator{},
		&snapshot.SnapshotValidator{},
		&dormantdatabase.DormantDatabaseValidator{},
	)
	cmd.Use = "run"
	cmd.Long = "Launch KubeDB server"
	cmd.Short = cmd.Long
	rootCmd.AddCommand(cmd)

	return rootCmd
}

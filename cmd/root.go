package cmd

import (
	"net/http"
	"time"

	ts3 "github.com/multiplay/go-ts3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rebuy-de/rebuy-go-sdk/cmdutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type App struct {
	ListenAddress string
	ListenPort    string
	Address       string
	Username      string
	Password      string
}

var (
	clientsConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "clients",
		Help:      "Number of connected clients",
	})
	uptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "uptime",
		Help:      "Uptime in seconds",
	})
	maxClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "max_number_of_clients",
		Help:      "Maximum number of clients the server is able to handle",
	})
	bytesSentTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "bytes_sent_total",
		Help:      "Total number of bytes sent",
	})
	bytesReceivedTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "bytes_received_total",
		Help:      "Total number of bytes received",
	})
	systemInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "system_info",
		Help:      "Information about server version",
	}, []string{"server_version", "platform", "status", "name"})
)

func (app *App) Run(cmd *cobra.Command, args []string) {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		srv := createServer(app.ListenAddress+":"+app.ListenPort, metricsMux)
		log.Fatal(srv.ListenAndServe())
	}()

	prometheus.MustRegister(clientsConnected)
	prometheus.MustRegister(uptime)
	prometheus.MustRegister(maxClients)
	prometheus.MustRegister(bytesSentTotal)
	prometheus.MustRegister(bytesReceivedTotal)
	prometheus.MustRegister(systemInfo)

	client, err := ts3.NewClient(app.Address)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Login(app.Username, app.Password)
	if err != nil {
		log.Fatal(err)
	}
	client.Use(1)
	for {
		sm, err := client.Server.Info()
		if err != nil {
			log.Fatal(err)
		}
		sc, err := client.Server.ServerConnectionInfo()
		if err != nil {
			log.Fatal(err)
		}
		clientsConnected.Set(float64(sm.ClientsOnline - sm.QueryClientsOnline))
		uptime.Set(float64(sm.Uptime))
		maxClients.Set(float64(sm.MaxClients))
		bytesSentTotal.Set(float64(sc.BytesSentTotal))
		bytesReceivedTotal.Set(float64(sc.BytesReceivedTotal))
		systemInfo.WithLabelValues(sm.Version, sm.Platform, sm.Status, sm.Name).Set(1.0)
		time.Sleep(5 * time.Second)
	}

}

func (app *App) Bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&app.ListenAddress, "listen-address", "localhost", `IP on which the exporter should listen`)
	cmd.PersistentFlags().StringVar(
		&app.ListenPort, "listen-port", "8010", `Port on which the exporter should listen`)
	cmd.PersistentFlags().StringVar(
		&app.Address, "address", "localhost:10011", `Address of the teamspeak server`)
	cmd.PersistentFlags().StringVarP(
		&app.Username, "username", "u", "serveradmin", `Username for ServerQuery login`)
	cmd.PersistentFlags().StringVarP(
		&app.Password, "password", "p", "", `Password for ServerQuery login`)
}

func NewRootCommand() *cobra.Command {
	cmd := cmdutil.NewRootCommand(new(App))
	cmd.Short = "Metrics exporter for TeamSpeak 3 server"
	return cmd
}

func createServer(addr string, serveMux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      serveMux,
	}
}

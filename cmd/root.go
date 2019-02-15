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
	Port     string
	Address  string
	Username string
	Password string
}

func (app *App) Run(cmd *cobra.Command, args []string) {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		srv := createServer(":"+app.Port, metricsMux)
		log.Fatal(srv.ListenAndServe())
	}()

	clientsConnected := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "clients",
		Help:      "Number of connected clients",
	})
	prometheus.MustRegister(clientsConnected)
	uptime := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "uptime",
		Help:      "Uptime in seconds",
	})
	prometheus.MustRegister(uptime)
	maxClients := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "teamspeak",
		Name:      "maxClients",
		Help:      "blabla",
	})
	prometheus.MustRegister(maxClients)

	client, _ := ts3.NewClient(app.Address)
	_ = client.Login(app.Username, app.Password)
	client.Use(1)
	for {
		sm, _ := client.Server.Info()
		//sc, _ := client.Server.ServerConnectionInfo()
		clientsConnected.Set(float64(sm.ClientsOnline))
		uptime.Set(float64(sm.Uptime))
		maxClients.Set(float64(sm.MaxClients))
		time.Sleep(5 * time.Second)
	}

}

func (app *App) Bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&app.Port, "port", "p", "8010", `Port on which the server should listen`)
	cmd.PersistentFlags().StringVar(
		&app.Address, "address", "localhost:10011", `Address of the teamspeak server`)
	cmd.PersistentFlags().StringVarP(
		&app.Username, "username", "u", "", `Username for ServerQuery login`)
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

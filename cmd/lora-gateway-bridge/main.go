package main

//go:generate ./doc.sh

import (
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/TheThingsNetwork/ttn/api"
	"github.com/brocaar/lora-gateway-bridge/backend/thethingsnetwork"
	"github.com/brocaar/lora-gateway-bridge/gateway"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/codegangsta/cli"
)

var version string // set by the compiler

func run(c *cli.Context) error {
	log.SetLevel(log.Level(uint8(c.Int("log-level"))))

	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://docs.loraserver.io/lora-gateway-bridge/",
	}).Info("starting LoRa Gateway Bridge")

	if rootCAFile := c.String("root-ca-file"); rootCAFile != "" {
		roots, err := ioutil.ReadFile(rootCAFile)
		if err != nil {
			log.WithError(err).Fatal("Could not load Root CA file")
		}
		if !api.RootCAs.AppendCertsFromPEM(roots) {
			log.Warn("Could not load all CAs from the Root CA file")
		} else {
			log.Infof("Using Root CAs from %s", rootCAFile)
		}
	}

	ttn, err := thethingsnetwork.NewBackend(
		c.String("ttn-discovery-server"),
		c.String("ttn-router"),
	)
	if err != nil {
		log.Fatalf("could not setup ttn backend: %s", err)
	}
	defer ttn.Close()
	ttn.SetRxRateLimit(10)

	if c.String("ttn-account-server") != "" {
		if err := ttn.SetAccount(
			c.String("ttn-account-server"),
			c.String("ttn-account-client-id"),
			c.String("ttn-account-client-secret"),
		); err != nil {
			log.Errorf("could not set account server: %s", err)
		}
	}

	if c.String("ttn-gateway-id") != "" {
		if err := ttn.AddGateway(
			c.String("ttn-gateway-eui"),
			c.String("ttn-gateway-id"),
			c.String("ttn-gateway-key"),
			c.String("ttn-gateway-token"),
		); err != nil {
			log.Errorf("could not add gateway: %s", err)
		}
	}

	if bridge := c.String("ttn-inject-bridge"); bridge != "" {
		ttn.InjectBridge(bridge)
	}

	if region := c.String("ttn-inject-region"); region != "" {
		ttn.InjectRegion(region)
	}

	if rtt := c.Uint("ttn-inject-rtt"); rtt != 0 {
		ttn.InjectRTT(rtt)
	}

	onNew := func(mac lorawan.EUI64) error {
		return ttn.SubscribeGatewayTX(mac)
	}

	onDelete := func(mac lorawan.EUI64) error {
		return ttn.UnSubscribeGatewayTX(mac)
	}

	gwBackend, err := gateway.NewBackend(c.String("udp-bind"), onNew, onDelete)
	if err != nil {
		log.Fatalf("could not setup gateway backend: %s", err)
	}
	defer gwBackend.Close()

	go func() {
		for rxPacket := range gwBackend.RXPacketChan() {
			go func(rxPacket gw.RXPacketBytes) {
				if err := ttn.PublishGatewayRX(rxPacket.RXInfo.MAC, rxPacket); err != nil {
					log.Errorf("could not publish RXPacket: %s", err)
				}
			}(rxPacket)
		}
	}()

	go func() {
		for stats := range gwBackend.StatsChan() {
			go func(stats gw.GatewayStatsPacket) {
				if err := ttn.PublishGatewayStats(stats.MAC, stats); err != nil {
					log.Errorf("could not publish GatewayStatsPacket: %s", err)
				}
			}(stats)
		}
	}()

	go func() {
		for txPacket := range ttn.TXPacketChan() {
			if err := gwBackend.Send(txPacket); err != nil {
				log.Errorf("could not send TXPacket: %s", err)
			}
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "lora-gateway-bridge"
	app.Usage = "abstracts the packet_forwarder protocol into protocol buffers over gRPC"
	app.Copyright = "See http://github.com/brocaar/lora-gateway-bridge for copyright information"
	app.Version = version
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "udp-bind",
			Usage:  "ip:port to bind the UDP listener to",
			Value:  "0.0.0.0:1700",
			EnvVar: "UDP_BIND",
		},
		cli.StringFlag{
			Name:   "ttn-account-server",
			Usage:  "TTN Account Server",
			Value:  "https://account.thethingsnetwork.org",
			EnvVar: "TTN_ACCOUNT_SERVER",
		},
		cli.StringFlag{
			Name:   "ttn-account-client-id",
			Usage:  "TTN Account Client ID",
			Value:  "",
			EnvVar: "TTN_ACCOUNT_CLIENT_ID",
		},
		cli.StringFlag{
			Name:   "ttn-account-client-secret",
			Usage:  "TTN Account Client Secret",
			Value:  "",
			EnvVar: "TTN_ACCOUNT_CLIENT_SECRET",
		},
		cli.StringFlag{
			Name:   "ttn-discovery-server",
			Usage:  "TTN Discovery Server",
			Value:  "localhost:1900",
			EnvVar: "TTN_DISCOVERY_SERVER",
		},
		cli.StringFlag{
			Name:   "root-ca-file",
			Usage:  "Root CA file",
			Value:  "",
			EnvVar: "ROOT_CA_FILE",
		},
		cli.StringFlag{
			Name:   "ttn-router",
			Usage:  "TTN Router ID",
			Value:  "dev",
			EnvVar: "TTN_ROUTER",
		},
		cli.StringFlag{
			Name:   "ttn-gateway-eui",
			Usage:  "TTN Gateway EUI",
			Value:  "",
			EnvVar: "TTN_GATEWAY_EUI",
		},
		cli.StringFlag{
			Name:   "ttn-gateway-id",
			Usage:  "TTN Gateway ID",
			Value:  "",
			EnvVar: "TTN_GATEWAY_ID",
		},
		cli.StringFlag{
			Name:   "ttn-gateway-key",
			Usage:  "TTN Gateway Key",
			Value:  "",
			EnvVar: "TTN_GATEWAY_KEY",
		},
		cli.StringFlag{
			Name:   "ttn-gateway-token",
			Usage:  "TTN Gateway Token",
			Value:  "",
			EnvVar: "TTN_GATEWAY_TOKEN",
		},
		cli.StringFlag{
			Name:   "ttn-inject-bridge",
			Usage:  "TTN Bridge string to inject into Status messages",
			Value:  "",
			EnvVar: "TTN_INJECT_BRIDGE",
		},
		cli.StringFlag{
			Name:   "ttn-inject-region",
			Usage:  "TTN Region to inject into Status messages",
			Value:  "", // Possible values: EU_863_870 US_902_928 CN_779_787 EU_433 AU_915_928 CN_470_510 AS_923 KR_920_923
			EnvVar: "TTN_INJECT_REGION",
		},
		cli.UintFlag{
			Name:   "ttn-inject-rtt",
			Usage:  "TTN RTT to inject into Status messages",
			Value:  0,
			EnvVar: "TTN_INJECT_RTT",
		},
		cli.IntFlag{
			Name:   "log-level",
			Value:  4,
			Usage:  "debug=5, info=4, warning=3, error=2, fatal=1, panic=0",
			EnvVar: "LOG_LEVEL",
		},
	}
	app.Run(os.Args)
}

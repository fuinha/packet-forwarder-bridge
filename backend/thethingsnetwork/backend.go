package thethingsnetwork

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"golang.org/x/net/context"

	"github.com/TheThingsNetwork/ttn/api"
	"github.com/TheThingsNetwork/ttn/core/types"

	log "github.com/Sirupsen/logrus"
	pb_discovery "github.com/TheThingsNetwork/ttn/api/discovery"
	pb_gateway "github.com/TheThingsNetwork/ttn/api/gateway"
	pb_protocol "github.com/TheThingsNetwork/ttn/api/protocol"
	pb_lorawan "github.com/TheThingsNetwork/ttn/api/protocol/lorawan"
	pb_router "github.com/TheThingsNetwork/ttn/api/router"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
)

type gatewayConn struct {
	uplink         pb_router.Router_UplinkClient
	downlink       pb_router.Router_SubscribeClient
	downlinkCancel context.CancelFunc
	stat           pb_router.Router_GatewayStatusClient
}

// Backend implements the TTN backend
type Backend struct {
	token        string
	conn         *grpc.ClientConn
	client       pb_router.RouterClient
	txPacketChan chan gw.TXPacketBytes
	gateways     map[lorawan.EUI64]*gatewayConn
	mutex        sync.RWMutex
}

func (b *Backend) getConn(mac lorawan.EUI64) *gatewayConn {
	// We're going to be optimistic and guess that the gateway is already active
	b.mutex.RLock()
	gtw, ok := b.gateways[mac]
	b.mutex.RUnlock()
	if ok {
		return gtw
	}
	// If it doesn't we still have to lock
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if _, ok := b.gateways[mac]; !ok {
		b.gateways[mac] = &gatewayConn{}
	}
	return b.gateways[mac]
}

// NewBackend creates a new Backend.
func NewBackend(discovery, router, token string) (*Backend, error) {
	b := Backend{
		token:        token,
		txPacketChan: make(chan gw.TXPacketBytes),
		gateways:     make(map[lorawan.EUI64]*gatewayConn),
	}

	var routerConn *grpc.ClientConn
	if discovery != "" {
		discoveryConn, err := grpc.Dial(discovery, append(api.DialOptions, grpc.WithInsecure())...)
		if err != nil {
			return nil, err
		}
		discovery := pb_discovery.NewDiscoveryClient(discoveryConn)
		md := metadata.Pairs(
			"service-name", "lora-gateway-bridge",
		)
		ctx := metadata.NewContext(context.Background(), md)
		router, err := discovery.Get(ctx, &pb_discovery.GetRequest{
			ServiceName: "router",
			Id:          router,
		})
		if err != nil {
			return nil, err
		}
		routerConn, err = router.Dial()
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		routerConn, err = grpc.Dial(router, append(api.DialOptions, grpc.WithInsecure())...)
		if err != nil {
			return nil, err
		}
	}

	b.conn = routerConn
	b.client = pb_router.NewRouterClient(routerConn)

	return &b, nil
}

// Close closes the backend.
func (b *Backend) Close() {
	for _, gtw := range b.gateways {
		if gtw.uplink != nil {
			gtw.uplink.CloseSend()
		}
		if gtw.downlink != nil {
			gtw.downlinkCancel()
		}
		if gtw.stat != nil {
			gtw.stat.CloseSend()
		}
	}
	<-time.After(1 * time.Second)
	b.conn.Close()
	<-time.After(1 * time.Second)
}

// TXPacketChan returns the TXPacketBytes channel.
func (b *Backend) TXPacketChan() chan gw.TXPacketBytes {
	return b.txPacketChan
}

// SubscribeGatewayTX subscribes the backend to the gateway TXPacket
// topic (packets the gateway needs to transmit).
func (b *Backend) SubscribeGatewayTX(mac lorawan.EUI64) error {
	conn := b.getConn(mac)
	if conn.downlink != nil {
		return nil
	}

	ctx, cancel := b.getContext(mac)
	conn.downlinkCancel = cancel

	go func() {
		for {
			stream, err := b.client.Subscribe(ctx, &pb_router.SubscribeRequest{})
			if err != nil {
				<-time.After(api.Backoff)
				continue
			}
			conn.downlink = stream

			for {
				in, err := stream.Recv()
				if err != nil && grpc.Code(err) == codes.Canceled {
					break
				}
				if err != nil {
					log.Errorf("backend/thethingsnetwork: error in downlink stream: %s", err)
					<-time.After(api.Backoff)
					break
				}
				log.WithField("gateway", mac).Info("backend/thethingsnetwork: message received")
				lora := in.ProtocolConfiguration.GetLorawan()
				if lora == nil {
					log.Error("backend/thethingsnetwork: received non-Lora message")
					continue
				}

				var dataRate band.DataRate

				if lora.Modulation == pb_lorawan.Modulation_LORA {
					dr, _ := types.ParseDataRate(lora.DataRate)
					dataRate.Modulation = band.LoRaModulation
					dataRate.SpreadFactor = int(dr.SpreadingFactor)
					dataRate.Bandwidth = int(dr.Bandwidth)
				}

				if lora.Modulation == pb_lorawan.Modulation_FSK {
					dataRate.Modulation = band.FSKModulation
					dataRate.BitRate = int(lora.BitRate)
				}

				var txPacket gw.TXPacketBytes
				txPacket.TXInfo = gw.TXInfo{
					MAC:       mac,
					Timestamp: in.GatewayConfiguration.Timestamp,
					Frequency: int(in.GatewayConfiguration.Frequency),
					Power:     int(in.GatewayConfiguration.Power),
					DataRate:  dataRate,
					CodeRate:  lora.CodingRate,
				}
				txPacket.PHYPayload = in.Payload
				b.txPacketChan <- txPacket
			}
		}
	}()

	return nil
}

// UnSubscribeGatewayTX unsubscribes the backend from the gateway TXPacket
// topic.
func (b *Backend) UnSubscribeGatewayTX(mac lorawan.EUI64) error {
	conn := b.getConn(mac)
	if conn.downlink != nil {
		conn.downlinkCancel()
		conn.downlink = nil
	}
	return nil
}

func convertRXPacket(rxPacket gw.RXPacketBytes) *pb_router.UplinkMessage {
	// Convert some Modulation-dependent fields
	var modulation pb_lorawan.Modulation
	var datarate string
	var bitrate uint32
	switch rxPacket.RXInfo.DataRate.Modulation {
	case band.LoRaModulation:
		modulation = pb_lorawan.Modulation_LORA
		datarate = fmt.Sprintf("SF%dBW%d", rxPacket.RXInfo.DataRate.SpreadFactor, rxPacket.RXInfo.DataRate.Bandwidth)
	case band.FSKModulation:
		modulation = pb_lorawan.Modulation_FSK
		bitrate = uint32(rxPacket.RXInfo.DataRate.BitRate)
	}

	return &pb_router.UplinkMessage{
		Payload: rxPacket.PHYPayload,
		ProtocolMetadata: &pb_protocol.RxMetadata{Protocol: &pb_protocol.RxMetadata_Lorawan{Lorawan: &pb_lorawan.Metadata{
			Modulation: modulation,
			DataRate:   datarate,
			BitRate:    bitrate,
			CodingRate: rxPacket.RXInfo.CodeRate,
		}}},
		GatewayMetadata: &pb_gateway.RxMetadata{
			GatewayId: fmt.Sprintf("eui-%s", rxPacket.RXInfo.MAC),
			Timestamp: rxPacket.RXInfo.Timestamp,
			Time:      rxPacket.RXInfo.Time.UnixNano(),
			RfChain:   uint32(rxPacket.RXInfo.RFChain),
			Channel:   uint32(rxPacket.RXInfo.Channel),
			Frequency: uint64(rxPacket.RXInfo.Frequency),
			Rssi:      float32(rxPacket.RXInfo.RSSI),
			Snr:       float32(rxPacket.RXInfo.LoRaSNR),
		},
	}
}

// PublishGatewayRX publishes a RX packet to the MQTT broker.
func (b *Backend) PublishGatewayRX(mac lorawan.EUI64, rxPacket gw.RXPacketBytes) error {
	conn := b.getConn(mac)
	if conn.uplink == nil {
		ctx, _ := b.getContext(mac)
		uplink, err := b.client.Uplink(ctx)
		if err != nil {
			return err
		}
		conn.uplink = uplink
	}
	pkt := convertRXPacket(rxPacket)
	err := conn.uplink.Send(pkt)
	if err != nil && (grpc.Code(err) == codes.Canceled || grpc.Code(err) == codes.Internal) {
		conn.uplink = nil
		// TODO: Maybe retry?
	}
	return err
}

func convertStatsPacket(stats gw.GatewayStatsPacket) *pb_gateway.Status {
	status := &pb_gateway.Status{
		Time: stats.Time.UnixNano(),
		RxIn: uint32(stats.RXPacketsReceived),
		RxOk: uint32(stats.RXPacketsReceivedOK),
	}

	if platform, ok := stats.CustomData["platform"]; ok {
		if platform, ok := platform.(string); ok {
			status.Platform = string(platform)
		}
	}
	if contactEmail, ok := stats.CustomData["contactEmail"]; ok {
		if contactEmail, ok := contactEmail.(string); ok {
			status.ContactEmail = string(contactEmail)
		}
	}
	if description, ok := stats.CustomData["description"]; ok {
		if description, ok := description.(string); ok {
			status.Description = string(description)
		}
	}

	if stats.Latitude != 0 || stats.Longitude != 0 || stats.Altitude != 0 {
		status.Gps = &pb_gateway.GPSMetadata{
			Latitude:  float32(stats.Latitude),
			Longitude: float32(stats.Longitude),
			Altitude:  int32(stats.Altitude),
		}
	}

	return status
}

// PublishGatewayStats publishes a GatewayStatsPacket to the MQTT broker.
func (b *Backend) PublishGatewayStats(mac lorawan.EUI64, stats gw.GatewayStatsPacket) error {
	conn := b.getConn(mac)
	if conn.stat == nil {
		ctx, _ := b.getContext(mac)
		stat, err := b.client.GatewayStatus(ctx)
		if err != nil {
			return err
		}
		conn.stat = stat
	}
	err := conn.stat.Send(convertStatsPacket(stats))
	if err != nil && (grpc.Code(err) == codes.Canceled || grpc.Code(err) == codes.Internal) {
		conn.stat = nil
		// TODO: Maybe retry?
	}
	return err
}

func (b *Backend) getContext(mac lorawan.EUI64) (context.Context, context.CancelFunc) {
	md := metadata.Pairs(
		"token", b.token,
		"id", fmt.Sprintf("eui-%s", mac),
	)
	return context.WithCancel(metadata.NewContext(context.Background(), md))
}

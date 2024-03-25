package proxy

import (
	"context"
	"fmt"
	"net"

	"github.com/bedrock-tool/bedrocktool/locale"
	"github.com/bedrock-tool/bedrocktool/ui/messages"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"github.com/sirupsen/logrus"
)

func (p *Context) onResourcePacksInfo() {
	p.ui.HandleMessage(&messages.Message{
		Source: "proxy",
		Data:   messages.ConnectStateReceivingResources,
	})
}

func (p *Context) onFinishedPack(pack *resource.Pack) {
	p.ui.HandleMessage(&messages.Message{
		Source: "proxy",
		Data:   messages.FinishedPack{Pack: pack},
	})
}

func (p *Context) connectServer(ctx context.Context) (err error) {
	if p.withClient {
		select {
		case <-p.clientConnecting:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	p.ui.HandleMessage(&messages.Message{
		Source: "proxy",
		Data:   messages.ConnectStateServerConnecting,
	})
	logrus.Info(locale.Loc("connecting", locale.Strmap{"Address": p.serverAddress}))
	d := minecraft.Dialer{
		TokenSource: p.tokenSource,
		PacketFunc:  p.packetFunc,
		GetClientData: func() login.ClientData {
			if p.withClient {
				select {
				case <-p.haveClientData:
				case <-ctx.Done():
				}
			}
			return p.clientData
		},
		EarlyConnHandler: func(c *minecraft.Conn) {
			p.Server = c
			p.rpHandler.SetServer(c)
			c.ResourcePackHandler = p.rpHandler
		},
	}
	for retry := 0; retry < 3; retry++ {
		d.ChainKey, d.ChainData, err = minecraft.CreateChain(ctx, p.tokenSource)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	server, err := d.DialContext(ctx, "raknet", p.serverAddress)
	if err != nil {
		return err
	}
	p.Server = server

	p.ui.HandleMessage(&messages.Message{
		Source: "proxy",
		Data:   messages.ConnectStateEstablished,
	})
	logrus.Debug(locale.Loc("connected", nil))
	return nil
}

func (p *Context) connectClient(ctx context.Context, serverAddress string) (err error) {
	var extraClientDebug func(pk packet.Packet)
	var extraClientDebugEnd func()
	if p.ExtraDebug {
		extraClientDebug, extraClientDebugEnd = newExtraDebug("packets-client.log")
	}

	p.listener, err = minecraft.ListenConfig{
		StatusProvider: minecraft.NewStatusProvider(fmt.Sprintf("%s Proxy", serverAddress)),
		PacketFunc: func(header packet.Header, payload []byte, src, dst net.Addr) {
			if extraClientDebug != nil {
				pk, ok := DecodePacket(header, payload)
				if !ok {
					return
				}
				extraClientDebug(pk)
			}
		},
		OnClientData: func(c *minecraft.Conn) {
			p.clientData = c.ClientData()
			close(p.haveClientData)
		},
		EarlyConnHandler: func(c *minecraft.Conn) {
			p.Client = c
			p.rpHandler.SetClient(c)
			c.ResourcePackHandler = p.rpHandler
			close(p.clientConnecting)
		},
	}.Listen("raknet", ":19132")
	if err != nil {
		return err
	}

	p.ui.HandleMessage(&messages.Message{
		Source: "proxy",
		Data:   messages.ConnectStateListening,
	})
	logrus.Infof(locale.Loc("listening_on", locale.Strmap{"Address": p.listener.Addr()}))
	logrus.Infof(locale.Loc("help_connect", nil))

	var accepted = false

	go func() {
		<-ctx.Done()
		if extraClientDebugEnd != nil {
			extraClientDebugEnd()
		}
		if !accepted {
			_ = p.listener.Close()
		}
	}()

	c, err := p.listener.Accept()
	if err != nil {
		return err
	}
	accepted = true
	p.Client = c.(*minecraft.Conn)
	return nil
}

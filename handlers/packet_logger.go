package handlers

import (
	"bufio"
	"io"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/bedrock-tool/bedrocktool/utils"
	"github.com/bedrock-tool/bedrocktool/utils/crypt"
	"github.com/bedrock-tool/bedrocktool/utils/proxy"
	"github.com/fatih/color"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

var MutedPackets = []uint32{
	packet.IDUpdateBlock,
	packet.IDMoveActorAbsolute,
	packet.IDSetActorMotion,
	packet.IDSetTime,
	packet.IDRemoveActor,
	packet.IDAddActor,
	packet.IDUpdateAttributes,
	packet.IDInteract,
	packet.IDLevelEvent,
	packet.IDSetActorData,
	packet.IDMoveActorDelta,
	packet.IDMovePlayer,
	packet.IDBlockActorData,
	packet.IDPlayerAuthInput,
	packet.IDLevelChunk,
	packet.IDLevelSoundEvent,
	packet.IDActorEvent,
	packet.IDNetworkChunkPublisherUpdate,
	packet.IDUpdateSubChunkBlocks,
	packet.IDSubChunk,
	packet.IDSubChunkRequest,
	packet.IDAnimate,
	packet.IDNetworkStackLatency,
	packet.IDInventoryTransaction,
	packet.IDPlaySound,
}

var dirS2C = color.GreenString("S") + "->" + color.CyanString("C")
var dirC2S = color.CyanString("C") + "->" + color.GreenString("S")

func NewDebugLogger(extraVerbose bool) *proxy.Handler {
	var logPlain, logCrypt, logCryptEnc io.WriteCloser
	var packetsLogF *bufio.Writer
	var dmpLock sync.Mutex

	if extraVerbose {
		// open plain text log
		logPlain, err := os.Create("packets.log")
		if err != nil {
			logrus.Error(err)
		}
		// open gpg log
		logCryptEnc, err = crypt.Encer("packets.log.gpg")
		if err != nil {
			logrus.Error(err)
		}
		if logPlain != nil || logCryptEnc != nil {
			packetsLogF = bufio.NewWriter(io.MultiWriter(logPlain, logCryptEnc))
		}
	}

	return &proxy.Handler{
		Name: "Debug",
		PacketCB: func(pk packet.Packet, toServer bool, timeReceived time.Time, preLogin bool) (packet.Packet, error) {
			if packetsLogF != nil {
				dmpLock.Lock()
				packetsLogF.Write([]byte(utils.DumpStruct(0, pk, true, false) + "\n\n\n"))
				dmpLock.Unlock()
			}

			if !slices.Contains(MutedPackets, pk.ID()) {
				var dir string = dirS2C
				if toServer {
					dir = dirC2S
				}
				pkName := reflect.TypeOf(pk).String()[1:]
				logrus.Debugf("%s 0x%02x, %s", dir, pk.ID(), pkName)
			}
			return pk, nil
		},
		Deferred: func() {
			dmpLock.Lock()
			if packetsLogF != nil {
				packetsLogF.Flush()
			}
			if logPlain != nil {
				logPlain.Close()
			}
			if logCryptEnc != nil {
				logCryptEnc.Close()
			}
			if logCrypt != nil {
				logCrypt.Close()
			}
			dmpLock.Unlock()
		},
	}
}

func init() {
	// inject to proxy to avoid loop
	proxy.NewDebugLogger = NewDebugLogger
}

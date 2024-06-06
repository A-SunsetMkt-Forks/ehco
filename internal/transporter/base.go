package transporter

import (
	"fmt"
	"net"

	"github.com/Ehco1996/ehco/internal/cmgr"
	"github.com/Ehco1996/ehco/internal/conn"
	"github.com/Ehco1996/ehco/internal/metrics"
	"github.com/Ehco1996/ehco/internal/relay/conf"

	"github.com/Ehco1996/ehco/pkg/lb"
	"go.uber.org/zap"
)

type baseTransporter struct {
	cfg *conf.Config
	l   *zap.SugaredLogger

	cmgr       cmgr.Cmgr
	tCPRemotes lb.RoundRobin
}

func NewBaseTransporter(cfg *conf.Config, cmgr cmgr.Cmgr) *baseTransporter {
	return &baseTransporter{
		cfg:        cfg,
		cmgr:       cmgr,
		tCPRemotes: cfg.ToTCPRemotes(),
		l:          zap.S().Named(cfg.GetLoggerName()),
	}
}

func (b *baseTransporter) GetTCPListenAddr() (*net.TCPAddr, error) {
	return net.ResolveTCPAddr("tcp", b.cfg.Listen)
}

func (b *baseTransporter) GetRemote() *lb.Node {
	return b.tCPRemotes.Next()
}

func (b *baseTransporter) RelayTCPConn(c net.Conn, handshakeF TCPHandShakeF) error {
	remote := b.GetRemote()
	metrics.CurConnectionCount.WithLabelValues(remote.Label, metrics.METRIC_CONN_TYPE_TCP).Inc()
	defer metrics.CurConnectionCount.WithLabelValues(remote.Label, metrics.METRIC_CONN_TYPE_TCP).Dec()

	// check limit
	if b.cfg.MaxConnection > 0 && b.cmgr.CountConnection(cmgr.ConnectionTypeActive) >= b.cfg.MaxConnection {
		c.Close()
		return fmt.Errorf("relay:%s active connection count exceed limit %d", b.cfg.Label, b.cfg.MaxConnection)
	}

	clonedRemote := remote.Clone()
	rc, err := handshakeF(clonedRemote)
	if err != nil {
		return err
	}

	b.l.Infof("RelayTCPConn from %s to %s", c.LocalAddr(), remote.Address)
	relayConn := conn.NewRelayConn(
		b.cfg.Label, c, rc, conn.WithHandshakeDuration(clonedRemote.HandShakeDuration))
	b.cmgr.AddConnection(relayConn)
	defer b.cmgr.RemoveConnection(relayConn)
	return relayConn.Transport(remote.Label)
}

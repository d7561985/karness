package executor

import (
	"fmt"
	"testing"
)

func TestGRPC(t *testing.T) {
	const target = "sportsbook-settings-sv.odds-compiler.svc.cluster.local:9000"
	const symbol = "egt.oddscompiler.sportsbooksettings.v3.public.InfoService/GetEventInfo"

	g := NewGRPC()

	res := g.Go(target, symbol, `{"id":"1"}`)

	fmt.Println(res)
}

package netboxparser

import (
	"log/slog"

	"github.com/mattieserver/netbox-oxidized-sync/internal/httphelper"
	"github.com/mattieserver/netbox-oxidized-sync/internal/model"
)

func ParseFortigateInterfaces(fortiInterfaces *model.FortigateInterfaces, netboxDeviceInterfaces *[]httphelper.NetboxInterface) {
	slog.Info("ok")
}
package v2rpc

import (
	"context"
	"fmt"

	"github.com/xtls/xray-core/proxy/vless"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"v2ray.com/core/app/proxyman/command"
	stat "v2ray.com/core/app/stats/command"
	"v2ray.com/core/common/protocol"
	"v2ray.com/core/common/serial"
)

func GetGrpcConn(addr string) *grpc.ClientConn {

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	return cc
}

func Adduser(uuidstr, mail string, cc *grpc.ClientConn) (*command.AlterInboundResponse, error) {

	hsc := command.NewHandlerServiceClient(cc)
	req := command.AlterInboundRequest{
		Tag: "proxy",
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: &protocol.User{
				Level: 0,
				Email: mail,
				Account: serial.ToTypedMessage(&vless.Account{
					Id: uuidstr,
				}),
			},
		}),
	}
	return hsc.AlterInbound(context.Background(), &req)

}

func RemoveUser(mail string, cc *grpc.ClientConn) (*command.AlterInboundResponse, error) {
	hsc := command.NewHandlerServiceClient(cc)

	req := command.AlterInboundRequest{
		Tag:       "proxy",
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{Email: mail}),
	}
	return hsc.AlterInbound(context.Background(), &req)

}

func GetUserStat(mail string, cc *grpc.ClientConn) uint64 {
	ssc := stat.NewStatsServiceClient(cc)

	req := stat.GetStatsRequest{
		Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", mail),
		Reset_: false,
	}

	rsp, err := ssc.GetStats(context.Background(), &req)
	if err != nil {
		return 0
	}

	return uint64(rsp.Stat.Value)
}

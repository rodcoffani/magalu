/*
import "magalu.cloud/lib/products/network/port/vpcs_ports"
*/
package vpcsPorts

import (
	"context"

	mgcClient "magalu.cloud/lib"
)

type service struct {
	ctx    context.Context
	client *mgcClient.Client
}

type Service interface {
	Create(parameters CreateParameters, configs CreateConfigs) (result CreateResult, err error)
	List(parameters ListParameters, configs ListConfigs) (result ListResult, err error)
}

func NewService(ctx context.Context, client *mgcClient.Client) Service {
	return &service{ctx, client}
}
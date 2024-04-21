package gqlapi

import (
	"context"

	"github.com/algonode/reti-algodrop/internal/config"
	"github.com/hasura/go-graphql-client"
	"github.com/sirupsen/logrus"
)

type GqlAPI struct {
	cfg    *config.GraphQLConfig
	log    *logrus.Logger
	client *graphql.Client
}

func Make(ctx context.Context, acfg *config.GraphQLConfig, log *logrus.Logger) (*GqlAPI, error) {

	// Create an algod client
	client := graphql.NewClient(acfg.Address, nil)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to make algod client: %s\n", err)
	// }

	return &GqlAPI{
		cfg:    acfg,
		log:    log,
		client: client,
	}, nil

}

package gqlapi

import (
	"context"

	"github.com/hasura/go-graphql-client"
)

type OnlineAccounts struct {
	Accounts struct {
		Nodes []struct {
			Addr     string
			IsOnline bool
			VoteLast uint64
			Algos    float64
		}
	} `graphql:"accounts(condition:{isOnline:true} filter:{voteLast:{greaterThanOrEqualTo: $round}} first:5000 orderBy: ALGOS_DESC)"`
}

func (g *GqlAPI) OnlineAccounts(ctx context.Context, round uint64) (*OnlineAccounts, error) {
	var q OnlineAccounts
	variables := map[string]interface{}{
		"round": graphql.Int(round),
	}
	err := g.client.Query(ctx, &q, variables)
	if err != nil {
		g.log.WithError(err).Error("Error executing GraphQL query")
		return nil, err
	}
	//log.Infof("%#v", q)
	return &q, nil
}

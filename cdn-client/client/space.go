package client

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/pkg/errors"
)

func (c *Client) QueryTerritoryInfo(name string) (chain.TerritoryInfo, error) {

	info, err := c.ChainClient.QueryTerritory(c.ChainClient.GetSignatureAccPulickey(), name, -1)
	if err != nil {
		return info, errors.Wrap(err, "query user space info error")
	}
	return info, nil
}

func (c *Client) MintTerritory(gibCount, days uint32, territoryName string) (string, error) {

	res, err := c.ChainClient.MintTerritory(gibCount, territoryName, days)
	if err != nil {
		return res, errors.Wrap(err, "buy space error")
	}
	return res, nil
}

func (c *Client) RenewalTerritory(day uint32, territoryName string) (string, error) {

	res, err := c.ChainClient.RenewalTerritory(territoryName, day)
	if err != nil {
		return res, errors.Wrap(err, "renewal space error")
	}
	return res, nil
}

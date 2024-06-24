package client

import (
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/pkg/errors"
)

func (c *Client) QueryUserSpaceInfo() (chain.UserSpaceInfo, error) {
	info, err := c.QueryUserOwnedSpace(c.PublicKey, -1)
	if err != nil {
		return info, errors.Wrap(err, "query user space info error")
	}
	return info, nil
}

func (c *Client) BuySpace(gibCount uint32) (string, error) {
	res, err := c.ChainClient.BuySpace(gibCount)
	if err != nil {
		return res, errors.Wrap(err, "buy space error")
	}
	return res, nil
}

func (c *Client) RenewalSpace(day uint32) (string, error) {
	res, err := c.ChainClient.RenewalSpace(day)
	if err != nil {
		return res, errors.Wrap(err, "renewal space error")
	}
	return res, nil
}

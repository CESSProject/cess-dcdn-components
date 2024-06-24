package client

import "github.com/pkg/errors"

func (c *Client) CreateBucket(name string) (string, error) {
	hash, err := c.ChainClient.CreateBucket(c.PublicKey, name)
	if err != nil {
		return "", errors.Wrap(err, "create bucket error")
	}
	return hash, nil
}

func (c *Client) DeleteBucket(name string) (string, error) {
	hash, err := c.ChainClient.DeleteBucket(c.PublicKey, name)
	if err != nil {
		return "", errors.Wrap(err, "delete bucket error")
	}
	return hash, nil
}

func (c *Client) ListBuckets() ([]string, error) {
	list, err := c.QueryAllBucketName(c.PublicKey, -1)
	if err != nil {
		return nil, errors.Wrap(err, "list buckets error")
	}
	return list, nil
}

func (c *Client) ListFilesInBucket(name string) ([]string, error) {
	info, err := c.QueryBucket(c.PublicKey, name, -1)
	if err != nil {
		return nil, errors.Wrap(err, "list files in bucket error")
	}
	files := make([]string, len(info.FileList))
	for i, file := range info.FileList {
		files[i] = string(file[:])
	}
	return files, nil
}

func (c *Client) ListUserFiles() ([]string, error) {
	list, err := c.QueryAllUserFiles(c.PublicKey, -1)
	if err != nil {
		return nil, errors.Wrap(err, "list user files error")
	}
	return list, nil
}

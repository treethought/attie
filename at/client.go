package at

import (
	"context"
	"fmt"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	log "github.com/sirupsen/logrus"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type Client struct {
	dir identity.Directory
	c   *atclient.APIClient
}

func NewClient(service string) *Client {
	dir := &identity.BaseDirectory{}
	cacheDir := identity.NewCacheDirectory(dir, 0, 0, 0, 0)
	if service == "" {
		service = "https://bsky.social"
	}
	client := atclient.NewAPIClient(service)
	return &Client{
		dir: cacheDir,
		c:   client,
	}
}

func (c *Client) withIdentifier(ctx context.Context, raw string) (*atclient.APIClient, error) {
	id, err := syntax.ParseAtIdentifier(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repo identifier: %w", err)
	}
	idd, err := c.dir.Lookup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup identifier: %w", err)
	}
	log.WithFields(log.Fields{
		"handle": idd.Handle,
		"DID":    idd.DID,
	}).Info("identifier resolved")
	return c.c.WithService(idd.PDSEndpoint()), nil
}

func (c *Client) GetRepo(ctx context.Context, repo string) (*comatproto.RepoDescribeRepo_Output, error) {
	log.WithFields(log.Fields{
		"repo": repo,
	}).Info("describe repo")

	client, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	// TODO: download repo as car
	// https://github.com/bluesky-social/cookbook/blob/main/go-repo-export/main.go#L46
	resp, err := comatproto.RepoDescribeRepo(ctx, client, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to describe repo: %w", err)
	}
	return resp, nil
}

func (c *Client) ListRecords(ctx context.Context, collection, repo string) ([]*agnostic.RepoListRecords_Record, error) {
	log.WithFields(log.Fields{
		"collection": collection,
		"repo":       repo,
	}).Info("list records")

	client, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	resp, err := agnostic.RepoListRecords(ctx, client, collection, "", 100, repo, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	return resp.Records, nil
}

func (c *Client) GetRecord(ctx context.Context, collection, repo, rkey string) (*agnostic.RepoGetRecord_Output, error) {
	log.WithFields(log.Fields{
		"collection": collection,
		"repo":       repo,
		"rkey":       rkey,
	}).Info("get record")

	client, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	resp, err := agnostic.RepoGetRecord(ctx, client, "", collection, repo, rkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	return resp, nil
}

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

// response wrappers with identity for easier navigation of views

type RepoWithIdentity struct {
	Identity *identity.Identity
	Repo     *comatproto.RepoDescribeRepo_Output
}

type RecordsWithIdentity struct {
	Identity *identity.Identity
	Records  []*agnostic.RepoListRecords_Record
}

type RecordWithIdentity struct {
	Identity *identity.Identity
	Record   *agnostic.RepoGetRecord_Output
}

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

func (c *Client) GetIdentity(ctx context.Context, raw string) (*identity.Identity, error) {
	id, err := syntax.ParseAtIdentifier(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse identifier: %w", err)
	}
	idd, err := c.dir.Lookup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup identifier: %w", err)
	}
	log.WithFields(log.Fields{
		"handle": idd.Handle,
		"DID":    idd.DID,
		"PDS":    idd.PDSEndpoint(),
	}).Info("identifier resolved")
	return idd, nil
}

func (c *Client) withIdentifier(ctx context.Context, raw string) (*atclient.APIClient, error) {
	idd, err := c.GetIdentity(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup identifier: %w", err)
	}
	return atclient.NewAPIClient(idd.PDSEndpoint()), nil
}

func (c *Client) GetRepo(ctx context.Context, repo string) (*RepoWithIdentity, error) {
	id, err := c.GetIdentity(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup identifier: %w", err)
	}

	client, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	log.WithFields(log.Fields{
		"client_host": client.Host,
		"repo":        repo,
		"pds":         id.PDSEndpoint(),
	}).Info("describe repo")

	// TODO: download repo as car
	// https://github.com/bluesky-social/cookbook/blob/main/go-repo-export/main.go#L46
	resp, err := comatproto.RepoDescribeRepo(ctx, client, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to describe repo: %w", err)
	}
	return &RepoWithIdentity{
		Identity: id,
		Repo:     resp,
	}, nil
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

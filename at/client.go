package at

import (
	"context"
	"encoding/json"
	"fmt"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	log "github.com/sirupsen/logrus"

	"github.com/bluesky-social/indigo/api/agnostic"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type Record struct {
	Uri   string
	Cid   string
	Value *json.RawMessage
}

func (r *Record) Collection() string {
	uri, err := syntax.ParseATURI(r.Uri)
	if err != nil {
		return ""
	}
	return uri.Collection().String()
}

func NewRecordFromList(r *agnostic.RepoListRecords_Record) *Record {
	return &Record{
		Uri:   r.Uri,
		Cid:   r.Cid,
		Value: r.Value,
	}
}

func NewRecordFromGet(r *agnostic.RepoGetRecord_Output) *Record {
	cid := ""
	if r.Cid != nil {
		cid = *r.Cid
	}
	return &Record{
		Uri:   r.Uri,
		Cid:   cid,
		Value: r.Value,
	}
}

type RepoWithIdentity struct {
	Identity *identity.Identity
	Repo     *comatproto.RepoDescribeRepo_Output
}

type RecordsWithIdentity struct {
	Identity *identity.Identity
	Records  []*Record
}

func (r *RecordsWithIdentity) Collection() string {
	if len(r.Records) == 0 {
		return ""
	}
	return r.Records[0].Collection()
}

type RecordWithIdentity struct {
	Identity *identity.Identity
	Record   *Record
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

func (c *Client) withIdentifier(ctx context.Context, raw string) (*atclient.APIClient, *identity.Identity, error) {
	idd, err := c.GetIdentity(ctx, raw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup identifier: %w", err)
	}
	return atclient.NewAPIClient(idd.PDSEndpoint()), idd, nil
}

func (c *Client) GetRepo(ctx context.Context, repo string) (*RepoWithIdentity, error) {
	client, id, err := c.withIdentifier(ctx, repo)
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

func (c *Client) ListRecords(ctx context.Context, collection, repo string) (*RecordsWithIdentity, error) {
	log.WithFields(log.Fields{
		"collection": collection,
		"repo":       repo,
	}).Info("list records")

	client, id, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	resp, err := agnostic.RepoListRecords(ctx, client, collection, "", 100, repo, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	records := make([]*Record, len(resp.Records))
	for i, r := range resp.Records {
		records[i] = NewRecordFromList(r)
	}

	return &RecordsWithIdentity{
		Identity: id,
		Records:  records,
	}, nil
}

func (c *Client) GetRecord(ctx context.Context, collection, repo, rkey string) (*RecordWithIdentity, error) {
	log.WithFields(log.Fields{
		"collection": collection,
		"repo":       repo,
		"rkey":       rkey,
	}).Info("get record")

	client, id, err := c.withIdentifier(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get client with identifier: %w", err)
	}

	resp, err := agnostic.RepoGetRecord(ctx, client, "", collection, repo, rkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	return &RecordWithIdentity{
		Identity: id,
		Record:   NewRecordFromGet(resp),
	}, nil
}

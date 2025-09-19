# InceptionDB Go Client User Guide

This guide explains how to work with the `github.com/inceptiondb/go-inceptiondb` package to talk to the [InceptionDB](https://inceptiondb.io) HTTP API. Every client method is documented with real examples that were executed against the public demo instance available at `https://inceptiondb.io`.

> **Note:** The public database is reset on a regular basis. Output values in the examples can change over time, but each request and payload shown here was validated against the service in September 2025.

## Requirements and installation

1. **Go 1.20 or newer.**
2. Add the client to your module:

   ```bash
   go get github.com/inceptiondb/go-inceptiondb
   ```

3. Import the package in your Go code:

   ```go
   import "github.com/inceptiondb/go-inceptiondb"
   ```

## Creating and configuring a client

### `NewClient`

```go
func NewClient(baseURL string, opts ...Option) (*Client, error)
```

Creates a client instance that targets the provided base URL. The function validates that the URL contains both the scheme (`http` or `https`) and the host.

```go
ctx := context.Background()
client, err := inceptiondb.NewClient("https://inceptiondb.io")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Client ready for", client)
```

### `Option` and `WithHTTPClient`

Options let you customize how the client behaves when it is created.

```go
type Option func(*Client)
```

The package currently ships with the following option:

```go
func WithHTTPClient(h *http.Client) Option
```

Use it to inject a custom `http.Client` (for instance to add timeouts, authentication, or metrics):

```go
httpClient := &http.Client{Timeout: 10 * time.Second}
client, err := inceptiondb.NewClient(
    "https://inceptiondb.io",
    inceptiondb.WithHTTPClient(httpClient),
)
```

If you do not provide an HTTP client, `http.DefaultClient` is used by default.

## Working with collections

### `ListCollections`

```go
func (c *Client) ListCollections(ctx context.Context) ([]Collection, error)
```

Returns the list of available collections and their metadata.

```go
collections, err := client.ListCollections(ctx)
if err != nil {
    log.Fatal(err)
}
for _, col := range collections {
    fmt.Printf("%s → %d documents, %d indexes\n", col.Name, col.Total, col.Indexes)
}
```

Sample output:

```
items → 2 documents, 0 indexes
multivaluated → 3 documents, 1 indexes
```

### `CreateCollection`

```go
func (c *Client) CreateCollection(ctx context.Context, req *CreateCollectionRequest) (*Collection, error)
```

Creates a new collection. `CreateCollectionRequest` accepts the collection name and an optional map of default values that will be applied to each inserted document.

```go
uniqueName := fmt.Sprintf("manual-%d", time.Now().UnixNano())
created, err := client.CreateCollection(ctx, &inceptiondb.CreateCollectionRequest{
    Name:     uniqueName,
    Defaults: map[string]any{"id": "uuid()"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Collection %s created with defaults %v\n", created.Name, created.Defaults)
```

Real example:

```
Collection example-e5f03c8a-bb82-43d0-9f19-812850c94ab1 created with defaults map[id:uuid()]
```

### `GetCollection`

```go
func (c *Client) GetCollection(ctx context.Context, collection string) (*Collection, error)
```

Fetches the metadata for a specific collection.

```go
col, err := client.GetCollection(ctx, "items")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%s has %d documents\n", col.Name, col.Total)
```

### `DropCollection`

```go
func (c *Client) DropCollection(ctx context.Context, collection string) error
```

Deletes the collection and all of its data. Handy for cleaning up temporary collections created during tests.

```go
if err := client.DropCollection(ctx, created.Name); err != nil {
    log.Fatal(err)
}
```

### `SetDefaults`

```go
func (c *Client) SetDefaults(ctx context.Context, collection string, defaults map[string]any) (map[string]any, error)
```

Sets the default document values that will be automatically added to inserted documents. The method returns the effective defaults stored by the server.

```go
newDefaults, err := client.SetDefaults(ctx, collectionName, map[string]any{
    "status": "draft",
    "tags":   []string{},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Active defaults:", newDefaults)
```

Real response:

```
Active defaults: map[id:uuid() status:draft tags:[]]
```

### `Size`

```go
func (c *Client) Size(ctx context.Context, collection string) (map[string]any, error)
```

Retrieves experimental usage statistics (for example disk and memory usage) for a collection.

```go
stats, err := client.Size(ctx, "items")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Approximate usage:", stats)
```

Example output: `map[disk:35417 memory:296]`.

## Index management

### Related types

- `Index` describes an existing index. Additional fields that depend on the index type are stored inside `Options`.
- `CreateIndexRequest` represents the payload used to create a new index. Its options are flattened when the request is sent.

### `ListIndexes`

```go
func (c *Client) ListIndexes(ctx context.Context, collection string) ([]Index, error)
```

Lists the indexes registered in a collection.

```go
indexes, err := client.ListIndexes(ctx, "multivaluated")
if err != nil {
    log.Fatal(err)
}
for _, idx := range indexes {
    fmt.Printf("%s (%s) → %v\n", idx.Name, idx.Type, idx.Options)
}
```

Real example: `colors (map) → map[field:colors sparse:false]`.

### `CreateIndex`

```go
func (c *Client) CreateIndex(ctx context.Context, collection string, req *CreateIndexRequest) (*Index, error)
```

Registers a new index. The accepted options depend on the index type; for example, a simple B-tree index only needs the field name:

```go
idx, err := client.CreateIndex(ctx, collectionName, &inceptiondb.CreateIndexRequest{
    Name: "by-id",
    Type: "btree",
    Options: map[string]any{
        "field": "id",
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Index %s created with type %s\n", idx.Name, idx.Type)
```

Sample response: `Index by-id created with type btree`.

### `GetIndex`

```go
func (c *Client) GetIndex(ctx context.Context, collection, name string) (*Index, error)
```

Fetches the configuration for a specific index.

```go
idx, err := client.GetIndex(ctx, collectionName, "by-id")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%s → options %v\n", idx.Name, idx.Options)
```

### `DropIndex`

```go
func (c *Client) DropIndex(ctx context.Context, collection, name string) error
```

Removes an existing index.

```go
if err := client.DropIndex(ctx, collectionName, "by-id"); err != nil {
    log.Fatal(err)
}
```

## Document operations

### Request types

The `find`, `patch`, and `remove` operations share the fields defined by `QueryOptions`:

```go
type QueryOptions struct {
    Mode    string
    Index   string
    Filter  map[string]any
    Skip    int64
    Limit   int64
    Reverse bool
    From    map[string]any
    To      map[string]any
    Value   string
}
```

- `FindRequest` only embeds `QueryOptions`.
- `PatchRequest` adds a `Patch any` field with the changes to apply.
- `RemoveRequest` also reuses `QueryOptions` as-is.

### Streaming inserts: `InsertStream`

```go
func (c *Client) InsertStream(ctx context.Context, collection string, reader io.Reader) (*JSONStream, error)
```

Sends a JSON Lines payload to the insert endpoint and returns a `JSONStream` with the inserted documents (including generated fields such as `id`).

```go
payload := strings.NewReader(`{"title":"Primer artículo","category":"guides"}
{"title":"Segundo artículo","category":"guides","tags":["beta"]}`)
stream, err := client.InsertStream(ctx, collectionName, payload)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()
var doc map[string]any
for {
    if err := stream.Next(&doc); err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        log.Fatal(err)
    }
    fmt.Println("Inserted:", doc)
}
```

Real output:

```
Inserted: map[category:guides id:310d97a7-5b46-4313-9f8c-0f7ef1acf493 title:Primer artículo]
Inserted: map[category:guides id:8a34e5a2-1f72-45bb-b29a-f7d6ce4d16fa tags:[beta] title:Segundo artículo]
```

### Convenience inserts: `InsertDocuments`

```go
func (c *Client) InsertDocuments(ctx context.Context, collection string, documents ...any) (*JSONStream, error)
```

Automatically serializes the provided documents to JSON Lines before delegating to `InsertStream`.

```go
stream, err := client.InsertDocuments(ctx, collectionName,
    map[string]any{"title": "Tercer artículo"},
    struct {
        Title string   `json:"title"`
        Tags  []string `json:"tags"`
    }{Title: "Cuarto artículo", Tags: []string{"nuevo"}},
)
if err != nil {
    log.Fatal(err)
}
```

### Queries: `Find`

```go
func (c *Client) Find(ctx context.Context, collection string, req *FindRequest) (*JSONStream, error)
```

Executes a query and streams the matching documents back as a `JSONStream`.

```go
stream, err := client.Find(ctx, collectionName, &inceptiondb.FindRequest{
    QueryOptions: inceptiondb.QueryOptions{
        Filter: map[string]any{"category": "guides"},
        Limit:  1,
    },
})
if err != nil {
    log.Fatal(err)
}
```

Real response (first document):

```
{"category":"guides","id":"310d97a7-5b46-4313-9f8c-0f7ef1acf493","title":"Primer artículo"}
```

### Partial updates: `Patch`

```go
func (c *Client) Patch(ctx context.Context, collection string, req *PatchRequest) (*JSONStream, error)
```

Applies partial modifications to the documents that match the filter and streams each updated document.

```go
stream, err := client.Patch(ctx, collectionName, &inceptiondb.PatchRequest{
    QueryOptions: inceptiondb.QueryOptions{
        Filter: map[string]any{"title": "Primer artículo"},
    },
    Patch: map[string]any{"status": "published"},
})
if err != nil {
    log.Fatal(err)
}
```

Real example:

```
{"category":"guides","id":"310d97a7-5b46-4313-9f8c-0f7ef1acf493","status":"published","title":"Primer artículo"}
```

### Deletions: `Remove`

```go
func (c *Client) Remove(ctx context.Context, collection string, req *RemoveRequest) (*JSONStream, error)
```

Deletes the matching documents and streams each removed document back to the caller.

```go
stream, err := client.Remove(ctx, collectionName, &inceptiondb.RemoveRequest{
    QueryOptions: inceptiondb.QueryOptions{
        Filter: map[string]any{"category": "guides"},
    },
})
if err != nil {
    log.Fatal(err)
}
```

Real output (removed document):

```
{"category":"guides","id":"8a34e5a2-1f72-45bb-b29a-f7d6ce4d16fa","tags":["beta"],"title":"Segundo artículo"}
```

## Working with JSON streams (`JSONStream`)

Operations that return many rows stream data back as JSON Lines. The `JSONStream` type wraps the HTTP response so you can consume it incrementally.

### Available methods

- `Close() error`: releases the underlying resource. It is called automatically once `io.EOF` is reached.
- `Next(v any) error`: decodes the next element into `v`. Returns `io.EOF` when the stream ends.
- `StatusCode() int`: exposes the HTTP status code received from the server.

`JSONStream` also works together with the helper `ErrStopIteration` value, which lets you stop iteration early without treating it as an error.

### `Iterate`

```go
func Iterate[T any](s *JSONStream, fn func(*T) error) error
```

A generic helper that iterates over the stream, decodes each element into a value of type `T`, and executes the callback `fn`. You can stop the loop early by returning `ErrStopIteration` from the callback.

```go
err := inceptiondb.Iterate(stream, func(article *Article) error {
    fmt.Println("Title:", article.Title)
    if article.Title == "Primer artículo" {
        return inceptiondb.ErrStopIteration
    }
    return nil
})
if err != nil {
    log.Fatal(err)
}
```

## Error handling

When the API responds with a status code `>= 400`, the client returns an error of type `*inceptiondb.Error`, which exposes:

- `StatusCode`
- `Message`
- `Description`
- `Body` (the full server response)

```go
_, err := client.GetCollection(ctx, "non-existent-collection")
if err != nil {
    if apiErr, ok := err.(*inceptiondb.Error); ok {
        fmt.Printf("Error %d: %s (%s)\n", apiErr.StatusCode, apiErr.Message, apiErr.Description)
    } else {
        log.Fatal(err)
    }
}
```

Real HTTP response example:

```
HTTP/1.1 404 Not Found
{"error":{"description":"Unexpected error","message":"collection not found"}}
```

## Cleanup

Remember to delete any temporary collections created during your tests with `DropCollection` so the shared instance remains tidy.

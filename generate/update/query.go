package update

import (
	"errors"
	"sync"

	"github.com/safe-waters/docker-lock/generate/parse"
	"github.com/safe-waters/docker-lock/registry"
)

// QueryExecutor queries for digests, caching results of past queries
type QueryExecutor struct {
	WrapperManager *registry.WrapperManager
	cache          map[parse.Image]*QueryResult
	mutex          *sync.RWMutex
}

// IQueryExecutor provides an interface for QueryExecutor's exported methods,
// which are used by ImageDigestUpdaters.
type IQueryExecutor interface {
	QueryRegistry(image *parse.Image) *QueryResult
}

// QueryResult contains an image with the updated digest and any error
// associated with querying for the digest.
type QueryResult struct {
	*parse.Image
	Err error
}

// NewQueryExecutor returns a QueryExecutor after validating its fields.
func NewQueryExecutor(
	wrapperManager *registry.WrapperManager,
) (*QueryExecutor, error) {
	if wrapperManager == nil {
		return nil, errors.New("wrapperManager cannot be nil")
	}

	var mutex sync.RWMutex

	cache := map[parse.Image]*QueryResult{}

	return &QueryExecutor{
		WrapperManager: wrapperManager,
		cache:          cache,
		mutex:          &mutex,
	}, nil
}

// QueryRegistry queries the appropriate registry for a digest.
func (q *QueryExecutor) QueryRegistry(image *parse.Image) *QueryResult {
	q.mutex.RLock()

	queryResult, ok := q.cache[*image]
	if ok {
		q.mutex.RUnlock()

		return queryResult
	}

	q.mutex.RUnlock()

	q.mutex.Lock()
	defer q.mutex.Unlock()

	wrapper := q.WrapperManager.Wrapper(image.Name)

	digest, err := wrapper.Digest(image.Name, image.Tag)

	queryResult = &QueryResult{
		Image: &parse.Image{
			Name:   image.Name,
			Tag:    image.Tag,
			Digest: digest,
		},
		Err: err,
	}
	q.cache[*image] = queryResult

	return queryResult
}

package ingestors

/*

Load latest releases from the PyPI XML-RPC endpoint
(https://warehouse.pypa.io/api-reference/xml-rpc.html#changelog-since-with-ids-false)
and return ingestion results.

The XML-RPC endpoint is mostly deprecated. Methods that have to do with
mirroring, like the "changelog" endpoint we're using, are still supported
as of 2023-09-05.

This feed continually delivers new information on changes to the PyPI database.
We're most concerned with actions that:

* Add a release
* Yank (remove from listings but still allow downloads) a release
* Remove (remove from listings and prevent downloads) a release

Once we see one of these actions, we create an ingestion event for the release.

*/

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/librariesio/depper/data"
	log "github.com/sirupsen/logrus"

	"github.com/kolo/xmlrpc"
)

const pyPiRpcServer = "https://pypi.org/pypi"

type PyPiXmlRpc struct {
	LatestRun time.Time
}

func NewPyPiXmlRpc() *PyPiXmlRpc {
	return &PyPiXmlRpc{}
}

func (ingestor *PyPiXmlRpc) Name() string {
	return "pypiXmlRpc"
}

func (ingestor *PyPiXmlRpc) Schedule() string {
	return "@every 5m"
}

// Structured storage for the tuple returned by the xmlrpc client
type PyPiXmlRpcResponse struct {
	Name      string
	Version   string
	Timestamp int64
	Action    string
}

// Return trhe if this response is an ingestable action
func (response *PyPiXmlRpcResponse) IsIngestionAction() bool {
	switch response.Action {
	case "new release", "yank release", "remove release", "unyank release":
		return true
	}

	return false
}

// Get the PackageVersion struct for this response
func (response *PyPiXmlRpcResponse) GetPackageVersion() data.PackageVersion {
	createdAt := time.Unix(response.Timestamp, 0)
	discoveryLag := time.Since(createdAt)

	return data.PackageVersion{
		Platform:     "pypi",
		Name:         response.Name,
		Version:      response.Version,
		CreatedAt:    createdAt,
		DiscoveryLag: discoveryLag,
	}
}

// Validate and then create a PyPiXmlRpcResponse from a log struct
func createResponseStruct(log []any) (*PyPiXmlRpcResponse, error) {
	if _, ok := log[0].(string); !ok {
		return nil, errors.New("package name is not a string")
	}

	if _, ok := log[1].(string); !ok {
		return nil, errors.New("version is not a string")
	}

	if _, ok := log[2].(int64); !ok {
		return nil, errors.New("created at date is not an int64 number")
	}

	if _, ok := log[3].(string); !ok {
		return nil, errors.New("action is not a string")
	}

	return &PyPiXmlRpcResponse{log[0].(string), log[1].(string), log[2].(int64), log[3].(string)}, nil
}

// Retrieve a list of [name, version, timestamp, action] since the given since. All since timestamps
// are UTC values. The argument is a UTC integer seconds since the epoch (e.g., the timestamp method
// to a datetime.datetime object).
// calls "changelog(since, with_ids=False)" RPC
func (ingestor *PyPiXmlRpc) Ingest() []data.PackageVersion {
	// An array of interface arrays. Each log entry contains:
	// * name(string), version(string), timestamp(int64), action(string)
	// These are converted to PyPiXmlRpcResponse structs
	var response [][]any
	var results []data.PackageVersion

	// Get the current bookmark
	since, err := getBookmarkTime(ingestor, time.Now().AddDate(0, 0, -1))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	client, _ := xmlrpc.NewClient(pyPiRpcServer, nil)
	defer client.Close()

	err = client.Call("changelog_since_serial", int(since.Unix()), &response)
	if err != nil {
		if strings.Contains(fmt.Sprint(err), "illegal character code") {
			// If we encounter illegal characters in the XML, ignore this page and treat it like an empty response.
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Info(fmt.Sprintf("Skipping page from timestamp %d", since.Unix()))
			response = [][]any{}
		} else {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	for _, log := range response {
		responseStruct, _ := createResponseStruct(log)

		if responseStruct != nil {
			if responseStruct.IsIngestionAction() {
				results = append(results, responseStruct.GetPackageVersion())
			}
		}
	}

	if _, err := setBookmarkTime(ingestor, time.Now()); err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	return results
}

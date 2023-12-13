package ingestors

/*

Load latest releases from the PyPI XML-RPC endpoint
(https://warehouse.pypa.io/api-reference/xml-rpc.html#changelog-since-with-ids-false)
and return ingestion results.

The XML-RPC endpoint is mostly deprecated. We previously used the "changelog" method,
but it was deprecated so we switched to "changelog_since_serial" on 2023-12-08.

This feed continually delivers new information on changes to the PyPI database.
We're most concerned with actions that:

* Add a release
* Yank (remove from listings but still allow downloads) a release
* Unyank (re-added to listings) a release
* Remove (remove from listings and prevent downloads) a release

Once we see one of these actions, we create an ingestion event for the release.

*/

import (
	"errors"
	"fmt"
	"strconv"
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
	Serial    int64
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
	var name, version, action string
	var createdAt, serial int64
	var ok bool

	if name, ok = log[0].(string); !ok {
		return nil, errors.New("package name is not a string")
	}

	if version, ok = log[1].(string); !ok {
		return nil, errors.New("version is not a string")
	}

	if createdAt, ok = log[2].(int64); !ok {
		return nil, errors.New("created at date is not an int64 number")
	}

	if action, ok = log[3].(string); !ok {
		return nil, errors.New("action is not a string")
	}

	if serial, ok = log[4].(int64); !ok {
		return nil, errors.New("serial is not an int")
	}

	return &PyPiXmlRpcResponse{name, version, createdAt, action, serial}, nil
}

// Retrieve a list of [name, version, timestamp, action] since the given since. All since timestamps
// are UTC values. The argument is a UTC integer seconds since the epoch (e.g., the timestamp method
// to a datetime.datetime object).
// calls "changelog(since, with_ids=False)" RPC
func (ingestor *PyPiXmlRpc) Ingest() []data.PackageVersion {
	// An array of interface arrays. Each log entry contains:
	// * name(string), version(string), timestamp(int64), action(string), serial(int)
	// These are converted to PyPiXmlRpcResponse structs
	var response [][]any
	var results []data.PackageVersion

	// Get the current bookmark
	bookmark, err := getBookmark(ingestor, "")
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	// Bookmark type migration: the old bookmarks were ISO8601 timestamps, which were 25 chars longs,
	// but we're keeping the serial ints as bookmarks now, which are 8 chars long currently,
	// so keep this around just to migrate the bookmark in redis. Safe to remove at some point in future.
	var serial int64
	if len(bookmark) > 20 {
		serial = 0
	} else {
		serial, err = strconv.ParseInt(bookmark, 10, 64)
		if err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(fmt.Sprintf("Couldn't convert bookmark to serial: %s", err))
		}
	}

	client, _ := xmlrpc.NewClient(pyPiRpcServer, nil)
	defer client.Close()

	if serial == 0 {
		serial, err = getLastSerial(client)
		if err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Info("Couldn't fetch last serial")
		} else {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Info("Fetched default serial: ", serial)
		}
	}

	err = client.Call("changelog_since_serial", serial, &response)
	if err != nil {
		if strings.Contains(fmt.Sprint(err), "illegal character code") {
			// If we encounter illegal characters in the XML, ignore this page and treat it like an empty response.
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Error(err)
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Info(fmt.Sprintf("Skipping page from serial %d", serial))
			response = [][]any{}
		} else {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	for _, changelogRow := range response {
		// NOTE: we're swallowing the error here, because e.g. some rows won't have a "version" field
		// and type-casting will fail. But the error can be useful for debugging when needed.
		responseStruct, _ := createResponseStruct(changelogRow)

		if responseStruct != nil {
			if responseStruct.IsIngestionAction() {
				results = append(results, responseStruct.GetPackageVersion())
				if responseStruct.Serial > serial {
					serial = responseStruct.Serial
				}
			}
		}
	}

	if _, err := setBookmark(ingestor, strconv.Itoa(int(serial))); err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	return results
}

// Serials for events from pypa are ints (e.g. 20972215).
func getLastSerial(client *xmlrpc.Client) (int64, error) {
	var serial int64
	var args any
	err := client.Call("changelog_last_serial", args, &serial)
	if err != nil {
		return 0, err
	}
	return serial, nil
}

package ingestors

import (
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

// changelog(since, with_ids=False)
// 	Retrieve a list of [name, version, timestamp, action] since the given since. All since timestamps
//  are UTC values. The argument is a UTC integer seconds since the epoch (e.g., the timestamp method
//  to a datetime.datetime object).
func (ingestor *PyPiXmlRpc) Ingest() []data.PackageVersion {
	// An array of interface arrays. Each log entry contains:
	// name(string), version(string), timestamp(int64), action(string)
	var response [][]interface{}
	var results []data.PackageVersion

	// Get the current bookmark
	since, err := getBookmarkTime(ingestor, time.Now().AddDate(0, 0, -1))
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	client, _ := xmlrpc.NewClient(pyPiRpcServer, nil)
	defer client.Close()

	err = client.Call("changelog", int(since.Unix()), &response)
	if err != nil {
		log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
	}

	for _, log := range response {
		//fmt.Printf("%s %s %v %s\n", log[0], log[1], log[2], log[3])
		switch log[3].(string) {
		case "new release", "yank release", "remove release":
			results = append(results,
				data.PackageVersion{
					Platform:  "pypi",
					Name:      log[0].(string),
					Version:   log[1].(string),
					CreatedAt: time.Unix(log[2].(int64), 0),
				})
		}
	}

	if len(results) > 0 {
		if _, err := setBookmarkTime(ingestor, data.MaxCreatedAt(results)); err != nil {
			log.WithFields(log.Fields{"ingestor": ingestor.Name()}).Fatal(err)
		}
	}

	return results
}

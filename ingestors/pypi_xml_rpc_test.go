package ingestors

import (
	"testing"
	"time"
)

type ingestionTest struct {
	action   string
	expected bool
}

var ingestionTests = []ingestionTest{
	{"new release", true},
	{"yank release", true},
	{"unyank release", true},
	{"remove release", true},
	{"something else", false},
}

func TestPyPiXmlRpcResponse_IsIngestionAction(t *testing.T) {
	for _, test := range ingestionTests {
		response := PyPiXmlRpcResponse{
			"what", "ever", 1, test.action, 1234,
		}

		if response.IsIngestionAction() != test.expected {
			t.Errorf("for %s, got %t, wanted %t", test.action, response.IsIngestionAction(), test.expected)
		}
	}
}

func TestPyPiXmlRpcResponse_GetPackageVersion(t *testing.T) {
	// get a number a set number of seconds ago
	// this is the fastest way to strip off microseconds
	now := time.Unix(time.Now().Unix(), 0)
	fiveSecondsAgoDuration, err := time.ParseDuration("-5s")
	if err != nil {
		t.Error("Unable to parse test duration string")
	}

	fiveSecondsAgo := now.Add(fiveSecondsAgoDuration)

	response := PyPiXmlRpcResponse{
		"name", "version", fiveSecondsAgo.Unix(), "whatever", 1235,
	}

	packageVersion := response.GetPackageVersion()

	// check if this value is within one second
	discoveryLagVsFiveSecondDuration := packageVersion.DiscoveryLag + fiveSecondsAgoDuration

	// 1 second = 1000000000 microseconds
	if discoveryLagVsFiveSecondDuration >= 1000000000 {
		t.Errorf("DiscoveryLag is not correct, %d is not within 1 second", discoveryLagVsFiveSecondDuration)
	}

	if packageVersion.Name != "name" {
		t.Error("name is not correct")
	}
	if packageVersion.Version != "version" {
		t.Error("version is not correct")
	}
	if packageVersion.CreatedAt != fiveSecondsAgo {
		t.Errorf("CreatedAt is not correct, expected %#v, got %#v", fiveSecondsAgo, packageVersion.CreatedAt)
	}
}

type logResponseTest struct {
	log     []any
	message string
}

var logResponsesFailures = []logResponseTest{
	{[]any{nil, "1.0.0", int64(100), "action", 1000}, "package name is not a string"},
	{[]any{"name", nil, int64(100), "action", 1001}, "version is not a string"},
	{[]any{"name", "1.0.0", nil, "action", 1002}, "created at date is not an int64 number"},
	{[]any{"name", "1.0.0", int64(100), nil, 1003}, "action is not a string"},
}

func TestCreateResponseStruct_Failure(t *testing.T) {
	for _, test := range logResponsesFailures {
		_, err := createResponseStruct(test.log)
		if err.Error() != test.message {
			t.Errorf("expect message %s, got %s", test.message, err.Error())
		}
	}
}

func TestCreateResponseStruct_Success(t *testing.T) {
	response, err := createResponseStruct([]any{"name", "1.0.0", int64(100), "action", int64(1000)})

	if err != nil {
		t.Errorf("expected err to be nil, got %#v", err)
	}

	if response.Name != "name" {
		t.Errorf("expected name to equal %s, got %s", "name", response.Name)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version to equal %s, got %s", "1.0.0", response.Version)
	}

	if response.Timestamp != int64(100) {
		t.Errorf("expected timestamp to equal %d, got %d", 100, response.Timestamp)
	}

	if response.Action != "action" {
		t.Errorf("expected action to equal %s, got %s", "action", response.Action)
	}

	if response.Serial != 1000 {
		t.Errorf("expected serial to equal %d, got %d", 1000, response.Serial)
	}
}

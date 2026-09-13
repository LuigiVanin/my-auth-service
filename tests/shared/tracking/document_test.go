package tracking_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"auth_service/shared/tracking"

	"github.com/stretchr/testify/assert"
)

func at(year int, month int) time.Time {
	return time.Date(year, time.Month(month), 15, 12, 0, 0, 0, time.UTC)
}

func signup(t *testing.T, document string, appId string, moment time.Time) tracking.Document {
	t.Helper()

	written, err := tracking.RecordSignup(json.RawMessage(document), appId, moment)

	assert.NoError(t, err)

	parsed, err := tracking.Parse(written)

	assert.NoError(t, err)

	return parsed
}

func login(t *testing.T, document string, event tracking.LoginEvent) tracking.Document {
	t.Helper()

	written, err := tracking.RecordLogin(json.RawMessage(document), event)

	assert.NoError(t, err)

	parsed, err := tracking.Parse(written)

	assert.NoError(t, err)

	return parsed
}

// An empty column is what a row written before the migration hands back, and it
// has to be the same as an empty document rather than an error.
func TestAnEmptyOrNullColumnStartsAnEmptyDocument(t *testing.T) {
	for _, document := range []string{``, `null`, `{}`} {
		parsed := signup(t, document, "app-1", at(2026, 9))

		assert.Equal(t, 1, parsed.Signup.Periods["2026-09"].Total, "from `%s`", document)
	}
}

func TestSignupCountsTheMonthAndTheApp(t *testing.T) {
	parsed := signup(t, `{}`, "app-1", at(2026, 9))

	assert.Equal(t, 1, parsed.Signup.Periods["2026-09"].Total)
	assert.Equal(t, map[string]int{"app-1": 1}, parsed.Signup.Periods["2026-09"].Apps)
}

func TestSignupIncrementsAnExistingMonth(t *testing.T) {
	stored := `{"signup":{"periods":{"2026-09":{"total":4,"apps":{"app-1":4}}}}}`

	parsed := signup(t, stored, "app-1", at(2026, 9))

	assert.Equal(t, 5, parsed.Signup.Periods["2026-09"].Total)
	assert.Equal(t, 5, parsed.Signup.Periods["2026-09"].Apps["app-1"])
}

// The whole point of the app breakdown: two apps of the same pool are two entries
// inside one month, and the total is the sum.
func TestSignupSplitsBetweenAppsInsideOneMonth(t *testing.T) {
	stored := `{"signup":{"periods":{"2026-09":{"total":4,"apps":{"app-1":4}}}}}`

	parsed := signup(t, stored, "app-2", at(2026, 9))

	assert.Equal(t, 5, parsed.Signup.Periods["2026-09"].Total)
	assert.Equal(t, map[string]int{"app-1": 4, "app-2": 1}, parsed.Signup.Periods["2026-09"].Apps)
}

func TestSignupOpensANewMonthWithoutTouchingTheOldOne(t *testing.T) {
	stored := `{"signup":{"periods":{"2026-08":{"total":4,"apps":{"app-1":4}}}}}`

	parsed := signup(t, stored, "app-1", at(2026, 9))

	assert.Equal(t, 4, parsed.Signup.Periods["2026-08"].Total)
	assert.Equal(t, 1, parsed.Signup.Periods["2026-09"].Total)
}

// The eleventh month evicts the oldest, and "oldest" is decided by sorting the
// keys - which is why the period is formatted YYYY-MM.
func TestTheEleventhMonthEvictsTheOldest(t *testing.T) {
	document := json.RawMessage(`{}`)

	// Ten months of 2026, January through October.
	for month := 1; month <= tracking.Limit; month++ {
		written, err := tracking.RecordSignup(document, "app-1", at(2026, month))
		assert.NoError(t, err)
		document = written
	}

	parsed, err := tracking.Parse(document)

	assert.NoError(t, err)
	assert.Len(t, parsed.Signup.Periods, tracking.Limit)
	assert.Contains(t, parsed.Signup.Periods, "2026-01")

	written, err := tracking.RecordSignup(document, "app-1", at(2026, 11))

	assert.NoError(t, err)

	parsed, err = tracking.Parse(written)

	assert.NoError(t, err)
	assert.Len(t, parsed.Signup.Periods, tracking.Limit)
	assert.NotContains(t, parsed.Signup.Periods, "2026-01", "the oldest month has to be the one dropped")
	assert.Contains(t, parsed.Signup.Periods, "2026-11")
}

// A month arriving out of order must not evict a newer one.
func TestAnOlderMonthDoesNotEvictANewerOne(t *testing.T) {
	document := json.RawMessage(`{}`)

	for month := 2; month <= tracking.Limit+1; month++ {
		written, err := tracking.RecordSignup(document, "app-1", at(2026, month))
		assert.NoError(t, err)
		document = written
	}

	written, err := tracking.RecordSignup(document, "app-1", at(2026, 1))

	assert.NoError(t, err)

	parsed, err := tracking.Parse(written)

	assert.NoError(t, err)
	assert.Len(t, parsed.Signup.Periods, tracking.Limit)
	assert.NotContains(t, parsed.Signup.Periods, "2026-01", "the month just written is itself the oldest")
	assert.Contains(t, parsed.Signup.Periods, "2026-11")
}

// The period is keyed in UTC, so the bucket does not depend on the timezone the
// process happens to run in.
func TestThePeriodIsUtc(t *testing.T) {
	// 23:30 on the last day of September in UTC+3 is still September in UTC.
	moment := time.Date(2026, 9, 30, 23, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))

	assert.Equal(t, "2026-09", tracking.Period(moment))
}

func TestLoginPushesNewestFirst(t *testing.T) {
	stored := `{"login":{"events":[{"app_id":"app-1","at":"2026-09-01T00:00:00Z"}]}}`

	parsed := login(t, stored, tracking.LoginEvent{AppId: "app-2", At: at(2026, 9)})

	assert.Len(t, parsed.Login.Events, 2)
	assert.Equal(t, "app-2", parsed.Login.Events[0].AppId, "the newest login is the first element")
	assert.Equal(t, "app-1", parsed.Login.Events[1].AppId)
}

func TestLoginKeepsTheWholeEvent(t *testing.T) {
	moment := at(2026, 9)

	parsed := login(t, `{}`, tracking.LoginEvent{
		AppId:     "app-1",
		At:        moment,
		Ip:        "203.0.113.7",
		UserAgent: "curl/8.0",
		LoginType: "WITH_OTP",
	})

	event := parsed.Login.Events[0]

	assert.Equal(t, "app-1", event.AppId)
	assert.True(t, moment.Equal(event.At))
	assert.Equal(t, "203.0.113.7", event.Ip)
	assert.Equal(t, "curl/8.0", event.UserAgent)
	assert.Equal(t, "WITH_OTP", event.LoginType)
}

func TestTheEleventhLoginDropsTheOldest(t *testing.T) {
	document := json.RawMessage(`{}`)

	for index := 1; index <= tracking.Limit+1; index++ {
		written, err := tracking.RecordLogin(document, tracking.LoginEvent{
			AppId: fmt.Sprintf("app-%d", index),
			At:    at(2026, 9),
		})

		assert.NoError(t, err)
		document = written
	}

	parsed, err := tracking.Parse(document)

	assert.NoError(t, err)
	assert.Len(t, parsed.Login.Events, tracking.Limit)
	assert.Equal(t, "app-11", parsed.Login.Events[0].AppId, "newest first")
	assert.Equal(t, "app-2", parsed.Login.Events[tracking.Limit-1].AppId, "app-1 is the one dropped")
}

// A users pool serializes no login half and a user serializes no signup half, so
// the column of each entity says only what that entity tracks.
func TestTheUnusedHalfIsNotSerialized(t *testing.T) {
	written, err := tracking.RecordSignup(json.RawMessage(`{}`), "app-1", at(2026, 9))

	assert.NoError(t, err)
	assert.NotContains(t, string(written), "login")

	written, err = tracking.RecordLogin(json.RawMessage(`{}`), tracking.LoginEvent{AppId: "app-1"})

	assert.NoError(t, err)
	assert.NotContains(t, string(written), "signup")
}

// Recording never discards a half it does not touch: a pool that somehow carries
// both keeps both.
func TestRecordingOneTagLeavesTheOtherAlone(t *testing.T) {
	stored := `{"signup":{"periods":{"2026-09":{"total":1,"apps":{"app-1":1}}}},"login":{"events":[{"app_id":"app-1"}]}}`

	parsed := login(t, stored, tracking.LoginEvent{AppId: "app-2"})

	assert.Equal(t, 1, parsed.Signup.Periods["2026-09"].Total)
	assert.Len(t, parsed.Login.Events, 2)
}

func TestAMalformedColumnIsAnError(t *testing.T) {
	_, err := tracking.RecordSignup(json.RawMessage(`{"signup":`), "app-1", at(2026, 9))

	assert.Error(t, err)
}

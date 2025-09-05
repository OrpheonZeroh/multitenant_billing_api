Go
Skip to Main Content
Search packages or symbols

Why Gosubmenu dropdown icon
Learn
Docssubmenu dropdown icon
Packages
Communitysubmenu dropdown icon
Discover Packages
 
github.com/inngest/inngestgo

Go
inngestgo
package
module


Main
Details
checked Valid go.mod file 
checked Redistributable license 
checked Tagged version 
unchecked Stable version 
Learn more about best practices
Repository
github.com/inngest/inngestgo
Links
Open Source Insights Logo Open Source Insights

type ConfigRateLimit
 README ¶


Write durable functions in Go via the Inngest SDK.
Read the documentation and get started in minutes.


GoDoc discord twitter

inngestgo: Durable execution in Go
inngestgo allows you to create durable functions in your existing HTTP handlers or via outbound TCP connections, without managing orchestrators, state, scheduling, or new infrastructure.

It's useful if you want to build reliable software without worrying about queues, events, subscribers, workers, or other complex primitives such as concurrency, parallelism, event batching, or distributed debounce. These are all built in.

Godoc docs
Inngest docs
Features
Type safe functions, durable workflows, and steps using generics
Event stream sampling built in
Declarative flow control (concurrency, prioritization, batching, debounce, rate limiting)
Zero-infrastructure. Inngest handles orchestration and calls your functions.
Examples
The following is the bare minimum setup for a fully distributed durable workflow:

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
)

func main() {
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID: "core",
	})
	if err != nil {
		panic(err)
	}

	_, err = inngestgo.CreateFunction(
		client,
		inngestgo.FunctionOpts{
			ID: "account-created",
		},
		// Run on every api/account.created event.
		inngestgo.EventTrigger("api/account.created", nil),
		AccountCreated,
	)
	if err != nil {
		panic(err)
	}

	http.ListenAndServe(":8080", client.Serve())
}

// AccountCreated is a durable function which runs any time the "api/account.created"
// event is received by Inngest.
//
// It is invoked by Inngest, with each step being backed by Inngest's orchestrator.
// Function state is automatically managed, and persists across server restarts,
// cloud migrations, and language changes.
func AccountCreated(
	ctx context.Context,
	input inngestgo.Input[AccountCreatedEventData],
) (any, error) {
	// Sleep for a second, minute, hour, week across server restarts.
	step.Sleep(ctx, "initial-delay", time.Second)

	// Run a step which emails the user.  This automatically retries on error.
	// This returns the fully typed result of the lambda.
	result, err := step.Run(ctx, "on-user-created", func(ctx context.Context) (bool, error) {
		// Run any code inside a step.
		result, err := emails.Send(emails.Opts{})
		return result, err
	})
	if err != nil {
		// This step retried 5 times by default and permanently failed.
		return nil, err
	}
	// `result` is  fully typed from the lambda
	_ = result

	// Sample from the event stream for new events.  The function will stop
	// running and automatially resume when a matching event is found, or if
	// the timeout is reached.
	fn, err := step.WaitForEvent[FunctionCreatedEvent](
		ctx,
		"wait-for-activity",
		step.WaitForEventOpts{
			Name:    "Wait for a function to be created",
			Event:   "api/function.created",
			Timeout: time.Hour * 72,
			// Match events where the user_id is the same in the async sampled event.
			If: inngestgo.StrPtr("event.data.user_id == async.data.user_id"),
		},
	)
	if err == step.ErrEventNotReceived {
		// A function wasn't created within 3 days.  Send a follow-up email.
		step.Run(ctx, "follow-up-email", func(ctx context.Context) (any, error) {
			// ...
			return true, nil
		})
		return nil, nil
	}

	// The event returned from `step.WaitForEvent` is fully typed.
	fmt.Println(fn.Data.FunctionID)

	return nil, nil
}

// AccountCreatedEvent represents the fully defined event received when an account is created.
//
// This is shorthand for defining a new Inngest-conforming struct:
//
//	type AccountCreatedEvent struct {
//		Name      string                  `json:"name"`
//		Data      AccountCreatedEventData `json:"data"`
//		User      map[string]any          `json:"user"`
//		Timestamp int64                   `json:"ts,omitempty"`
//		Version   string                  `json:"v,omitempty"`
//	}
type AccountCreatedEvent inngestgo.GenericEvent[AccountCreatedEventData]
type AccountCreatedEventData struct {
	AccountID string
}

type FunctionCreatedEvent inngestgo.GenericEvent[FunctionCreatedEventData]
type FunctionCreatedEventData struct {
	FunctionID string
}
Expand ▾
 Documentation ¶
Index ¶
Constants
Variables
func BoolPtr(b bool) *bool
func Connect(ctx context.Context, opts ConnectOpts) (connect.WorkerConnection, error)
func CronTrigger(cron string) fn.Trigger
func DevServerURL() string
func EventTrigger(name string, expression *string) fn.Trigger
func IntPtr(i int) *int
func IsDev() bool
func NowMillis() int64
func Ptr[T any](i T) *T
func SetBasicRequestHeaders(req *http.Request)
func SetBasicResponseHeaders(w http.ResponseWriter)
func Sign(ctx context.Context, at time.Time, key, body []byte) (string, error)
func Slugify(s string) string
func StrPtr(i string) *string
func Timestamp(t time.Time) int64
func ValidateRequestSignature(ctx context.Context, sig string, signingKey string, signingKeyFallback string, ...) (bool, string, error)
func ValidateResponseSignature(ctx context.Context, sig string, key, body []byte) (bool, error)
type Client
func NewClient(opts ClientOpts) (Client, error)
type ClientOpts
type ConfigBatchEvents
type ConfigCancel
type ConfigDebounce
type ConfigPriority
type ConfigRateLimit
type ConfigSingleton
type ConfigStepConcurrency
type ConfigThrottle
type ConfigTimeouts
type ConnectOpts
type Event
type FunctionOpts
type GenericEvent
type Input
type InputCtx
type MultipleTriggers
type SDKFunction
type ServableFunction
func CreateFunction[T any](c Client, fc FunctionOpts, trigger fn.Triggerable, f SDKFunction[T]) (ServableFunction, error)
type ServeOpts
type StepError
type StreamResponse
type Trigger
Constants ¶
View Source
const (
	SDKAuthor         = "inngest"
	SDKLanguage       = "go"
	SyncKindInBand    = "in_band"
	SyncKindOutOfBand = "out_of_band"
)
View Source
const (
	// ExternalID is the field name used to reference the user's ID within your
	// systems.  This is _your_ UUID or ID for referencing the user, and allows
	// Inngest to match contacts to your users.
	ExternalID = "external_id"

	// Email is the field name used to reference the user's email.
	Email = "email"
)
View Source
const (
	HeaderKeyAuthorization      = "Authorization"
	HeaderKeyContentType        = "Content-Type"
	HeaderKeyEnv                = "X-Inngest-Env"
	HeaderKeyEventIDSeed        = "x-inngest-event-id-seed"
	HeaderKeyExpectedServerKind = "X-Inngest-Expected-Server-Kind"
	HeaderKeyNoRetry            = "X-Inngest-No-Retry"
	HeaderKeyReqVersion         = "x-inngest-req-version"
	HeaderKeyRetryAfter         = "Retry-After"
	HeaderKeySDK                = "X-Inngest-SDK"
	HeaderKeyServerKind         = "X-Inngest-Server-Kind"
	HeaderKeySignature          = "X-Inngest-Signature"
	HeaderKeySyncKind           = "x-inngest-sync-kind"
	HeaderKeyUserAgent          = "User-Agent"
)
View Source
const SDKVersion = "0.13.1"
Variables ¶
View Source
var (
	ErrTypeMismatch = fmt.Errorf("cannot invoke function with mismatched types")

	// DefaultMaxBodySize is the default maximum size read within a single incoming
	// invoke request (100MB).
	DefaultMaxBodySize = 1024 * 1024 * 100
)
View Source
var (
	ErrExpiredSignature = fmt.Errorf("expired signature")
	ErrInvalidSignature = fmt.Errorf("invalid signature")
	ErrInvalidTimestamp = fmt.Errorf("invalid timestamp")
)
View Source
var (
	HeaderValueSDK = fmt.Sprintf("%s:v%s", SDKLanguage, SDKVersion)
)
View Source
var NoRetryError = errors.NoRetryError
Re-export internal errors for users

View Source
var RetryAtError = errors.RetryAtError
Functions ¶
func BoolPtr ¶
added in v0.8.0
func BoolPtr(b bool) *bool
func Connect ¶
added in v0.8.0
func Connect(ctx context.Context, opts ConnectOpts) (connect.WorkerConnection, error)
func CronTrigger ¶
added in v0.5.0
func CronTrigger(cron string) fn.Trigger
func DevServerURL ¶
added in v0.5.0
func DevServerURL() string
DevServerURL returns the URL for the Inngest dev server. This uses the INNGEST_DEV environment variable, or defaults to 'http://127.0.0.1:8288' if unset.

func EventTrigger ¶
added in v0.5.0
func EventTrigger(name string, expression *string) fn.Trigger
func IntPtr ¶
added in v0.5.0
func IntPtr(i int) *int
func IsDev ¶
added in v0.5.0
func IsDev() bool
IsDev returns whether to use the dev server, by checking the presence of the INNGEST_DEV environment variable.

To use the dev server, set INNGEST_DEV to any non-empty value OR the URL of the development server, eg:

INNGEST_DEV=1
INNGEST_DEV=http://192.168.1.254:8288
func NowMillis ¶
added in v0.5.1
func NowMillis() int64
NowMillis returns a timestamp with millisecond precision used for the Event.Timestamp field.

func Ptr ¶
added in v0.8.0
func Ptr[T any](i T) *T
Ptr converts the given type to a pointer. Nil pointers are sometimes used for optional arguments within configuration, meaning we need pointers within struct values. This util helps.

func SetBasicRequestHeaders ¶
added in v0.5.2
func SetBasicRequestHeaders(req *http.Request)
func SetBasicResponseHeaders ¶
added in v0.5.2
func SetBasicResponseHeaders(w http.ResponseWriter)
func Sign ¶
added in v0.5.0
func Sign(ctx context.Context, at time.Time, key, body []byte) (string, error)
Sign signs a request body with the given key at the given timestamp.

func Slugify ¶
added in v0.8.0
func Slugify(s string) string
Slugify converts a string to a slug. This is only useful for replicating the legacy slugification logic for function IDs, aiding in migration to a newer SDK version.

func StrPtr ¶
added in v0.5.0
func StrPtr(i string) *string
func Timestamp ¶
func Timestamp(t time.Time) int64
Timestamp converts a go time.Time into a timestamp with millisecond precision used for the Event.Timestamp field.

func ValidateRequestSignature ¶
added in v0.7.4
func ValidateRequestSignature(
	ctx context.Context,
	sig string,
	signingKey string,
	signingKeyFallback string,
	body []byte,
	isDev bool,
) (bool, string, error)
ValidateRequestSignature ensures that the signature for the given body is signed with the given key within a given time period to prevent invalid requests or replay attacks. A signing key fallback is used if provided. Returns the correct signing key, which is useful when signing responses

func ValidateResponseSignature ¶
added in v0.7.4
func ValidateResponseSignature(ctx context.Context, sig string, key, body []byte) (bool, error)
ValidateResponseSignature validates the response signature. It's the same as request signature validation except doesn't perform canonicalization.

Types ¶
type Client ¶
type Client interface {
	AppID() string

	// Send sends the specific event to the ingest API.
	Send(ctx context.Context, evt any) (string, error)
	// Send sends a batch of events to the ingest API.
	SendMany(ctx context.Context, evt []any) ([]string, error)

	Serve() http.Handler
	ServeWithOpts(opts ServeOpts) http.Handler
	SetOptions(opts ClientOpts) error
	SetURL(u *url.URL)
}
Client represents a client used to send events to Inngest.

func NewClient ¶
func NewClient(opts ClientOpts) (Client, error)
NewClient returns a concrete client initialized with the given ingest key, which can immediately send events to the ingest API.

type ClientOpts ¶
added in v0.5.0
type ClientOpts struct {
	AppID string

	// HTTPClient is the HTTP client used to send events.
	HTTPClient *http.Client
	// EventKey is your Inngest event key for sending events.  This defaults to the
	// `INNGEST_EVENT_KEY` environment variable if nil.
	EventKey *string

	// EventURL is the URL of the event API to send events to.  This defaults to
	// https://inn.gs if nil.
	//
	// Deprecated: Use EventAPIBaseURL instead.
	EventURL *string

	// Env is the branch environment to deploy to.  If nil, this uses
	// os.Getenv("INNGEST_ENV").  This only deploys to branches if the
	// signing key is a branch signing key.
	Env *string

	// Logger is the structured logger to use from Go's builtin structured
	// logging package.
	Logger *slog.Logger

	// SigningKey is the signing key for your app.  If nil, this defaults
	// to os.Getenv("INNGEST_SIGNING_KEY").
	SigningKey *string

	// SigningKeyFallback is the fallback signing key for your app. If nil, this
	// defaults to os.Getenv("INNGEST_SIGNING_KEY_FALLBACK").
	SigningKeyFallback *string

	// APIOrigin is the specified host to be used to make API calls
	APIBaseURL *string

	// EventAPIOrigin is the specified host to be used to send events to
	EventAPIBaseURL *string

	// RegisterURL is the URL to use when registering functions.  If nil
	// this defaults to Inngest's API.
	//
	// This only needs to be set when self hosting.
	RegisterURL *string

	// AppVersion supplies an application version identifier. This should change
	// whenever code within one of your Inngest function or any dependency thereof changes.
	AppVersion *string

	// MaxBodySize is the max body size to read for incoming invoke requests
	MaxBodySize int

	// URL that the function is served at.  If not supplied this is taken from
	// the incoming request's data.
	URL *url.URL

	// UseStreaming enables streaming - continued writes to the HTTP writer.  This
	// differs from true streaming in that we don't support server-sent events.
	UseStreaming bool

	// AllowInBandSync allows in-band syncs to occur. If nil, in-band syncs are
	// disallowed.
	AllowInBandSync *bool

	// Dev is whether to use the Dev Server.
	Dev *bool

	// Middleware is a list of middleware to apply to the client.
	Middleware []func() middleware.Middleware
}
type ConfigBatchEvents ¶
added in v0.13.0
type ConfigBatchEvents = fn.EventBatchConfig
ConfigBatchEvents allows you run functions with a batch of events, instead of executing a new run for every event received.

The MaxSize option configures how many events will be collected into a batch before executing a new function run.

The timeout option limits how long Inngest waits for a batch to fill to MaxSize before executing the function with a smaller batch. This allows you to ensure functions run without waiting for a batch to fill indefinitely.

Inngest will execute your function as soon as MaxSize is reached or the Timeout is reached.

type ConfigCancel ¶
added in v0.13.0
type ConfigCancel = fn.Cancel
ConfigCancel represents a cancellation signal for a function. When specified, this will set up pauses which automatically cancel the function based off of matching events and expressions.

type ConfigDebounce ¶
added in v0.13.0
type ConfigDebounce = fn.Debounce
ConfigDebounce represents debounce configuration.

type ConfigPriority ¶
added in v0.13.0
type ConfigPriority = fn.Priority
ConfigPriority allows you to dynamically execute some runs ahead or behind others based on any data. This allows you to prioritize some jobs ahead of others without the need for a separate queue. Some use cases for priority include:

- Giving higher priority based on a user's subscription level, for example, free vs. paid users. - Ensuring that critical work is executed before other work in the queue. - Prioritizing certain jobs during onboarding to give the user a better first-run experience.

type ConfigRateLimit ¶
added in v0.13.0
type ConfigRateLimit = fn.RateLimit
ConfigRateLimit rate limits a function to a maximum number of runs over a given period. Any runs over the limit are ignored and are NOT enqueued for the future.

type ConfigSingleton ¶
added in v0.13.0
type ConfigSingleton = fn.Singleton
ConfigSingleton configures a function to run as a singleton, ensuring that only one instance of the function is active at a time for a given key. This is useful for deduplicating runs or enforcing exclusive execution.

If a new run is triggered while another instance with the same key is active, it will either be skipped or replace the existing instance depending on the mode.

type ConfigStepConcurrency ¶
added in v0.13.0
type ConfigStepConcurrency = fn.Concurrency
Concurrency keys: virtual queues.
ConfigStepConcurrency represents a single concurrency limit for a function. Concurrency limits the number of running steps for a given key at a time. Other steps will be enqueued for the future and executed as soon as there's capacity.

Concurrency keys: virtual queues. ¶
The `Key` parameter is an optional CEL expression evaluated using the run's events. The output from the expression is used to create new virtual queues, which limits the number of runs for each virtual queue.

For example, to limit the number of running steps for every account in your system, you can send the `account_id` in the triggering event and use the following key:

event.data.account_id
Concurrency is then limited for each unique account_id field in parent events.

type ConfigThrottle ¶
added in v0.13.0
type ConfigThrottle = fn.Throttle
ConfigThrottle represents concurrency over time. This limits the maximum number of new function runs over time. Any runs over the limit are enqueued for the future.

Note that this does not limit the number of steps executing at once and only limits how frequently runs can start. To limit the number of steps executing at once, use concurrency limits.

type ConfigTimeouts ¶
added in v0.13.0
type ConfigTimeouts = fn.Timeouts
ConfigTimeouts represents timeouts for the function. If any of the timeouts are hit, the function will be marked as cancelled with a cancellation reason.

type ConnectOpts ¶
added in v0.8.0
type ConnectOpts struct {
	Apps []Client

	// InstanceID represents a stable identifier to be used for identifying connected SDKs.
	// This can be a hostname or other identifier that remains stable across restarts.
	//
	// If nil, this defaults to the current machine's hostname.
	InstanceID *string

	RewriteGatewayEndpoint func(endpoint url.URL) (url.URL, error)

	// MaxConcurrency defines the maximum number of requests the worker can process at once.
	// This affects goroutines available to handle connnect workloads, as well as flow control.
	// Defaults to 1000.
	MaxConcurrency int
}
type Event ¶
type Event = event.Event
type FunctionOpts ¶
added in v0.5.0
type FunctionOpts = fn.FunctionOpts
FunctionOpts represents the options available to configure functions. This includes concurrency, retry, and flow control configuration.

type GenericEvent ¶
added in v0.5.0
type GenericEvent[DATA any] = event.GenericEvent[DATA]
type Input ¶
added in v0.5.0
type Input[T any] = fn.Input[T]
Input is the input for a given function run.

type InputCtx ¶
added in v0.5.0
type InputCtx = fn.InputCtx
InputCtx is the additional context for a given function run, including the run ID, function ID, step ID, attempt, etc.

type MultipleTriggers ¶
added in v0.13.0
type MultipleTriggers = fn.MultipleTriggers
MultipleTriggers represents the configuration for a function that can be triggered by multiple triggers.

type SDKFunction ¶
added in v0.5.0
type SDKFunction[T any] func(ctx context.Context, input Input[T]) (any, error)
SDKFunction represents a user-defined function to be called based off of events or on a schedule.

The function is registered with the SDK by calling `CreateFunction` with the function name, the trigger, the event type for marshalling, and any options.

This uses generics to strongly type input events:

func(ctx context.Context, input gosdk.Input[SignupEvent]) (any, error) {
	// .. Your logic here.  input.Event will be strongly typed as a SignupEvent.
}
type ServableFunction ¶
added in v0.5.0
type ServableFunction = fn.ServableFunction
ServableFunction defines a function which can be called by a handler's Serve method.

func CreateFunction ¶
added in v0.5.0
func CreateFunction[T any](
	c Client,
	fc FunctionOpts,
	trigger fn.Triggerable,
	f SDKFunction[T],
) (ServableFunction, error)
CreateFunction creates a new function which can be registered within a handler.

This function uses generics, allowing you to supply the event that triggers the function. For example, if you have a signup event defined as a struct you can use this to strongly type your input:

type SignupEvent struct {
	Name string
	Data struct {
		Email     string
		AccountID string
	}
}

f := CreateFunction(
	inngestgo.FunctionOptions{Name: "Post-signup flow"},
	inngestgo.EventTrigger("user/signed.up"),
	func(ctx context.Context, input gosdk.Input[SignupEvent]) (any, error) {
		// .. Your logic here.  input.Event will be strongly typed as a SignupEvent.
		// step.Run(ctx, "Do some logic", func(ctx context.Context) (string, error) { return "hi", nil })
	},
)
type ServeOpts ¶
added in v0.8.0
type ServeOpts struct {
	// Origin is the host to used for HTTP base function invoking.
	// It's used to specify the host were the functions are hosted on sync.
	// e.g. https://example.com
	Origin *string

	// Path is the path to use for HTTP base function invoking
	// It's used to specify the path were the functions are hosted on sync.
	// e.g. /api/inngest
	Path *string
}
type StepError ¶
added in v0.6.0
type StepError = errors.StepError
type StreamResponse ¶
added in v0.5.0
type StreamResponse struct {
	StatusCode int               `json:"status"`
	Body       any               `json:"body"`
	RetryAt    *time.Time        `json:"retryAt"`
	NoRetry    bool              `json:"noRetry"`
	Headers    map[string]string `json:"headers"`
}
type Trigger ¶
added in v0.12.0
type Trigger = fn.Trigger
Trigger represents a function trigger - either an EventTrigger or a CronTrigger

 Source Files ¶
View all Source files
client.go
connect.go
consts.go
env.go
errors.go
event.go
expose_internal.go
funcs.go
handler.go
headers.go
net.go
platform.go
signature.go
version.go
 Directories ¶
Show internal
Expand all
connect
errors
examples
experimental
group
realtime
step
tests
Why Go
Use Cases
Case Studies
Get Started
Playground
Tour
Stack Overflow
Help
Packages
Standard Library
Sub-repositories
About Go Packages
About
Download
Blog
Issue Tracker
Release Notes
Brand Guidelines
Code of Conduct
Connect
Twitter
GitHub
Slack
r/golang
Meetup
Golang Weekly
Gopher in flight goggles
Copyright
Terms of Service
Privacy Policy
Report an Issue
System theme
Theme Toggle


Shortcuts Modal

Google logo
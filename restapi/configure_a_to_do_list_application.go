// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	"github.com/newrelic/go-agent/v3/newrelic"

	"todo-list-server/models"
	"todo-list-server/restapi/operations"
	"todo-list-server/restapi/operations/todos"
)

//go:generate swagger generate server --target ..\..\server-complete --name AToDoListApplication --spec ..\swagger.yml --principal interface{}

var exampleFlags = struct {
	Example1 string `long:"example1" description:"Sample for showing how to configure cmd-line flags"`
	Example2 string `long:"example2" description:"Further info at https://github.com/jessevdk/go-flags"`
}{}

func configureFlags(api *operations.AToDoListApplicationAPI) {
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		swag.CommandLineOptionsGroup{
			ShortDescription: "Example Flags",
			LongDescription:  "",
			Options:          &exampleFlags,
		},
	}
}

var items = make(map[int64]*models.Item)
var lastID int64

var itemsLock = &sync.Mutex{}
var AppNR *newrelic.Application
var err error


func newItemID() int64 {
	return atomic.AddInt64(&lastID, 1)
}

func addItem(item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	newID := newItemID()
	item.ID = newID
	items[newID] = item

	return nil
}

func updateItem(id int64, item *models.Item) error {
	if item == nil {
		return errors.New(500, "item must be present")
	}

	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	item.ID = id
	items[id] = item
	return nil
}

func deleteItem(id int64) error {
	itemsLock.Lock()
	defer itemsLock.Unlock()

	_, exists := items[id]
	if !exists {
		return errors.NotFound("not found: item %d", id)
	}

	delete(items, id)
	return nil
}

func allItems(since int64, limit int32) (result []*models.Item) {
	txn := AppNR.StartTransaction("GET ALL TODOS")
	defer txn.End()
	result = make([]*models.Item, 0)
	for id, item := range items {
		if len(result) >= int(limit) {
			return
		}
		if since == 0 || id > since {
			result = append(result, item)
		}
	}
	return
}

func configureAPI(api *operations.AToDoListApplicationAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.TodosAddOneHandler = todos.AddOneHandlerFunc(func(params todos.AddOneParams) middleware.Responder {
		if err := addItem(params.Body); err != nil {
			return todos.NewAddOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewAddOneCreated().WithPayload(params.Body)
	})
	api.TodosDestroyOneHandler = todos.DestroyOneHandlerFunc(func(params todos.DestroyOneParams) middleware.Responder {
		if err := deleteItem(params.ID); err != nil {
			return todos.NewDestroyOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewDestroyOneNoContent()
	})
	api.TodosFindTodosHandler = todos.FindTodosHandlerFunc(func(params todos.FindTodosParams) middleware.Responder {
		mergedParams := todos.NewFindTodosParams()
		mergedParams.Since = swag.Int64(0)
		if params.Since != nil {
			mergedParams.Since = params.Since
		}
		if params.Limit != nil {
			mergedParams.Limit = params.Limit
		}
		return todos.NewFindTodosOK().WithPayload(allItems(*mergedParams.Since, *mergedParams.Limit))
	})
	api.TodosUpdateOneHandler = todos.UpdateOneHandlerFunc(func(params todos.UpdateOneParams) middleware.Responder {
		if err := updateItem(params.ID, params.Body); err != nil {
			return todos.NewUpdateOneDefault(500).WithPayload(&models.Error{Code: 500, Message: swag.String(err.Error())})
		}
		return todos.NewUpdateOneOK().WithPayload(params.Body)
	})

	api.ServerShutdown = func() {}
	println(exampleFlags.Example1)
	println(exampleFlags.Example2)

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

func newRelicMiddleware(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
            txn := AppNR.StartTransaction(r.URL.String())
            defer txn.End()

            // req is a *http.Request, this marks the transaction as a web transaction
            txn.SetWebRequestHTTP(r)

            // writer is a http.ResponseWriter, use the returned writer in place of the original
            writer := txn.SetWebResponse(w)

            // do some middleware logic here
            handler.ServeHTTP(writer, r)
        })
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return newRelicMiddleware(handler)
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	//_, h := newrelic.WrapHandleFunc(AppNR, "/", handler.ServeHTTP)
	//return http.HandlerFunc(h)
	return handler
}

func init() {
	fmt.Println("Initializing new relic configs")
	AppNR, err = newrelic.NewApplication(
		newrelic.ConfigAppName("Todo-List-Server"),
		newrelic.ConfigLicense(""),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigAppLogEnabled(true),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
		func(config *newrelic.Config) {
			config.CustomInsightsEvents.Enabled = true
		},
		func(config *newrelic.Config) {
			config.TransactionEvents.Attributes.Enabled = true
		},
	)
}

# NewRelic Integration with swagger generated rest api application
 - This repo has pre swagger generated go files, which uses swagger.yaml as spec.
## Pre-Requisites
 - Install latest golang (Pls follow the doc https://go.dev/doc/install)
## How to Compile and run
- Once the code has been cloned, pls run `go mod tidy` form the NW-POC directory.
- This repo has pre-generated go files, no need to generate them again
- Go to cmd\a-to-do-list-application-server 
  `cd cmd\a-to-do-list-application-server`
- run `go build -o nwpoc.exe main.go`, this will genreate a binary nwpoc.exe for windows env
- For Linux/MAC env, just build it with `go build -o nwpoc main.go`, this will geneate binary nwpoc that could run on Linxu or MAC OS
- Run the binary with the command  `.\nwpoc --port 3000`, this will serve application on port 3000
## Integration of NewRelic Agent with swagger generated codebase
- The swagger geneated code has a part which could be editable, once it is edited, swagger tool wont regenerate/overwrite the code again.
- Open this file restapi\configure_a_to_do_list_application.go which has an go init function, that initializes the new relice go-agent with app name, license key etc
```
func init() {
	fmt.Println("Initializing new relic configs")
	NewRelicAppClient, err = newrelic.NewApplication(
		newrelic.ConfigAppName("Todo-List-Server"),
		newrelic.ConfigLicense(""),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigAppLogEnabled(true),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
}`
```
- The middleware function is where we are trying to get the rest endpoints dynamically 
```
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}
```





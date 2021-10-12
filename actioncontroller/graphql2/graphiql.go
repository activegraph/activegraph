package graphql

import (
	"html/template"
	"net/http"
)

const indexTemplate = `
{{ define "index" }}
<!doctype html>
<html>
  <head>
    <title>Active Graph Console</title>
    <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
  </head>
  <body style="margin: 0;">
    <div id="graphiql" style="height: 100vh;"></div>

    <script
      crossorigin
      src="https://unpkg.com/react/umd/react.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/graphiql/graphiql.min.js"
    ></script>

    <script>
	  const fetcher = GraphiQL.createFetcher({ url: '{{ .Endpoint }}' });

      ReactDOM.render(
        React.createElement(GraphiQL, { fetcher: fetcher }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
{{ end }}
`

type graphiqlData struct {
	Endpoint string
}

func handleGraphiQL(rw http.ResponseWriter, r *http.Request) {
	t := template.New("graphiql")
	t, err := t.Parse(indexTemplate)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	data := graphiqlData{
		Endpoint: r.URL.Path,
	}

	err = t.ExecuteTemplate(rw, "index", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

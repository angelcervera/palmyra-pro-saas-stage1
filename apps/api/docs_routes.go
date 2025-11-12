package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// docSpecs maps public documentation names to their contract files.
var docSpecs = map[string]string{
	// Expose only mounted domains in docs
	"schema-categories": "contracts/schema-categories.yaml",
	"schema-repository": "contracts/schema-repository.yaml",
	"users":             "contracts/users.yaml",
}

const swaggerUITemplate = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>TCG Land API – Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>body{margin:0} #swagger-ui{max-width:1400px;margin:0 auto}</style>
  </head>
  <body>
    <div style="padding:10px 16px;background:#f7f7f7;border-bottom:1px solid #eaeaea;display:flex;align-items:center;gap:8px;">
      <label for="apiPicker" style="font-weight:600;">Select API:</label>
      <select id="apiPicker" style="padding:6px 8px;"></select>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
      const specs = [/*__SPECS__*/];
      (function init(){
        const picker = document.getElementById('apiPicker');
        const urls = specs.map(s => ({ url: s.url, name: s.name }));
        urls.forEach(u => { const o=document.createElement('option'); o.value=u.url; o.textContent=u.name; picker.appendChild(o); });
        const first = urls[0] ? urls[0].url : '';
        window.ui = SwaggerUIBundle({
          urls: urls,
          "urls.primaryName": urls[0]?.name || '',
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
          layout: 'StandaloneLayout'
        });
        if (first) picker.value = first;
        picker.addEventListener('change', (e) => {
          const url = e.target.value; if (window.ui && window.ui.specActions) { window.ui.specActions.updateUrl(url); window.ui.specActions.download(url); }
        });
      })();
    </script>
  </body>
</html>`

func registerDocsRoutes(router chi.Router, logger *zap.Logger) {
	router.Get("/docs", docsUIHandler())
	router.Get("/openapi/{name}.json", openapiJSONHandler(logger))
}

func docsUIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		injected := buildDocSpecsList()
		ui := strings.Replace(swaggerUITemplate, "/*__SPECS__*/", injected, 1)
		_, _ = w.Write([]byte(ui))
	}
}

func openapiJSONHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		path, ok := docSpecs[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		spec := mustLoadSpec(logger, path)
		b, err := spec.MarshalJSON()
		if err != nil {
			logger.Error("marshal openapi json", zap.String("name", name), zap.Error(err))
			http.Error(w, "failed to marshal OpenAPI", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}
}

func buildDocSpecsList() string {
	names := make([]string, 0, len(docSpecs))
	for name := range docSpecs {
		names = append(names, name)
	}
	sort.Strings(names)

	var builder strings.Builder
	for i, name := range names {
		if i > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString(fmt.Sprintf("        { url: '/openapi/%s.json', name: '%s' }", name, name))
	}
	return builder.String()
}

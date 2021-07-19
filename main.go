package main

import (
	"encoding/json"
	context2 "github.com/Unleash/unleash-client-go/v3/context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Unleash/unleash-client-go/v3"
	"github.com/Unleash/unleash-client-go/v3/api"
)

type FeatureVariant struct {
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Payload api.Payload `json:"payload"`
}

type FeatureToggle struct {
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Variant FeatureVariant `json:"variant"`
}

type ProxyResponse struct {
	Toggles []FeatureToggle `json:"toggles"`
}

func contains(arr []string, needle string) bool {
	for _, el := range arr {
		if el == needle {
			return true
		}
	}
	return false
}


func main() {
	proxySecrets := strings.Split(os.Getenv("UNLEASH_PROXY_SECRETS"), ",")

	uc, err := unleash.NewClient(
		unleash.WithAppName("unleash-proxy-go"),
		unleash.WithUrl(os.Getenv("UNLEASH_API_URL")),
		unleash.WithCustomHeaders(http.Header{"Authorization": {os.Getenv("UNLEASH_SECRET")}}),
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithDisableMetrics(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !contains(proxySecrets, authHeader) {
			w.WriteHeader(401)
			return
		}
		var toggles []FeatureToggle
		props := map[string]string{}

		for k, v := range r.URL.Query() {
			props[k] = strings.Join(v, ",")
		}
		context := context2.Context{
			UserId:        r.URL.Query().Get("userId"),
			SessionId:     r.URL.Query().Get("sessionId"),
			RemoteAddress: r.RemoteAddr,
			Properties:    props,
		}
		for _, feature := range uc.ListFeatures() {
			enabled := uc.IsEnabled(feature.Name, unleash.WithContext(context))
			vari := uc.GetVariant(feature.Name, unleash.WithVariantContext(context))
			toggles = append(toggles, FeatureToggle{
				Name:    feature.Name,
				Enabled: enabled,
				Variant: FeatureVariant{
					Name:    vari.Name,
					Enabled: vari.Enabled,
					Payload: vari.Payload,
				},
			})
			uc.IsEnabled(feature.Name)
		}
		proxyResponse := ProxyResponse{
			Toggles: toggles,
		}
		json.NewEncoder(w).Encode(proxyResponse)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if true {
			w.WriteHeader(200)
		}
	})

	http.ListenAndServe(":1982", nil)
}

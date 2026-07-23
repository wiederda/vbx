// ------------------------
// stdlib_geo.go (OSM-only, Array, interne URL-Kodierung)
// ------------------------

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Basis-URL für Nominatim
var geoBaseURL = "https://nominatim.openstreetmap.org/search?format=json&limit=1&q="

// InitGeoFunctions registriert Geo-Funktionen
func InitGeoFunctions() {
	if builtins == nil {
		builtins = make(map[string]BuiltinInfo)
	}

	ns := "geo."

	Register(ns+"Lookup", "geo", "city", "Sucht Geodaten via OSM. Gibt Array zurück: [Stadt, Region, Land, Lat, Lon].", func(args []Value) Value {
		if len(args) < 1 {
			return Value{Kind: KindArr, Arr: []Value{StrVal("error: no city provided")}}
		}

		city := ToString(args[0])
		urlStr := geoBaseURL + url.QueryEscape(city) // interne URL-Kodierung

		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return Value{Kind: KindArr, Arr: []Value{StrVal("error: request creation failed: " + err.Error())}}
		}
		req.Header.Set("User-Agent", "vbx-geo")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return Value{Kind: KindArr, Arr: []Value{StrVal("error: network error: " + err.Error())}}
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return Value{Kind: KindArr, Arr: []Value{StrVal("error: read response failed: " + err.Error())}}
		}

		var result []map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil || len(result) == 0 {
			return Value{Kind: KindArr, Arr: []Value{StrVal("error: no results or invalid response")}}
		}

		place := result[0]

		// Werte extrahieren
		cityName := "unknown"
		state := "unknown"
		country := "unknown"
		lat := "unknown"
		lon := "unknown"

		if displayName, ok := place["display_name"].(string); ok {
			parts := strings.Split(displayName, ",")
			if len(parts) >= 1 {
				cityName = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 {
				state = strings.TrimSpace(parts[1])
			}
			if len(parts) >= 3 {
				country = strings.TrimSpace(parts[len(parts)-1])
			}
		}

		if latStr, ok := place["lat"].(string); ok {
			lat = latStr
		}
		if lonStr, ok := place["lon"].(string); ok {
			lon = lonStr
		}

		return Value{Kind: KindArr, Arr: []Value{
			StrVal(cityName),
			StrVal(state),
			StrVal(country),
			StrVal(lat),
			StrVal(lon),
		}}
	})
}

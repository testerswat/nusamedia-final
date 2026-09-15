package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DiscoverResult struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Category         string  `json:"category"`
	Address          string  `json:"address"`
	Lat              float64 `json:"lat"`
	Lon              float64 `json:"lon"`
	DistanceMeters   int     `json:"distance_meters"`
	Source           string  `json:"source"`
	OpenStreetMapURL string  `json:"openstreetmap_url"`
}

type nominatimItem struct {
	PlaceID     int    `json:"place_id"`
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Name        string `json:"name"`
}

func (s Social) DiscoverNearby(ctx context.Context, lat, lon float64, radiusMeters int, intent string, limit int) ([]DiscoverResult, string, error) {
	if radiusMeters < 250 {
		radiusMeters = 250
	}
	if radiusMeters > 10000 {
		radiusMeters = 10000
	}
	if limit < 1 {
		limit = 12
	}
	if limit > 30 {
		limit = 30
	}
	queries, label := discoveryQueries(intent)
	results := make([]DiscoverResult, 0, limit)
	seen := map[string]bool{}
	client := &http.Client{Timeout: 7 * time.Second}
	for _, q := range queries {
		u := "https://nominatim.openstreetmap.org/search?format=jsonv2&addressdetails=1&limit=30&bounded=1&q=" + url.QueryEscape(q) + "&lat=" + strconv.FormatFloat(lat, 'f', 6, 64) + "&lon=" + strconv.FormatFloat(lon, 'f', 6, 64)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "NusaMedia/1.0 (nearby-discovery)")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var items []nominatimItem
		err = json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if err != nil {
			continue
		}
		for _, it := range items {
			rlat, e1 := strconv.ParseFloat(it.Lat, 64)
			rlon, e2 := strconv.ParseFloat(it.Lon, 64)
			if e1 != nil || e2 != nil {
				continue
			}
			d := haversineMeters(lat, lon, rlat, rlon)
			if d > float64(radiusMeters) {
				continue
			}
			name := strings.TrimSpace(it.Name)
			if name == "" {
				name = strings.Split(it.DisplayName, ",")[0]
			}
			key := fmt.Sprintf("%s:%0.5f:%0.5f", name, rlat, rlon)
			if seen[key] {
				continue
			}
			seen[key] = true
			cat := friendlyCategory(it.Category, it.Type, label)
			results = append(results, DiscoverResult{ID: strconv.Itoa(it.PlaceID), Name: name, Category: cat, Address: it.DisplayName, Lat: rlat, Lon: rlon, DistanceMeters: int(math.Round(d)), Source: "OpenStreetMap", OpenStreetMapURL: "https://www.openstreetmap.org/?mlat=" + strconv.FormatFloat(rlat, 'f', 6, 64) + "&mlon=" + strconv.FormatFloat(rlon, 'f', 6, 64) + "#map=18/" + strconv.FormatFloat(rlat, 'f', 6, 64) + "/" + strconv.FormatFloat(rlon, 'f', 6, 64)})
		}
		if len(results) >= limit {
			break
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].DistanceMeters < results[j].DistanceMeters })
	if len(results) > limit {
		results = results[:limit]
	}
	return results, label, nil
}

func discoveryQueries(intent string) ([]string, string) {
	x := strings.ToLower(strings.TrimSpace(intent))
	switch {
	case strings.Contains(x, "makan"), strings.Contains(x, "kuliner"), strings.Contains(x, "lapar"), strings.Contains(x, "food"):
		return []string{"restaurant", "cafe", "fast food", "food court", "bakery"}, "Tempat makan di sekitar"
	case strings.Contains(x, "kopi"), strings.Contains(x, "coffee"), strings.Contains(x, "ngopi"):
		return []string{"cafe", "coffee shop"}, "Kopi & cafe di sekitar"
	case strings.Contains(x, "apotik"), strings.Contains(x, "apotek"), strings.Contains(x, "obat"):
		return []string{"pharmacy", "clinic", "hospital"}, "Kesehatan di sekitar"
	case strings.Contains(x, "belanja"), strings.Contains(x, "mall"), strings.Contains(x, "toko"), strings.Contains(x, "shopping"):
		return []string{"mall", "supermarket", "convenience store", "department store"}, "Belanja di sekitar"
	case strings.Contains(x, "bank"), strings.Contains(x, "atm"), strings.Contains(x, "tarik"):
		return []string{"bank", "atm"}, "Bank & ATM di sekitar"
	case strings.Contains(x, "hotel"), strings.Contains(x, "nginep"), strings.Contains(x, "menginap"):
		return []string{"hotel", "guest house", "hostel"}, "Tempat menginap di sekitar"
	case strings.Contains(x, "pom"), strings.Contains(x, "bensin"), strings.Contains(x, "isi bensin"):
		return []string{"fuel"}, "SPBU di sekitar"
	default:
		return []string{x, "restaurant", "cafe", "shop", "tourism"}, "Yang mungkin kamu cari di sekitar"
	}
}

func friendlyCategory(category, typ, label string) string {
	if strings.Contains(label, "makan") {
		return "Kuliner"
	}
	if strings.Contains(label, "Kopi") {
		return "Kafe & Kopi"
	}
	switch typ {
	case "restaurant":
		return "Restoran"
	case "cafe":
		return "Kafe"
	case "pharmacy":
		return "Apotek"
	case "bank":
		return "Bank"
	case "atm":
		return "ATM"
	case "hotel":
		return "Hotel"
	}
	if category != "" {
		return strings.Title(strings.ReplaceAll(category, "_", " "))
	}
	return "Tempat"
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp := (lat2 - lat1) * math.Pi / 180
	dl := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * R * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

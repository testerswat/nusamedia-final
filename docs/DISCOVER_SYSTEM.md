# NusaMedia Discover — Real System Contract

Discover is a contextual local-discovery system, not a static search screen.

## User journey

1. A visitor opens NusaMedia without signing in.
2. Landing UI explains social + local discovery and offers Discover immediately.
3. Discover requests browser geolocation only after user permission.
4. User enters natural language such as `saya sedang cari makan`, `cari kopi`, `butuh apotek`, or `cari ATM`.
5. Backend maps intent into multiple local categories.
6. Backend queries a real geographic provider (OpenStreetMap/Nominatim) using the supplied coordinates.
7. Results are filtered by the requested radius and sorted by distance.
8. UI renders name, category, address, distance and map link.
9. The user can change radius and intent without creating an account.
10. Registration/login remains one click away from discovery.

## Route

- Public web: `/#discover`
- API: `GET /api/v1/discover/nearby`

Query:

`lat`, `lon`, `radius` (250–10000 m), `intent`

Response contains `title`, `radius_meters`, `source`, and `items[]` with `name`, `category`, `address`, coordinates, distance and map URL.

## Context mapping

- makan / kuliner / lapar → restaurant, cafe, fast food, food court, bakery
- kopi / coffee / ngopi → cafe, coffee shop
- apotek / obat → pharmacy, clinic, hospital
- belanja / toko / shopping → mall, supermarket, convenience store, department store
- bank / ATM / tarik → bank, ATM
- hotel / menginap → hotel, guest house, hostel
- bensin / pom → fuel
- other phrases → original intent plus broad place categories

## Privacy

The server does not need a stored user location for this public flow. Coordinates are supplied by the browser after explicit permission and are used for the request. The current implementation does not persist coordinates.

## Provider note

OpenStreetMap/Nominatim is a real external data provider. Production deployment must comply with its usage policy, caching requirements and attribution. A provider adapter should be retained so a commercial/local provider can be configured later without changing the UI contract.

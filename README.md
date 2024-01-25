Tinder for films. Simple: it recommends you films and either you like or you don't.

Endpoints:
1. Get films
GET https://j5szh4ivo1.execute-api.eu-north-1.amazonaws.com/default/get-films

Query parameters:
- id - string type, must be UUID v4

3. Update film
If you like or do not like recommended film.

GET https://54zfj2agze.execute-api.eu-north-1.amazonaws.com/default/update-user-films?method=unlike&film=The Perfect Man

Query params:
- id - string type, must be UUID v4
- method=unlike
- film=The Perfect Man

Headers:
authorization: Bearer ${yourAccessToken}

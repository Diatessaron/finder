Tinder for films. Simple: it recommends you films and either you like or you don't.

Endpoints:
1. Get films
To get recommended films

GET https://j5szh4ivo1.execute-api.eu-north-1.amazonaws.com/default/get-films

Query parameters:
- id=0165fb5f-9341-44fd-99b2-9828be80488f - string type, must be UUID v4

2. Update film
If you like or do not like recommended film.

GET https://54zfj2agze.execute-api.eu-north-1.amazonaws.com/default/update-user-films?method=unlike&film=The Perfect Man

Query params:
- id=0165fb5f-9341-44fd-99b2-9828be80488f - string type, must be UUID v4
- method=unlike - method you choose, either *like* or *unlike*
- film=The Perfect Man - string, name of the film to perform the chosen method

3. Delete one liked film
To delete one liked film

GET https://7575yvzd67.execute-api.eu-north-1.amazonaws.com/default/delete-one-liked-films?id=0165fb5f-9341-44fd-99b2-9828be80488f&filmToRemove=The Social Network

Query params:
- id=0165fb5f-9341-44fd-99b2-9828be80488f - string type, must be UUID v4
- filmToRemove=Her - string type, name of the film to remoe from the liked films

4. Get liked films
To get liked films. Allows pagination

GET https://wgc146jtpb.execute-api.eu-north-1.amazonaws.com/default/get-liked-films?id=0165fb5f-9341-44fd-99b2-9828be80488f

Query params:
- id=0165fb5f-9341-44fd-99b2-9828be80488f - string type, must be UUID v4
- page=1 - optional - int, number of the page
- size=2 - optional - int, number of the entries per page

5. Clear state films
To clear all liked and unliked films, to clear user recommendations

GET https://3yje4cfzq8.execute-api.eu-north-1.amazonaws.com/default/clear-state-films?id=0165fb5f-9341-44fd-99b2-9828be80488f

Query params:
- id=0165fb5f-9341-44fd-99b2-9828be80488f - string type, must be UUID v4

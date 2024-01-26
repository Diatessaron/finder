package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var movieSearchUrl = "https://api.themoviedb.org/3/search/movie?query={query}&include_adult=true&page=1&language=en-US"
var movieImageUrl = "https://api.themoviedb.org/3/movie/{movieId}/images"
var movieDirectorSearchUrl = "https://api.themoviedb.org/3/movie/{movie_id}/credits"
var movieDetailsSearchUrl = "https://api.themoviedb.org/3/movie/{movie_id}"
var imagePrefix = "https://image.tmdb.org/t/p/w500"

var tmdbToken = os.Getenv("TMDBReadToken")

func NormalizeFilms(recommendedFilms []string) (string, error) {
	var normalizedFilms []ResultRecommendedFilm

	for _, recommendedFilm := range recommendedFilms {
		movieId, err := searchForMovie(recommendedFilm)
		if err != nil {
			return "", err
		}

		movieDetails, err := searchForMovieDetails(movieId)
		if err != nil {
			return "", err
		}

		directorName, err := searchForDirector(movieId)
		if err != nil {
			return "", err
		}

		images, err := getImages(movieId)
		if err != nil {
			return "", err
		}

		normalizedFilms = constructFilmAndAppend(movieDetails, normalizedFilms, recommendedFilm, directorName, images)
	}

	bytes, err := json.Marshal(normalizedFilms)
	return string(bytes[:]), err
}

func searchForMovieDetails(movieId int) (Movie, error) {
	req, _ := http.NewRequest("GET", strings.ReplaceAll(movieDetailsSearchUrl, "{movie_id}", fmt.Sprint(movieId)), nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbToken)
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	byteResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return Movie{}, err
	}

	var movie Movie
	err = json.Unmarshal(byteResponse, &movie)
	if err != nil {
		return Movie{}, err
	}

	return movie, nil
}

func searchForMovie(recommendedFilm string) (int, error) {
	url := strings.ReplaceAll(strings.ReplaceAll(movieSearchUrl, "{query}", recommendedFilm), " ", "%20")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbToken)
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	byteResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var response searchForMovieResponse
	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return 0, err
	}
	if len(response.Results) == 0 {
		return 0, errors.New(fmt.Sprintf("Response results is empty. Recommended film - %s. Response results - %v", recommendedFilm, response.Results))
	}

	return response.Results[0].ID, nil
}

func searchForDirector(movieId int) (string, error) {
	req, _ := http.NewRequest("GET", strings.ReplaceAll(movieDirectorSearchUrl, "{movie_id}", fmt.Sprint(movieId)), nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbToken)
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	byteResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response MovieDetail
	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return "", err
	}

	for _, director := range response.Crew {
		if director.KnownForDepartment == "Directing" {
			return director.Name, nil
		}
	}

	return "", errors.New("Film director not found. Response crew - " + string(byteResponse))
}

func getImages(movieId int) (MovieImages, error) {
	req, _ := http.NewRequest("GET", strings.ReplaceAll(movieImageUrl, "{movieId}", fmt.Sprint(movieId)), nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbToken)
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	byteResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return MovieImages{}, err
	}

	var response MovieImages
	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return MovieImages{}, err
	}

	for index, image := range response.Posters {
		if index == 2 {
			break
		}

		image.FilePath = imagePrefix + image.FilePath
		response.Posters[index] = image
	}
	for index, image := range response.Logos {
		if index == 2 {
			break
		}

		image.FilePath = imagePrefix + image.FilePath
		response.Logos[index] = image
	}
	for index, image := range response.Backdrops {
		if index == 2 {
			break
		}

		image.FilePath = imagePrefix + image.FilePath
		response.Backdrops[index] = image
	}

	return response, nil
}

func constructFilmAndAppend(movieDetails Movie, normalizedFilms []ResultRecommendedFilm, recommendedFilm string, directorName string, images MovieImages) []ResultRecommendedFilm {
	var genreNames []string

	for _, genre := range movieDetails.Genres {
		genreNames = append(genreNames, genre.Name)
	}

	return append(normalizedFilms, ResultRecommendedFilm{
		recommendedFilm,
		movieDetails.ReleaseDate[0:4],
		genreNames,
		directorName,
		movieDetails.Overview,
		images,
	})
}

type movieIdResponse struct {
	ID int `json:"id"`
}

type searchForMovieResponse struct {
	Results []movieIdResponse `json:"results"`
}

type Movie struct {
	Genres           []Genre `json:"genres"`
	ID               int     `json:"id"`
	OriginalLanguage string  `json:"original_language"`
	OriginalTitle    string  `json:"original_title"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	Title            string  `json:"title"`
}

type Genre struct {
	Name string `json:"name"`
}

type ResultRecommendedFilm struct {
	Name        string      `json:"name"`
	Year        string      `json:"year"`
	Genres      []string    `json:"genres"`
	DirectedBy  string      `json:"directedBy"`
	Description string      `json:"description"`
	MovieImages MovieImages `json:"movieImages"`
}

type MovieDetail struct {
	Crew []Person `json:"crew"`
}

type Person struct {
	Name               string `json:"name"`
	KnownForDepartment string `json:"known_for_department""`
}

type Image struct {
	AspectRatio float64 `json:"aspect_ratio"`
	Height      int     `json:"height"`
	ISO639_1    *string `json:"iso_639_1"` // Using a pointer to handle null values
	FilePath    string  `json:"file_path"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int     `json:"vote_count"`
	Width       int     `json:"width"`
}

type MovieImages struct {
	Backdrops []Image `json:"backdrops"`
	Logos     []Image `json:"logos"`
	Posters   []Image `json:"posters"`
}

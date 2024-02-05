package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

var movieSearchUrl = "https://api.themoviedb.org/3/search/movie?query={query}&include_adult=true&page=1&language=en-US&year={year}"
var movieImageUrl = "https://api.themoviedb.org/3/movie/{movieId}/images"
var movieDirectorSearchUrl = "https://api.themoviedb.org/3/movie/{movie_id}/credits"
var movieDetailsSearchUrl = "https://api.themoviedb.org/3/movie/{movie_id}"
var imagePrefix = "https://image.tmdb.org/t/p/w500"

var tmdbToken = os.Getenv("TMDBReadToken")

func NormalizeFilms(recommendedFilms []string) (string, error) {
	var normalizedFilms []ResultRecommendedFilm
	filmsChan := make(chan filmData, len(recommendedFilms))

	for _, recommendedFilm := range recommendedFilms {
		go func(film string) {
			var data filmData
			data.movieID, data.err = searchForMovie(film)
			if data.err != nil {
				filmsChan <- data
				return
			}

			var wg sync.WaitGroup
			wg.Add(3) // We are going to perform 3 concurrent operations

			// Fetch movie details
			go func() {
				defer wg.Done()
				data.movie, data.err = searchForMovieDetails(data.movieID)
				if data.err != nil {
					return
				}
			}()

			// Fetch director names
			go func() {
				defer wg.Done()
				data.directors, data.err = searchForDirector(data.movieID)
				if data.err != nil {
					return
				}
			}()

			// Fetch images
			go func() {
				defer wg.Done()
				data.images, data.err = getImages(data.movieID)
				if data.err != nil {
					return
				}
			}()

			wg.Wait() // Wait for all concurrent operations to complete
			filmsChan <- data
		}(recommendedFilm)
	}

	for i := 0; i < len(recommendedFilms); i++ {
		data := <-filmsChan
		if data.err != nil {
			return "", data.err
		}

		normalizedFilm := constructFilm(data.movie, data.directors, data.images, recommendedFilms[i])
		normalizedFilms = append(normalizedFilms, normalizedFilm)
	}

	bytes, err := json.Marshal(normalizedFilms)
	return string(bytes), err
}

func constructFilm(movie Movie, directors []string, images MovieImages, recommendedFilm string) ResultRecommendedFilm {
	var genreNames []string
	for _, genre := range movie.Genres {
		genreNames = append(genreNames, genre.Name)
	}

	year := ""
	if len(movie.ReleaseDate) >= 4 {
		year = movie.ReleaseDate[:4]
	}

	return ResultRecommendedFilm{
		Name:        recommendedFilm,
		Year:        year,
		Genres:      genreNames,
		DirectedBy:  directors,
		Description: movie.Overview,
		MovieImages: images,
	}
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

	filteredFilms := filterResponseResultByName(response.Results, response.Results[0].Title)
	sort.Slice(filteredFilms, func(i, j int) bool {
		return filteredFilms[i].Popularity > filteredFilms[j].Popularity
	})

	return filteredFilms[0].ID, nil
}

func filterResponseResultByName(results []movieIdResponse, filmName string) []movieIdResponse {
	var filteredFilms []movieIdResponse
	for _, movie := range results {
		if movie.Title == filmName {
			filteredFilms = append(filteredFilms, movie)
		}
	}

	return filteredFilms
}

func searchForDirector(movieId int) ([]string, error) {
	req, _ := http.NewRequest("GET", strings.ReplaceAll(movieDirectorSearchUrl, "{movie_id}", fmt.Sprint(movieId)), nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+tmdbToken)
	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	byteResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return make([]string, 0), err
	}

	var response MovieDetail
	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return make([]string, 0), err
	}

	directors := make([]string, 0)
	for _, director := range response.Crew {
		if director.Job == "Director" {
			directors = append(directors, director.Name)
		}
	}

	if len(directors) == 0 {
		return make([]string, 0), errors.New("Film director not found. Response crew - " + string(byteResponse))
	} else {
		return directors, nil
	}
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
		if index == 4 {
			response.Posters = response.Posters[0:4]
			break
		}

		image.FilePath = imagePrefix + image.FilePath
		response.Posters[index] = image
	}
	for index, image := range response.Backdrops {
		if index == 4 {
			response.Backdrops = response.Backdrops[0:4]
			break
		}

		image.FilePath = imagePrefix + image.FilePath
		response.Backdrops[index] = image
	}

	return response, nil
}

func constructFilmAndAppend(movieDetails Movie, normalizedFilms []ResultRecommendedFilm, recommendedFilm string, directors []string, images MovieImages) []ResultRecommendedFilm {
	var genreNames []string

	for _, genre := range movieDetails.Genres {
		genreNames = append(genreNames, genre.Name)
	}

	return append(normalizedFilms, ResultRecommendedFilm{
		recommendedFilm,
		movieDetails.ReleaseDate[0:4],
		genreNames,
		directors,
		movieDetails.Overview,
		images,
	})
}

type movieIdResponse struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Popularity float64 `json:"popularity"`
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
	DirectedBy  []string    `json:"directedBy"`
	Description string      `json:"description"`
	MovieImages MovieImages `json:"movieImages"`
}

type MovieDetail struct {
	Crew []Person `json:"crew"`
}

type Person struct {
	Name string `json:"name"`
	Job  string `json:"job"`
}

type Image struct {
	FilePath string `json:"file_path"`
}

type MovieImages struct {
	Backdrops []Image `json:"backdrops"`
	Posters   []Image `json:"posters"`
}

type FilmRecommendation struct {
	FilmName string `json:"filmName"`
	Year     string `json:"year"`
}

type filmData struct {
	movieID   int
	movie     Movie
	directors []string
	images    MovieImages
	err       error
}

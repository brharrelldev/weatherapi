package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type coord struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}
type weather struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Coord   coord  `json:"coord"`
}

func downloaddata(url string) (int64, error) {

	resp, err := http.Get(url)

	fmt.Println("Downloading datasource")

	defer resp.Body.Close()
	if err != nil {
		fmt.Errorf("could not retrieve url %v", err)
	}

	urlSlice := strings.Split(url, "/")
	filename := urlSlice[len(urlSlice)-1]
	data, err := os.Create(filename)

	if err != nil {
		panic("Could not create file")
	}

	fmt.Println("Staging datasouce...")
	copyfile, err := io.Copy(data, resp.Body)

	if err != nil {
		panic("Could not generate file when copying, aborting ...")

	}
	fmt.Println("Datasource staged successfully!")

	return copyfile, nil

}

func unzip(file, result string) error {
	gz, err := os.Open(file)

	if err != nil {
		panic("something went wrong when trying to read file")
	}
	uz, err := gzip.NewReader(gz)
	if err != nil {
		fmt.Errorf("could not read file %v", err)
	}
	jsonf, err := os.Create(result)
	defer jsonf.Close()

	if err != nil {
		fmt.Errorf("could not unzip file %v", err)
	}

	fmt.Println("Unarchiving datasource...")
	io.Copy(jsonf, uz)

	fmt.Println("Datasource unarchived")


	return nil
}

func splitfile(bigfile string) ([]weather, error) {

	var weath []weather

	f, err := ioutil.ReadFile(bigfile)

	if err != nil {
		fmt.Errorf("could not read file %v", err)
	}

	e := json.Unmarshal(f, &weath)

	if e != nil {
		fmt.Println(e)
	}

	return weath, nil

}
func dbOps(db string, data []weather) error {

	if _, err := os.Stat(db); err != nil {
		f, err := os.Create(db)
		if err != nil {
			fmt.Errorf("could not create db file %v", err)
			return nil
		}
		defer f.Close()

	}

	sqldb, err := sql.Open("sqlite3", db)

	if err != nil {
		fmt.Errorf("could not open db file %v", err)
		return err
	}

	fmt.Println("Creating weather table...")
	createWeatherDb := `CREATE TABLE WEATHER (id integer not null primary key, city text)`
	_, err = sqldb.Exec(createWeatherDb)
	fmt.Println("weather table has been created")

	if err != nil {
		fmt.Errorf("could not execute statement %v", err)
		return err
	}

	stmt, err := sqldb.Begin()

	if err != nil {
		fmt.Errorf("error occured in transaction %v", err)
	}

	insert, err := stmt.Prepare("insert into weather(id, city) values(?,?)")
	if err != nil {
		fmt.Errorf("error with db driver %v")
		return err
	}

	for _, recs := range data {
		insert.Exec(recs.Id, recs.Name)
	}
	stmt.Commit()
	fmt.Println("Database load complete!")

	defer sqldb.Close()

	return nil
}

func GetWeather(writer http.ResponseWriter, request *http.Request) {
	request.Header.Set("Context-Type", "application/json")
	cities := mux.Vars(request)
	var cityId int

	db, err := sql.Open("sqlite3", "weather.db")
	if err != nil {
		fmt.Errorf("error connecting to database %v", err)
	}

	query, err := db.Prepare("select id from weather where city=? order by id")

	defer query.Close()

	err = query.QueryRow(cities["city"]).Scan(&cityId)

	if err != nil {
		fmt.Errorf("could not retrieve records from database %v")
	}
	fmt.Println(cityId)
	reqUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?id=%d&appid=6fbe6282bf59f3cf7bed8c66bdd3a63c", cityId)

	req, err := http.Get(reqUrl)

	if err != nil {
		fmt.Errorf("could not complete request %v", err)
		writer.WriteHeader(http.StatusBadRequest)
	}

	defer req.Body.Close()

	respBody, err := ioutil.ReadAll(req.Body)
	var weathermap map[string]map[string]float32

	json.Unmarshal([]byte(respBody), &weathermap)

	temperatureMap := weathermap["main"]
	currentTemp := fmt.Sprintf("%d", int(temperatureMap["temp"]*9/5-459.67))
	if err != nil {
		fmt.Errorf("could not read response body %v", err)
	}

	writer.Write([]byte("Current weather is: " + currentTemp + "F"))

}
func main() {

	url := "http://bulk.openweathermap.org/sample"
	arc := "city.list.json.gz"
	result := "city.list.json"
	dbfile := "weather.db"

	if _, err := os.Stat(result); os.IsNotExist(err) {

		fmt.Println("Necessary files not detected, beginning setup ...")
		var urlArr = []string{url, arc}

		endpoint := strings.Join(urlArr, "/")
		_, err := downloaddata(endpoint)

		if err != nil {
			fmt.Errorf("could not create arc %v", err)
		}

		err = unzip(arc, result)
		if err != nil {
			fmt.Errorf("something happened in trying to decompress arc %v", err)
		}
		datalist, err := splitfile(result)
		if err != nil {
			fmt.Errorf("could not decode data %v\n", err)
		}

		err = dbOps(dbfile, datalist)
		fmt.Println(err)

		if err != nil {
			fmt.Errorf("could not operate on database %v", err)
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/weather/{city}", GetWeather).Methods("GET")

	fmt.Println("Launching server")
	http.ListenAndServe(":9000", r)

}

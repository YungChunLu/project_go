package main

import (
	"encoding/json"
	"errors"
    "fmt"
	"log"
	"os"
	"net/http"
	"strconv"
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type PlaceOrderArgument struct {
	Origin [2]string 
	Destination [2]string 
}

type Order struct {
	Id int64 `json:"id"`
	Distance int64 `json:"distance"`
	Status string `json:"status"`
}

type Range struct {
	Lower float64
	Upper float64
}

type ErrorMessage struct {
	Error string `json:"error"`
}

type StatusMessage struct {
	Status string `json:"status"`
}

type Distance struct {
	Value int64 `json:"value"`
	Text string `json:"text"`
}

type Element struct {
	Status string `json:"status"`
	DistanceVal Distance `json:"distance"` 
}

type Row struct {
	Elements []Element
}

type MapResponse struct {
	Status string `json:"status"`
	Rows []Row `json:rows`
}

func checkErr(err error) {
	if err != nil {
		log.Panic(err)
		panic(err)
	}
}

func is_valid_coordinates(arr [2]string) error{
	var is_valid = true
	for idx, val := range arr {
		_val, err := strconv.ParseFloat(val, 10)
		_range := coordinates_ranges[idx]
		if err != nil || _val<_range.Lower || _val>_range.Upper{
			is_valid = false
			break
		}
	}
	if is_valid {
		return nil
	} else {
		return errors.New("Invalid coordinates.")
	}		
}

func is_valid_value(val string, lower_bound int64) (int64, bool){
	_val, err := strconv.ParseInt(val, 10, 32)
	if err!=nil || _val<lower_bound{
		return _val, false
	} else {
		return _val, true
	} 
}

func init_db() *sql.DB{
	psqlInfo := fmt.Sprintf("host=172.29.0.2 user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PGUSER"), os.Getenv("PGPASSWORD"), os.Getenv("PGDATABASE"))
	db, err := sql.Open("postgres", psqlInfo)
	checkErr(err)
	err = db.Ping()
	checkErr(err)
	fmt.Println("Successfully connected!")
	return db
}

func GetDistance(param *PlaceOrderArgument) int64{
	origin := fmt.Sprintf("%s,%s", param.Origin[0], param.Origin[1])
	destination := fmt.Sprintf("%s,%s", param.Destination[0], param.Destination[1])
	log.Printf("Origin: %s -> Destination: %s", origin, destination)
	end_point := "https://maps.googleapis.com/maps/api/distancematrix/json"
	url := fmt.Sprintf("%s?origins=%s&destinations=%s&key=%s",
		end_point, origin, destination, os.Getenv("APIKEY"))
	r, _ := http.Get(url)
	var response MapResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err==nil{
		if response.Status == "OK" && response.Rows[0].Elements[0].Status == "OK" {
			return response.Rows[0].Elements[0].DistanceVal.Value
		} else {
			return -1
		}
	} else {
		return -1
	}
}

func PlaceOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var param PlaceOrderArgument
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorMessage{"Invalid coordinates."})
	} else if param.Origin[0] == "" || param.Destination[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorMessage{"Missing required arguments."})
	} else {
		// check if origin/destination has valid coordinates
		if is_valid_coordinates(param.Origin) != nil || is_valid_coordinates(param.Destination) != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorMessage{"Invalid coordinates."})
			return
		}
		
		distance := GetDistance(&param)
		if distance == -1 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorMessage{"Can't get the distance."})
		} else {
			var lastInsertId int64
			err = db.QueryRow("INSERT INTO orders(distance, status) VALUES($1, 'UNASSIGNED') returning id;", distance).Scan(&lastInsertId)
			if err == nil {
				json.NewEncoder(w).Encode(Order{lastInsertId, distance, "UNASSIGNED"})
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorMessage{err.Error()})
			}
			
		}
	}
}

func GetOrderList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query()
	_page := q.Get("page")
	_limit := q.Get("limit")
	if _page=="" || _limit=="" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorMessage{"Missing required arguments."})
		return
	}
	page, is_valid_page := is_valid_value(_page, 0)
	limit, is_valid_limit := is_valid_value(_limit, 1)
	if !is_valid_page || !is_valid_limit{
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorMessage{"Invalid values."})
	} else {
		lower_bound, upper_bound := page*limit, (page+1)*limit
		sql_query := "SELECT id, distance, status FROM orders where id>$1 and id<=$2"
		orders := []Order{}
		switch rows, err := db.Query(sql_query, lower_bound, upper_bound); err {
			case sql.ErrNoRows:
				json.NewEncoder(w).Encode(orders)
			case nil:
				for rows.Next() {
					var id int64
					var distance int64
					var status string
					err := rows.Scan(&id, &distance, &status)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(ErrorMessage{err.Error()})
						return
					}
					orders = append(orders, Order{id, distance, status})
				}
				json.NewEncoder(w).Encode(orders)
			default:
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorMessage{err.Error()})
		}		
	}
}

func PlaceGetOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		GetOrderList(w, &*r)
	} else if r.Method == "POST" {
		PlaceOrder(w, &*r)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func TakeOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PATCH" {
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 10, 32)
		sql_query := "SELECT status FROM orders where id=$1;"
		var status string
		switch err := db.QueryRow(sql_query, id).Scan(&status); err {
			case sql.ErrNoRows:
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorMessage{"The order doesn't exist."})
			case nil:
				if status=="TAKEN" {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(ErrorMessage{"Oops! The order has been taken."})
				} else {
					if tx, err := db.Begin(); err==nil {
						sql_query := "SELECT status FROM orders where id=$1 and status='UNASSIGNED' FOR UPDATE OF orders NOWAIT;"
						err := tx.QueryRow(sql_query, id).Scan(&status)
						if err==sql.ErrNoRows {
							w.WriteHeader(http.StatusBadRequest)
							json.NewEncoder(w).Encode(ErrorMessage{"Oops! The order has been taken."})
						} else if err, ok := err.(*pq.Error); ok {
							if err.Code.Name()=="lock_not_available" {
								w.WriteHeader(http.StatusBadRequest)
								json.NewEncoder(w).Encode(ErrorMessage{"Oops! The order has been taken."})
							} else {
								w.WriteHeader(http.StatusInternalServerError)
								json.NewEncoder(w).Encode(ErrorMessage{err.Code.Name()})
							}
							tx.Rollback()
						} else {
							sql_query := "UPDATE orders SET status='TAKEN' WHERE id=$1;"
							if _, err := tx.Exec(sql_query, id); err!=nil {
								w.WriteHeader(http.StatusInternalServerError)
								json.NewEncoder(w).Encode(ErrorMessage{"The order is not successfully taken."})
								tx.Rollback()
							} else {
								if err := tx.Commit(); err!= nil {
									w.WriteHeader(http.StatusInternalServerError)
									json.NewEncoder(w).Encode(ErrorMessage{"The order is not successfully taken."})
								} else {
									json.NewEncoder(w).Encode(StatusMessage{"SUCCESS"})
								}
							}
						}
					} else {
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(ErrorMessage{"DB is non-operational."})
						tx.Rollback()
					}
				}
			default:
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorMessage{"DB is non-operational."})
		}
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

var db *sql.DB
// Valid value ranges for coordinates
var coordinates_ranges = [2]Range{Range{-90.0, 90.0}, Range{-180.0, 180.0}}

func main() {
	db = init_db()
	r := mux.NewRouter()
	r.HandleFunc("/orders", PlaceGetOrderHandler)
	r.HandleFunc("/orders/{id:[0-9]+}", TakeOrderHandler)
    http.Handle("/", r)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

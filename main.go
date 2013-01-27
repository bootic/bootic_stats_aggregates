package main

import(
  "log"
  "fmt"
  "bootic_stats_observer/socket"
  "bootic_stats_observer/redis_stats"
  "net/http"
  "github.com/gorilla/mux"
  "github.com/vmihailenco/redis"
  "encoding/json"
  "strconv"
  "strings"
)

func AllKeysHandler(client *redis.Client) (handle func(http.ResponseWriter, *http.Request)) {
  return func(res http.ResponseWriter, req *http.Request) {
    res.Header().Add("Content-Type", "application/json")
    res.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate, private, proxy-revalidate")
    res.Header().Add("Pragma", "no-cache")
    res.Header().Add("Expires", "Fri, 24 Nov 2000 01:00:00 GMT")
    
    vars := mux.Vars(req)
    splitPath := []string{vars["base"]}
    if vars["key"] != "" {
      splitPath = append(splitPath, vars["key"])
    }
    if vars["evt"] != "" {
      splitPath = append(splitPath, vars["evt"])
    }
    
    splitPath = append(splitPath, "*")
    
    redisPath := strings.Join(splitPath, ":")
    f := client.Keys(redisPath)
    
    urls := []string{}
    for _, v := range(f.Val()) {
      url := fmt.Sprintf("http://%s/%s", req.Host, strings.Replace(v, ":", "/", -1))
      urls = append(urls, url)
    }
    
    json, err := json.Marshal(urls)
    if err != nil {
      panic(err)
    }

    res.Write(json)
    
  }
}

func KeyHandler(client *redis.Client) (handle func(http.ResponseWriter, *http.Request)) {
  return func(res http.ResponseWriter, req *http.Request) {
    res.Header().Add("Content-Type", "application/json")
    res.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate, private, proxy-revalidate")
    res.Header().Add("Pragma", "no-cache")
    res.Header().Add("Expires", "Fri, 24 Nov 2000 01:00:00 GMT")
    
    vars := mux.Vars(req)
    splitPath := []string{vars["base"], vars["key"]}
    
    if vars["evt"] != "" {
      splitPath = append(splitPath, vars["evt"])
    }
    if vars["year"] != "" {
      splitPath = append(splitPath, vars["year"])
    }
    if vars["month"] != "" {
      splitPath = append(splitPath, vars["month"])
    }
    if vars["day"] != "" {
      splitPath = append(splitPath, vars["day"])
    }
    
    redisPath := strings.Join(splitPath, ":")

    f := client.HGetAll(redisPath)
    
    keys   := []string{}
    vals   := []int64{}
    
    counts  := make(map[string]int64)
    
    for i, v := range(f.Val()) {
      if i % 2 == 0 {
        keys = append(keys, v)
      } else {
        i, _ := strconv.ParseInt(v, 10, 0)
        vals = append(vals, i)
      }
    }
    
    for i, v := range(keys) {
      counts[v] = vals[i]
    }
    
    
    json, err := json.Marshal(counts)
    if err != nil {
      panic(err)
    }
    
    res.Write(json)
  }
  
}


func main() {
  
  topic := "all"
  
  daemon, _  := socket.NewZMQSubscriber("tcp://127.0.0.1:6000", topic)
  
  log.Println("ZMQ socket started")
  
  tracker, err := redis_stats.NewTracker("localhost:6379")
  
  log.Println("Redis tracker started")
  
  if err != nil {
    panic(err)
  }
  
  daemon.SubscribeToType(tracker.Notifier, "pageview")
  daemon.SubscribeToType(tracker.Funnels, "order")
  
  log.Println("Redis tracking 'pageview' and 'order' events")
  
  http_host := "localhost:8001"
  
  router := mux.NewRouter()
  keyHandler := KeyHandler(tracker.Conn)
  allKeysHandler := AllKeysHandler(tracker.Conn)
  
  router.HandleFunc("/{base}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{base}/{key}", allKeysHandler).Methods("GET")
  router.HandleFunc("/{base}/{key}/{evt}", allKeysHandler).Methods("GET")
  
  router.HandleFunc("/{base}/{key}/{evt}/{year}", keyHandler).Methods("GET")
  router.HandleFunc("/{base}/{key}/{evt}/{year}/{month}", keyHandler).Methods("GET")
  router.HandleFunc("/{base}/{key}/{evt}/{year}/{month}/{day}", keyHandler).Methods("GET")
  
  http.Handle("/", router)
      
  log.Fatal(http.ListenAndServe(http_host, nil))
}
package handlers

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "github.com/vmihailenco/redis"
  "log"
  "net/http"
  "net/url"
  "strconv"
  "strings"
)

type Payload struct {
  Type    string           `json:"type"`
  Account string           `json:"account"`
  Event   string           `json:"event"`
  Year    string           `json:"year"`
  Month   string           `json:"month"`
  Day     string           `json:"day"`
  Links   []string         `json:"links"`
  Data    map[string]int64 `json:"data"`
}

func respond(req *http.Request, res http.ResponseWriter, payload *Payload) {
  defer func() {
    if err := recover(); err != nil {
      log.Println("Goroutine failed:", err)
    }
  }()

  res.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate, private, proxy-revalidate")
  res.Header().Add("Pragma", "no-cache")
  res.Header().Add("Expires", "Fri, 24 Nov 2000 01:00:00 GMT")

  query, _ := url.ParseQuery(req.URL.RawQuery)
  jsonpCallback := query["callback"]

  json, err := json.Marshal(payload)

  if err != nil {
    panic(err)
  }

  if len(jsonpCallback) > 0 {
    res.Header().Add("Content-Type", "text/javascript")
    jsonAsString := string(json)
    wrapped := fmt.Sprintf("%s(%s)", jsonpCallback[0], jsonAsString)
    res.Write([]byte(wrapped))
  } else {
    res.Header().Add("Content-Type", "application/json")
    res.Write(json)
  }
}

func redisLinksLookup(client *redis.Client, req *http.Request, redisPath, prefix string) []string {
  f := client.Keys(redisPath)
  urls := []string{}

  for _, v := range f.Val() {
    url := fmt.Sprintf("http://%s%s/%s", req.Host, prefix, strings.Replace(v, ":", "/", -1))
    urls = append(urls, url)
  }

  return urls
}

func redisIntHash(client *redis.Client, redisPath string) map[string]int64 {
  f := client.HGetAll(redisPath)

  keys := []string{}
  vals := []int64{}

  counts := make(map[string]int64)

  for i, v := range f.Val() {
    if i%2 == 0 {
      keys = append(keys, v)
    } else {
      i, _ := strconv.ParseInt(v, 10, 0)
      vals = append(vals, i)
    }
  }

  for i, v := range keys {
    counts[v] = vals[i]
  }

  return counts
}

func Favicon(res http.ResponseWriter, req *http.Request) {
  res.Header().Add("Content-Type", "text/plain")
  res.Write([]byte(""))
}

func RootHandler(client *redis.Client, prefix string) (handle func(http.ResponseWriter, *http.Request)) {
  return func(res http.ResponseWriter, req *http.Request) {

    links := []string{
      fmt.Sprintf("http://%s%s/%s", req.Host, prefix, "track"),
      fmt.Sprintf("http://%s%s/%s", req.Host, prefix, "funnels"),
    }

    payload := &Payload{
      Links: links,
    }

    respond(req, res, payload)
  }
}

func AllKeysHandler(client *redis.Client, prefix string) (handle func(http.ResponseWriter, *http.Request)) {
  return func(res http.ResponseWriter, req *http.Request) {

    vars := mux.Vars(req)

    payload := &Payload{
      Type:    vars["chartType"],
      Account: vars["key"],
      Event:   vars["evt"],
    }

    splitPath := []string{vars["chartType"]}
    if vars["key"] != "" {
      splitPath = append(splitPath, vars["key"])
    }
    if vars["evt"] != "" {
      splitPath = append(splitPath, vars["evt"])
    }

    splitPath = append(splitPath, "*")

    redisPath := strings.Join(splitPath, ":")

    payload.Links = redisLinksLookup(client, req, redisPath, prefix)

    respond(req, res, payload)

  }
}

func KeyHandler(client *redis.Client, prefix string) (handle func(http.ResponseWriter, *http.Request)) {
  return func(res http.ResponseWriter, req *http.Request) {

    vars := mux.Vars(req)
    splitPath := []string{vars["chartType"], vars["key"]}

    payload := &Payload{
      Type:    vars["chartType"],
      Account: vars["key"],
      Event:   vars["evt"],
      Year:    vars["year"],
      Month:   vars["month"],
      Day:     vars["day"],
    }

    if vars["evt"] != "" {
      splitPath = append(splitPath, vars["evt"])
    }
    if vars["year"] != "" {
      splitPath = append(splitPath, vars["year"])
      // load links for year
    }
    if vars["month"] != "" {
      splitPath = append(splitPath, vars["month"])
      // load links for month
    }
    if vars["day"] != "" {
      splitPath = append(splitPath, vars["day"])
    }

    redisPath := strings.Join(splitPath, ":")

    payload.Data = redisIntHash(client, redisPath)

    keysPattern := fmt.Sprintf("%s:*", redisPath)
    payload.Links = redisLinksLookup(client, req, keysPattern, prefix)

    respond(req, res, payload)
  }

}

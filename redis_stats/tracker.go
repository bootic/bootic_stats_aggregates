package redis_stats

import (
  "fmt"
  data "github.com/bootic/bootic_go_data"
  "github.com/vmihailenco/redis"
  "log"
  "strconv"
  "time"
)

type Tracker struct {
  Conn     *redis.Client
  Notifier data.EventsChannel
  Funnels  data.EventsChannel
}

func (self *Tracker) TrackTime(accountStr, evtType string, now time.Time) {

  go func(key, evtType string, now time.Time) {

    defer func() {
      if err := recover(); err != nil {
        log.Println("Goroutine failed:", err)
      }
    }()

    yearAsString := strconv.Itoa(now.Year())
    monthAsString := strconv.Itoa(int(now.Month()))
    dayAsString := strconv.Itoa(now.Day())
    hourAsString := strconv.Itoa(now.Hour())

    // increment current month in year
    yearKey := fmt.Sprintf("track:%s:%s:%s", key, evtType, yearAsString)
    self.Conn.HIncrBy(yearKey, monthAsString, 1)

    // increment current day in month
    monthKey := fmt.Sprintf("track:%s:%s:%s:%s", key, evtType, yearAsString, monthAsString)
    self.Conn.HIncrBy(monthKey, dayAsString, 1)

    // increment current hour in day
    dayKey := fmt.Sprintf("track:%s:%s:%s:%s:%s", key, evtType, yearAsString, monthAsString, dayAsString)
    self.Conn.HIncrBy(dayKey, hourAsString, 1)

    // Expire day entry after a month
    // self.Conn.Expire(dayKey, 2592000)

  }(accountStr, evtType, now)
}

func (self *Tracker) TrackFunnel(accountStr, evtType, statusStr string, now time.Time) {

  go func(key, evtType, statusStr string, now time.Time) {

    defer func() {
      if err := recover(); err != nil {
        log.Println("Goroutine failed:", err)
      }
    }()

    yearAsString := strconv.Itoa(now.Year())
    monthAsString := strconv.Itoa(int(now.Month()))

    // increment current month in year
    yearKey := fmt.Sprintf("funnels:%s:%s:%s", key, evtType, yearAsString)
    self.Conn.HIncrBy(yearKey, statusStr, 1)

    // increment current day in month
    monthKey := fmt.Sprintf("funnels:%s:%s:%s:%s", key, evtType, yearAsString, monthAsString)
    self.Conn.HIncrBy(monthKey, statusStr, 1)

  }(accountStr, evtType, statusStr, now)
}

func getLocalTime(event *data.Event) time.Time {
  tzstr, e1 := event.Get("data").Get("tz").String()
  now := time.Now()
  if e1 != nil {
    return now
  }
  if tzstr == "" {
    return now
  }
  tzhours := fmt.Sprintf("%sh", tzstr)
  tzdur, e2 := time.ParseDuration(tzhours)
  if e2 != nil {
    return time.Now()
  }

  then := now.Add(tzdur)

  return then
}

func (self *Tracker) listenForPageviews() {
  for {
    event := <-self.Notifier
    evtType, _ := event.Get("type").String()
    evtAccount, _ := event.Get("data").Get("account").String()
    then := getLocalTime(event)

    self.TrackTime(evtAccount, evtType, then)
    self.TrackTime("all", evtType, then)

    unique, _ := event.Get("data").Get("unq").String()

    if unique == "1" {
      self.TrackTime(evtAccount, "unique", then)
      self.TrackTime("all", "unique", then)
    }
  }
}

func (self *Tracker) listenForFunnels() {
  for {
    event := <-self.Funnels
    evtType, _ := event.Get("type").String()
    evtAccount, _ := event.Get("data").Get("account").String()
    evtStatus, _ := event.Get("data").Get("status").String()

    then := getLocalTime(event)

    self.TrackFunnel(evtAccount, evtType, evtStatus, then)
    self.TrackFunnel("all", evtType, evtStatus, then)
  }
}

func NewTracker(redisAddress string) (tracker *Tracker, err error) {
  password := "" // no password set
  conn := redis.NewTCPClient(redisAddress, password, -1)

  defer conn.Close()

  tracker = &Tracker{
    Conn:     conn,
    Notifier: make(data.EventsChannel, 1),
    Funnels:  make(data.EventsChannel, 1),
  }

  go tracker.listenForPageviews()
  go tracker.listenForFunnels()

  return
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"

	auth "github.com/abbot/go-http-auth"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

const backlogDays = 8

type station struct {
	Data   measurementData `json:"data"`
	Config waterConfig     `json:"config"`

	mutex         sync.RWMutex
	whitelistNets []net.IPNet
	sht           *i2c.SHT3xDriver
	wuc           *Wuc
	serverConfig  `json:"-"`
}

type measurementData struct {
	Moisture    []int `json:"moisture"`
	Temperature []int `json:"temperature"`
	Humidity    []int `json:"humidity"`
	Watering    []int `json:"water"`
	Level       []int `json:"level"`
	Time        int   `json:"time"`
}

type waterConfig struct {
	WaterHour  int `json:"hour"`
	MinWater   int `json:"min"`
	MaxWater   int `json:"max"`
	LowLevel   int `json:"low"`
	DstLevel   int `json:"dst"`
	LevelRange int `json:"range"`
}

type loginConfig struct {
	User string
	Pass string
}

type httpsConfig struct {
	Addr string
	Cert string
	Key  string
}

type filesConfig struct {
	Watering string
	Data     string
}

type serverConfig struct {
	HTTPS httpsConfig
	Login loginConfig
	Files filesConfig
}

func main() {

	// waitForTimeSync()

	log.Print("start")

	var sconfFile string
	flag.StringVar(&sconfFile, "c", "server.conf", "server config file")
	flag.Parse()

	r := raspi.NewAdaptor()
	w, err := NewWuc(r)
	if err != nil {
		log.Fatalf("failed to create connection to microcontroller: %v", err)
	}

	s := station{
		serverConfig: serverConfig{
			Login: loginConfig{
				User: "user",
				Pass: "",
			},
			HTTPS: httpsConfig{
				Addr: ":443",
				Cert: "localhost.crt",
				Key:  "localhost.key",
			},
			Files: filesConfig{
				Watering: "/var/opt/plantstation/watering.conf",
				Data:     "/var/opt/plantstation/data.json",
			},
		},
		Config: waterConfig{
			WaterHour:  7,
			MinWater:   2000,
			MaxWater:   20000,
			LowLevel:   1400,
			DstLevel:   1500,
			LevelRange: 100,
		},
		sht: i2c.NewSHT3xDriver(r),
		wuc: w,
		Data: measurementData{
			Time:        time.Now().Hour(),
			Moisture:    make([]int, 0),
			Temperature: make([]int, 0),
			Humidity:    make([]int, 0),
			Watering:    make([]int, 0),
			Level:       make([]int, 0),
		},
	}

	s.parseServerConfigFile(sconfFile)
	s.parseWaterConfigFile()
	s.readData()

	err = s.sht.Start()
	if err != nil {
		log.Fatalf("failed to create connection to SHT31: %v", err)
	}

	authenticator := auth.NewBasicAuthenticator("plant", s.secret())

	// TODO: create own server instance and do graceful shutdown on signal
	// server := &http.Server{
	// 	Addr: s.serverConfig.HTTPS.Addr,
	// }

	http.Handle("/", http.FileServer(http.Dir("web")))
	http.HandleFunc("/calc", calcWateringHandler(&s))
	http.HandleFunc("/moist", moistureHandler(&s))
	http.HandleFunc("/level", waterLevelHandler(&s))
	http.HandleFunc("/ht", htHandler(&s))
	http.HandleFunc("/data", dataHandler(&s))
	http.HandleFunc("/config", auth.JustCheck(authenticator, configHandler(&s)))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go s.run()

	go func() {
		log.Fatal(http.ListenAndServeTLS(
			s.serverConfig.HTTPS.Addr,
			s.serverConfig.HTTPS.Cert,
			s.serverConfig.HTTPS.Key,
			nil))
	}()

	<-sigs
	log.Print("shutting down")

	s.saveData()
	s.mutex.Lock()
}

func (s *station) parseWaterConfigFile() {
	fw := s.serverConfig.Files.Watering
	b, err := ioutil.ReadFile(fw)
	if err != nil && os.IsNotExist(err) {
		log.Printf("watering config %s not found, using default", fw)
		return
	} else if err != nil {
		log.Fatalf("failed to read %s: %v", fw, err)
	}

	err = json.Unmarshal(b, &s.Config)
	if err != nil {
		log.Fatalf("failed to parse watering config: %v", err)
	}
}

func (s *station) parseServerConfigFile(serverConf string) {
	b, err := ioutil.ReadFile(serverConf)
	if err != nil {
		log.Fatalf("failed to read %s: %v", serverConf, err)
	}

	err = toml.Unmarshal(b, &s.serverConfig)
	if err != nil {
		log.Fatalf("failed to parse server config: %v", err)
	}
}

func (s *station) readData() {
	b, err := ioutil.ReadFile(s.serverConfig.Files.Data)
	if err != nil && os.IsNotExist(err) {
		log.Printf("no old measurement data found at %s",
			s.serverConfig.Files.Data)
		return
	} else if err != nil {
		log.Fatalf("failed to read measurement data to %s: %v",
			s.serverConfig.Files.Data, err)
	}

	err = json.Unmarshal(b, &s.Data)
	if err != nil {
		log.Fatalf("failed to marshal measurement data: %v", err)
	}
}

func (s *station) saveData() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	b, err := json.Marshal(s.Data)
	if err != nil {
		log.Fatalf("failed to marshal measurement data: %v", err)
	}

	err = ioutil.WriteFile(s.serverConfig.Files.Data, b, 0600)
	if err != nil {
		log.Fatalf("failed to save measurement data to %s: %v",
			s.serverConfig.Files.Data, err)
	}
}

func (s *station) run() {
	n := time.Now().Add(60 * time.Minute)
	timer := time.NewTimer(time.Until(time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), 0, 0, 0, n.Location())))

	tch := timer.C

	for {
		select {
		case <-tch:
			// get current hour
			h := time.Now().Add(30 * time.Minute).Hour()
			// next hour
			n := time.Now().Add(90 * time.Minute)
			log.Printf("update %v", h)
			s.update(h)
			// reset timer to next hour
			timer.Reset(time.Until(time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), 0, 0, 0, n.Location())))
		}
	}
}

func pushSlice(s []int, v int, maxLen int) []int {
	n := len(s) + 1
	if n > maxLen {
		copy(s, s[n-maxLen:])
		s = s[:maxLen-1]
	}
	return append(s, v)
}

func (s *station) calculateWatering(hour int, m int) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	lastw := (s.Config.MinWater + s.Config.MaxWater) / 2
	durw := 0

	if len(s.Data.Watering) > 0 {
		for i := len(s.Data.Watering) - 1; i >= 0; i-- {
			if s.Data.Watering[i] > 0 {
				lastw = s.Data.Watering[i]
				break
			}
			durw = len(s.Data.Watering) - i
		}
	}

	log.Printf("last watered %v hours ago", durw+1)

	sum := m
	for i := len(s.Data.Moisture) - durw; i < len(s.Data.Moisture); i++ {
		sum += s.Data.Moisture[i]
	}

	avg := sum / (durw + 1)

	log.Printf("average moisture since last watering: %v", avg)

	dl := float32(s.Config.DstLevel - avg)
	rl := float32(s.Config.LevelRange)
	rw := float32(s.Config.MaxWater - s.Config.MinWater)
	dw := dl / rl * rw

	log.Printf("adjusting watering time by %v", dw)

	wt := lastw + int(dw+0.5)
	return clamp(wt, s.Config.MinWater, s.Config.MaxWater)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func (s *station) update(hour int) {

	m, err := s.wuc.ReadMoisture()

	if err != nil {
		log.Printf("failed to read moisture: %v", err)

		// fallback to last read moisture value
		n := len(s.Data.Moisture)
		if n > 0 {
			m = s.Data.Moisture[n-1]
		}
	}

	t, h, err := s.sht.Sample()
	if err != nil {
		log.Printf("failed to read humidity and temperature: %v", err)
		// fallback to last read values
		n := len(s.Data.Humidity)
		if n > 0 {
			h = float32(s.Data.Humidity[n-1]) / 100
		}
		n = len(s.Data.Temperature)
		if n > 0 {
			t = float32(s.Data.Temperature[n-1]) / 100
		}
	}

	l, err := s.wuc.ReadWaterLevel()
	if err != nil {
		log.Printf("failed to read water level: %v", err)
		// fallback to last read value
		n := len(s.Data.Level)
		if n > 0 {
			l = s.Data.Level[n-1]
		}
	}

	// calculate watering time
	wt := 0
	if hour == s.Config.WaterHour && m <= s.Config.LowLevel {
		wt = s.calculateWatering(hour, m)
	}
	if wt > 0 {
		wt = s.wuc.DoWatering(wt)
	}

	// update values
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Data.Time = hour
	const maxHours = backlogDays * 24
	s.Data.Moisture = pushSlice(s.Data.Moisture, m, maxHours)
	s.Data.Humidity = pushSlice(s.Data.Humidity, int(h*100), maxHours)
	s.Data.Temperature = pushSlice(s.Data.Temperature, int(t*100), maxHours)
	s.Data.Watering = pushSlice(s.Data.Watering, wt, maxHours)
	s.Data.Level = pushSlice(s.Data.Level, l, maxHours)
}

func dataHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		s.mutex.RLock()
		defer s.mutex.RUnlock()

		js, err := json.Marshal(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func checkAuth(user, pass string) bool {
	return user == "user" && pass == "pass"
}

func configHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			s.saveConfig(w, r.Body)
		case http.MethodGet:
			s.sendConfig(w)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (s *station) saveConfig(w http.ResponseWriter, r io.Reader) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var c waterConfig
	err = json.Unmarshal(b, &c)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	err = ioutil.WriteFile(s.serverConfig.Files.Watering, b, 0600)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
	}

	s.Config = c
	fmt.Fprint(w, "config saved")
}

func (s *station) sendConfig(w http.ResponseWriter) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	js, err := json.Marshal(s.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func moistureHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := s.wuc.ReadMoisture()
		if err != nil {
			log.Println("failed to read soil moisture: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintf(w, "%v", m)
	}
}

func waterLevelHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := s.wuc.ReadWaterLevel()
		if err != nil {
			log.Println("failed to read water level: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintf(w, "%v", m)
	}
}

func htHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		t, h, err := s.sht.Sample()
		if err != nil {
			log.Println("failed to read humidity and temperature: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintf(w, "%v, %v", h, t)
	}
}

func calcWateringHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		m, err := s.wuc.ReadMoisture()
		if err != nil {
			log.Println("failed to read soil moisture: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}

		s.mutex.RLock()
		defer s.mutex.RUnlock()

		fmt.Fprintf(w, "%v", s.calculateWatering(time.Now().Hour()+1, m))
	}
}

func (s *station) secret() func(user, realm string) string {
	return func(user, realm string) string {
		if user == s.serverConfig.Login.User {
			return s.serverConfig.Login.Pass
		}
		return ""
	}
}

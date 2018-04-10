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
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"

	auth "github.com/abbot/go-http-auth"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

const backlogMinutes = 8 * 60
const backlogDays = 8

type station struct {
	Data    measurementData `json:"data"`
	MinData measurementData `json:"mindata"`
	Config  [2]waterConfig  `json:"config"`

	mutex         sync.RWMutex
	whitelistNets []net.IPNet
	sht           *i2c.SHT3xDriver
	wuc           *Wuc
	serverConfig  `json:"-"`
}

type measurementData struct {
	Weight      [2][]int `json:"weight"`
	Temperature []int    `json:"temperature"`
	Humidity    []int    `json:"humidity"`
	Watering    [2][]int `json:"water"`
	Time        int      `json:"time"`
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
		Config: [2]waterConfig{{
			WaterHour:  7,
			MinWater:   2000,
			MaxWater:   20000,
			LowLevel:   1400,
			DstLevel:   1500,
			LevelRange: 100,
		}, {
			WaterHour:  7,
			MinWater:   2000,
			MaxWater:   20000,
			LowLevel:   1400,
			DstLevel:   1500,
			LevelRange: 100,
		}},
		sht: i2c.NewSHT3xDriver(r),
		wuc: w,
		Data: measurementData{
			Time:        time.Now().Hour(),
			Weight:      [2][]int{make([]int, 0), make([]int, 0)},
			Temperature: make([]int, 0),
			Humidity:    make([]int, 0),
			Watering:    [2][]int{make([]int, 0), make([]int, 0)},
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
	http.HandleFunc("/water", auth.JustCheck(authenticator, wateringHandler(&s)))
	http.HandleFunc("/calc", calcWateringHandler(&s))
	http.HandleFunc("/weight", weightHandler(&s))
	http.HandleFunc("/limit", waterLimitHandler(&s))
	http.HandleFunc("/ht", htHandler(&s))
	http.HandleFunc("/data", dataHandler(&s))
	http.HandleFunc("/config", auth.JustCheck(authenticator, configHandler(&s)))
	http.HandleFunc("/echo", echoHandler(&s))

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

	nm := time.Now().Add(60 * time.Second)
	mintimer := time.NewTimer(time.Until(time.Date(nm.Year(), nm.Month(), nm.Day(), nm.Hour(), nm.Minute(), 0, 0, nm.Location())))

	tch := timer.C
	mtch := mintimer.C

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

		case <-mtch:
			// get current hour
			m := time.Now().Add(30 * time.Second).Minute()
			// next hour
			n := time.Now().Add(90 * time.Second)
			log.Printf("minute %v", m)
			s.updateMinute(m)
			mintimer.Reset(time.Until(time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), 0, 0, n.Location())))
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

func (s *station) calculateWatering(index int, hour int, weight int) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	lastw := (s.Config[index].MinWater + s.Config[index].MaxWater) / 2
	durw := 0

	if len(s.Data.Watering[index]) > 0 {
		for i := len(s.Data.Watering[index]) - 1; i >= 0; i-- {
			if s.Data.Watering[index][i] > 0 {
				lastw = s.Data.Watering[index][i]
				break
			}
			durw = len(s.Data.Watering[index]) - i
		}
	}

	log.Printf("last watered %v hours ago", durw+1)

	sum := weight
	for i := len(s.Data.Weight[index]) - durw; i < len(s.Data.Weight[index]); i++ {
		sum += s.Data.Weight[index][i]
	}

	avg := sum / (durw + 1)

	log.Printf("average weight since last watering: %v", avg)

	dl := float32(s.Config[index].DstLevel - avg)
	rl := float32(s.Config[index].LevelRange)
	rw := float32(s.Config[index].MaxWater - s.Config[index].MinWater)
	dw := dl / rl * rw

	log.Printf("adjusting watering time by %v", dw)

	wt := lastw + int(dw+0.5)
	return clamp(wt, s.Config[index].MinWater, s.Config[index].MaxWater)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	} else if v > max {
		return max
	}
	return v
}

func hourMedian(mindata []int) int {
	i0 := 0
	n := len(mindata)

	if n == 0 {
		panic(fmt.Errorf("empty slice"))
	}

	if n > 60 {
		i0 = n - 60
	}
	d := make([]int, n-i0)
	copy(d, mindata[i0:])
	sort.Ints(d)
	return d[len(d)/2]
}

func (s *station) update(hour int) {
	var err error
	w := [2]int{}

	if len(s.MinData.Weight[0]) == 0 || len(s.MinData.Weight[1]) == 0 {
		w[0], w[1], err = s.wuc.ReadWeights()
		if err != nil {
			log.Printf("failed to read weight: %v", err)

			// fallback to last read weight
			n := len(s.Data.Weight)
			if n > 0 {
				w[0] = s.Data.Weight[0][n-1]
				w[1] = s.Data.Weight[1][n-1]
			}
		}
	} else {
		for index := 0; index < 2; index++ {
			w[index] = hourMedian(s.MinData.Weight[index])
		}
	}

	var t, h int
	if len(s.MinData.Humidity) == 0 || len(s.MinData.Temperature) == 0 {
		tf, hf, err := s.sht.Sample()
		if err != nil {
			log.Printf("failed to read humidity and temperature: %v", err)
			// fallback to last read values
			n := len(s.Data.Humidity)
			if n > 0 {
				h = s.Data.Humidity[n-1]
			}
			n = len(s.Data.Temperature)
			if n > 0 {
				t = s.Data.Temperature[n-1]
			}
		} else {
			t = int(tf * 100)
			h = int(hf * 100)
		}
	} else {
		h = hourMedian(s.MinData.Humidity)
		t = hourMedian(s.MinData.Temperature)
	}

	// calculate watering time
	wt := [2]int{}
	for i := range wt {
		if hour == s.Config[i].WaterHour && w[i] <= s.Config[i].LowLevel {
			wt[i] = s.calculateWatering(i, hour, w[i])
		}
		if wt[i] > 0 {
			wt[i] = s.wuc.DoWatering(i, wt[i])
		}
	}

	// update values
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Data.Time = hour
	const maxHours = backlogDays * 24
	for i := range w {
		s.Data.Weight[i] = pushSlice(s.Data.Weight[i], w[i], maxHours)
		s.Data.Watering[i] = pushSlice(s.Data.Watering[i], wt[i], maxHours)
	}
	s.Data.Humidity = pushSlice(s.Data.Humidity, h, maxHours)
	s.Data.Temperature = pushSlice(s.Data.Temperature, t, maxHours)
}

func (s *station) updateMinute(min int) {
	var err error
	w := [2]int{}
	w[0], w[1], err = s.wuc.ReadWeights()
	if err != nil {
		log.Printf("failed to read weight: %v", err)
		// fallback to last read weight
		n := len(s.MinData.Weight)
		if n > 0 {
			w[0] = s.MinData.Weight[0][n-1]
			w[1] = s.MinData.Weight[1][n-1]
		}
	}

	t, h, err := s.sht.Sample()
	if err != nil {
		log.Printf("failed to read humidity and temperature: %v", err)
		// fallback to last read values
		n := len(s.MinData.Humidity)
		if n > 0 {
			h = float32(s.MinData.Humidity[n-1]) / 100
		}
		n = len(s.MinData.Temperature)
		if n > 0 {
			t = float32(s.MinData.Temperature[n-1]) / 100
		}
	}

	// update values
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.MinData.Time = min
	for i := range w {
		s.MinData.Weight[i] = pushSlice(s.MinData.Weight[i], w[i], backlogMinutes)
	}
	s.MinData.Humidity = pushSlice(s.MinData.Humidity, int(h*100), backlogMinutes)
	s.MinData.Temperature = pushSlice(s.MinData.Temperature, int(t*100), backlogMinutes)
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

func getRequestIndex(r *http.Request) int {
	var index int
	if indexStr, ok := r.URL.Query()["i"]; ok {
		index, _ = strconv.Atoi(indexStr[0])
	}

	if index != 1 {
		index = 0
	}

	return index
}

func checkAuth(user, pass string) bool {
	return user == "user" && pass == "pass"
}

func configHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		index := getRequestIndex(r)
		switch r.Method {
		case http.MethodPut:
			s.saveConfig(index, w, r.Body)
		case http.MethodGet:
			s.sendConfig(index, w)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (s *station) saveConfig(index int, w http.ResponseWriter, r io.Reader) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c := s.Config
	err = json.Unmarshal(b, &c[index])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	b, err = json.Marshal(c)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	err = ioutil.WriteFile(s.serverConfig.Files.Watering, b, 0600)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
	}

	s.Config[index] = c[index]
	fmt.Fprint(w, "config saved")
}

func (s *station) sendConfig(index int, w http.ResponseWriter) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	js, err := json.Marshal(s.Config[index])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func wateringHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		index := getRequestIndex(r)
		tq, ok := r.URL.Query()["t"]

		if !ok || len(tq) < 1 {
			t, err := s.wuc.ReadLastWatering(index)
			if err != nil {
				log.Println("failed to read last watering time: ", err)
			}
			fmt.Fprintf(w, "%v", t)
			return
		}
		t, err := strconv.Atoi(tq[0])
		if err != nil {
			fmt.Fprintf(w, "invalid parameter: %v", err)
			return
		}

		log.Printf("watering %v", t)
		t = s.wuc.DoWatering(index, t)
		log.Printf("watered %v", t)
		fmt.Fprintf(w, "%v", t)
	}
}

func weightHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w1, w2, err := s.wuc.ReadWeights()
		if err != nil {
			log.Println("failed to read weight: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprintf(w, "%v, %v", w1, w2)
	}
}

func waterLimitHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		index := getRequestIndex(r)
		m, err := s.wuc.ReadWateringLimit(index)
		if err != nil {
			log.Println("failed to read watering limit: ", err)
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
		index := getRequestIndex(r)
		var err error
		var we [2]int
		we[0], we[1], err = s.wuc.ReadWeights()
		if err != nil {
			log.Println("failed to read soil moisture: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
			return
		}

		s.mutex.RLock()
		defer s.mutex.RUnlock()

		fmt.Fprintf(w, "%v", s.calculateWatering(index, time.Now().Hour()+1, we[index]))
	}
}

func echoHandler(s *station) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d := r.URL.Query()["d"]

		var buf []byte

		for _, s := range d {
			i, err := strconv.Atoi(s)
			if err != nil {
				fmt.Fprintf(w, "invalid parameter %v: %v", s, err)
				return
			}
			buf = append(buf, byte(i))
		}

		fmt.Fprintf(w, "sending: %v\n", buf)

		buf, err := s.wuc.Echo(buf)
		if err != nil {
			fmt.Fprintf(w, "echo failed: %v", err)
			return
		}

		fmt.Fprintf(w, "%v", buf)
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

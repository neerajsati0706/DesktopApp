package main

import (
	c "Stefano/copy"
	gv "Stefano/gv"
	"Stefano/usbdrivedetector"
	"Stefano/websocket"
	socket "Stefano/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

var (
	destination         = ""
	LocalFolder360      = ""
	LocalFolderStandard = ""
	destination360      = ""
	destinationStandard = ""
	USBFolder360        = ""
	USBFolderStandard   = ""
)

type Detail struct {
	VideoType string `json:"video_type"`
	FromDate  string `json:"from_date"`
	EndDate   string `json:"end_date"`
}

type Download struct {
	Filename string `json:"filename"`
}

type VideoInfo struct {
	Path     string    `json:"video_link"`
	Filename string    `json:"video_name"`
	Filesize int64     `json:"filesize"`
	FileDate time.Time `json:"filedate"`
}

type Progress struct {
	Status     string
	Pending    int
	Percent    int
	DeviceName string
}

func getUSBDrivePath() ([]gv.ReadWrite, error) {
	var folders = make([]gv.ReadWrite, 0)

	if drives, err := usbdrivedetector.Detect(); err == nil {
		fmt.Printf("%d USB Devices Found\n", len(drives))
		notifyMessage := "" + strconv.Itoa(len(drives)) + "USB Devices Found\n"
		socket.Notify(notifyMessage)

		if gv.TotalUSB < len(drives) {
			gv.TotalUSB = len(drives)

			for i, d := range drives {
				ReadFolder360 := d + USBFolder360
				ReadFolderStandard := d + USBFolderStandard

				folders = append(folders, gv.ReadWrite{ReadFolder: ReadFolder360, WriteFolder: destination360})
				folders = append(folders, gv.ReadWrite{ReadFolder: ReadFolderStandard, WriteFolder: destinationStandard})
				USBFiles := []gv.ReadWrite{{ReadFolder: ReadFolder360, WriteFolder: destination360}, {ReadFolder: ReadFolderStandard, WriteFolder: destinationStandard}}
				gv.DeviceList = append(gv.DeviceList, gv.DeviceListStruct{i, USBFiles})
			}
		} else {
			return nil, errors.New("No new device")
		}
	}
	return folders, nil
}

func moveFiles(folders []gv.ReadWrite) {
	currentTime := time.Now()
	dateFolder := currentTime.Format("2006_01_02")
	for _, f := range folders {
		wildCardFolder, err := filepath.Glob(f.ReadFolder)

		if err != nil {
			fmt.Println(err)
		}
		if len(wildCardFolder) > 1 {
			actualFolder := wildCardFolder[0]
			actualDestinationFolder := f.WriteFolder + "/" + dateFolder
			c.CreateDir(actualDestinationFolder, 0755)
			err1 := c.CopyDirectory(actualFolder, actualDestinationFolder)
			if err1 == nil {

				fmt.Println("\n Successfully copied to " + actualFolder + " from:" + actualDestinationFolder)

			}
			if err1 != nil {
				socket.Notify("\n Error copied to " + actualFolder + " from:" + actualDestinationFolder)
				fmt.Println("\n Error copied to " + actualFolder + " from:" + actualDestinationFolder)
			}
		} else {
			socket.Notify("\n Error in reading data from USB")
			fmt.Println("PushStart", "Error", "\n Error in reading data from USB", "")
		}

	}
}

func copyData() {

	gv.CopyingData = true

	VideoFolders, err := getUSBDrivePath()
	gv.VideoFolders = VideoFolders
	if err == nil {
		moveFiles(gv.VideoFolders)
	}
	gv.CopyingData = false
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	destination = os.Getenv("DESTINATION")
	LocalFolder360 = os.Getenv("LOCALFOLDER_360")
	LocalFolderStandard = os.Getenv("LOCALFOLDER_STANDARD")
	destination360 = os.Getenv("DESTINATION") + LocalFolder360
	destinationStandard = os.Getenv("DESTINATION") + LocalFolderStandard
	USBFolder360 = os.Getenv("USBFOLDER_360")
	USBFolderStandard = os.Getenv("USBFOLDER_STANDARD")

	// go copyData()

	c := cron.New()
	c.AddFunc("* * * * *", RunEverySecond)
	RunEverySecond()
	go c.Start()

	startHTTPServer()
}

func RunEverySecond() {
	if !gv.CopyingData {
		copyData()
	}
}
func startHTTPServer() {
	http.HandleFunc("/api/videos", videoDownloadAPI)
	http.HandleFunc("/api/test", testFunction)
	http.HandleFunc("/api/scan", scanUSBAPI)

	http.HandleFunc("/api/download", DownloadFile)
	http.HandleFunc("/api/progress", getProgressValueAPI)

	websocket.Socket()
	panic(http.ListenAndServe(":8000", nil))
}
func testFunction(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Ping Pong!!!!"))
}
func scanUSBAPI(w http.ResponseWriter, r *http.Request) {
	go copyData()
	w.Write([]byte("Scanning started!!!!"))
}
func getProgressValueAPI(w http.ResponseWriter, r *http.Request) {
	var response = make([]Progress, 0)
	var progress Progress
	for i, device := range gv.DeviceList {
		var localCount int = 0
		var message string = ""
		for _, f := range device.Files {
			wildCardFolder, err := filepath.Glob(f.ReadFolder)
			if err != nil {
				fmt.Println(err)
			}
			actualFolder := wildCardFolder[0]
			localCount = localCount + c.GetTotalFiles(actualFolder)
			message = strconv.Itoa(localCount) + " files are pending "
		}
		deviceName := "Device " + strconv.Itoa(i+1)
		progress = Progress{message, localCount, 0, deviceName}
		response = append(response, progress)
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func PercentageChange(old, new int) (delta float64) {
	diff := float64(new - old)
	delta = (diff / float64(old)) * 100
	return
}

func videoDownloadAPI(w http.ResponseWriter, r *http.Request) {
	var details Detail

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("\nUnable to read body")
	}

	err = json.Unmarshal(body, &details)
	if err != nil {
		fmt.Println("Error in Unmarshalling data")
	}
	fmt.Println("\nUnmarshalled Data is", details)
	loc, _ := time.LoadLocation(os.Getenv("TIMEZONE"))
	fromDate, err := time.ParseInLocation("2006-01-02", details.FromDate, loc)
	if err != nil {
		fmt.Println("Unable to parse fromdate")
	}
	endDate, err := time.ParseInLocation("2006-01-02", details.EndDate, loc)
	//endDate = endDate.Add(time.Hour * 24 * 1)

	if err != nil {
		fmt.Println("Unable to parse enddate")
	}

	if fromDate.After(endDate) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var folderDates = make([]string, 0)

	for i := 0; i <= 90; i++ {
		date := fromDate.AddDate(0, 0, i)
		fmt.Println(date)
		datefolder := date.Format("2006_01_02")
		folderDates = append(folderDates, datefolder)
		if endDate == date {
			break
		}
	}
	var dstfd = ""
	if details.VideoType == "360" {
		dstfd = os.Getenv("DESTINATION") + os.Getenv("LOCALFOLDER_360")
	} else {
		dstfd = os.Getenv("DESTINATION") + os.Getenv("LOCALFOLDER_STANDARD")
	}

	var videoList = make([]VideoInfo, 0)
	for _, folders := range folderDates {
		folderLocation := dstfd + "/" + folders
		if _, err := os.Stat(folderLocation); os.IsNotExist(err) {
			fmt.Println(folderLocation + " path doesn't exist")
			continue
		}

		files, err := c.FilePathWalkDir(folderLocation)
		if err != nil {
			fmt.Println("Error in reading folder")
			panic(err)
		}

		for _, file := range files {

			v, err := os.Stat(file)
			if err != nil {
				fmt.Println("Unable to determine file attributes")
			}
			filepath := path.Join(path.Dir(file))
			//if v.ModTime().Unix() >= from && v.ModTime().Unix() <= end {
			videoList = append(videoList, VideoInfo{Path: filepath, Filename: v.Name(), Filesize: v.Size(), FileDate: v.ModTime()})
		}
	}
	js, err := json.Marshal(videoList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func DownloadFile(w http.ResponseWriter, r *http.Request) {

	var filename Download

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error in reading body")
	}

	err = json.Unmarshal(body, &filename)
	if err != nil {
		fmt.Println("Error in unmarshalling data")
	}

	fmt.Println("\nClient requests: ", filename.Filename)

	files, err := c.FilePathWalkDir(os.Getenv("DESTINATION"))
	if err != nil {
		panic(err)
	}

	for _, file := range files {

		v, err := os.Stat(file)
		if err != nil {
			fmt.Println("Unable to determine file attributes")
		}
		if v.Name() == filename.Filename {

			Openfile, err := os.Open(file)
			if err != nil {
				fmt.Println("Unable to open file")
			}
			defer Openfile.Close()
			n, err := io.Copy(w, Openfile) //'Copy' the file to the client
			if err == nil {
				fmt.Fprintf(w, "Succesfully downloaded")
				fmt.Println("Bytes transfered is", n)
			}

		}

	}

}

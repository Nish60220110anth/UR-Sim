package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"test_app/lib"
	"time"
)

var closeChan chan struct{}
var closeChanAll []chan struct{}

const UpTime int = 10
const NoOfThreads int = 5
const ServerDelayTime int = 1 // in seconds
const UpDelayTime int = 1     // in second

func init() {
	fmt.Println("Init function Called")

	closeChan = make(chan struct{}, 1)
	closeChanAll = make([]chan struct{}, NoOfThreads)

	for i := range closeChanAll {
		closeChanAll[i] = make(chan struct{}, 1)
	}
}

type WorkerData struct {
	GroundStationNum string
	HelmetNum        string
	GasLevel         float32
	Spo2Level        float32
	Temperature      float32
	HeartRate        int
}

func (wd *WorkerData) String() string {
	return fmt.Sprintf("GroundStationNum %v\nHelmetNum %v\nGasLevel "+
		"%v\nSpo2Level %v\nTemperature %v\n"+
		"HeartRate %v\n", wd.GroundStationNum, wd.HelmetNum, wd.GasLevel, wd.Spo2Level, wd.Temperature, wd.HeartRate)
}

func GenRandom() *WorkerData {
	return &WorkerData{
		GroundStationNum: lib.RandString(10),
		HelmetNum:        lib.RandString(10),
		GasLevel:         float32(math.Abs(float64(lib.RandFloat32()))),
		HeartRate:        lib.RandInt(),
		Temperature:      float32(math.Abs(float64(lib.RandFloat32()))),
		Spo2Level:        float32(math.Abs(float64(lib.RandFloat32()))),
	}
}

func GenAllThreads(logger *log.Logger, noOfThreads int, serverDelay int, conn []*net.TCPConn) {
	logger.Println("GenAllThreads Called")

	for i := 0; i < noOfThreads; i++ {
		go GenOneThread(logger, i, serverDelay, conn[i])
	}

outer:
	for {
		select {
		case <-closeChan:
			logger.Println("Close Main Channel Initiated")

			for i := 0; i < noOfThreads; i++ {
				closeChanAll[i] <- struct{}{}
			}

			logger.Println("Close Info informed to all threads")
			break outer
		default:

		}
		time.Sleep(time.Duration(500) * time.Millisecond)
	}

}

func GenOneThread(logger *log.Logger, threadIndex int, serveDelay int, conn *net.TCPConn) {
	logger.Printf("GenOneThread Index %d Called\n", threadIndex)

outer:
	for {
		select {
		case <-closeChanAll[threadIndex]:
			logger.Printf("Closing Thread Index %d\n", threadIndex)
			err := conn.Close()
			if err != nil {
				logger.Fatalln(err.Error())
			}
			break outer
		default:
			logger.Printf("Data Sent from %d\n", threadIndex)
			_, err := conn.Write([]byte(GenRandom().String()))
			if err != nil {
				logger.Fatalln(err.Error())
			}
			time.Sleep(time.Second * time.Duration(serveDelay))
		}
	}

}

func MakeOneConn(lAddr *net.TCPAddr, rAddr *net.TCPAddr, threadIndex int) *net.TCPConn {
	const NO_OF_TRIES int = 2

	for i := 0; i < NO_OF_TRIES; i++ {
		ltcp, _ := net.DialTCP("tcp", lAddr, rAddr)

		if ltcp != nil {
			return ltcp
		}

		time.Sleep(time.Duration(100) * time.Millisecond)
	}

	log.Fatalf("Unable to create connection for thread Index %d", threadIndex)
	return nil
}

func LoadConnections(lAddr *net.TCPAddr, rAddr *net.TCPAddr) []*net.TCPConn {
	var myTcpConn []*net.TCPConn = make([]*net.TCPConn, NoOfThreads)

	for i := 0; i < NoOfThreads; i++ {
		myTcpConn[i] = MakeOneConn(lAddr, rAddr, i)
	}

	return myTcpConn
}

func main() {
	logger := log.Default()
	logger.Println("Server Started")

	var laddrs net.TCPAddr
	var raddrs net.TCPAddr

	laddrs.IP = net.ParseIP("127.0.0.1")
	laddrs.Port = 6000

	raddrs.IP = net.ParseIP("127.0.0.1")
	raddrs.Port = 5000

	var tcpAllConn []*net.TCPConn
	tcpAllConn = LoadConnections(&laddrs, &raddrs)

	go GenAllThreads(logger, NoOfThreads, ServerDelayTime, tcpAllConn)
	time.Sleep(time.Duration(UpTime) * time.Second)
	closeChan <- struct{}{}
	log.Println("Server Closing Initiated")
	time.Sleep(time.Duration(UpDelayTime*2) * time.Second)
}
